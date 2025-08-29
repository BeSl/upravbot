@echo off
echo Building and installing CupBot as Windows service...
echo.

echo Step 1: Building project...
call build.bat
if errorlevel 1 (
    echo Build failed! Please check errors above.
    pause
    exit /b 1
)

echo.
echo Step 2: Installing service...
call install-service.bat

echo.
echo Installation completed!
pause