# BFR-WEBUI_GO

High-performance, ultra-lightweight Android System Control Panel & WebUI built with Go, Alpine.js, and Tailwind CSS for Magisk, KernelSU, and APatch.

---

## 🌟 Key Features

- ⚡ **Ultra-lightweight Single Binary**: ~7MB - 10MB binary size with minimal RAM consumption (~12MB).
- 📶 **Network & Carrier Aggregation Tweaks**:
  - LTE-A / 5G Carrier Aggregation (`gsm.lte.ca.support=1`, `persist.radio.lte_ca=1`).
  - TCP BBR2 / BBR Congestion Control & network stack buffer optimizations.
  - TTL Spoofing (`iptables -t mangle -A POSTROUTING -j TTL --ttl-set 64`) to bypass tethering throttling.
  - Per-interface MTU & TxQueue Length tuning.
  - One-click DNS Resolver Switcher (Cloudflare, Google, AdGuard, Quad9).
  - Built-in Ping Diagnostic tool.
- 🚀 **Proxy Core Manager**:
  - Automatic Mihomo / Clash process detection (Box4Magisk, Clash For Magisk).
  - Service lifecycle management (Start, Stop, Restart).
  - Core Mode selector (`Rule`, `Global`, `Direct`).
  - Real-time live log streaming via Server-Sent Events (SSE).
- 💻 **Interactive Web Terminal**:
  - Full-featured root shell (`/system/bin/sh` / `su`) powered by xterm.js via WebSockets (`/api/terminal/ws`).
- 📂 **Native Web File Manager**:
  - Full directory navigation with breadcrumbs and shortcuts (`/sdcard`, `/data/adb`, `/modules`).
  - Text & code editor for modifying config files (`system.prop`, `.sh`, `config.yaml`).
  - Drag-and-drop / file upload, file downloads, directory creation, and file deletion.
- 📊 **Real-time System Monitoring**:
  - Per-core CPU frequencies & usage percentages.
  - Thermal zone temperature monitoring.
  - Storage & disk partition metrics (`/data`, `/system`, `/sdcard`).
  - Battery capacity, temperature, voltage, health, and status.
  - Device model & Android release details.
- 🛡️ **100% Offline Ready**:
  - Fully self-contained with embedded static assets (`Alpine.js`, `Tailwind CSS`, `xterm.js`) — zero external CDN dependencies required.
- 🔒 **Cookie-based Session Authentication**:
  - Secure HTTP cookie session authentication (`/api/auth/login`, `/api/auth/logout`, `/api/auth/status`).

---

## 🛠️ Installation & Build Instructions

### Prerequisites
- [Go 1.21+](https://go.dev/dl/) installed on your build machine.

### Building for Android ARM64
Run the following build command from project root:

```bash
# Windows PowerShell
$env:GOOS="android"; $env:GOARCH="arm64"; go build -ldflags="-s -w" -o webui main.go

# Linux / macOS
GOOS=android GOARCH=arm64 go build -ldflags="-s -w" -o webui main.go
```

### Packaging into Magisk / KernelSU Module ZIP
Create a release ZIP archive containing the following files at the root of the archive:
- `module.prop`
- `service.sh`
- `customize.sh`
- `system.prop`
- `webui` (the compiled Android ARM64 binary)

Flash the created ZIP module via Magisk App, KernelSU, or APatch and reboot.

---

## 🌐 Default Access

Once installed and booted, open your browser and navigate to:

- **URL**: `http://<device-ip>:8080` (or `http://localhost:8080`)
- **Default Password**: `bfr` *(configurable via `BFR_PASSWORD` environment variable)*

---

## 📄 License
Released under the MIT License. Developed by BFR.
