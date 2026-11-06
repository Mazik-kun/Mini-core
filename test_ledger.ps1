# --- Settings ---
$protoPath = "./proto"
$accFile = "bank/account/v1/account.proto"
$ledgerFile = "bank/ledger/v1/ledger.proto"
$accAddr = "localhost:50056"
$ledgerAddr = "localhost:50057"

# Helper functions to avoid PowerShell quoting issues
function Call-Account {
    param($method, $jsonData)
    $jsonData | grpcurl -plaintext -import-path $protoPath -proto $accFile -d "@" $accAddr "bank.account.v1.AccountService/$method"
}
function Call-Ledger {
    param($method, $jsonData)
    $jsonData | grpcurl -plaintext -import-path $protoPath -proto $ledgerFile -d "@" $ledgerAddr "bank.ledger.v1.LedgerService/$method"
}

Write-Host "=== TESTING LEDGER SERVICE ===" -ForegroundColor Cyan

try {
    # 1. Создаем два счета
    Write-Host "1. Creating two accounts..." -ForegroundColor Yellow
    $cust1 = [guid]::NewGuid().ToString()
    $cust2 = [guid]::NewGuid().ToString()

    $acc1Raw = Call-Account -method "OpenAccount" -jsonData ('{"customer_id":"' + $cust1 + '","currency":"GEL","account_type":"ACCOUNT_TYPE_CHECKING"}')
    $acc1 = ($acc1Raw | ConvertFrom-Json).accountId

    $acc2Raw = Call-Account -method "OpenAccount" -jsonData ('{"customer_id":"' + $cust2 + '","currency":"GEL","account_type":"ACCOUNT_TYPE_CHECKING"}')
    $acc2 = ($acc2Raw | ConvertFrom-Json).accountId

    Write-Host "Account 1 ID: $acc1" -ForegroundColor DarkGray
    Write-Host "Account 2 ID: $acc2" -ForegroundColor DarkGray

    # 2. Депозит на первый счет (100 000)
    Write-Host "`n2. Deposit 100000 to Account 1..." -ForegroundColor Yellow
    $depositKey = [guid]::NewGuid().ToString()
    $depositJson = '{"account_id":"' + $acc1 + '","amount":{"amount":100000,"currency":"GEL"},"idempotency_key":"' + $depositKey + '"}'
    $depositRaw = Call-Ledger -method "Deposit" -jsonData $depositJson
    Write-Host $depositRaw -ForegroundColor Gray

    # 3. Проверяем баланс 1 (должно быть 100 000)
    Write-Host "`n3. Checking Balance of Account 1 (Expect 100000)..." -ForegroundColor Yellow
    $bal1Raw = Call-Ledger -method "GetBalance" -jsonData ('{"account_id":"' + $acc1 + '"}')
    Write-Host $bal1Raw -ForegroundColor Gray

    # 4. Перевод 25 000 на второй счет
    Write-Host "`n4. Transfer 25000 from Account 1 to Account 2..." -ForegroundColor Yellow
    $transferKey = [guid]::NewGuid().ToString()
    $transferJson = '{"from_account_id":"' + $acc1 + '","to_account_id":"' + $acc2 + '","amount":{"amount":25000,"currency":"GEL"},"idempotency_key":"' + $transferKey + '"}'
    $transferRaw = Call-Ledger -method "Transfer" -jsonData $transferJson
    Write-Host $transferRaw -ForegroundColor Gray

    # 5. Проверяем баланс 1 (должно стать 75 000)
    Write-Host "`n5. Checking Balance of Account 1 (Expect 75000)..." -ForegroundColor Yellow
    $bal1AfterRaw = Call-Ledger -method "GetBalance" -jsonData ('{"account_id":"' + $acc1 + '"}')
    Write-Host $bal1AfterRaw -ForegroundColor Gray

    # 6. Проверяем баланс 2 (должно стать 25 000)
    Write-Host "`n6. Checking Balance of Account 2 (Expect 25000)..." -ForegroundColor Yellow
    $bal2AfterRaw = Call-Ledger -method "GetBalance" -jsonData ('{"account_id":"' + $acc2 + '"}')
    Write-Host $bal2AfterRaw -ForegroundColor Gray

    # 7. Проверка идемпотентности (повторяем перевод с тем же ключом)
    Write-Host "`n7. Testing Idempotency (repeating transfer with same key)..." -ForegroundColor Yellow
    $transferRepeatRaw = Call-Ledger -method "Transfer" -jsonData $transferJson
    Write-Host "Response should be identical, balance should NOT change." -ForegroundColor DarkGray
    Write-Host $transferRepeatRaw -ForegroundColor Gray

    # 8. Финальная проверка первого баланса
    Write-Host "`n8. Final Check Balance of Account 1 (Still 75000)..." -ForegroundColor Yellow
    $bal1FinalRaw = Call-Ledger -method "GetBalance" -jsonData ('{"account_id":"' + $acc1 + '"}')
    Write-Host $bal1FinalRaw -ForegroundColor Gray

    Write-Host "`n=== TESTING COMPLETED ===" -ForegroundColor Green
} catch {
    Write-Host "`n=== ERROR ===" -ForegroundColor Red
    Write-Host $_ -ForegroundColor Red
}

Write-Host "`nPress Enter to exit..."
Read-Host