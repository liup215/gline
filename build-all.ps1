# gline Unified Build Script (CLI + GUI)
# Fixed, full build. No parameters. No options. No skips.
# Usage: powershell -ExecutionPolicy Bypass -File build-all.ps1

$ErrorActionPreference = "Stop"

$Output = "bin\gline.exe"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  gline Unified Build (CLI + GUI)"       -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# ===========================================================================
# 1. Generate Wails Bindings (always, never skip)
# ===========================================================================
Write-Host "`n[1/5] Generating Wails bindings..." -ForegroundColor Yellow
if (-not (Get-Command wails3 -ErrorAction SilentlyContinue)) {
    Write-Error "wails3 CLI not found. Install with: go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
    exit 1
}

Push-Location "$PSScriptRoot\cmd\gline"
    cmd /c "wails3 generate bindings --ts -d ..\..\frontend\bindings"
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Bindings generation reported issues, but continuing..."
    }
Pop-Location

# ===========================================================================
# 2. Build frontend (production, always)
# ===========================================================================
Write-Host "`n[2/5] Building frontend..." -ForegroundColor Yellow
Push-Location "$PSScriptRoot\frontend"
    if (-not (Test-Path "node_modules")) {
        Write-Host "  Installing dependencies..." -ForegroundColor Gray
        cmd /c "npm install"
        if ($LASTEXITCODE -ne 0) { Write-Error "npm install failed!"; exit 1 }
    }

    cmd /c "npm run build"
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Frontend build failed!"
        exit 1
    }
Pop-Location

# ===========================================================================
# 3. Sync to Go embed directory
# ===========================================================================
Write-Host "`n[3/5] Syncing frontend to embed path..." -ForegroundColor Yellow
$src = "$PSScriptRoot\frontend\dist"
$dst = "$PSScriptRoot\cmd\gline\frontend\dist"

if (Test-Path $dst) {
    Remove-Item -Recurse -Force "$dst\*" -ErrorAction SilentlyContinue
} else {
    New-Item -ItemType Directory -Path $dst | Out-Null
}
Copy-Item -Recurse -Force "$src\*" $dst
Write-Host "  Copied to cmd/gline/frontend/dist" -ForegroundColor Gray

# ===========================================================================
# 4. Build Go binary
# ===========================================================================
Write-Host "`n[4/5] Building Go binary -> $Output ..." -ForegroundColor Yellow

$version = git describe --tags --always --dirty 2>$null
if (-not $version) { $version = "dev" }
$commit  = git rev-parse --short HEAD 2>$null
if (-not $commit)  { $commit = "unknown" }
$buildTime = (Get-Date -Format "yyyy-MM-dd_HH:mm:ss").ToString()

$ldflags = "-X github.com/liup215/gline/internal/version.Version=$version " +
           "-X github.com/liup215/gline/internal/version.Commit=$commit " +
           "-X github.com/liup215/gline/internal/version.BuildTime=$buildTime " +
           "-s -w -H=windowsgui"

cmd /c "go build -ldflags `"$ldflags`" -o $Output ./cmd/gline"
if ($LASTEXITCODE -ne 0) {
    Write-Error "Go build failed!"
    exit 1
}

# ===========================================================================
# 5. Verify
# ===========================================================================
Write-Host "`n[5/5] Verifying binary..." -ForegroundColor Yellow
$binary = Get-Item $Output -ErrorAction SilentlyContinue
if (-not $binary) {
    Write-Error "Binary not found!"
    exit 1
}

Write-Host "`n==========================================" -ForegroundColor Green
Write-Host "  Build Success!"                         -ForegroundColor Green
Write-Host "  Binary : $($binary.FullName)"           -ForegroundColor Green
Write-Host "  Size   : $([math]::Round($binary.Length/1KB,1)) KB" -ForegroundColor Green
Write-Host "  Version: $version ($commit)"            -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
