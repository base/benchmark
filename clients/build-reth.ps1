# PowerShell wrapper for building reth on Windows
param(
    [string]$RethRepo = "https://github.com/paradigmxyz/reth/",
    [string]$RethVersion = "main",
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

Write-Host "Building reth binary..." -ForegroundColor Green
Write-Host "Repository: $RethRepo"
Write-Host "Version/Commit: $RethVersion"
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
if (Test-Path "reth") {
    Write-Host "Updating existing reth repository..." -ForegroundColor Yellow
    Set-Location "reth"
    git fetch origin
} else {
    Write-Host "Cloning reth repository..." -ForegroundColor Yellow
    git clone $RethRepo "reth"
    Set-Location "reth"
}

# Checkout specified version/commit
Write-Host "Checking out version: $RethVersion" -ForegroundColor Yellow
git checkout $RethVersion

# Build the binary using cargo
Write-Host "Building reth with cargo..." -ForegroundColor Yellow
cargo build --bin op-reth --profile maxperf --manifest-path crates/optimism/bin/Cargo.toml

# Copy binary to output directory
Write-Host "Copying binary to output directory..." -ForegroundColor Yellow
if (!(Test-Path "../../$OutputDir")) {
    New-Item -ItemType Directory -Path "../../$OutputDir" -Force
}

$binaryPath = "target/maxperf/op-reth.exe"
if (Test-Path $binaryPath) {
    Copy-Item $binaryPath "../../$OutputDir/reth.exe"
    Write-Host "reth binary built successfully and placed in $OutputDir/reth.exe" -ForegroundColor Green
} else {
    Write-Error "Could not find reth binary at $binaryPath"
    exit 1
}