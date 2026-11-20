@echo off
setlocal enabledelayedexpansion

:: Scripts live in scripts/; always operate from the repository root.
cd /d "%~dp0.."

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
for /f "tokens=2 delims==" %%i in ('findstr "version=" module\module.prop') do set VER=%%i

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

echo 4. Staging Magisk module files and packaging zip using tar.exe...
set ZIP_NAME=BFR-WEBUI-Magisk-%VER%-local.zip
if /i "%FLAVOR%"=="nomodem" (
    set ZIP_NAME=BFR-WEBUI-Magisk-%VER%-nomodem.zip
)

if exist "%ZIP_NAME%" del "%ZIP_NAME%"
if exist ".tmp\pkg-local" rmdir /s /q ".tmp\pkg-local"
mkdir ".tmp\pkg-local"
copy /y module\customize.sh .tmp\pkg-local\ >nul
copy /y module\module.prop .tmp\pkg-local\ >nul
copy /y module\service.sh .tmp\pkg-local\ >nul
copy /y module\system.prop .tmp\pkg-local\ >nul
copy /y module\update.json .tmp\pkg-local\ >nul
copy /y module\env.example .tmp\pkg-local\ >nul
copy /y module\uninstall.sh .tmp\pkg-local\ >nul
xcopy /e /i /y module\bin .tmp\pkg-local\bin >nul
xcopy /e /i /y module\META-INF .tmp\pkg-local\META-INF >nul
copy /y webui .tmp\pkg-local\ >nul
if /i "%FLAVOR%"=="nomodem" (
    if exist ".tmp\pkg-local\bin\band_manager.dex" del /f /q ".tmp\pkg-local\bin\band_manager.dex"
    if exist ".tmp\pkg-local\bin\sim_switcher.dex" del /f /q ".tmp\pkg-local\bin\sim_switcher.dex"
)
tar.exe -a -c -f "%ZIP_NAME%" -C .tmp\pkg-local META-INF customize.sh module.prop service.sh system.prop update.json env.example uninstall.sh bin webui
if %errorlevel% neq 0 (
    echo [ERROR] Packing failed.
    del webui
    rmdir /s /q ".tmp\pkg-local" >nul 2>&1
    pause
    exit /b %errorlevel%
)
rmdir /s /q ".tmp\pkg-local" >nul 2>&1

echo 5. Cleaning up temporary binary...
del webui

echo [SUCCESS] %ZIP_NAME% created successfully!
endlocal
