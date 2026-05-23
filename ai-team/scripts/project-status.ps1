# project-status.ps1
# Текущее состояние проекта: git, сервисы, миграции, тесты.
# Использование: .\ai-team\scripts\project-status.ps1

Write-Host ""
Write-Host "=== Clinic Scheduler — Project Status ===" -ForegroundColor Cyan
Write-Host ""

# --- Git ---
Write-Host "Git:" -ForegroundColor Yellow

$branch = git rev-parse --abbrev-ref HEAD 2>&1
Write-Host "  Branch:  $branch"

$lastCommit = git log --oneline -1 2>&1
Write-Host "  Last:    $lastCommit"

$status = git status --short 2>&1
if ($status) {
    Write-Host "  Changes:" -ForegroundColor DarkYellow
    $status | ForEach-Object { Write-Host "    $_" -ForegroundColor DarkYellow }
} else {
    Write-Host "  Working tree: clean" -ForegroundColor Green
}

$ahead = git rev-list --count @{u}..HEAD 2>&1
if ($LASTEXITCODE -eq 0 -and $ahead -ne "0") {
    Write-Host "  Ahead of remote: $ahead commit(s)" -ForegroundColor DarkYellow
}

Write-Host ""

# --- Docker services ---
Write-Host "Docker services:" -ForegroundColor Yellow

$dockerRunning = docker compose ps 2>&1
if ($LASTEXITCODE -eq 0) {
    $lines = $dockerRunning -split "`n" | Where-Object { $_ -match '\S' }
    foreach ($line in $lines) {
        if ($line -match "running|healthy") {
            Write-Host "  $line" -ForegroundColor Green
        } elseif ($line -match "exited|unhealthy|dead") {
            Write-Host "  $line" -ForegroundColor Red
        } else {
            Write-Host "  $line"
        }
    }
} else {
    Write-Host "  Docker Compose not running or not available" -ForegroundColor DarkGray
}

Write-Host ""

# --- Backend health ---
Write-Host "Backend health:" -ForegroundColor Yellow

try {
    $response = Invoke-WebRequest -Uri "http://localhost:8000/health" -TimeoutSec 3 -ErrorAction Stop
    Write-Host "  GET /health → $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "  GET /health → unreachable" -ForegroundColor DarkGray
}

Write-Host ""

# --- Migrations ---
Write-Host "Migrations:" -ForegroundColor Yellow

$migrationFiles = Get-ChildItem -Path "backend/migrations" -Filter "*.sql" 2>$null | Sort-Object Name
if ($migrationFiles) {
    Write-Host "  Files: $($migrationFiles.Count)"
    $migrationFiles | ForEach-Object { Write-Host "    $($_.Name)" }
} else {
    Write-Host "  No migration files found" -ForegroundColor DarkGray
}

Write-Host ""

# --- Go modules ---
Write-Host "Go modules:" -ForegroundColor Yellow

$backendMod = Get-Content "backend/go.mod" -First 3 2>$null
if ($backendMod) {
    $goVer = $backendMod | Select-String "^go "
    Write-Host "  backend: $goVer"
} else {
    Write-Host "  backend/go.mod: not found" -ForegroundColor DarkGray
}

$botMod = Get-Content "bot/go.mod" -First 3 2>$null
if ($botMod) {
    $goVerBot = $botMod | Select-String "^go "
    Write-Host "  bot:     $goVerBot"
} else {
    Write-Host "  bot/go.mod: not found" -ForegroundColor DarkGray
}

Write-Host ""

# --- Frontend ---
Write-Host "Frontend:" -ForegroundColor Yellow

$pkgJson = Get-Content "frontend/package.json" 2>$null | ConvertFrom-Json
if ($pkgJson) {
    Write-Host "  React: $($pkgJson.dependencies.react)"
    Write-Host "  Vite:  $($pkgJson.devDependencies.vite)"
} else {
    Write-Host "  frontend/package.json: not found" -ForegroundColor DarkGray
}

Write-Host ""

# --- Recent commits ---
Write-Host "Recent commits:" -ForegroundColor Yellow
git log --oneline -8 2>&1 | ForEach-Object { Write-Host "  $_" }

Write-Host ""

# --- Roadmap hint ---
Write-Host "Roadmap:" -ForegroundColor Yellow
if (Test-Path "docs/ROADMAP.md") {
    $roadmap = Get-Content "docs/ROADMAP.md" | Select-String "^\- \[[ x]\]" | Select-Object -First 10
    $roadmap | ForEach-Object {
        $line = $_.Line
        if ($line -match "\[x\]") {
            Write-Host "  $line" -ForegroundColor Green
        } else {
            Write-Host "  $line" -ForegroundColor DarkYellow
        }
    }
} else {
    Write-Host "  docs/ROADMAP.md not found" -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "=== End of Status ===" -ForegroundColor Cyan
Write-Host ""
