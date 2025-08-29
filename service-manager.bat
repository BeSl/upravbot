@echo off
setlocal enabledelayedexpansion

echo ===============================================
echo          CupBot Service Manager
echo ===============================================
echo.

set SERVICE_NAME=CupBot

:: Check for administrator privileges
net session >nul 2>&1
if errorlevel 1 (
    echo ERROR: This script must be run as Administrator
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

:MENU
cls
echo ===============================================
echo          CupBot Service Manager
echo ===============================================
echo.

:: Check service status
sc query "%SERVICE_NAME%" >nul 2>&1
if errorlevel 1 (
    echo Service Status: NOT INSTALLED
    echo.
    echo Available options:
    echo [1] Install Service
    echo [2] Build Project
    echo [3] Exit
) else (
    echo Service Status:
    sc query "%SERVICE_NAME%" | find "STATE"
    echo.
    echo Available options:
    echo [1] Start Service
    echo [2] Stop Service
    echo [3] Restart Service
    echo [4] Uninstall Service
    echo [5] View Service Logs
    echo [6] Build Project
    echo [7] Exit
)

echo.
set /p choice="Select option: "

if "%choice%"=="1" (
    sc query "%SERVICE_NAME%" >nul 2>&1
    if errorlevel 1 (
        call install-service.bat
    ) else (
        echo Starting service...
        net start "%SERVICE_NAME%"
    )
    pause
    goto MENU
)

if "%choice%"=="2" (
    sc query "%SERVICE_NAME%" >nul 2>&1
    if errorlevel 1 (
        call build.bat
    ) else (
        echo Stopping service...
        net stop "%SERVICE_NAME%"
    )
    pause
    goto MENU
)

if "%choice%"=="3" (
    sc query "%SERVICE_NAME%" >nul 2>&1
    if errorlevel 1 (
        exit /b 0
    ) else (
        echo Restarting service...
        net stop "%SERVICE_NAME%" >nul 2>&1
        timeout /t 2 >nul
        net start "%SERVICE_NAME%"
    )
    pause
    goto MENU
)

if "%choice%"=="4" (
    call uninstall-service.bat
    pause
    goto MENU
)

if "%choice%"=="5" (
    echo Viewing Windows Event Log for CupBot...
    echo Recent events:
    wevtutil qe Application /c:10 /rd:true /f:text /q:"*[System[Provider[@Name='CupBot']]]" 2>nul
    if errorlevel 1 (
        echo No CupBot events found in Application log
        echo Check Service logs in Event Viewer manually
    )
    pause
    goto MENU
)

if "%choice%"=="6" (
    call build.bat
    pause
    goto MENU
)

if "%choice%"=="7" (
    exit /b 0
)

echo Invalid option. Please try again.
pause
goto MENU