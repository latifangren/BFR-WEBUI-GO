# Changelog

All notable changes to the **BFR-WEBUI-GO** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

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
