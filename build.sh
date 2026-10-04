#!/usr/bin/env bash
set -e

FLAVOR="${1:-}"
if [ -z "$FLAVOR" ]; then
    CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "dev")
    if [[ "$CURRENT_BRANCH" == *"no-modem"* ]]; then
        FLAVOR="nomodem"
    else
        FLAVOR="full"
    fi
fi

echo "========================================================"
echo " BFR-WEBUI-GO Builder [Target Flavor: ${FLAVOR}]"
echo "========================================================"

echo "1. Compiling frontend (Svelte 5 + Vite)..."
(cd frontend && (bun run build || pnpm build || npm run build))

echo "2. Building Go binary for Android ARM64..."
export GOOS=android
export GOARCH=arm64
export CGO_ENABLED=0

BUILD_TAGS=""
if [ "$FLAVOR" = "nomodem" ]; then
    echo "[INFO] Applying build tag: nomodem (zero vendor modem code)"
    BUILD_TAGS="-tags nomodem"
else
    echo "[INFO] Full private build (all vendor baseband tools included)"
fi

go build $BUILD_TAGS -ldflags "-s -w" -o bfr-webui-android-arm64 .

echo "[SUCCESS] Build successful: bfr-webui-android-arm64 (${FLAVOR})"
