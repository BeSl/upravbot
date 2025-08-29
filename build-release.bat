@echo off
echo Building CupBot...

REM Set Go environment
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

REM Clean previous builds
if exist cupbot.exe del cupbot.exe
if exist cupbot-windows-amd64.exe del cupbot-windows-amd64.exe

REM Download dependencies
echo Downloading dependencies...
go mod download
if errorlevel 1 (
    echo Failed to download dependencies
    goto error
)

REM Verify dependencies
echo Verifying dependencies...
go mod verify
if errorlevel 1 (
    echo Failed to verify dependencies
    goto error
)

REM Build debug version
echo Building debug version...
go build -v -o cupbot.exe .
if errorlevel 1 (
    echo Failed to build debug version
    goto error
)

REM Build release version
echo Building release version...
go build -ldflags="-w -s" -o cupbot-windows-amd64.exe .
if errorlevel 1 (
    echo Failed to build release version
    goto error
)

REM Test executable
echo Testing executable...
cupbot.exe -help
if errorlevel 1 (
    echo Warning: Executable test failed, but continuing...
)

echo.
echo ✓ Build completed successfully!
echo.
echo Files created:
echo   - cupbot.exe (debug version)
echo   - cupbot-windows-amd64.exe (release version)
echo.
goto end

:error
echo.
echo ✗ Build failed!
echo.
pause
exit /b 1

:end
pause