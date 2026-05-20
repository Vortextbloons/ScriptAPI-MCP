$dest = "$env:USERPROFILE\.local\bin\script-api-helper.exe"

Write-Host "Building script-api-helper.exe..."
go build -o script-api-helper.exe ./cmd/script-api-helper
if (-not $?) { Write-Host "Build FAILED"; exit 1 }

Write-Host "Stopping any running script-api-helper processes..."
Get-Process -Name "script-api-helper" -ErrorAction SilentlyContinue | Stop-Process -Force

Write-Host "Deploying to $dest..."
Copy-Item -LiteralPath ".\script-api-helper.exe" -Destination $dest -Force
if (-not $?) { Write-Host "Deploy FAILED"; exit 1 }

Write-Host "Done. Built and deployed script-api-helper.exe"
