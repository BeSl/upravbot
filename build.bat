@echo off
setlocal enabledelayedexpansion

echo ===============================================
echo           CupBot Build Script
echo ===============================================
echo.

:: Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

echo [1/4] Checking Go installation...
go version
echo.

echo [2/4] Installing dependencies...
go mod tidy
if errorlevel 1 (
    echo ERROR: Failed to install dependencies
    pause
    exit /b 1
)
echo Dependencies installed successfully.
echo.

echo [3/4] Building CupBot...
go build -ldflags="-s -w" -o cupbot.exe .
if errorlevel 1 (
    echo ERROR: Build failed
    pause
    exit /b 1
)
echo Build completed successfully.
echo.

echo [4/4] Verifying build...
if exist cupbot.exe (
    echo ✓ cupbot.exe created successfully
    for %%A in (cupbot.exe) do echo   Size: %%~zA bytes
) else (
    echo ✗ Build failed - cupbot.exe not found
    pause
    exit /b 1
)
echo.

echo ===============================================
echo Build completed successfully!
echo.
echo Next steps:
echo 1. Configure your bot token and admin IDs
echo 2. Run install-service.bat as Administrator to install as service
echo 3. Or run cupbot.exe directly for testing
echo ===============================================
pause