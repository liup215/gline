# gline Build Script
# Builds both CLI (gline.exe) and GUI (gline-gui.exe) binaries.
# Usage: powershell -ExecutionPolicy Bypass -File build-all.ps1

$ErrorActionPreference = "Stop"

$Output    = "bin\gline.exe"
$OutputGUI = "bin\gline-gui.exe"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  gline Build (CLI + GUI)"                 -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# ===========================================================================
# 1. Generate Wails Bindings (always, never skip)
# ===========================================================================
Write-Host "`n[1/6] Generating Wails bindings..." -ForegroundColor Yellow
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
Write-Host "`n[2/6] Building frontend..." -ForegroundColor Yellow
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
# 3. Sync frontend build output to Go embed directories
# ===========================================================================
Write-Host "`n[3/6] Syncing frontend to embed paths..." -ForegroundColor Yellow
$src = "$PSScriptRoot\frontend\dist"

foreach ($dstRel in @("cmd\gline\frontend\dist", "cmd\gline-gui\frontend\dist")) {
    $dst = "$PSScriptRoot\$dstRel"
    if (Test-Path $dst) {
        Remove-Item -Recurse -Force "$dst\*" -ErrorAction SilentlyContinue
    } else {
        New-Item -ItemType Directory -Path $dst -Force | Out-Null
    }
    Copy-Item -Recurse -Force "$src\*" $dst
    Write-Host "  -> $dstRel" -ForegroundColor Gray
}

# Copy icon for the GUI entry point (needs its own build/appicon.png)
$iconSrc = "$PSScriptRoot\cmd\gline\build\appicon.png"
$iconDst = "$PSScriptRoot\cmd\gline-gui\build\appicon.png"
New-Item -ItemType Directory -Path (Split-Path $iconDst) -Force | Out-Null
Copy-Item -Force $iconSrc $iconDst
Write-Host "  -> cmd/gline-gui/build/appicon.png" -ForegroundColor Gray

# ===========================================================================
# 4. Build CLI binary
# ===========================================================================
Write-Host "`n[4/6] Building CLI binary -> $Output ..." -ForegroundColor Yellow

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
    Write-Error "CLI build failed!"
    exit 1
}

# ===========================================================================
# 5. Build GUI binary (standalone, no console window)
# ===========================================================================
Write-Host "`n[5/6] Building GUI binary -> $OutputGUI ..." -ForegroundColor Yellow

$ldflagsGUI = "-X github.com/liup215/gline/internal/version.Version=$version " +
              "-X github.com/liup215/gline/internal/version.Commit=$commit " +
              "-X github.com/liup215/gline/internal/version.BuildTime=$buildTime " +
              "-s -w -H=windowsgui"

cmd /c "go build -tags gui -ldflags `"$ldflagsGUI`" -o $OutputGUI ./cmd/gline-gui"
if ($LASTEXITCODE -ne 0) {
    Write-Error "GUI build failed!"
    exit 1
}

# ===========================================================================
# 6. Verify both binaries
# ===========================================================================
Write-Host "`n[6/6] Verifying binaries..." -ForegroundColor Yellow
$cliBin  = Get-Item $Output    -ErrorAction SilentlyContinue
$guiBin  = Get-Item $OutputGUI -ErrorAction SilentlyContinue
if (-not $cliBin) { Write-Error "CLI binary not found!"; exit 1 }
if (-not $guiBin) { Write-Error "GUI binary not found!"; exit 1 }

Write-Host "`n==========================================" -ForegroundColor Green
Write-Host "  Build Success!"                                          -ForegroundColor Green
Write-Host "  CLI  : $($cliBin.FullName)  ($([math]::Round($cliBin.Length/1KB,1)) KB)" -ForegroundColor Green
Write-Host "  GUI  : $($guiBin.FullName)  ($([math]::Round($guiBin.Length/1KB,1)) KB)" -ForegroundColor Green
Write-Host "  Version: $version ($commit)"                             -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
