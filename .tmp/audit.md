# Technical Audit & Architecture Roadmap: BFR-WEBUI-GO

> **Audit Scope**: Complete backend codebase under `internal/`, `main.go`, and architecture layers.  
> **Status Update**: August 4, 2026 (Updated with Advanced Go Feature Proposals & Reordered Roadmap).

---

## 📌 Executive Summary

**BFR-WEBUI-GO** is an ultra-lightweight, high-performance Go backend operating as a Magisk / KernelSU / APatch root module for Android systems. With a compiled binary footprint of **21 MB** and a physical RAM RSS of **~30.8 MB** (Dedicated PSS ~11.4 MB), it demonstrates exceptional resource efficiency on target ARM64 Android devices.

Because it runs as a native compiled Go binary with root privileges rather than a limited shell script, it unlocks powerful system capabilities that are impossible in conventional Magisk modules.

---

## 🌟 Proposed Advanced Feature Roadmap (Next Major Releases)

Below are 5 proposed high-value features leveraging native Go concurrency and Linux syscalls placed at the top of the development roadmap:

### 1. 📡 Cellular Modem AT Controller & Band Locking (4G/5G)
- **Concept**: Direct serial interaction with Qualcomm/MediaTek modem interfaces (`/dev/ttyUSB*`, `/dev/smd*`, `/dev/atcmd*`).
- **Capability**:
  - Real-time display of advanced cellular metrics: **RSRP, RSRQ, SINR, Cell ID, Active Band**.
  - **Band Locking**: Lock specific 4G/5G LTE bands (e.g. Band 1, 3, 8, 40) directly from WebUI to force connection to the fastest BTS tower.
- **Feasibility**: High (Go handles raw serial port IO & non-blocking concurrency natively).

### 2. 💿 Dynamic USB Gadget Emulator (ISO Mount to PC & Virtual HID Input)
- **Concept**: Control Linux kernel `/sys/kernel/config/usb_gadget/` via Go state machine.
- **Capability**:
  - **DriveDroid-style ISO Mount**: Select `.iso` files in File Manager and mount the phone as a bootable USB CD-ROM / Flashdrive to PCs.
  - **Virtual USB HID Keyboard & Mouse**: Inject touch/click inputs from WebUI directly to a connected PC as a hardware USB HID device.
- **Feasibility**: High (Relies on standard Linux USB Gadget `configfs`).

### 3. 💬 Telegram Remote SMS OTP Auto-Forwarder & Dialer Listener
- **Concept**: Event-driven background monitoring of system SQLite inbox (`/data/user_de/0/com.android.providers.telephony/databases/mmssms.db`).
- **Capability**:
  - Automatically parse incoming OTP tokens, bank verification codes, and secret pins using regex.
  - Instantly forward OTP messages to your private Telegram Bot in <1 second with zero battery drain (*0% wakelock*).
- **Feasibility**: High (Go SQLite parser runs seamlessly in background worker).

### 4. 📈 Real-time Traffic DPI (Deep Packet Inspection) & App Bandwidth Throttler
- **Concept**: Tap into netfilter queues (`nfqueue`) or raw socket interfaces for packet inspection.
- **Capability**:
  - Real-time WebSocket bandwidth usage graphs grouped per application PID / UID.
  - Bandwidth throttling & rate-limiting per application or target IP address.
- **Feasibility**: Moderate (Requires `iptables` / `nftables` queue hooks).

### 5. ⚡ Event-driven Smart Thermal & System Governor Tuning
- **Concept**: Event-driven CPU governor switching based on screen touch inputs (`/dev/input/`) or rendering frame rates (`surfaceflinger`).
- **Capability**:
  - Dynamic CPU boost on screen touch, instant governor scaling down on idle without polling timer loops.
  - Zero wakelock thermal management.
- **Feasibility**: High (Go event channel listener on `/dev/input/event*`).

---

## 🛠️ Technical Audit & Performance Roadmap (v1.3.0+)

Below are the remaining technical optimization items from the codebase audit:

| Priority | Category | Target File | Actionable Target |
| :--- | :--- | :--- | :--- |
| 🔴 **HIGH** | **Performance** | `internal/charger/charger.go:172` | Replace `su -c "test -w"` shell commands with native `unix.Access` / `os.Stat`. |
| 🔴 **HIGH** | **Resilience** | `internal/worker/worker.go` | Add universal `defer recover()` panic safety wrappers to background workers. |
| 🔴 **HIGH** | **Performance** | `internal/sysinfo/sysinfo.go` | Implement a 500ms-1s in-memory TTL cache for `/proc` reads. |
| 🟡 **MEDIUM** | **Adaptability** | `internal/charger/charger.go` | Implement dynamic `/sys/class/power_supply/` attribute scanner. |
| 🟡 **MEDIUM** | **Modularity** | `internal/handlers/*` | Refactor handlers to use Struct Dependency Injection. |
| 🟡 **MEDIUM** | **Performance** | `internal/speedtest/speedtest.go` | Implement `sync.Pool` buffer recycling to optimize GC during speed tests. |

---

## ✅ Completed Features & Historical Roadmap (v1.2.0)

All features below have been fully implemented, verified, and deployed:

- [x] **Persistent Storage Directory (`/data/adb/bfr_webui_go/data/`)**: All module configs (`auth.json`, `speedtest_history.json`, `telegram_config.json`, `charger_config.json`, `ssh_config.json`, `cloud_config.json`, `tweaks_config.json`, `vnstat_data.json`) saved in persistent storage outside module path with auto-migration.
- [x] **Encrypted Password Management & Change Password API**: Salted SHA-256 password hashing, `POST /api/auth/change-password`, `GET /api/auth/status`, and adaptive login quick-fill badge (`Default: bfr` vs `🔒 Password Kustom Aktif`).
- [x] **Mobile Responsive Navigation System**: Candidate A 5-column grid bottom category bar (`grid grid-cols-5`) and Slide-Up Bottom Sheet Modal (`z-[999]`).
- [x] **Collapsible Navigation Toggle (`🧭 Nav`)**: Slide-down bottom navbar with floating pill toggle button for 100% full-screen mobile view.
- [x] **Tabbed Settings Modal (3-Tab System)**: Compact 3-tab Settings dialog (📦 **Backup & Restore**, 🔐 **Security**, ☁️ **Cloud Sync**) with a bottom `Close ✕` exit button.
- [x] **URL Hash Tab Persistence & Hardware Back Button Support**: Enabled `#tab` hash navigation sync (`#files`, `#terminal`, `#proxy`, etc.) so browser refreshes stay on the active tab and Android hardware Back button navigates between previous tabs smoothly.
- [x] **File Manager & Storage Usage Fixes**: Fixed `storageInfo.used_pct` mapping for 100% accurate amber/green storage usage bar, added responsive table horizontal scroll container, fixed action button text wrapping.
- [x] **Login Page Redesign (Candidate 1)**: Neo-Brutalist Glassmorphism login card with device badge, show/hide password eye toggle, and local offline inline SVG social links (Telegram, Facebook, GitHub).
- [x] **PWA Service Worker Cache Buster (`sw.js`)**: Versioned cache `bfr-webui-v1.2.0-b4` with Network-First HTML template fetching.
- [x] **Native Speedtest Engine & WebDAV Cloud Sync**: Multi-threaded Go speedtest engine with Ookla-style IP/ISP/Colo trace and WebDAV archive sync.
- [x] **LuCI OpenWrt Category Dropdown Navigation**: 5-category desktop navigation bar (**Status ▾**, **System ▾**, **Services ▾**, **Network ▾**, **Extras ▾**).

---

*Report updated for `BFR-WEBUI-GO` roadmap organization.*
