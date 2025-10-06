# PowerShell wrapper for building op-geth on Windows
param(
    [string]$GethRepo = "https://github.com/ethereum-optimism/op-geth/",
    [string]$GethVersion = "optimism",
    [string]$BuildDir = "./build",
    [string]$OutputDir = "../bin"
)

# Source versions if available
if (Test-Path "versions.env") {
    Get-Content "versions.env" | ForEach-Object {
        if ($_ -match "^([^=]+)=(.*)$") {
            Set-Variable -Name $matches[1] -Value $matches[2]
        }
    }
}

Write-Host "Building op-geth binary..." -ForegroundColor Green
Write-Host "Repository: $GethRepo"
Write-Host "Version/Commit: $GethVersion"
Write-Host "Build directory: $BuildDir"
Write-Host "Output directory: $OutputDir"

# Create build directory if it doesn't exist
if (!(Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir -Force
}

Set-Location $BuildDir

# Clone or update repository
if (Test-Path "op-geth") {
    Write-Host "Updating existing op-geth repository..." -ForegroundColor Yellow
    Set-Location "op-geth"
    git fetch origin
} else {
    Write-Host "Cloning op-geth repository..." -ForegroundColor Yellow
    git clone $GethRepo "op-geth"
    Set-Location "op-geth"
}

# Checkout specified version/commit
Write-Host "Checking out version: $GethVersion" -ForegroundColor Yellow
git checkout $GethVersion

# Build the binary using Go
Write-Host "Building op-geth with Go..." -ForegroundColor Yellow
go run build/ci.go install -static ./cmd/geth

# Copy binary to output directory
Write-Host "Copying binary to output directory..." -ForegroundColor Yellow
if (!(Test-Path "../../$OutputDir")) {
    New-Item -ItemType Directory -Path "../../$OutputDir" -Force
}

# Find and copy the binary
$binaryPaths = @("build/bin/geth.exe", "bin/geth.exe", "geth.exe")
$found = $false

foreach ($path in $binaryPaths) {
    if (Test-Path $path) {
        Copy-Item $path "../../$OutputDir/geth.exe"
        $found = $true
        break
    }
}

if (!$found) {
    # Search for geth binary
    $gethBinary = Get-ChildItem -Recurse -Name "*geth*.exe" | Select-Object -First 1
    if ($gethBinary) {
        Copy-Item $gethBinary "../../$OutputDir/geth.exe"
        $found = $true
    }
}

if ($found) {
    Write-Host "op-geth binary built successfully and placed in $OutputDir/geth.exe" -ForegroundColor Green
} else {
    Write-Error "Could not find geth binary after build"
    exit 1
}