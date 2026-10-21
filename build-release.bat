@echo off
setlocal enabledelayedexpansion

echo ========================================================
echo  BFR-WEBUI-GO Dual-Variant Release Builder
echo ========================================================

:: Extract version from module.prop
set "VERSION=v1.2.8"
for /f "tokens=2 delims==" %%I in ('findstr "version=" module.prop 2^>nul') do (
    set "VERSION=%%I"
)
echo [INFO] Detected Version: !VERSION!

if not exist ".tmp" mkdir .tmp

:: ========================================================
:: 1. Build Flavor 1: Full Modem (dev target)
:: ========================================================
echo.
echo [1/4] Preparing Full Modem frontend assets...
git checkout frontend/src/config/features.ts frontend/src/components/tabs/vendor_tabs.ts >nul 2>&1
cd frontend
call bun run build
if !ERRORLEVEL! neq 0 (
    echo [ERROR] Frontend build failed for Full Modem!
    cd ..
    exit /b 1
)
cd ..

echo [2/4] Cross-compiling Full Android ARM64 binary...
set "GOOS=android"
set "GOARCH=arm64"
set "CGO_ENABLED=0"

if exist ".tmp\pkg-full" rmdir /s /q ".tmp\pkg-full"
mkdir ".tmp\pkg-full"

go build -ldflags="-s -w" -o .tmp\pkg-full\webui .
if !ERRORLEVEL! neq 0 (
    echo [ERROR] Go build failed for Full Modem!
    exit /b 1
)

echo [INFO] Packaging BFR-WEBUI-Magisk-!VERSION!.zip...
copy /y customize.sh .tmp\pkg-full\ >nul
copy /y module.prop .tmp\pkg-full\ >nul
copy /y service.sh .tmp\pkg-full\ >nul
copy /y system.prop .tmp\pkg-full\ >nul
copy /y update.json .tmp\pkg-full\ >nul
copy /y env.example .tmp\pkg-full\ >nul
copy /y uninstall.sh .tmp\pkg-full\ >nul
xcopy /e /i /y bin .tmp\pkg-full\bin >nul
xcopy /e /i /y META-INF .tmp\pkg-full\META-INF >nul

if exist "BFR-WEBUI-Magisk-!VERSION!.zip" del /f /q "BFR-WEBUI-Magisk-!VERSION!.zip"
tar -a -c -f "BFR-WEBUI-Magisk-!VERSION!.zip" -C .tmp\pkg-full META-INF customize.sh module.prop service.sh system.prop update.json env.example uninstall.sh bin webui
if !ERRORLEVEL! neq 0 (
    powershell -NoProfile -Command "Compress-Archive -Path '.tmp\pkg-full\*' -DestinationPath 'BFR-WEBUI-Magisk-!VERSION!.zip' -Force"
)
echo [SUCCESS] Full Modem Zip created: BFR-WEBUI-Magisk-!VERSION!.zip

:: ========================================================
:: 2. Build Flavor 2: Clean No-Modem (public target)
:: ========================================================
echo.
echo [3/4] Preparing Clean No-Modem frontend assets...
echo export const HAS_VENDOR_MODEM = false;> frontend\src\config\features.ts
echo import type { Component } from 'svelte';> frontend\src\components\tabs\vendor_tabs.ts
echo.>> frontend\src\components\tabs\vendor_tabs.ts
echo export const VENDOR_TABS: Record^<string, Component^> = {};>> frontend\src\components\tabs\vendor_tabs.ts

cd frontend
call bun run build
if !ERRORLEVEL! neq 0 (
    echo [ERROR] Frontend build failed for No-Modem!
    cd ..
    git checkout frontend/src/config/features.ts frontend/src/components/tabs/vendor_tabs.ts >nul 2>&1
    exit /b 1
)
cd ..

echo [4/4] Cross-compiling No-Modem Android ARM64 binary...
if exist ".tmp\pkg-nomodem" rmdir /s /q ".tmp\pkg-nomodem"
mkdir ".tmp\pkg-nomodem"

go build -tags nomodem -ldflags="-s -w" -o .tmp\pkg-nomodem\webui .
if !ERRORLEVEL! neq 0 (
    echo [ERROR] Go build failed for No-Modem!
    git checkout frontend/src/config/features.ts frontend/src/components/tabs/vendor_tabs.ts >nul 2>&1
    exit /b 1
)

:: Restore frontend files immediately
git checkout frontend/src/config/features.ts frontend/src/components/tabs/vendor_tabs.ts >nul 2>&1

:: Run security audit if python available
python --version >nul 2>&1
if !ERRORLEVEL! equ 0 (
    echo [INFO] Running zero-leak forensic audit on No-Modem binary...
    python scripts\audit_nomodem.py .tmp\pkg-nomodem\webui
    if !ERRORLEVEL! neq 0 (
        echo [SECURITY WARNING] Audit reported signatures in No-Modem binary!
    )
)

echo [INFO] Packaging BFR-WEBUI-Magisk-!VERSION!-nomodem.zip...
copy /y customize.sh .tmp\pkg-nomodem\ >nul
copy /y module.prop .tmp\pkg-nomodem\ >nul
copy /y service.sh .tmp\pkg-nomodem\ >nul
copy /y system.prop .tmp\pkg-nomodem\ >nul
copy /y update.json .tmp\pkg-nomodem\ >nul
copy /y env.example .tmp\pkg-nomodem\ >nul
copy /y uninstall.sh .tmp\pkg-nomodem\ >nul
xcopy /e /i /y bin .tmp\pkg-nomodem\bin >nul
xcopy /e /i /y META-INF .tmp\pkg-nomodem\META-INF >nul

:: Strip proprietary DEX files
if exist ".tmp\pkg-nomodem\bin\band_manager.dex" del /f /q ".tmp\pkg-nomodem\bin\band_manager.dex"
if exist ".tmp\pkg-nomodem\bin\sim_switcher.dex" del /f /q ".tmp\pkg-nomodem\bin\sim_switcher.dex"

if exist "BFR-WEBUI-Magisk-!VERSION!-nomodem.zip" del /f /q "BFR-WEBUI-Magisk-!VERSION!-nomodem.zip"
tar -a -c -f "BFR-WEBUI-Magisk-!VERSION!-nomodem.zip" -C .tmp\pkg-nomodem META-INF customize.sh module.prop service.sh system.prop update.json env.example uninstall.sh bin webui
if !ERRORLEVEL! neq 0 (
    powershell -NoProfile -Command "Compress-Archive -Path '.tmp\pkg-nomodem\*' -DestinationPath 'BFR-WEBUI-Magisk-!VERSION!-nomodem.zip' -Force"
)
echo [SUCCESS] Clean No-Modem Zip created: BFR-WEBUI-Magisk-!VERSION!-nomodem.zip

:: Clean up staging dirs
rmdir /s /q ".tmp\pkg-full" >nul 2>&1
rmdir /s /q ".tmp\pkg-nomodem" >nul 2>&1

echo.
echo ========================================================
echo  All 2 Release Variants Successfully Built!
echo   1. BFR-WEBUI-Magisk-!VERSION!.zip
echo   2. BFR-WEBUI-Magisk-!VERSION!-nomodem.zip
echo ========================================================
