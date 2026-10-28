# =====================================================================
# E2E TEST VIA API GATEWAY (REST)
# =====================================================================

# 0. Settings
$baseUrl = "http://localhost:8080"
$rand = Get-Random
$clientEmail = "client_$rand@test.com"
$pass = "Password123!"

# Decode JWT function
function Get-JwtClaims($token) {
    $parts = $token.Split('.')
    $payload = $parts[1].Replace('-', '+').Replace('_', '/')
    switch ($payload.Length % 4) {
        2 { $payload += '==' }
        3 { $payload += '=' }
    }
    return ConvertFrom-Json ([System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($payload)))
}

Write-Host "`n=== STEP 1: Prepare KYC Officer ===" -ForegroundColor Cyan
# 1. Login as admin
$adminBody = @{ email = "admin@bank.com"; password = "changeme123" } | ConvertTo-Json
$adminResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/login" -ContentType "application/json" -Body $adminBody
$adminToken = $adminResp.accessToken
Write-Host "Admin logged in." -ForegroundColor Green

# 2. Create officer
$officerEmail = "officer_$rand@test.com"
$offBody = @{ email = $officerEmail; password = $pass } | ConvertTo-Json
$headers = @{ Authorization = "Bearer $adminToken" }
try {
    Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/officers" -Headers $headers -ContentType "application/json" -Body $offBody | Out-Null
    Write-Host "Officer $officerEmail created." -ForegroundColor Green
} catch {
    Write-Host "Error creating officer: $($_.Exception.Message)" -ForegroundColor Red
    exit
}

# 3. Login as officer
$offLoginBody = @{ email = $officerEmail; password = $pass } | ConvertTo-Json
$offLoginResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/login" -ContentType "application/json" -Body $offLoginBody
$officerToken = $offLoginResp.accessToken
Write-Host "Officer token received." -ForegroundColor Green

Write-Host "`n=== STEP 2: Register & Login Client ===" -ForegroundColor Cyan
# 1. Register client
$cliBody = @{ email = $clientEmail; password = $pass } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/register" -ContentType "application/json" -Body $cliBody | Out-Null
Write-Host "Client $clientEmail registered. Waiting for Kafka (3 sec)..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

# 2. Login client
$cliLoginResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/login" -ContentType "application/json" -Body $cliBody
$clientToken = $cliLoginResp.accessToken
$clientHeaders = @{ Authorization = "Bearer $clientToken" }

$claims = Get-JwtClaims $clientToken
$customerId = $claims.customer_id
if (-not $customerId) {
    Write-Host "ERROR: No customer_id in token! Kafka failed." -ForegroundColor Red
    exit
}
Write-Host "Customer ID received: $customerId" -ForegroundColor Green

Write-Host "`n=== STEP 3: Update Profile ===" -ForegroundColor Cyan
$profileBody = @{
    fullName = "Ivan Ivanov"
    dateOfBirth = "1995-05-15T00:00:00Z"
    address = "Tbilisi"
    phone = "+995555000000"
    citizenship = "GE"
} | ConvertTo-Json
$updateResp = Invoke-RestMethod -Method Patch -Uri "$baseUrl/v1/customers/me" -Headers $clientHeaders -ContentType "application/json" -Body $profileBody
Write-Host "Profile status: $($updateResp.status)" -ForegroundColor Green

Write-Host "`n=== STEP 4: Upload Documents ===" -ForegroundColor Cyan
# Generate valid PNG (1x1 pixel)
$base64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="
[IO.File]::WriteAllBytes("$PWD\test.png", [Convert]::FromBase64String($base64))

# Function to upload a single document
function Upload-Document($docType) {
    Write-Host "Uploading $docType..." -ForegroundColor Gray
    
    # Init Upload
    $initBody = @{ documentType = $docType; filename = "file.png"; contentType = "image/png"; sizeBytes = 1000 } | ConvertTo-Json
    $initResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/documents/init-upload" -Headers $clientHeaders -ContentType "application/json" -Body $initBody
    
    # PUT to MinIO
    curl.exe -s -X PUT -T "test.png" -H "Content-Type: image/png" $initResp.uploadUrl | Out-Null
    
    # Confirm
    $confirmBody = @{ documentId = $initResp.documentId } | ConvertTo-Json
    $confirmResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/documents/confirm-upload" -Headers $clientHeaders -ContentType "application/json" -Body $confirmBody
    
    return $initResp.documentId
}

$docId1 = Upload-Document "DOCUMENT_TYPE_PASSPORT"
Write-Host "Passport confirmed (ID: $docId1)" -ForegroundColor Green

$docId2 = Upload-Document "DOCUMENT_TYPE_SELFIE"
Write-Host "Selfie confirmed (ID: $docId2)" -ForegroundColor Green

Write-Host "`n=== STEP 5: Submit KYC Application ===" -ForegroundColor Cyan
$submitBody = @{ documentIds = @($docId1, $docId2) } | ConvertTo-Json
$submitResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/kyc/applications" -Headers $clientHeaders -ContentType "application/json" -Body $submitBody
$kycId = $submitResp.applicationId
Write-Host "Application created. ID: $kycId. Status: $($submitResp.status)" -ForegroundColor Green
Write-Host "Waiting for auto-check worker (6 sec)..." -ForegroundColor Yellow
Start-Sleep -Seconds 6

Write-Host "`n=== STEP 6: Officer Workflow ===" -ForegroundColor Cyan
$officerHeaders = @{ Authorization = "Bearer $officerToken" }

# 1. Claim
$claimResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/kyc/applications/$kycId/claim" -Headers $officerHeaders -ContentType "application/json" -Body "{}"
Write-Host "Officer claimed application. Status: $($claimResp.status)" -ForegroundColor Green

# 2. Approve
$approveResp = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/kyc/applications/$kycId/approve" -Headers $officerHeaders -ContentType "application/json" -Body "{}"
Write-Host "Application approved! Status: $($approveResp.status)" -ForegroundColor Green

Write-Host "`n=== STEP 7: Check Client Status ===" -ForegroundColor Cyan
Start-Sleep -Seconds 2 # Wait for Kafka (kyc.application_approved -> customer)
$statusResp = Invoke-RestMethod -Method Get -Uri "$baseUrl/v1/customers/me" -Headers $clientHeaders
Write-Host "Final client status: $($statusResp.status)" -ForegroundColor Magenta

if ($statusResp.status -eq "CUSTOMER_STATUS_ACTIVE") {
    Write-Host "`nSUCCESS! Full E2E scenario via API Gateway works perfectly!" -ForegroundColor Green
} else {
    Write-Host "`nWARNING: Status is not ACTIVE. Check customer-service logs." -ForegroundColor Red
}