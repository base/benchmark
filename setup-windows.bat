@echo off
echo Setting up Base Benchmark on Windows...

REM Check if Git is installed
git --version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Git is not installed or not in PATH
    echo Please install Git from https://git-scm.com/
    pause
    exit /b 1
)

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/
    pause
    exit /b 1
)

REM Check if Node.js is installed
node --version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Node.js is not installed or not in PATH
    echo Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

REM Initialize Git submodules
echo Initializing Git submodules...
git submodule update --init --recursive
if errorlevel 1 (
    echo ERROR: Failed to initialize Git submodules
    pause
    exit /b 1
)

REM Update Go dependencies
echo Updating Go dependencies...
go mod tidy
if errorlevel 1 (
    echo ERROR: Failed to update Go dependencies
    pause
    exit /b 1
)

REM Build the main application
echo Building main application...
go build -v -o bin\base-bench.exe .\benchmark\cmd
if errorlevel 1 (
    echo ERROR: Failed to build main application
    pause
    exit /b 1
)

REM Install frontend dependencies
echo Installing frontend dependencies...
cd report
npm install --legacy-peer-deps
if errorlevel 1 (
    echo ERROR: Failed to install frontend dependencies
    cd ..
    pause
    exit /b 1
)
cd ..

echo.
echo ✅ Setup completed successfully!
echo.
echo Next steps:
echo 1. Build client binaries: make build-binaries
echo 2. Run a benchmark: .\bin\base-bench.exe run --config .\configs\public\basic.yml --root-dir .\data-dir --output-dir .\output
echo 3. View results: cd report && npm run dev
echo.
pause