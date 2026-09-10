# BFR-WEBUI-GO

> **Ultra-Lightweight, High-Performance Android System Control Panel & WebUI**  
> Specially engineered as a 100% offline-ready Magisk / KernelSU / APatch module. Built with a modular native Go backend and a modern Svelte 5 + TypeScript + Tailwind CSS frontend embedded into a single self-contained binary (~10-12MB).

---

## 📊 Empirical Resource Footprint (Pixel 5 ARM64)

Benchmarked directly on live Android target hardware:

| Metric | Measured Value | Architecture Highlights |
| :--- | :--- | :--- |
| **Binary Footprint** | **~16 MB** | **100% Standalone** (Single compiled binary, zero Python/Node.js runtime dependencies) |
| **Physical RSS RAM** | **~21 MB** | **Ultra-Efficient** (Runs smoothly on low-end 2GB/3GB RAM Android devices) |
| **Private PSS RAM** | **~11 MB** | Extremely low dedicated memory footprint |
| **Idle CPU Usage** | **0.0% - 0.2%** | Minimal kernel context switches, zero background polling overhead |
| **Swap / Storage Wear** | **0 KB** | Zero disk write thrashing, protects eMMC / UFS flash lifespan |

---

## ⚡ Core Feature Highlights

### 📁 Modular Dual-Pane File Manager (Dual Commander)
- **Dual-Pane Split View**: True side-by-side two-column explorer on desktop/tablet, touch-friendly tab switcher (`[ Panel A | Panel B ]`) on mobile devices.
- **One-Click Cross-Pane Transfer**: Instant `Copy to Other Pane` and `Move to Other Pane` batch operations backed by atomic `/api/files/batch` endpoint.
- **Quick Bookmarks Bar**: Built-in Android & Magisk system presets (`/`, `/sdcard`, `/data/adb`, `/data/adb/modules`, `/data/local/tmp`) plus persistent custom bookmarks pinned via `localStorage`.
- **Drag & Drop Upload**: Full-screen reactive dropzone overlay with automatic multipart chunk streaming directly to the active directory.
- **Built-in File Tools**: Monospace syntax code editor, octal chmod permission presets (`0755`, `0644`, `0777`), ZIP/TAR compress and extract.

### 🎨 Customization Matrix (Appearance Studio)
- **Compact Appearance Studio**: Fast, minimalist modal configuration with zero layout clutter.
- **9 Curated Color Palettes**: `Dark Navy`, `Pure AMOLED`, `Clean Light`, `Dracula`, `Nordic Frost`, `Cyberpunk 2077`, `Matrix Emerald`, `Retro Sunset`, and **`Retro Deck`** (pastel ice with tactile high-contrast borders).
- **2 UI Paradigm Styles**:
  - **Neobrutalism**: 2px solid borders, hard mechanical drop-shadows, 4px corners, and tactile retro badges.
  - **Modern Clean**: 1px subtle borders, smooth curves, and ambient shadows.
- **2 Navigation Layouts**:
  - **Classic Top Bar**: Desktop category pills with hover flyouts; mobile 5-column bottom navigation with touch-friendly popovers.
  - **Modern Sidebar**: Desktop collapsible accordion drawer groups (Core, Network, System, Tools); mobile clean slide-over drawer.

### 🚀 Kernel & System Optimization
- **BBR2 TCP Congestion Control**: Automated socket optimization tailored for high-bandwidth, low-latency wireless networks.
- **System Optimizer Tweaks**: Persistent kernel sysctl tuning, TCP FastOpen, Queue limit allocations, and dynamic SDK-aware TTL spoofing (Android 11+ compatible).
- **Dynamic Hardware Charge Limiter**: Multi-vendor sysfs auto-scanner with Qualcomm PMIC hardware bypass (`force_main_fcc` 0 mA) and custom sysfs override support.
- **Root Daemon Services Telemetry**: Real-time PID, multi-core normalized CPU %, and resident RAM (MB/KB) tracking for `webui`, `mihomo`, `dropbear`, and `adbd`.

### 🌐 Connectivity & Networking
- **Cellular Modem & Band Locking**: Hybrid multi-engine band locking via Qualcomm AT serial (`/dev/smd11`, `/dev/ttyUSB*`), `cmd phone`, and secret codes. Real-time RSRP, RSRQ, SINR, and EARFCN signal metrics.
- **Proxy Core Controller**: Daemon manager for Clash / Mihomo with live stream logs, config editing, watchdog loop, and rule/global/direct mode switching.
- **VnStat Traffic Accounting**: Real-time interface bandwidth rate meters, daily/monthly consumption charts, and billing cycle quota trackers.
- **Interactive Web Terminal & Scrcpy Screen Mirror**: Full PTY root terminal over WebSockets (`xterm.js`) and low-latency H.264 canvas screen mirror with gesture touch injection.

### 🤖 Remote Management & Cloud Sync
- **Interactive Telegram Bot**: Remote commands (`/stats`, `/charger`, `/ssh`, `/proxy`, `/reboot`), persistent 4-row keyboard menus, and instant security push alerts (overheat, battery, IP change, SSH login).
- **WebDAV Cloud Backup**: Automated background compression and encrypted sync of configuration bundles (`charger`, `ssh`, `telegram`, `tweaks`) to private cloud servers.
- **Bundled Dropbear SSH**: Precompiled static ARM64 daemon with automated host-key generation and root authentication (`bfr`).
- **Support & Donation Hub**: Clean QRIS donation showcase (`qris.jpg`) and direct 1-tap confirmation via Telegram and Facebook.

---

## 🛠️ Tech Stack & Architecture

```
BFR-WEBUI-GO Architecture
├── Frontend (Embedded SPA in single Go binary)
│   ├── Svelte 5 (Runes: $state, $derived, $props)
│   ├── TypeScript (Strict typing across all components)
│   ├── Tailwind CSS (Neo-Brutalist & Modern Design Tokens)
│   └── Vite (Optimized production asset pipeline)
│
└── Backend (Go Native Module)
    ├── Go 1.22+ (Native concurrency & low memory overhead)
    ├── Standard Library HTTP Mux & Middleware (Gzip, CSRF, Rate Limiter)
    ├── Subsystem Controllers (charger, network, proxy, terminal, vnstat, modem)
    └── Magisk / KernelSU / APatch Module Runtime (/data/adb/modules/bfr_webui_go)
```

---

## 📦 Building & Packaging

### Prerequisites
- **Go**: Version 1.22 or higher
- **Node.js & pnpm**: Node.js 18+ and pnpm/npm installed

### Quick Build (Magisk ZIP Package)
On Windows:
```powershell
.\build_zip.bat
```
On Linux / macOS:
```bash
chmod +x build.sh
./build.sh
```
The output Magisk module ZIP (`BFR-WEBUI-Magisk-v1.2.2-local.zip`) will be generated at the workspace root, ready to flash via Magisk, KernelSU, or APatch manager.

### Cross-Compiling Standalone Binary
To compile the Android ARM64 binary directly:
```powershell
cd frontend
npm run build
cd ..
$env:CGO_ENABLED="0"; $env:GOOS="android"; $env:GOARCH="arm64"; go build -ldflags "-s -w" -o webui .
```

---

## 🔒 Security Architecture

- **Session Authentication**: Cookie-based authentication with `HttpOnly`, `SameSite=Lax`, and double-submit CSRF tokens.
- **Brute-Force Shield**: Per-IP rate limiter enforcing a maximum of 5 failed attempts per minute window (`HTTP 429`).
- **Input Sanitization**: Strict path traversal validation on file operations (`CleanPath` & `AllowedDirs` isolation).
- **Streamlined Media Delivery**: Non-compressed media bypass on Gzip middleware preventing `Content-Length` mismatches.

---

## 📄 License & Maintainer

- **Author / Maintainer**: [latifangren](https://github.com/latifangren)
- **License**: MIT Open Source License
- **Project Repository**: [https://github.com/latifangren/BFR-WEBUI-GO](https://github.com/latifangren/BFR-WEBUI-GO)
