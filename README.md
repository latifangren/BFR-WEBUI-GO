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

## ⚡ Feature Highlights

<details>
<summary><b>✨ Click to expand / collapse all project features</b></summary>
<br>

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

### 🚀 Kernel, Power & Hardware Controls
- **Dynamic Hardware Charge Limiter**: Multi-vendor sysfs auto-scanner with Qualcomm PMIC hardware bypass (`force_main_fcc` 0 mA) and custom sysfs override support.
- **Automated System Reboot Scheduler**: Dual-mode scheduler (`Uptime Interval` from boot or `Specific Time Daily`) with live countdown timer and instant power controls (`Reboot`, `Recovery`, `Bootloader`, `Power Off`).
- **SoC & Governor Tuner**: Multi-cluster CPU scaling, governor policies, and real-time frequency distribution tracking.
- **Root Daemon Services Telemetry**: Real-time PID, multi-core normalized CPU %, and resident RAM (MB/KB) tracking for `webui`, `mihomo`, `dropbear`, and `adbd`.

### 📶 Baseband Modem, Carrier Aggregation & Cell Locking
- **Multi-Tier RAT Locking (Android 7–15)**: Strict **4G LTE Only** via native binary bitmasks eliminating 3G/GSM fallback leaks, dedicated **5G & 4G Dual Lock** mode, and graceful legacy fallback.
- **eNodeB Tower Cell Locking**: Hardware-level cell tower locking (`EARFCN` + `PCI`) with multi-CA Primary Component Carrier (PCC) awareness and persistent storage across reboots (`cell_lock.json`).
- **Tower Cell Scanner**: Real-time discovery of serving, secondary CA, and neighbor cell towers with live signal power metrics (RSRP, RSRQ, RSSI, SINR).
- **Proprietary Vendor Baseband Engines**: Full private build support for Qualcomm QRTR/DIAG/EFS2, Samsung SecRIL (`band_manager.dex`), and Tensor Shannon modems.

### 🛡️ Box for Magisk (BFM) Proxy Suite
- **Multi-Core Daemon Controller**: Unified manager for `mihomo`, `sing-box`, and `clash` with real-time memory RSS tracking, lifecycle controls, and watchdog auto-restart.
- **Transparent Routing Modes**: Support for `TPROXY`, `REDIRECT`, `TUN`, and `MIXED` with IPv6 transparent redirection and UDP 443 QUIC blocking.
- **Per-App Routing Matrix**: Interactive Blacklist vs Whitelist policy editor with instant search across installed 3rd-party Android user applications.
- **Hotspot / Tethering Proxying**: Direct Wi-Fi and USB tethered client transparent interception via `ap.list.cfg`.
- **Remote Subscription & GeoX Synchronizer**: Instant subscription URL updater, automated GeoIP (`Country.mmdb`) and GeoSite sync, with direct links to upstream MetaCubeX releases.
- **File Manager Integration**: 1-click shortcut from Proxy tab directly into `/data/adb/box` in BFR File Manager.

### 🌐 Networking & Diagnostics
- **Persistent Network Settings**: Persistent Dynamic TTL modifier (bypassing carrier hotspot quota limits) and Custom DNS resolver (Cloudflare, Google, AdGuard, Quad9) saved in `network_config.json`.
- **DNS Reset to Default**: 1-click flush to cleanly purge custom iptables NAT redirections and restore carrier/DHCP resolution.
- **Hardened Ping Diagnostics**: Multi-platform ping engine with 7-second timeout context, packet loss resilience, and root ICMP execution fallback.
- **Router-Grade QoS Bandwidth Manager**: 1-click router traffic shaping profiles (*Gaming Anti-Lag*, *Fair Share*, *Quota Saver*) via native Linux TC HTB and ingress policing.
- **VnStat Traffic Accounting**: Real-time interface bandwidth rate meters, daily/monthly consumption charts, and billing cycle quota trackers.

### 📺 Remote Management, Screen & Shell
- **Interactive Web Terminal & Scrcpy Screen Mirror**: Full PTY root terminal over WebSockets (`xterm.js`) and low-latency H.264 canvas screen mirror with gesture touch injection.
- **Bundled Dropbear SSH**: Static ARM64 daemon with automated host-key generation and root authentication (`bfr`).
- **Interactive Telegram Bot**: Remote commands (`/stats`, `/charger`, `/ssh`, `/proxy`, `/reboot`), keyboard menus, and instant security push alerts (overheat, battery, IP change, SSH login).
- **Interactive Supporter Hub**: Pokemon TCG-style floating idle card and inspection zoom modal for QRIS supporter contributions with 1-click image download.

</details>
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

## 📥 Installation

1. Download `BFR-WEBUI-Magisk-v1.2.7.zip` from the [Releases](https://github.com/latifangren/BFR-WEBUI-GO-release/releases) page.
2. Open the **Magisk / KernelSU / APatch** app -> navigate to the **Modules** tab -> select **Install from storage** -> choose the downloaded ZIP file -> wait for the installation process to complete -> **Reboot** your device.
3. Open a browser on any device connected to the same Wi-Fi or hotspot network and navigate to:
   - **HTTP**: `http://<DEVICE-IP>` (standard port 80 default)
   - **HTTPS**: `https://<DEVICE-IP>` (standard port 443 default, direct access without extra port)

---

## 📺 Scrcpy Remote Screen & HTTPS Notice

> ⚠️ **IMPORTANT: HTTPS (Secure Context) Required for Scrcpy WebCodecs**

- The **Scrcpy Remote Screen Control** feature utilizes the modern browser **WebCodecs API** for ultra-low latency H.264 video decoding.
- Modern browsers (Google Chrome, Microsoft Edge, Mozilla Firefox) require a **Secure Context** (`window.isSecureContext`) for WebCodecs. When accessing via plain HTTP (`http://<DEVICE-IP>`), browsers automatically block the video hardware decoder, resulting in a **blank black screen**.
- **Solution**:
  1. Access the WebUI via **HTTPS**: `https://<DEVICE-IP>` (native HTTPS runs on standard port 443, so simply type the URL directly without specifying any port).
  2. Since BFR-WEBUI-GO generates a self-signed SSL certificate automatically (`internal/tlsgen/`), your browser will present a security warning on the first visit. Click **"Advanced"** and select **"Proceed to <DEVICE-IP> (unsafe)"** to continue.
  3. *(Chromium alternative)*: Add `http://<DEVICE-IP>` to `chrome://flags/#unsafely-treat-insecure-origin-as-secure` and restart the browser.

---

## 📦 Building & Packaging (Dual Build Flavors)

`BFR-WEBUI-GO` supports two official build flavors:
1. **Full Private Flavor (`dev` branch)**: Includes all baseband diagnostics, Qualcomm QRTR/DIAG/EFS2, Samsung SecRIL, and Google Tensor Shannon tools.
2. **Public Core Flavor (`release/no-modem` branch)**: 100% zero proprietary vendor baseband code or strings, retaining generic Cellular & Radio Telemetry (`dumpsys`, SIM routing, 5G thermals) and SMS.

### Prerequisites
- **Go**: Version 1.22 or higher
- **Bun or Node.js**: Bun (recommended) or Node.js 18+ (pnpm/npm)

### One-Click Build & Packaging
The build scripts automatically detect your active Git branch and compile with the appropriate build tags:
- On Windows:
  ```cmd
  :: Auto-detects branch (or pass 'full' / 'nomodem')
  build_zip.bat
  ```
- On Linux / macOS / Android Termux:
  ```bash
  chmod +x build.sh
  ./build.sh
  ```

### Manual Cross-Compilation
- **Full Private Build**:
  ```bash
  cd frontend && bun run build && cd ..
  CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags "-s -w" -o webui .
  ```
- **Public Clean Non-Modem Build**:
  ```bash
  cd frontend && bun run build && cd ..
  CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -tags nomodem -ldflags "-s -w" -o webui .
  ```

For detailed architectural isolation specs and zero-leak forensic verification, see [`docs/MODEM-ISOLATION.md`](docs/MODEM-ISOLATION.md).
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
- **Project Repository**: [https://github.com/latifangren/BFR-WEBUI-GO-release](https://github.com/latifangren/BFR-WEBUI-GO-release)
