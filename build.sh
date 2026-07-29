#!/bin/bash

set -e

GOOS=android
GOARCH=arm64

echo "Building for $GOOS/$GOARCH..."
go build -ldflags "-s -w" -o bfr-webui-android-arm64 .

echo "Build successful: bfr-webui-android-arm64"
