$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$dist = Join-Path $repoRoot "dist"
$stage = Join-Path $repoRoot "build\portable"

New-Item -ItemType Directory -Force $dist | Out-Null
New-Item -ItemType Directory -Force $stage | Out-Null

$exe = Join-Path $stage "codex-context-meter-lite.exe"
go build -trimpath -ldflags "-s -w -H=windowsgui" -o $exe ./cmd/meter

Copy-Item (Join-Path $repoRoot "README.md") $stage -Force
Copy-Item (Join-Path $repoRoot "LICENSE") $stage -Force
Copy-Item (Join-Path $repoRoot "THIRD_PARTY_NOTICES.md") $stage -Force
$stageImages = Join-Path $stage "docs\images"
New-Item -ItemType Directory -Force $stageImages | Out-Null
Copy-Item (Join-Path $repoRoot "docs\images\*.png") $stageImages -Force

$zip = Join-Path $dist "codex-context-meter-lite-windows-amd64-v0.1.0.zip"
if (Test-Path $zip) {
    throw "Release archive already exists: $zip"
}
$releaseFiles = @(
    "codex-context-meter-lite.exe",
    "README.md",
    "LICENSE",
    "THIRD_PARTY_NOTICES.md",
    "docs"
)
Push-Location $stage
try {
    Compress-Archive -Path $releaseFiles -DestinationPath $zip -CompressionLevel Optimal
}
finally {
    Pop-Location
}

$size = (Get-Item $exe).Length
if ($size -gt 15MB) {
    throw "Executable exceeds 15 MB: $size bytes"
}

Write-Output "EXE: $exe"
Write-Output "ZIP: $zip"
Write-Output "EXE bytes: $size"
