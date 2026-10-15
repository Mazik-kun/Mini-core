$modules = @("./pkg", "./contracts", "./services/auth", "./services/customer")

foreach ($module in $modules) {
    Write-Host "==> $module (unit)" -ForegroundColor Cyan
    Push-Location $module
    go test -short ./...
    $exitCode = $LASTEXITCODE
    Pop-Location
    if ($exitCode -ne 0) { exit $exitCode }
}
Write-Host "Unit tests passed" -ForegroundColor Green