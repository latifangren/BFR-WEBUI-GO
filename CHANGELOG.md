# Changelog

All notable changes to the **BFR-WEBUI-GO** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

## [1.2.1] - 2026-08-26

### Changed
- **Default HTTP Server Port Change**: Changed default WebUI server HTTP bind port from `8080` to `80` across server flag fallbacks, sysinfo checkers, tunnel forwarders, shell setup scripts, and system documentation.

### Fixed
- **Navbar Dropdown Submenu Stacking Context in Modern Clean Style**: Fixed desktop navigation category dropdown submenu clipping where main content cards overlay dropdown menus due to missing z-index rules in precompiled Tailwind build. Added CSS z-index governance rules in `base.css` (`header` at `z-index: 1000`, `nav div.absolute` at `z-index: 1050`, `.nav-dropdown` at `z-index: 1060`) and static utility classes `.z-60` to `.z-999`.
- **Dropdown Hover Disappear Fix**: Eliminated `button:hover` `transform: translateY(-1px)` positional shift on category trigger buttons in Modern Clean style to prevent premature Alpine `@mouseleave` events.
- **SQM / CAKE Engine Badge Contrast**: Fixed dark mode text contrast on SQM / CAKE status badges in QoS tab.
- **English Translation Standardization**: Standardized all UI texts, modal dialogs, and error messages to English.

### Added
- **Modem Signal Quality Metrics & Percentage Calculations**: Added EARFCN parsing, signal percentages (`RSRPPct`, `RSRQPct`, `SINRPct`), and quality rating indicators (`QualityRSRP`, `QualityRSRQ`, `QualitySINR`) to cellular modem status engine.
- **Multi-Preset Color Theme Architecture & Appearance Options Modal**: Added support for 7 preset color themes (**Dark**, **Light**, **Dracula**, **Nord**, **Cyberpunk**, **Emerald**, **Sunset**) managed via `data-theme="..."` CSS variables on `<html>`. Replaced separate theme/style buttons in Header with a unified **🎨 Appearance** modal dialog featuring real-time component previews.
- **Dynamic System Properties via `tweaks.json`**: Converted `system.prop` into a clean placeholder file to prevent Magisk/KernelSU from forcing static boot properties (`ro.telephony.default_network`, `dalvik.vm.*`, `net.tcp.buffersize.*`) when network tweaks are set to `false`. All optimizations are now dynamically driven 100% by `tweaks.json` & WebUI toggles.

### Added
- **Modular UI Theme Architecture (Neobrutal vs Modern Clean)**:
  - Added support for 2 UI design paradigms: **⚡ Neobrutal** (bold 2px borders, 4px hard offset shadows, sharp corners) and **✨ Modern Clean** (soft drop shadows, smooth `rounded-xl` corners, subtle 1px borders, glassmorphism backdrop blur).
  - Split CSS into modular structure (`web/static/css/base.css`, `web/static/css/styles/neobrutal.css`, `web/static/css/styles/modern.css`, and `web/static/css/style.css`).
  - Added UI Style switcher toggle in Header (`web/templates/layout/header.html`) and persistent `localStorage.setItem('uiStyle', ...)` state manager in `web/static/js/modules/common/common.js`.
  - Added anti-FOUC inline script in `web/index.html` to prevent theme flicker on page load.
- **1-Click Auto Install Daemon Binary Remote Tunnel**: Penambahan fungsi pengunduhan biner persisten ARM64 resmi (`cloudflared`, `tailscaled`, `zerotier-one`) langsung dari WebUI dengan hak akses eksekusi `0755` serta UI indikator status biner interaktif.
- **Dedicated Remote Access Tunnel Tab (`internal/tunnel`)**: Dukungan Hybrid Tunnel (Cloudflare Quick Tunnel / Token, Tailscale, ZeroTier) dengan deteksi biner dinamis, status URL publik real-time, dan kontrol via API `/api/tunnel/*`.
- **Dedicated Local NAS Lite & Media Share Tab (`internal/nas`)**: Native Go WebDAV & HTTP File Sharing Server dengan opsi Basic Auth, Mode Read-Only, kalkulasi kapasitas memori internal/SD Card/USB OTG, dan kontrol via API `/api/nas/*`.
- **Telegram SMS Auto-Forwarder (OTP & Kuota)**: Pilihan auto-forward SMS masuk ke chat Telegram, perintah bot `/sms`, filter OTP-only (`\b\d{4,8}\b`), dan keyword custom filter.
- **Hotspot MAC Address Filtering & Access Rules**: Mode Whitelist & Blacklist menggunakan `iptables -m mac --mac-source` dengan aksi instant block dari tabel client Hotspot.
- **Comprehensive Unit Test Suite**: Added 20 unit test files covering 100% of internal packages (`auth`, `charger`, `config`, `handlers`, `hotspot`, `modem`, `modules`, `network`, `power`, `qos`, `scrcpy`, `smsviewer`, `sysinfo`, `telegram`, `terminal`, `vnstat`). Fully verified with 100% pass rate and Android `arm64` cross-compilation compatibility.
- **Dedicated QoS Bandwidth Control & Traffic Prioritization Tab (`internal/qos`)**:
  - Implemented dedicated **QoS Control** UI tab with Neo-Brutalist / AMOLED Dark styling.
  - Built Hybrid QoS Engine with high-precision `tc HTB/IFB` qdisc bandwidth shaper and automatic fallback to `iptables Mangle / MARK`.
  - Added per-client bandwidth limits with quick presets (`Gaming` 5M, `Stream` 25M, `Browsing` 10M, `Strict` 2M, `Unlimited`) and custom Download/Upload Mbps caps.
  - Implemented DSCP/TOS traffic classification for Low-Latency Gaming (ICMP/UDP) and VoIP/Streaming.
  - Added Time-based Bandwidth Scheduler for automated peak-hours throttling.
  - Added persistent state saving (`qos.json`) and automatic background QoS rule application during server boot in `main.go`.
- **Dedicated Cellular Modem & Band Locking Tab (`internal/modem`)**:
  - Implemented dedicated **Modem & Band Lock** UI tab featuring real-time signal gauges and a Neo-Brutalist AT terminal interface.
  - Built Hybrid Multi-Engine band locking supporting Universal Android Framework (`cmd phone lte-set-band-mode`), Qualcomm Snapdragon AT Serial Direct (`/dev/smd11`, `/dev/ttyUSB*`, `/dev/atcmd*`), and Vendor Secret Code intent hooks.
  - Built Hex Bitmask Auto-Calculator for LTE & 5G NR band combinations (`0x8000000005`, etc.) with manual Hex override and copy functionality.
  - Built Interactive AT Command Console with live serial response stream (`OK`/`ERROR`) and built-in AT presets (`AT+CPSI?`, `AT+QENG="servingcell"`, `AT+QCAINFO`, `AT^SYSCONFIG`, `AT+EGMR=1,7,"..."`).
  - Added real-time cellular signal metrics parser (`RSRP`, `RSRQ`, `SINR`, `RSSI`, Operator, Band, Cell ID, TAC, PCI, EARFCN).
- **Default Network Tweaks Reset**: Updated `internal/network/tweaks.json` to disable all initial network optimization flags by default.
- **Authentication IP Rate Limiting**: Implemented per-IP rate limiter in `Manager.Authenticate()` allowing a maximum of 5 failed login attempts per minute window to prevent brute-force attacks (`HTTP 429`).
- **Sysctl Initial Defaults Backup & Restoration**: Added `BackupSysctlDefaults()` and `RestoreSysctlDefaults()` in `internal/network/tweaks.go` (`POST /api/network/tweaks/restore` and `♻️ Restore Original Sysctl Defaults` button) to restore initial pre-tune sysctl parameters and reset persistent `tweaks.json` state.
- **Per-Core CPU Frequency & Usage Diagnostics**: Added per-core CPU frequency reading from sysfs (`scaling_cur_freq`) and real-time CPU core usage badge grid rendering in `tab_sysinfo.html`.

### Improved
- **HTTP Gzip Response Compression & Asset Caching**: Added Gzip response writer middleware with lazy initialization and `Flush()` support for SSE streams, alongside `Cache-Control` static asset headers.
- **Non-blocking Sysinfo Background Ticker & Static Hardware Caching**: Implemented 1.5s background ticker loop updating `cachedStats` with non-blocking `RLock()` reads, cached static hardware info (`wm size`, `wm density`, model, SDK/kernel versions) via `sync.Once`, and replaced `df` subprocesses with native `syscall.Statfs`.
- **WebSocket Terminal Security & Token Verification**: Replaced presence-only cookie checks in `HandleWebsocket()` with full session token validation (`ValidateSession`) and Origin host verification (`EqualFold`).
- **Network Tweaks Batching & Dynamic RAM TCP Buffer Scaling**: Batched all root shell executions in `ApplyAllTweaks()` into a single `su -c` command chain and scaled TCP max buffers based on physical RAM capacity (`<4GB` RAM capped at 16MB max buffer).
- **Subshell Elimination & Dynamic Sysfs Scanner** (`internal/charger/charger.go`): Replaced wasteful `su test -w` subshells with native Go `os.OpenFile` checks and added dynamic fallback sysfs scanner (`/sys/class/power_supply/*/`) matching keywords (`charging`, `suspend`, `limit`, `fcc`, `store_mode`, `switch`).
- **Universal Worker Panic Recovery** (`internal/worker/worker.go`): Added panic recovery handler to background worker pool and fallback goroutines.
- **In-Memory TTL Cache for `/proc` Reads** (`internal/sysinfo/sysinfo.go`): Implemented 750ms TTL in-memory cache for `GetStats()` using `sync.RWMutex` to eliminate unnecessary disk and `/proc` I/O.
- **Dynamic `$PATH` & Module Binary Scanner** (`internal/proxy/proxy.go`): Added dynamic `$PATH` lookup (`exec.LookPath`) and Magisk/KSU module directory scanner (`/data/adb/modules/*/bin/{mihomo,clash}`) alongside static core paths.
- **Speedtest Engine Buffer Recycling** (`internal/speedtest/speedtest.go`): Implemented `sync.Pool` for 32KB buffer recycling in download/upload multi-worker loops to eliminate GC allocations.
- **Service Panic Safety** (`internal/telegram/bot.go`, `internal/ssh/ssh.go`): Added `defer recover()` panic recovery handlers to Telegram polling, notification workers, and SSH daemon control loops.

## [1.2.0] - 2026-08-03

### Added
- **Mobile Responsive Navigation & Slide-Up Bottom Sheet System**:
  - Implemented 5-column grid Fixed Bottom Navigation Category Bar (`grid grid-cols-5`) and Slide-Up Bottom Sheet Modal for mobile devices.
  - Added Collapsible Navigation Toggle with Floating Pill Button (`🧭 Nav`) for 100% full-screen mobile viewing.
- **Tabbed Settings Modal (3-Tab System)**:
  - Redesigned monolithic Settings Modal into a compact 3-tab dialog (📦 **Backup & Restore**, 🔐 **Security**, ☁️ **Cloud Sync**) with a bottom `Close ✕` exit button, preventing content clipping on mobile and desktop screens.
- **Persistent Storage Directory (`/data/adb/bfr_webui_go/data/`)**:
  - Migrated storage of all module configs (`auth.json`, `speedtest_history.json`, `telegram_config.json`, `charger_config.json`, `ssh_config.json`, `cloud_config.json`, `tweaks_config.json`, `vnstat_data.json`) to persistent directory outside module path (`/data/adb/bfr_webui_go/data/`) to prevent data loss on module updates with automatic legacy file migration.
- **Encrypted Password Management & Change Password API**:
  - Implemented SHA-256 salted password hashing, `POST /api/auth/change-password`, `GET /api/auth/status` with `is_default_pass`, and dynamic Quick-Fill badge adaptation on the login page (`Default: bfr` vs `🔒 Password Kustom Aktif`).
- **URL Hash Tab Persistence & Hardware Back Button Support**:
  - Enabled `#tab` hash navigation sync (`#files`, `#terminal`, `#proxy`, etc.) so browser refreshes stay on the active tab and Android hardware Back button navigates between previous tabs smoothly.
- **File Manager & Storage Usage Fixes**:
  - Fixed `storageInfo.used_pct` mapping for 100% accurate amber/green storage usage bar, added responsive table horizontal scroll container, fixed action button text wrapping.
- **Login Page Redesign (Candidate 1)**:
  - Neo-Brutalist Glassmorphism login card with device badge, show/hide password eye toggle, and local offline inline SVG social links (Telegram, Facebook, GitHub).
- **SMS Viewer Mobile Card Layout**:
  - Redesigned SMS message items into mobile-friendly stacked cards with clean word wrapping.
- **Persistent Storage Directory & Password Hashing Management**:
  - Added central persistent data directory helper (`GetPersistentDataDir` & `GetPersistentFilePath`) targeting `/data/adb/bfr_webui_go/data` with automatic legacy config migration across `auth.go`, `charger.go`, `ssh.go`, `bot.go`, `cloud.go`, `speedtest.go`, and `vnstat.go`.
  - Added secure SHA-256 + Salt password hashing and persistent `auth.json` configuration in `internal/auth/auth.go`.
  - Implemented `IsDefaultPassword()` indicator and `ChangePassword(currentPass, newPass)` backend manager.
  - Added REST API endpoints `GET /api/auth/status` (returning `authenticated` and `is_default_pass`) and `POST /api/auth/change-password` (`/api/auth/change-password`).
  - Implemented multi-threaded Go network speedtest engine (`internal/speedtest/speedtest.go`) supporting concurrent Ping/Jitter probes, multi-worker HTTP GET download, and multi-worker HTTP POST upload speed testing with live progress tracking (`/api/speedtest/start`, `/api/speedtest/status`, `/api/speedtest/stop`, `/api/speedtest/history`).
  - Added Ookla-style Client IP, ISP/Carrier name, Location, and Server Data Center / IATA Colo mapping (`fetchClientAndISPInfo`) parsing Cloudflare `/cdn-cgi/trace`, `ip-api.com`, and `ipinfo.io` with fallback handlers.
  - Added `getSpeedtestHTTPClient` with custom `net.Resolver` fallback to public DNS (`1.1.1.1:53`, `8.8.8.8:53`, `9.9.9.9:53`) and `InsecureSkipVerify: true` to bypass Android IPv6 loopback (`[::1]:53`) DNS failures and TLS verification blocks.
  - Added fallback speedtest endpoints across Cloudflare (`https://1.1.1.1/cdn-cgi/trace`, `https://speed.cloudflare.com/__down?bytes=0`, `http://1.1.1.1/`), OVH (`http://proof.ovh.net/files/10Mb.dat`), and Hetzner (`http://speed.hetzner.de/10MB.bin`).
  - Added `HistoryEntry` storage in `speedtest.go` (max 20 entries) and fixed frontend JSON field mapping (`progress_pct`, `ping_ms`, `jitter_ms`, `download_mbps`, `upload_mbps`, `client_ip`, `isp`, `location`, `server_colo`, `server_name`) with auto-polling timer initialization on test start.
  - Implemented WebDAV Cloud Backup manager (`internal/backup/cloud.go`) supporting compressed `.tar.gz` archive generation and automated periodic PUT sync with WebDAV servers (`/api/backup/cloud/config`, `/api/backup/cloud/sync`).
  - Redesigned top navigation bar (`web/templates/layout/sidebar.html` & `web/templates/layout/header.html`) into 5 categorized dropdown menus: **Status ▾** (`overview`, `sysinfo`, `logs`), **System ▾** (`files`, `terminal`, `tools`), **Services ▾** (`telegram`, `charger`, `ssh`, `scrcpy`), **Network ▾** (`network`, `proxy`), and **Extras ▾** (`sms`, `about`).
  - Added smooth Alpine.js dropdown interaction (`x-data="{ open: false }"`, `@mouseenter`, `@mouseleave`, `@click.outside`) with visual active indicators matching AMOLED dark theme aesthetic.
  - Saved over 60% header space for clean layout scaling while maintaining 100% compatibility with existing tab IDs and backend APIs.
- **Universal Hardware Smart Charger Limiter & Custom Path Override**:
  - Expanded sysfs auto-scanner (`internal/charger/charger.go`) to detect Qualcomm PMIC hardware charge cutoff nodes (`/sys/class/power_supply/main/force_main_fcc`, `force_main_icl`), successfully cutting off charging current (`0 mA`) on devices like Google Pixel 5.
  - Added support for Google Pixel 5 / Tensor `charge_limit` percentage threshold nodes and expanded candidate sysfs paths across Samsung, Xiaomi, OnePlus, OPPO, Realme, ASUS ROG, and MediaTek devices.
  - Added `custom_path` override configuration in backend and UI to allow manual sysfs node specification if auto-scan misses vendor-specific nodes.
  - Added `EXPERIMENTAL` badge and kernel hardware dependency note in WebUI charger control card (`web/templates/charger/tab_charger.html`).
- **Telegram Bot Remote Management & System Alerts**:
  - Implemented bidirectional Telegram Bot daemon (`internal/telegram/bot.go`) with `/start`, `/stats`, `/charger`, `/ssh`, `/proxy`, `/hotspot`, `/modules`, `/ip`, `/reboot`, `/tweak`, and `/cmd` commands.
  - Added persistent custom Reply Keyboard menu (`ReplyKeyboardMarkup`) for single-tap navigation and interactive Inline Keyboards (`InlineKeyboardMarkup`) with callback queries for quick actions (`/charger`, `/ssh`, `/proxy`, `/reboot`).
  - Added granular notification toggles (`battery_guard`, `battery_overheat`, `ssh_status`, `ip_change`, `hotspot_client`) in `NotificationConfig` and background notification tickers for real-time alerts.
  - Added notification settings checkboxes under Telegram Bot card in WebUI Tools tab (`web/templates/tools/tab_tools.html`) and frontend state manager (`web/static/js/modules/telegram/telegram.js`).
  - Added custom Go `net.Resolver` fallback to public DNS (`1.1.1.1:53`, `8.8.8.8:53`, `9.9.9.9:53`) to bypass Android IPv6 loopback (`[::1]:53`) DNS lookup failures.
  - Added automated Telegram startup notifications on boot.
- **Bundled Static Dropbear SSH Daemon**:
  - Bundled standalone static `dropbear` and `dropbearkey` ARM64 binaries (`bin/arm64/`), removing reliance on external Termux or stock ROM SSH binaries.
  - Added automated host key generation (`/data/ssh/dropbear_ecdsa_host_key`).
  - Added root password authentication support (`bfr`) via automated `/data/ssh/passwd` mount bind to `/etc/passwd` with valid MD5 crypt hash and `/bin/sh` shell setup.
  - Added full LAN binding (`0.0.0.0:2222`) support and auto-start SSH daemon on server boot if enabled.

### Fixed & Improved
- **Module Config Persistence Across OTA Updates**:
  - Updated `customize.sh` to preserve all `*.json` configuration files (`telegram_config.json`, `ssh_config.json`, `charger_config.json`, `tweaks.json`) during Magisk/KernelSU module updates.
  - Added UTF-8 BOM (`\xef\xbb\xbf`) prefix stripping across JSON loaders to prevent parsing errors.
  - Fixed LAN API authentication issues by setting session cookies to `SameSite=Lax`.

## [1.1.0] - 2026-08-02

### Added
- **Magisk / KernelSU / APatch Module Manager**:
  - Full module scanning API (`/api/modules`) parsing properties (`module.prop`) and handling enabled/disabled/removed states.
  - POST endpoints for toggling modules (via touch/rm `/disable`) and installing uploaded module ZIP packages (`magisk --install-module` / `ksud` / `apatch`).
  - Dark AMOLED module manager UI dropdown view in Tools with ZIP flashing dropzone.
- **CPU Governor & Thermal Control**:
  - GET `/api/sysinfo/governor` listing available CPU scaling governors, current CPU frequencies per core, and GPU/Thermal zone temperatures.
  - POST `/api/sysinfo/governor` applying selected scaling governors to all CPU cores with timeout limits and regex safety.
  - Segmented control UI panel inside System Info.
- **Live Android Logcat Stream**:
  - Live logcat tail WebSocket handler (`/api/logs/logcat/stream`) executing `logcat` subprocess and streaming data in real-time under request context.
  - Real-time logging terminal component in Logs with severity level buttons and text search filters.
- **Config Backup & Restore**:
  - Implemented `/api/backup/export` and `/api/backup/import` APIs to securely back up and restore vital system settings (charger, ssh, tweaks, proxy).
  - Gear button (`⚙️`) settings overlay and Backup/Restore dropzone modal.
- **PWA (Progressive Web App) Support**:
  - Added `web/manifest.json` with standalone display configuration, AMOLED theme colors, and icons.
  - Added `web/sw.js` (Service Worker) with offline caching strategy for static assets and network-first strategy for `/api/*`.
  - Registered Service Worker and theme color meta tags in `web/index.html`.
- **Global Toast Notification Engine**:
  - Implemented Alpine.js store-driven floating Toast notification system (`$dispatch('notify', {type, title, message})`).
- **Embedded OpenAPI 3.0 & Interactive Docs Viewer**:
  - Created complete OpenAPI 3.0 specification (`web/openapi.json`).
  - Added `/docs` endpoint serving embedded Scalar API Reference documentation viewer.
- **Root Command Executor (`ExecSuContext`)**:
  - Added `config.ExecSuContext` and `config.ExecSuTimeout` helpers with `context.WithTimeout` to prevent process deadlocks when running root commands.
- **Background Worker Task Pool**:
  - Added lightweight 2-worker goroutine queue (`internal/worker/worker.go`) for executing CPU and disk-intensive operations asynchronously.
- **Development Roadmap**:
  - Added `roadmap.md` documenting future improvements across Performance, Architecture, UX, System Integrations, and Security.

### Performance & Memory
- **Procfs Direct Parsing**:
  - Replaced subprocess `pidof` and `cat /proc` shell pipes with direct Go procfs reading (`/proc/[pid]/comm` and `/proc/[pid]/status`) to reduce CPU overhead and latency.
- **`sync.Pool` Memory Buffer Reuse**:
  - Implemented `internal/bufferpool` reusing `bytes.Buffer` and bucketed byte slices (1KB, 2KB, 32KB) across terminal PTY, Scrcpy JPEG encoding, and file copy operations.
- **Adaptive Frame Throttling for Scrcpy**:
  - Implemented `writeMux.TryLock()` frame skipping to prevent buffer accumulation when streaming over slow Wi-Fi.

---

## [1.0.0] - 2026-08-02

### Added
- **Enhanced File Manager**:
  - Modular backend refactoring (`filemanager.go`, `ops.go`, `permissions.go`, `archive.go`, `search.go`).
  - Single & batch file operations (`CopyPath`, `MovePath`, `BatchDelete`, `BatchCopy`, `BatchMove`).
  - File permissions & ownership editor (`chmod` octal modes & `chown` owner/group with recursive toggle).
  - ZIP archive compression (`CompressZip`) and extraction (`ExtractZip`) with Zip Slip protection.
  - Case-insensitive bounded file search (`SearchFiles`).
  - Partition storage usage metrics (`GetStorageUsage`).
  - Interactive UI components: Storage Usage Bar, Cut/Copy/Paste Clipboard status pill, Multi-select checkboxes, Image preview modal with keyboard navigation, and custom Octal-to-Symbolic `rwx` permission editor matrix.
- **Dynamic Gateway & DNS Detection**:
  - Active gateway detection via `ip route get 8.8.8.8` (Tier 1) with fallbacks for Android 8–16+.
  - Multi-source DNS detection and transparent UI status labeling (`Active`, `Active / Wi-Fi`, `Active / Cellular`, `Wi-Fi Config`, `Cellular Config`).
  - Upgraded DNS Switcher supporting `ndc resolver` and dual-protocol TCP/UDP port 53 `iptables` DNAT redirection.
- **Centralized Environment Configuration (`internal/config`)**:
  - Added support for `BFR_SU_BIN`, `BFR_MODULE_DIR`, `BFR_CLASH_API`, `BFR_SMS_DB`, `BFR_LEASES_FILE`, and `BFR_ALLOWED_DIRS` environment variable overrides.
- **Application Event Logger**:
  - Implemented global `LogWriter` streaming standard Go logs into `logger.Get()`.
  - Added real-time audit event logging across Auth, Power, Network, Proxy, SoftAP Hotspot, File Manager, Terminal, SSH, Charger, and Scrcpy handlers.
- **Feature-Based Modular Web Structure**:
  - Reorganized frontend templates into `web/templates/<feature>/` subdirectories.
  - Reorganized JavaScript modules into `web/static/js/modules/<feature>/` subdirectories.
  - Refactored Go HTML template loader in `router.go` to parse templates recursively using `fs.WalkDir`.
- **Documentation Suite**:
  - Added `docs/PROJECT_STRUCTURE.md`, `docs/API_REFERENCE.md`, `docs/DEVELOPMENT.md`, `docs/SECURITY.md`, and `env.example`.

### Fixed
- **Deadlock in SSH Manager**: Fixed `sync.RWMutex` re-entrancy deadlock in `ssh.go` (`Start()` and `Stop()`).
- **Security Audit Findings (Group A, Medium, Low)**:
  - Resolved command injection vulnerabilities in `SetSysctl`, `SetDNS`, `SetInterfaceConfig`, `ConfigureRPS`, and `resolveDeviceName` via strict regex and IP validation.
  - Enforced `SanitizePath` with symlink resolution across file operations.
  - Hardened authentication tokens (no timestamp fallback) and updated session cookies to `SameSite=Strict`.
  - Enforced 1MB request body limit on POST handlers and added HTTP security headers (`nosniff`, `DENY`, `no-referrer`).
  - Set 512KB WebSocket ReadLimit for terminal sessions and added 5s timeout to Clash API HTTP client.
  - Fixed temporary SMS DB file permissions (`chmod 600`).
  - Added graceful shutdown handling for `SIGINT`/`SIGTERM` in `main.go`.

---

## [0.4.0] - 2026-07-31
- Initial public release of BFR-WEBUI-GO as a Magisk / KernelSU module.
