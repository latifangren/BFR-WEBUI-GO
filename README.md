# BFR-WEBUI-GO

> **Ultra-Lightweight Android System Control Panel & WebUI**  
> Specially designed as a 100% offline-ready Magisk / KernelSU / APatch module. Built with a modular Go backend, Alpine.js, and a LuCI OpenWrt Bootstrap-style Category Dropdown AMOLED dark interface.

---

## 📊 Live Resource Consumption on Android

Real-time empirical benchmarks measured directly on target Android hardware (Pixel 5 ARM64):

| Resource Parameter | Measured Value | Efficiency & Notes |
| :--- | :--- | :--- |
| **Binary Disk Footprint** | **21 MB** | **100% Standalone** (Single compiled Go binary, zero Python/NodeJS dependencies) |
| **RAM Physical RSS (Total)** | **~30.8 MB** | **Ultra-Efficient** (Runs smoothly on 2GB/3GB RAM Android devices) |
| **RAM Dedicated (Private PSS)** | **~11.4 MB** | Extremely low memory footprint |
| **Idle CPU Usage** | **0.0% CPU** | Zero CPU overhead in background |
| **Swap Memory Usage** | **0 KB** | Zero flash memory wear |

---

## ⚡ Key Features

- **Offline-Ready & Tiny Footprint**: Runs as a single compiled Go binary with extremely low memory usage (~11MB PSS / ~30MB RSS).
- **LuCI OpenWrt Category Dropdown Navigation**: Clean, 5-category dropdown header navigation navbar (**Status ▾**, **System ▾**, **Services ▾**, **Network ▾**, **Extras ▾**) saving over 60% header space.
- **Native Speedtest Engine (Ookla-Style Trace)**: Multi-threaded Go HTTP latency & bandwidth benchmark (Download/Upload/Ping/Jitter) with real-time Client IP, ISP/Carrier name, Location, and Server Data Center (IATA Colo) details.
- **WebDAV Cloud Backup & Auto-Sync**: Automated background sync of compressed `.tar.gz` configuration bundles (`charger`, `ssh`, `telegram`, `tweaks`) to private WebDAV cloud servers.
- **Telegram Bot Remote Management & Interactive Keyboards**:
  - Full remote control via `/start`, `/stats`, `/charger`, `/ssh`, `/proxy`, `/hotspot`, `/modules`, `/ip`, and `/reboot` commands.
  - Persistent 4-row Reply Keyboard menu buttons & interactive Inline Action buttons for single-tap execution.
  - Granular notification toggles for battery guard, overheat alert (>45°C), SSH status, IP change, and hotspot client connections.
- **Bundled Static Dropbear SSH Daemon**: Integrated prebuilt static `dropbear` ARM64 binary with automated host key generation, root password authentication (`bfr`), and full LAN binding (`0.0.0.0:2222`).
- **Universal Hardware Smart Charger Limiter**: Multi-vendor sysfs auto-scanner with Qualcomm PMIC hardware charge cutoff (`force_main_fcc` 0 mA) and custom path override support.
- **Security-First Model**: Robust input sanitization, double-submit CSS/Custom Header CSRF protection, IP-based rate limiting, and SameSite session security.
- **Proxy Core Controller**: Full daemon controller for Clash / Mihomo. Allows hot-swapping proxy modes (Rule, Global, Direct, Script) and real-time log monitoring over WebSockets.
- **Custom Android Network Tweaks**:
  - Live Sysctl tuning (TCP Congestion BBRv2, buffer optimization, core TCP properties).
  - Custom DNS configuration and iptables DNAT injection.
  - TTL & Hop Limit spoofing (supporting both IPv4 and IPv6).
  - Packet steering (RPS) auto-tuner for high-throughput mobile networks.
  - Dynamic MTU and TxQueueLen controllers.
- **Magisk / KernelSU / APatch Module Manager**: View, toggle, and install root modules (`.zip`) natively via web interfaces.
- **CPU Governor & Thermal Monitor**: Real-time core frequency gauges, thermal zone monitoring, and active CPU scaling governor switcher.
- **Live Android Logcat Live Tail**: Interactive WebSocket terminal view of system logs with level filters (Debug, Info, Warn, Error) and keyword search.
- **SoftAP & Hotspot Controller**: Full SoftAP configuration, Client DHCP/ARP lease table monitoring, and connection banning.
- **Root Web Terminal (PTY)**: Complete interactive Web-based root shell console driven by pty & WebSockets.
- **Sophisticated File Manager**: Full read, write, cut, copy, paste clipboard operations, file permissions grid (`chmod`/`chown`), search, and ZipSlip-hardened archiver.
- **PWA Support**: Fully installable PWA with service worker network-first API caching.

---

## 🌐 Documentation & Guides

Complete setup, customization, and usage guides are available in English and Indonesian:

- 🇬🇧 [English Installation & Operation Guide](./docs/INSTALLATION_EN.md)
- 🇮🇩 [Indonesian Installation Guide](./docs/INSTALLATION_ID.md)

---

## ⚙️ Environment Overrides

BFR-WEBUI-GO is fully configurable via system properties or environment flags. 

See[`env.example`](./env.example) for baseline templates:

| Variable | Default Value | Description |
|---|---|---|
| `PORT` | `8080` | HTTP WebUI server bind port |
| `BFR_PASSWORD` | `bfr` | Sign-in access password |
| `BFR_SU_BIN` | `su` | Root executor binary binary (e.g. `su`, `ksu`, `apatch`) |
| `BFR_MODULE_DIR` | `/data/adb/modules/bfr_webui_go` | Base module install path |
| `BFR_BOX_BASE` | `/data/adb/box` | Box framework base path |
| `BFR_CLASH_API` | `http://127.0.0.1:9090` | Clash API controller address |
| `BFR_LEASES_FILE` | `/data/misc/dhcp/dnsmasq.leases` | Hotspot leases file path |
| `BFR_ALLOWED_DIRS` | `/sdcard,/storage,/data/adb...` | FileManager path boundaries |

---

## 🛠️ Building From Source

To compile the target binary for Android ARM64 environments manually:

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o webui main.go
```

---

## 📄 License

This project is licensed under the terms of the MIT License.
