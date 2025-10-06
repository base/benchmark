# PowerShell wrapper for building op-rbuilder on Windows
param(
    [string]$RbuilderRepo = "https://github.com/haardikk21/op-rbuilder",
    [string]$RbuilderVersion = "a8bb38693ece585e7fa98d52f51290e7dcececff",
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

Write-Host "Building op-rbuilder binary..." -ForegroundColor Green
Write-Host "Repository: $RbuilderRepo"
Write-Host "Version/Commit: $RbuilderVersion"
Write-Host "Build directory: $BuildDir"
Write-Host "Output directory: $OutputDir"

# Check if Rust/Cargo is installed
try {
    $cargoVersion = cargo --version
    Write-Host "Found Cargo: $cargoVersion" -ForegroundColor Green
} catch {
    Write-Error "Cargo not found. Please install Rust from https://rustup.rs/"
    exit 1
}

# Create build directory if it doesn't exist
if (!(Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir -Force
}

Set-Location $BuildDir

# Clone or update repository
if (Test-Path "op-rbuilder") {
    Write-Host "Updating existing op-rbuilder repository..." -ForegroundColor Yellow
    Set-Location "op-rbuilder"
    git fetch origin
} else {
    Write-Host "Cloning op-rbuilder repository..." -ForegroundColor Yellow
    git clone $RbuilderRepo "op-rbuilder"
    Set-Location "op-rbuilder"
}

# Checkout specified version/commit
Write-Host "Checking out version: $RbuilderVersion" -ForegroundColor Yellow
git checkout $RbuilderVersion

# Build the binary using cargo
Write-Host "Building op-rbuilder with cargo..." -ForegroundColor Yellow
cargo build --release

# Copy binary to output directory
Write-Host "Copying binary to output directory..." -ForegroundColor Yellow
if (!(Test-Path "../../$OutputDir")) {
    New-Item -ItemType Directory -Path "../../$OutputDir" -Force
}

# Find the built binary and copy it
$binaryPaths = @("target/release/op-rbuilder.exe", "target/release/rbuilder.exe")
$found = $false

foreach ($path in $binaryPaths) {
    if (Test-Path $path) {
        Copy-Item $path "../../$OutputDir/op-rbuilder.exe"
        $found = $true
        break
    }
}

if (!$found) {
    # Search for rbuilder binary
    $rbuilderBinary = Get-ChildItem -Path "target/release" -Name "*rbuilder*.exe" | Select-Object -First 1
    if ($rbuilderBinary) {
        Copy-Item "target/release/$rbuilderBinary" "../../$OutputDir/op-rbuilder.exe"
        $found = $true
    }
}

if ($found) {
    Write-Host "op-rbuilder binary built successfully and placed in $OutputDir/op-rbuilder.exe" -ForegroundColor Green
} else {
    Write-Error "Could not find rbuilder binary after build"
    exit 1
}