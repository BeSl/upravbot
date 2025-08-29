@echo off
REM Test runner script for CupBot project
REM This script runs all tests with coverage and generates reports

echo ========================================
echo CupBot Test Runner
echo ========================================
echo.

REM Change to project directory
cd /d "%~dp0"

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

echo Go version:
go version
echo.

REM Install test dependencies if needed
echo Installing test dependencies...
go mod tidy
if errorlevel 1 (
    echo ERROR: Failed to install dependencies
    pause
    exit /b 1
)
echo.

REM Create test output directory
if not exist "test-results" mkdir test-results

REM Run tests with coverage
echo ========================================
echo Running unit tests with coverage...
echo ========================================
echo.

REM Run tests for each package separately with coverage
echo Running config package tests...
go test -v -coverprofile=test-results\config-coverage.out .\internal\config
if errorlevel 1 (
    echo ERROR: Config tests failed
    set TEST_FAILED=1
)
echo.

echo Running database package tests...
go test -v -coverprofile=test-results\database-coverage.out .\internal\database
if errorlevel 1 (
    echo ERROR: Database tests failed
    set TEST_FAILED=1
)
echo.

echo Running auth package tests...
go test -v -coverprofile=test-results\auth-coverage.out .\internal\auth
if errorlevel 1 (
    echo ERROR: Auth tests failed
    set TEST_FAILED=1
)
echo.

echo Running system package tests...
go test -v -coverprofile=test-results\system-coverage.out .\internal\system
if errorlevel 1 (
    echo ERROR: System tests failed
    set TEST_FAILED=1
)
echo.

echo Running bot integration tests...
go test -v -coverprofile=test-results\bot-coverage.out .\internal\bot
if errorlevel 1 (
    echo ERROR: Bot tests failed
    set TEST_FAILED=1
)
echo.

REM Run all tests together for combined coverage
echo ========================================
echo Running all tests with combined coverage...
echo ========================================
echo.

go test -v -coverprofile=test-results\coverage.out .\internal\...
if errorlevel 1 (
    echo ERROR: Combined test run failed
    set TEST_FAILED=1
)
echo.

REM Generate coverage report
echo ========================================
echo Generating coverage reports...
echo ========================================
echo.

REM Generate HTML coverage report
go tool cover -html=test-results\coverage.out -o test-results\coverage.html
if errorlevel 1 (
    echo WARNING: Failed to generate HTML coverage report
) else (
    echo HTML coverage report generated: test-results\coverage.html
)

REM Generate text coverage summary
go tool cover -func=test-results\coverage.out > test-results\coverage-summary.txt
if errorlevel 1 (
    echo WARNING: Failed to generate coverage summary
) else (
    echo Coverage summary generated: test-results\coverage-summary.txt
    echo.
    echo Coverage Summary:
    type test-results\coverage-summary.txt | findstr "total"
)
echo.

REM Run benchmarks
echo ========================================
echo Running benchmarks...
echo ========================================
echo.

go test -bench=. -benchmem .\internal\... > test-results\benchmark.txt 2>&1
if errorlevel 1 (
    echo WARNING: Some benchmarks failed
) else (
    echo Benchmark results saved to: test-results\benchmark.txt
)
echo.

REM Check for race conditions
echo ========================================
echo Running race condition tests...
echo ========================================
echo.

go test -race .\internal\... > test-results\race-test.txt 2>&1
if errorlevel 1 (
    echo WARNING: Race condition tests failed or detected issues
    echo Check test-results\race-test.txt for details
) else (
    echo Race condition tests passed
)
echo.

REM Generate test report
echo ========================================
echo Generating test report...
echo ========================================
echo.

echo CupBot Test Report > test-results\test-report.txt
echo Generated on: %DATE% %TIME% >> test-results\test-report.txt
echo. >> test-results\test-report.txt
echo Go Version: >> test-results\test-report.txt
go version >> test-results\test-report.txt
echo. >> test-results\test-report.txt
echo Coverage Summary: >> test-results\test-report.txt
type test-results\coverage-summary.txt | findstr "total" >> test-results\test-report.txt
echo. >> test-results\test-report.txt

REM Final results
echo ========================================
echo Test Results Summary
echo ========================================
echo.

if defined TEST_FAILED (
    echo ❌ SOME TESTS FAILED
    echo Please check the output above for details
    echo.
    exit /b 1
) else (
    echo ✅ ALL TESTS PASSED
    echo.
    echo Generated files:
    echo - test-results\coverage.html    (HTML coverage report)
    echo - test-results\coverage-summary.txt (Coverage summary)
    echo - test-results\benchmark.txt   (Benchmark results)
    echo - test-results\race-test.txt   (Race condition test results)
    echo - test-results\test-report.txt (Test report summary)
    echo.
    echo To view coverage report: test-results\coverage.html
)

echo ========================================
echo Test run completed
echo ========================================
pause