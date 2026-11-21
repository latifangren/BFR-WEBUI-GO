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
if [ "$FLAVOR" = "nomodem" ]; then
    export VITE_HAS_VENDOR_MODEM=false
else
    export VITE_HAS_VENDOR_MODEM=true
fi
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

if [ "$FLAVOR" = "nomodem" ]; then
    if command -v python3 >/dev/null 2>&1; then
        python3 scripts/audit_nomodem.py bfr-webui-android-arm64
    elif command -v python >/dev/null 2>&1; then
        python scripts/audit_nomodem.py bfr-webui-android-arm64
    else
        echo "[ERROR] Python is required to audit the no-modem binary."
        exit 1
    fi
fi

echo "[SUCCESS] Build successful: bfr-webui-android-arm64 (${FLAVOR})"
