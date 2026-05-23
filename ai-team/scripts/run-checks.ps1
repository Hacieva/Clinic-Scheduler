# run-checks.ps1
# Запуск всех проверок перед коммитом.
# Использование: .\ai-team\scripts\run-checks.ps1

$ErrorActionPreference = "Continue"
$passed = 0
$failed = 0

function Write-Check {
    param($label, $status, $detail = "")
    if ($status) {
        Write-Host "  [PASS] $label" -ForegroundColor Green
        $script:passed++
    } else {
        Write-Host "  [FAIL] $label" -ForegroundColor Red
        if ($detail) { Write-Host "         $detail" -ForegroundColor DarkRed }
        $script:failed++
    }
}

Write-Host ""
Write-Host "=== Clinic Scheduler — Pre-commit Checks ===" -ForegroundColor Cyan
Write-Host ""

# --- Backend ---
Write-Host "Backend (Go):" -ForegroundColor Yellow

# go build
$buildOut = & powershell -Command { $env:GO111MODULE='on'; go build ./... 2>&1 } 2>&1
Write-Check "go build ./..." ($LASTEXITCODE -eq 0) ($buildOut | Out-String)

# go vet
$vetOut = & powershell -Command { $env:GO111MODULE='on'; go vet ./... 2>&1 } 2>&1
Write-Check "go vet ./..." ($LASTEXITCODE -eq 0) ($vetOut | Out-String)

# go test
$testOut = & powershell -Command { $env:GO111MODULE='on'; go test ./internal/... 2>&1 } 2>&1
Write-Check "go test ./internal/..." ($LASTEXITCODE -eq 0) ($testOut | Select-String "FAIL" | Out-String)

Write-Host ""

# --- Frontend ---
Write-Host "Frontend (Node):" -ForegroundColor Yellow

$frontendPath = "frontend"
if (Test-Path $frontendPath) {
    Push-Location $frontendPath

    $buildOut = npm run build 2>&1
    Write-Check "npm run build" ($LASTEXITCODE -eq 0) ($buildOut | Select-String "error" | Out-String)

    $lintOut = npm run lint 2>&1
    Write-Check "npm run lint" ($LASTEXITCODE -eq 0) ($lintOut | Select-String "error" | Out-String)

    Pop-Location
} else {
    Write-Host "  [SKIP] frontend/ not found" -ForegroundColor DarkGray
}

Write-Host ""

# --- Bot ---
Write-Host "Bot (Go):" -ForegroundColor Yellow

$botPath = "bot"
if (Test-Path $botPath) {
    Push-Location $botPath

    $botBuild = & powershell -Command { $env:GO111MODULE='on'; go build ./... 2>&1 } 2>&1
    Write-Check "go build ./... (bot)" ($LASTEXITCODE -eq 0) ($botBuild | Out-String)

    $botVet = & powershell -Command { $env:GO111MODULE='on'; go vet ./... 2>&1 } 2>&1
    Write-Check "go vet ./... (bot)" ($LASTEXITCODE -eq 0) ($botVet | Out-String)

    Pop-Location
} else {
    Write-Host "  [SKIP] bot/ not found" -ForegroundColor DarkGray
}

Write-Host ""

# --- Secrets scan ---
Write-Host "Secrets scan:" -ForegroundColor Yellow

$secretPatterns = @(
    'password\s*=\s*"[^"]{4,}"',
    'secret\s*=\s*"[^"]{4,}"',
    'token\s*=\s*"[^"]{4,}"',
    'api_key\s*=\s*"[^"]{4,}"'
)

$secretFound = $false
foreach ($pattern in $secretPatterns) {
    $matches = Get-ChildItem -Recurse -Include "*.go","*.js","*.jsx","*.ts","*.tsx" `
        -Exclude "node_modules","dist",".git" |
        Select-String -Pattern $pattern -CaseSensitive:$false
    if ($matches) {
        $secretFound = $true
        Write-Host "  Potential secret: $($matches[0].Filename):$($matches[0].LineNumber)" -ForegroundColor DarkRed
    }
}
Write-Check "No hardcoded secrets" (-not $secretFound)

Write-Host ""

# --- Summary ---
$total = $passed + $failed
Write-Host "=== Results: $passed/$total passed ===" -ForegroundColor Cyan

if ($failed -eq 0) {
    Write-Host "All checks passed. Ready to commit." -ForegroundColor Green
    exit 0
} else {
    Write-Host "$failed check(s) failed. Fix before committing." -ForegroundColor Red
    exit 1
}
