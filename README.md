# BFR-WEBUI-GO PRO

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Alpine.js](https://img.shields.io/badge/Alpine.js-3.x-2F855A?style=for-the-badge&logo=alpine.js&logoColor=white)](https://alpinejs.dev)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.x-38B2AC?style=for-the-badge&logo=tailwind-css&logoColor=white)](https://tailwindcss.com)
[![Magisk](https://img.shields.io/badge/Magisk-Supported-green?style=for-the-badge&logo=android&logoColor=white)](https://github.com/topjohnwu/Magisk)
[![KernelSU](https://img.shields.io/badge/KernelSU-Supported-red?style=for-the-badge)](https://github.com/tiann/KernelSU)
[![APatch](https://img.shields.io/badge/APatch-Supported-blue?style=for-the-badge)](https://github.com/bmaxiyi/APatch)

High-performance, ultra-lightweight, 100% offline-ready Android System Control Panel & WebUI designed to run as a Magisk, KernelSU, or APatch module. It compiles into a single standalone Go binary, packaging frontend resources inside a tiny memory footprint.

---

## ✨ Features Highlight

### ⚡ Ultra-lightweight Single Binary
- Single compiled binary (~10MB) embedded with all assets.
- 100% offline ready with zero runtime dependencies or CDN connections.
- Highly optimized Go daemon with minimal CPU & RAM footprint (~12MB).

### 🎨 PRO Neobrutalism UI & Dark Mode
- **Atmospheric Palette**: High-contrast, Neobrutalist design supporting light mode and Amoled dark mode.
- **Yoinks-style PRO Badge**: Crisp black text on a gold accent badge.
- **Floating Toast Notifications**: In-app feedback system for all background API actions (saving settings, file actions, Core controls).
- **Custom Confirmation Modals**: Seamless in-app dialog confirm overlays, eliminating clunky native browser popups.

### 📶 Network & Carrier Optimization
- **Carrier Aggregation (LTE-A / 5G)**: Optimizations to lock carrier aggregation (`gsm.lte.ca.support=1`, `persist.radio.lte_ca=1`).
- **TCP Congestion Control**: BBR / BBR2 TCP Congestion options with sysctl buffer allocations.
- **Interactive TTL Spoofing**: Numeric TTL input control (1-255) with separate **Activate** and **Disable** actions to bypass carrier hotspot limits.
- **Multi-Core RPS (Receive Packet Steering)**: Distributes network packet traffic workloads. Includes a dynamic CPU core bitmask guide:
  - `0F` = Core 0-3 (Efficiency)
  - `F0` = Core 4-7 (Performance)
  - `FF` = All Cores (Max performance)
  - `00` = Disabled
- **One-Click DNS Switcher & Live Display**: Switch resolvers instantly + displays the active system DNS addresses (`net.dns1`/`net.dns2`) in real-time.
- **Built-in Ping Diagnostics**: Telemetry utility to test network connectivity and host latency.

### 💻 SELinux-Immune Web Terminal
- Bi-directional interactive root shell (spawning `/system/bin/sh` or target shell with `su` context) using xterm.js via WebSockets.
- Seamless terminal session management and PTY fallback mechanisms.

### 📂 Enhanced Web File Manager
- **Hierarchical Navigation**: Direct folder navigation featuring a parent directory directory link (`📁 ..`).
- **Dynamic Shortcuts**: Dynamically populated path bookmark shortcuts stored in client `localStorage`.
- **Full Scope Operations**: Support for **+ New File**, **+ New Folder**, **Rename**, **Edit** (with code editor modal), **Upload**, **Download**, and **Delete**.

### 📱 SMS OTP Viewer (Android 16 Ready)
- Telephony DB parser reading SQLite database directly or falling back to `content query --uri content://sms` to ensure compatibility with Android 10 to Android 16.
- **Auto OTP Highlight**: Soft yellow visual highlighting of numeric tokens and banking authentication codes.

### 🖥️ Remote Screen Projection (Scrcpy WS)
- Dynamic display projection streaming layout frame loops via WebSocket directly to a canvas.
- Fully injected control signals supporting touch movements, clicks, and physical key presses (Back, Home, Recents, Vol+/Vol-).

### 🔋 Smart Battery Charger Control
- Battery lifespan protection limiting power supplies when charge levels reach designated threshold inputs (e.g., stop charging at 80% to avoid thermal wear).

### 📊 vnStat Bandwidth Monitor
- Captures overall network throughput metrics providing Daily, Monthly, and Weekly data consumption structures.

### ℹ️ System Info & Telemetry Dashboard
- Exhaustive system readout showing Device Model, OS release, API levels, SELinux status, Security patch level, Screen resolution, Screen density (DPI), MTU config, Default TTL, Hostname, and Linux Kernel string details.
- **Bandwidth Throughput Monitor**: Live dynamic SVG graph tracing real-time network throughput rate (Rx/Tx).

### ☕ Support & QRIS Donation Integration
- Built-in QRIS payment code display (SekarRepku Store) supporting Indonesian wallets and banking transfers.
- Integrated confirm message builder sending auto-copied confirmation details directly to Telegram or Facebook.

---

## 📦 Installation

1. Compile the tool or download the release archive `BFR-WEBUI-Magisk-v0.1.0.zip`.
2. Flash the ZIP module in Magisk Manager, KernelSU, or APatch.
3. **Restartless Daemon Upgrades**: Automatically restarts the server daemon when updated/re-flashed, meaning you do not need to reboot your phone to apply updates!
4. Default login address: `http://127.0.0.1:8080` (or `http://<your-device-ip>:8080`) with password `bfr`.

---

## ⚙️ Configuration (`tweaks.json`)

All configuration parameters and system optimization properties are isolated in `tweaks.json` in the module directory, resolving configurations dynamically at startup. By default, toggles are set to `false`, providing safe operations out-of-the-box.

---

## 📜 License & Credits

- Developer: **Latifan** ([GitHub](https://github.com/latifangren/BFR-WEBUI-GO))
- Released under the MIT License.
