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
echo        CupBot Service Installation
echo ===============================================
echo.

set SERVICE_NAME=CupBot
set SERVICE_DISPLAY_NAME=CupBot Telegram Bot
set SERVICE_DESCRIPTION=Telegram bot for remote computer management
set CURRENT_DIR=%~dp0
set EXE_PATH=%CURRENT_DIR%cupbot.exe

:: Check if executable exists
if not exist "%EXE_PATH%" (
    echo ERROR: cupbot.exe not found in current directory
    echo Please run build.bat first to build the application
    pause
    exit /b 1
)

echo [1/4] Checking if service already exists...
sc query "%SERVICE_NAME%" >nul 2>&1
if not errorlevel 1 (
    echo Service already exists. Stopping and removing...
    net stop "%SERVICE_NAME%" >nul 2>&1
    sc delete "%SERVICE_NAME%" >nul 2>&1
    timeout /t 2 >nul
)

echo [2/4] Creating service...
sc create "%SERVICE_NAME%" binPath= "\"%EXE_PATH%\" -service" DisplayName= "%SERVICE_DISPLAY_NAME%" start= auto
if errorlevel 1 (
    echo ERROR: Failed to create service
    pause
    exit /b 1
)

echo [3/4] Configuring service...
sc description "%SERVICE_NAME%" "%SERVICE_DESCRIPTION%"
sc config "%SERVICE_NAME%" start= auto
sc failure "%SERVICE_NAME%" reset= 60 actions= restart/30000/restart/30000/restart/30000

echo [4/4] Starting service...
net start "%SERVICE_NAME%"
if errorlevel 1 (
    echo WARNING: Service created but failed to start
    echo Please check configuration and start manually
) else (
    echo Service started successfully!
)

echo.
echo ===============================================
echo Service installation completed!
echo.
echo Service Name: %SERVICE_NAME%
echo Service Status: 
sc query "%SERVICE_NAME%" | find "STATE"
echo.
echo Management commands:
echo   Start:   net start "%SERVICE_NAME%"
echo   Stop:    net stop "%SERVICE_NAME%"
echo   Remove:  uninstall-service.bat
echo ===============================================
pause