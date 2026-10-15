$modules = @("./pkg", "./services/auth", "./services/customer")

foreach ($module in $modules) {
    Write-Host "==> $module (integration)" -ForegroundColor Cyan
    Push-Location $module
    go test -tags=integration ./...
    $exitCode = $LASTEXITCODE
    Pop-Location
    if ($exitCode -ne 0) { exit $exitCode }
}
Write-Host "Integration tests passed" -ForegroundColor Green