# --- Settings ---
$baseUrl = "http://localhost:8080"
$protoPath = "./proto"
$accFile = "bank/account/v1/account.proto"
$ledgerFile = "bank/ledger/v1/ledger.proto"
$accAddr = "localhost:50056"
$ledgerAddr = "localhost:50057"

# Helper: Decode JWT
function Get-JwtClaims($token) {
    $parts = $token.Split('.')
    $payload = $parts[1].Replace('-', '+').Replace('_', '/')
    switch ($payload.Length % 4) {
        2 { $payload += '==' }
        3 { $payload += '=' }
    }
    return ConvertFrom-Json ([System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($payload)))
}

# Helper: Call gRPC
function Call-Grpc {
    param($addr, $file, $method, $jsonObj)
    $jsonStr = $jsonObj | ConvertTo-Json -Compress
    $jsonStr | grpcurl -plaintext -import-path $protoPath -proto $file -d "@" $addr $method
}

Write-Host "=== 1. SETUP (Auth & Profile) ===" -ForegroundColor Cyan
$email1 = "gateway_test_1_$((Get-Random))@test.com"
$email2 = "gateway_test_2_$((Get-Random))@test.com"
$pass = "Password123!"

# Регистрация
$regBody = @{ email = $email1; password = $pass } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/register" -ContentType "application/json" -Body $regBody | Out-Null
$regBody2 = @{ email = $email2; password = $pass } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/register" -ContentType "application/json" -Body $regBody2 | Out-Null

Write-Host "Waiting for Kafka (Customer Service)..."
Start-Sleep -Seconds 3 

# Логин
$loginBody = @{ email = $email1; password = $pass } | ConvertTo-Json
$token1 = (Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/login" -ContentType "application/json" -Body $loginBody).accessToken
$loginBody2 = @{ email = $email2; password = $pass } | ConvertTo-Json
$token2 = (Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/auth/login" -ContentType "application/json" -Body $loginBody2).accessToken

$cid1 = (Get-JwtClaims $token1).customer_id
$cid2 = (Get-JwtClaims $token2).customer_id

$h1 = @{ Authorization = "Bearer $token1" }
$h2 = @{ Authorization = "Bearer $token2" }

Write-Host "User 1 (CID: $cid1)" -ForegroundColor DarkGray
Write-Host "User 2 (CID: $cid2)" -ForegroundColor DarkGray

Write-Host "`n=== 2. SETUP (Accounts & Balance via gRPC) ===" -ForegroundColor Cyan
# Форсим статус ACTIVE (чтобы не проходить KYC)
docker exec -i mini-core-postgres psql -U admin -d customer_db -c "UPDATE customers SET status = 'ACTIVE' WHERE id = '$cid1';" | Out-Null
docker exec -i mini-core-postgres psql -U admin -d customer_db -c "UPDATE customers SET status = 'ACTIVE' WHERE id = '$cid2';" | Out-Null

# Создаем счета
$openAcc1 = @{ customer_id = $cid1; currency = "GEL"; account_type = "ACCOUNT_TYPE_CHECKING" }
$acc1Json = Call-Grpc $accAddr $accFile "bank.account.v1.AccountService/OpenAccount" $openAcc1
$accId1 = ($acc1Json | ConvertFrom-Json).accountId

$openAcc2 = @{ customer_id = $cid2; currency = "GEL"; account_type = "ACCOUNT_TYPE_CHECKING" }
$acc2Json = Call-Grpc $accAddr $accFile "bank.account.v1.AccountService/OpenAccount" $openAcc2
$accId2 = ($acc2Json | ConvertFrom-Json).accountId

# Пополняем счет первого пользователя на 100 000
$depositObj = @{
    account_id = $accId1
    amount = @{ amount = 100000; currency = "GEL" }
    idempotency_key = [guid]::NewGuid().ToString()
}
Call-Grpc $ledgerAddr $ledgerFile "bank.ledger.v1.LedgerService/Deposit" $depositObj | Out-Null

Write-Host "Account 1 ID: $accId1 (Balance: 100000)" -ForegroundColor Green
Write-Host "Account 2 ID: $accId2 (Balance: 0)" -ForegroundColor Green

Write-Host "`n=== 3. TESTING GATEWAY (HTTP) ===" -ForegroundColor Cyan

# 3.1. Получаем список своих счетов
Write-Host "`n3.1. GET /v1/accounts (User 1)" -ForegroundColor Yellow
$myAccs = Invoke-RestMethod -Method Get -Uri "$baseUrl/v1/accounts" -Headers $h1
Write-Host "Found accounts: $($myAccs.accounts.Count)" -ForegroundColor Gray

# 3.2. Получаем свой баланс
Write-Host "`n3.2. GET /v1/accounts/$accId1/balance (User 1)" -ForegroundColor Yellow
$bal1 = Invoke-RestMethod -Method Get -Uri "$baseUrl/v1/accounts/$accId1/balance" -Headers $h1
Write-Host "Balance: $($bal1.balance.amount) $($bal1.balance.currency)" -ForegroundColor Green

# 3.3. IDOR ТЕСТ: Пытаемся прочитать чужой баланс
Write-Host "`n3.3. IDOR TEST: GET /v1/accounts/$accId1/balance (User 2)" -ForegroundColor Yellow
try {
    Invoke-RestMethod -Method Get -Uri "$baseUrl/v1/accounts/$accId1/balance" -Headers $h2
    Write-Host "FAIL: User 2 was able to read User 1's balance!" -ForegroundColor Red
} catch {
    $err = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "SUCCESS: Access denied! Code: $($err.code), Message: $($err.message)" -ForegroundColor Green
}

# 3.4. Перевод денег через Gateway
Write-Host "`n3.4. POST /v1/transfers (User 1 -> Account 2)" -ForegroundColor Yellow
$transferBody = @{
    from_account_id = $accId1
    to_account_id = $accId2
    amount = @{ amount = 25000; currency = "GEL" }
    idempotency_key = [guid]::NewGuid().ToString()
} | ConvertTo-Json

$tx = Invoke-RestMethod -Method Post -Uri "$baseUrl/v1/transfers" -Headers $h1 -ContentType "application/json" -Body $transferBody
Write-Host "Transfer successful! Tx ID: $($tx.transactionId)" -ForegroundColor Green

# 3.5. Проверяем финальные балансы
Write-Host "`n3.5. Final Balances" -ForegroundColor Yellow
$finalBal1 = Invoke-RestMethod -Method Get -Uri "$baseUrl/v1/accounts/$accId1/balance" -Headers $h1
$finalBal2 = Invoke-RestMethod -Method Get -Uri "$baseUrl/v1/accounts/$accId2/balance" -Headers $h2
Write-Host "User 1 Balance: $($finalBal1.balance.amount) (Expect 75000)" -ForegroundColor Green
Write-Host "User 2 Balance: $($finalBal2.balance.amount) (Expect 25000)" -ForegroundColor Green

Write-Host "`n=== TESTING COMPLETED ===" -ForegroundColor Cyan
Read-Host "Press Enter to exit"