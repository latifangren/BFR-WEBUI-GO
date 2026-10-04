@echo off
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
go build -tags nomodem -ldflags "-s -w" -o bfr-webui-android-arm64 .
if %errorlevel% neq 0 (
    echo Build failed.
    exit /b %errorlevel%
)
echo Build successful: bfr-webui-android-arm64
