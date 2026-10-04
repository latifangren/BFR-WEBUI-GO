@echo off
setlocal enabledelayedexpansion

set FLAVOR=%1
if "%FLAVOR%"=="" (
    set CURRENT_BRANCH=dev
    for /f %%b in ('git branch --show-current 2^>nul') do set CURRENT_BRANCH=%%b
    echo !CURRENT_BRANCH! | findstr /i "no-modem" >nul
    if !errorlevel! equ 0 (
        set FLAVOR=nomodem
    ) else (
        set FLAVOR=full
    )
)

echo ========================================================
echo  BFR-WEBUI-GO Builder [Target Flavor: %FLAVOR%]
echo ========================================================

echo 1. Compiling frontend (Svelte 5 + Vite)...
cd frontend
where bun >nul 2>nul
if %errorlevel% equ 0 (
    call bun run build
) else (
    call npm run build
)
if %errorlevel% neq 0 (
    echo [ERROR] Frontend build failed.
    cd ..
    exit /b %errorlevel%
)
cd ..

echo 2. Building Go binary for Android ARM64...
set GOOS=android
set GOARCH=arm64
set CGO_ENABLED=0

set BUILD_TAGS=
if /i "%FLAVOR%"=="nomodem" (
    echo [INFO] Applying build tag: nomodem (zero vendor modem code)
    set BUILD_TAGS=-tags nomodem
) else (
    echo [INFO] Full private build (all vendor baseband tools included)
)

go build %BUILD_TAGS% -ldflags "-s -w" -o bfr-webui-android-arm64 .
if %errorlevel% neq 0 (
    echo [ERROR] Go build failed.
    exit /b %errorlevel%
)

echo [SUCCESS] Build successful: bfr-webui-android-arm64 (%FLAVOR%)
endlocal
