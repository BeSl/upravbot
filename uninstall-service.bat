@echo off
setlocal enabledelayedexpansion

:: Check for administrator privileges
net session >nul 2>&1
if errorlevel 1 (
    echo ERROR: This script must be run as Administrator
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

echo ===============================================
echo       CupBot Service Uninstallation
echo ===============================================
echo.

set SERVICE_NAME=CupBot

echo [1/3] Checking service status...
sc query "%SERVICE_NAME%" >nul 2>&1
if errorlevel 1 (
    echo Service "%SERVICE_NAME%" is not installed.
    pause
    exit /b 0
)

echo [2/3] Stopping service...
net stop "%SERVICE_NAME%" >nul 2>&1
if errorlevel 1 (
    echo Service was already stopped or failed to stop
) else (
    echo Service stopped successfully
)

echo [3/3] Removing service...
sc delete "%SERVICE_NAME%"
if errorlevel 1 (
    echo ERROR: Failed to remove service
    pause
    exit /b 1
) else (
    echo Service removed successfully
)

echo.
echo ===============================================
echo Service uninstallation completed!
echo ===============================================
pause