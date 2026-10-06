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

set GOOS=android
set GOARCH=arm64
set CGO_ENABLED=0

echo ========================================================
echo  BFR-WEBUI-GO Magisk Zip Packager [Flavor: %FLAVOR%]
echo ========================================================

echo 1. Reading version from module.prop...
set VER=local
for /f "tokens=2 delims==" %%i in ('findstr "^version=" module.prop') do set VER=%%i

echo 2. Compiling frontend production bundle (Svelte 5 + Vite)...
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
    pause
    exit /b %errorlevel%
)
cd ..

echo 3. Building Go binary for Android ARM64 (webui)...
set BUILD_TAGS=
if /i "%FLAVOR%"=="nomodem" (
    echo [INFO] Applying build tag: nomodem
    set BUILD_TAGS=-tags nomodem
) else (
    echo [INFO] Full private build
)

go build %BUILD_TAGS% -ldflags "-s -w" -o webui .
if %errorlevel% neq 0 (
    echo [ERROR] Go build failed.
    pause
    exit /b %errorlevel%
)

echo 4. Packaging Magisk zip using tar.exe...
set ZIP_NAME=BFR-WEBUI-Magisk-%VER%-local.zip
if /i "%FLAVOR%"=="nomodem" (
    set ZIP_NAME=BFR-WEBUI-Magisk-%VER%-nomodem.zip
)

if exist "%ZIP_NAME%" del "%ZIP_NAME%"
if /i "%FLAVOR%"=="nomodem" (
    tar.exe -a -c --exclude="band_manager.dex" --exclude="sim_switcher.dex" -f "%ZIP_NAME%" customize.sh module.prop service.sh system.prop update.json env.example bin webui
) else (
    tar.exe -a -c -f "%ZIP_NAME%" customize.sh module.prop service.sh system.prop update.json env.example bin webui
)
    echo [ERROR] Packing failed.
    del webui
    pause
    exit /b %errorlevel%
)

echo 5. Cleaning up temporary binary...
del webui

echo [SUCCESS] %ZIP_NAME% created successfully!
endlocal
