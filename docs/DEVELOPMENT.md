# BFR-WEBUI-GO Development Guide

This guide explains how to set up, build, test, and contribute to **BFR-WEBUI-GO**.

---

## 🛠️ Development Prerequisites

- **Go**: Version 1.22 or newer (1.25.0 recommended).
- **Node.js & pnpm**: Node.js 20+ and pnpm 9+ (for frontend development & compilation).
- **Git**: For source version control.
- **Android Device or Emulator**: Rooted with Magisk, KernelSU, or APatch (for testing on real hardware over ADB).

---

## 🚀 Building & Cross-Compiling

The project uses Go's `embed.FS` (`frontend/embed.go`) to embed the compiled frontend bundle into a single self-contained binary (~10-12MB).

### 1. Build the Frontend Distribution
```bash
cd frontend
pnpm install
pnpm build
cd ..
```
This generates optimized static production assets inside `frontend/dist/`.

### 2. Compile for Android (ARM64)
Since the primary target is Android running on ARM64 architecture, compile using:

- **Linux / macOS**:
  ```bash
  GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o webui .
  ```
- **Windows (PowerShell)**:
  ```powershell
  $env:GOOS='android'; $env:GOARCH='arm64'; $env:CGO_ENABLED='0'; go build -ldflags="-s -w" -o webui .
  ```

### 3. Local Standalone Testing
You can run the backend and frontend locally for UI & non-root feature development:

- **Backend**:
  ```bash
  go run main.go
  ```
  The server starts at `http://localhost:80` using default fallback values.

- **Frontend Hot-Reloading (Vite Dev Server)**:
  ```bash
  cd frontend
  pnpm dev
  ```
  Vite will serve the frontend with HMR at `http://localhost:5173`, proxying API requests to the Go backend.

---

## 🏗️ Project Architecture Overview

The codebase is split into modular Go backend packages and a modern Svelte 5 SPA frontend:

### Backend Structure (`internal/`)
1. **`main.go`**: Program entry point. Parses flags, registers middleware, initializes HTTP server, and sets up graceful shutdown signals.
2. **`internal/config/`**: Centralized configuration module. Handles environment variable overrides (`BFR_*`) with hardcoded defaults.
3. **`internal/auth/`**: Cookie-based session authentication manager.
4. **`internal/charger/`**: Hardware PMIC charging limitation controllers with auto-detection.
5. **`internal/filemanager/`**: Path sanitization, directory listing, file CRUD, octal permissions, and zip archives.
6. **`internal/handlers/`**: Web controllers and REST API routers.
7. **`internal/hotspot/`**: SoftAP controls, connected clients ARP parsing, and MAC filtering.
8. **`internal/network/`**: Persistent sysctl values, DNS resolvers, and dynamic SDK-aware TTL configurations.
9. **`internal/proxy/`**: Clash/Mihomo configurations and watchdog monitoring.
10. **`internal/scrcpy/`**: Scrcpy web streaming canvas and touch input event channels.
11. **`internal/smsviewer/`**: AT modem SMS parsing and message inbox management.
12. **`internal/sysinfo/`**: Hardware counters `/proc` and `/sys` sensor reader.
13. **`internal/terminal/`**: WebSocket interactive PTY terminal.
14. **`internal/vnstat/`**: Vnstat bandwidth accounting and telemetry engine.

### Frontend Structure (`frontend/`)
- **`frontend/src/App.svelte`**: Main application shell with responsive sidebar, header, and tab router.
- **`frontend/src/components/layout/`**: Header, Sidebar, BottomNav, and Toast notifications.
- **`frontend/src/components/tabs/`**: Individual tab components categorized under System, Network, Services, Connectivity, Diagnostics, and About.
- **`frontend/src/components/ui/`**: Reusable Neo-Brutalist design components (Card, Button, Badge, Modal, Input, Toggle).
- **`frontend/src/stores/`**: Reactive Svelte 5 rune stores (`auth.svelte.ts`, `navigation.svelte.ts`, `toast.svelte.ts`, `theme.svelte.ts`).
- **`frontend/src/api/client.ts`**: Unified HTTP fetch wrapper with error handling and CSRF/auth management.
- **`frontend/embed.go`**: Single-binary asset embed point exporting `DistFS`.

---

## 🧪 Testing & Code Quality

Run tests and verification before submitting code:

```bash
# Frontend validation
cd frontend
pnpm check
pnpm build
cd ..

# Go formatting and verification
go fmt ./...
go vet ./...

# Run all Go package tests
go test -v ./...

# Android ARM64 compilation verification
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o webui .
```
