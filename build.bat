@echo off
set GOOS=android
set GOARCH=arm64
echo Building for %GOOS% %GOARCH%...
go build -ldflags "-s -w" -o bfr-webui-android-arm64 .
if %errorlevel% neq 0 (
    echo Build failed.
    exit /b %errorlevel%
)
echo Build successful: bfr-webui-android-arm64
