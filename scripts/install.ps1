# Deck automated 1-line installer for Windows PowerShell
$ErrorActionPreference = "Stop"

$repo = "Dolyyyy/deck"
$binary = "deck.exe"

Write-Host "⚡ Installing deck on Windows..." -ForegroundColor Cyan

$arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $arch = "arm64"
}

$installDir = Join-Path $HOME ".deck\bin"
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

$destPath = Join-Path $installDir $binary

try {
    $latest = (Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -UseBasicParsing).tag_name
} catch {
    $latest = "v0.1.0"
}

$cleanVer = $latest.TrimStart("v")
$zipName = "deck_${cleanVer}_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/$latest/$zipName"

$tempZip = Join-Path $env:TEMP $zipName

Write-Host "⬇️  Downloading $zipName from $url..." -ForegroundColor Yellow
try {
    Invoke-WebRequest -Uri $url -OutFile $tempZip -UseBasicParsing
    Expand-Archive -Path $tempZip -DestinationPath $installDir -Force
    Remove-Item -Path $tempZip -Force
} catch {
    Write-Host "⚠️  Pre-built release not found, trying go install..." -ForegroundColor Yellow
    if (Get-Command go -ErrorAction SilentlyContinue) {
        go install "github.com/$repo/cmd/deck@latest"
        Write-Host "✓ Successfully installed via go install!" -ForegroundColor Green
        exit 0
    } else {
        Write-Error "Failed to install pre-built binary and Go compiler is not available."
        exit 1
    }
}

# Ensure installDir is in user PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    $env:Path = "$env:Path;$installDir"
    Write-Host "✓ Added $installDir to User PATH." -ForegroundColor Green
}

Write-Host "🎉 Successfully installed deck to $destPath!" -ForegroundColor Green
Write-Host "👉 Run 'deck' to open the cockpit, or 'deck --help' for commands." -ForegroundColor Cyan
