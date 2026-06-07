# gline 统一构建脚本 (CLI + GUI)
# 先构建前端，再同步嵌入目录，最后编译 Go 应用

# gline 统一构建脚本 (CLI + GUI)
# PowerShell fallback for Windows (no make required)
# Usage: .\build-all.ps1 [-Dev] [-SkipBindings] [-Output bin\gline.exe]

param(
    [string]$Output = "bin\gline.exe",
    [switch]$Dev,          # 开发模式：不压缩前端
    [switch]$SkipBindings # 跳过 bindings 生成（如果已有）
)

$ErrorActionPreference = "Stop"

Write-Host "═══════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  gline Unified Build (CLI + GUI)" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════" -n

# ───────────────────────────────────────────
# 1. 生成 Wails Bindings
# ───────────────────────────────────────────
Write-Host "`n[1/5] Generating Wails bindings..." -ForegroundColor Yellow
if (-not (Get-Command wails3 -ErrorAction SilentlyContinue)) {
    Write-Error "wails3 CLI not found. Install with: go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
    exit 1
}
if (-not $SkipBindings) {
    Set-Location -Path "$PSScriptRoot\cmd\gline"
    wails3 generate bindings --ts -d "..\..\frontend\bindings" 2>&1 | ForEach-Object { Write-Host $_ }
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Bindings generation reported issues, but frontend may still build."
    }
    Set-Location -Path $PSScriptRoot
} else {
    Write-Host "  Skipped (--SkipBindings)" -ForegroundColor Gray
}

# ───────────────────────────────────────────
# 2. 构建前端
# ───────────────────────────────────────────
Write-Host "`n[2/5] Building frontend..." -ForegroundColor Yellow
Set-Location -Path "$PSScriptRoot\frontend"

if (-not (Test-Path "node_modules")) {
    Write-Host "  Installing dependencies first..." -ForegroundColor Gray
    npm install 2>&1 | ForEach-Object { Write-Host $_ }
}

if ($Dev) {
    npm run build:dev 2>&1 | ForEach-Object { Write-Host $_ }
} else {
    npm run build 2>&1 | ForEach-Object { Write-Host $_ }
}

if ($LASTEXITCODE -ne 0) {
    Write-Error "Frontend build failed!"
    exit 1
}
Set-Location -Path $PSScriptRoot

# ───────────────────────────────────────────
# 3. 同步到 Go embed 目录
# ───────────────────────────────────────────
Write-Host "`n[3/5] Syncing frontend to embed path..." -ForegroundColor Yellow
$src = "$PSScriptRoot\frontend\dist"
$dst = "$PSScriptRoot\cmd\gline\frontend\dist"

if (-not (Test-Path $dst)) {
    New-Item -ItemType Directory -Path $dst | Out-Null
} else {
    Remove-Item -Recurse -Force "$dst\*" -ErrorAction SilentlyContinue
}
Copy-Item -Recurse -Force "$src\*" $dst
Write-Host "  Copied to cmd/gline/frontend/dist" -ForegroundColor Gray

# ───────────────────────────────────────────
# 4. 编译 Go 应用
# ───────────────────────────────────────────
Write-Host "`n[4/5] Building Go binary -> $Output ..." -ForegroundColor Yellow

$version = git describe --tags --always --dirty 2>$null
if (-not $version) { $version = "dev" }
$commit  = git rev-parse --short HEAD 2>$null
if (-not $commit)  { $commit = "unknown" }
$buildTime = (Get-Date -Format "yyyy-MM-dd_HH:mm:ss").ToString()

$ldBase = "-X github.com/liup215/gline/internal/version.Version=$version " +
          "-X github.com/liup215/gline/internal/version.Commit=$commit " +
          "-X github.com/liup215/gline/internal/version.BuildTime=$buildTime" +
          " -s -w"

$ldflags = if ($IsWindows -or ($env:OS -eq "Windows_NT")) {
    "$ldBase -H=windowsgui"
} else {
    $ldBase
}

go build -ldflags "$ldflags" -o $Output ./cmd/gline 2>&1 | ForEach-Object { Write-Host $_ }

if ($LASTEXITCODE -ne 0) {
    Write-Error "Go build failed!"
    exit 1
}

# ───────────────────────────────────────────
# 5. 验证
# ───────────────────────────────────────────
Write-Host "`n[5/5] Verifying binary..." -ForegroundColor Yellow
$binary = Get-Item $Output -ErrorAction SilentlyContinue
if (-not $binary) {
    Write-Error "Binary not found!"
    exit 1
}

Write-Host "`n═══════════════════════════════════════════" -ForegroundColor Green
Write-Host "  Build Success!" -ForegroundColor Green
Write-Host "  Binary : $($binary.FullName)" -ForegroundColor Green
Write-Host "  Size   : $([math]::Round($binary.Length/1KB,1)) KB" -ForegroundColor Green
Write-Host "  Version: $version ($commit)" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════" -ForegroundColor Green
