@echo off
set GOOS=android
set GOARCH=arm64
set CGO_ENABLED=0

echo 1. Reading version from module.prop...
set VER=local
for /f "tokens=2 delims==" %%i in ('findstr "^version=" module.prop') do set VER=%%i

echo 2. Building Go binary for Android ARM64 (webui)...
go build -ldflags "-s -w" -o webui .
if %errorlevel% neq 0 (
    echo [ERROR] Go build failed.
    pause
    exit /b %errorlevel%
)

echo 3. Packaging Magisk zip using tar.exe...
if exist "BFR-WEBUI-Magisk-%VER%-local.zip" del "BFR-WEBUI-Magisk-%VER%-local.zip"
tar.exe -a -c -f "BFR-WEBUI-Magisk-%VER%-local.zip" customize.sh module.prop service.sh system.prop tweaks.json env.example bin webui
if %errorlevel% neq 0 (
    echo [ERROR] Packing failed.
    del webui
    pause
    exit /b %errorlevel%
)

echo 4. Cleaning up temporary binary...
del webui

echo [SUCCESS] BFR-WEBUI-Magisk-%VER%-local.zip created successfully!
pause
