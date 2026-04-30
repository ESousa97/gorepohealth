@echo off
SETLOCAL

echo ========================================
echo   GOREPOHEALTH - Automated Test Suite
echo ========================================

:: 1. Run Unit Tests
echo [1/3] Running Go Unit Tests...
go test ./... -v
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Unit tests failed!
    exit /b %ERRORLEVEL%
)
echo [OK] Unit tests passed.

:: 2. Build Check
echo.
echo [2/3] Building the application...
go build -o gorepohealth.exe ./cmd/gorepohealth/main.go
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Build failed!
    exit /b %ERRORLEVEL%
)
echo [OK] Application built successfully.

:: 3. Integration Check (Requires GITHUB_TOKEN)
echo.
echo [3/3] Running Integration Check (google/go-github)...
if "%GITHUB_TOKEN%"=="" (
    echo [SKIP] GITHUB_TOKEN not set, skipping integration check.
) else (
    .\gorepohealth.exe google/go-github
    if %ERRORLEVEL% NEQ 0 (
        echo [ERROR] Integration check failed!
        exit /b %ERRORLEVEL%
      )
    echo [OK] Integration check passed.
)

echo.
echo ========================================
echo   ALL TESTS PASSED SUCCESSFULLY!
echo ========================================
pause
