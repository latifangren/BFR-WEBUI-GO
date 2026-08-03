# Changelog

All notable changes to the **BFR-WEBUI-GO** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

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
