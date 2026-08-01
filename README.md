# BFR-WEBUI-GO

> **Ultra-Lightweight Android System Control Panel & WebUI**  
> Specially designed as a 100% offline-ready Magisk / KernelSU / APatch module. Built with a modular Go backend, Alpine.js, and a Neo-Brutalist Tailwind CSS dark AMOLED interface.

---

## ⚡ Key Features

- **Offline-Ready & Tiny Footprint**: Runs as a single compiled Go binary (~10MB) with extremely low memory usage (~15MB RAM RSS).
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
- **System Backups**: One-click configuration export and import backup (.json) bundle.
- **SoftAP & Hotspot Controller**: Full SoftAP configuration, Client DHCP/ARP lease table monitoring, and connection banning.
- **Smart Battery Charger Tuning**: Auto-control of battery thresholds via sysfs charger overrides.
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
