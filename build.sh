#!/bin/bash

set -e

echo "1. Compiling frontend (Svelte 5 + Vite)..."
(cd frontend && (pnpm build || npm run build))

echo "2. Building Go binary for Android ARM64..."
export GOOS=android
export GOARCH=arm64
export CGO_ENABLED=0
go build -ldflags "-s -w" -o bfr-webui-android-arm64 .

echo "Build successful: bfr-webui-android-arm64"
