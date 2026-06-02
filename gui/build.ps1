# gline GUI build script
# Usage: .\build.ps1 [dev|build]
param([string]$Action = "build")

switch ($Action.ToLower()) {
    "dev"     { wails3 dev -config .\build\config.yml }
    "build"   { wails3 generate bindings; wails3 build }
    default   { Write-Host "Usage: .\build.ps1 [dev|build]" }
}
