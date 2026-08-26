# BFR-WEBUI-GO Development Guide

This guide explains how to set up, build, test, and contribute to **BFR-WEBUI-GO**.

---

## 🛠️ Development Prerequisites

- **Go**: Version 1.20 or newer.
- **Git**: For source version control.
- **Android Device or Emulator**: Rooted with Magisk, KernelSU, or APatch (for testing on real hardware over ADB).

---

## 🚀 Building & Cross-Compiling

### Building for Android (ARM64)
Since the primary target is Android running on ARM64 architecture, compile using:

- **Linux / macOS**:
  ```bash
  GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o webui .
  ```
- **Windows (PowerShell)**:
  ```powershell
  $env:GOOS='android'; $env:GOARCH='arm64'; $env:CGO_ENABLED='0'; go build -ldflags="-s -w" -o webui .
  ```

### Local Standalone Testing
You can also compile and run the binary locally on your desktop operating system for UI & non-root feature development:

```bash
go run main.go
```
The server will start at `http://localhost:80` using default fallback values.

---

## 🏗️ Project Architecture Overview

The codebase is split into modular Go packages and feature-based frontend templates:

1. **`main.go`**: Program entry point. Parses environment flags, registers middlewares, initializes HTTP server, and sets up graceful shutdown signals.
2. **`internal/config/`**: Centralized configuration module. Handles environment variable overrides (`BFR_*`) with hardcoded defaults.
3. **`internal/filemanager/`**: Modular file operations engine split into:
   - `filemanager.go`: Path sanitization and directory listing.
   - `ops.go`: File/directory operations (copy, move, delete, create, upload, batch operations).
   - `permissions.go`: Octal permission (`chmod`) and ownership (`chown`) controls.
   - `archive.go`: ZIP compression and extraction with Zip Slip vulnerability protection.
   - `search.go`: Bounded file search and disk space metrics.
4. **`internal/sysinfo/`**: Direct `/proc` and `/sys` hardware counter parsing, featuring active Gateway & DNS resolver detection.
5. **`internal/handlers/`**: REST API endpoints, WebSocket handlers (`terminal`, `scrcpy`), and request middleware pipeline (`securityHeaders`, `maxBodySize`, `RequireAuth`).
6. **`web/`**: Single-Page Application (SPA) templates and assets embedded into the Go binary via `embed.FS`.
   - `templates/<feature>/`: Feature-scoped HTML template blocks.
   - `static/js/modules/<feature>/`: Modular Alpine.js stores and logic.

---

## 🧪 Code Quality & Verification

Before submitting PRs or commits, verify that the code passes linting and cross-compilation:

```bash
# 1. Run Go vet
go vet ./...

# 2. Test cross-compilation for Android
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build ./...
```

---

## 📲 Direct Device Deployment over ADB

If your rooted Android device has ADB wireless enabled (e.g. at `192.168.100.55:5555`):

```bash
# Connect ADB
adb connect 192.168.100.55:5555

# Build binary
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o webui .

# Stop running process, push new binary, and restart
adb shell "su -c 'killall -9 webui 2>/dev/null'"
adb push webui /data/local/tmp/webui_update
adb shell "su -c 'cp /data/local/tmp/webui_update /data/adb/modules/bfr_webui_go/webui; chmod 755 /data/adb/modules/bfr_webui_go/webui; rm /data/local/tmp/webui_update; /data/adb/modules/bfr_webui_go/webui &'"
```
