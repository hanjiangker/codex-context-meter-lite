$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$icon = Join-Path $repoRoot "assets\codex-context-meter-lite.ico"
$resource = Join-Path $repoRoot "cmd\meter\rsrc_windows_amd64.syso"

Push-Location $repoRoot
try {
    & go run .\cmd\icon-generator -output $icon
    if ($LASTEXITCODE -ne 0) {
        throw "Icon generator failed with exit code $LASTEXITCODE"
    }

    & go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico $icon -o $resource
    if ($LASTEXITCODE -ne 0) {
        throw "rsrc failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}

Write-Host "Generated $icon"
Write-Host "Generated $resource with github.com/akavel/rsrc@v0.10.2"
