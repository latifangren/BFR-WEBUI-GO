# Project-Specific Routing & Codebase Rules

## 1. Project Overview
`BFR-WEBUI_GO` is an ultra-lightweight, 100% offline-ready Android System Control Panel & WebUI designed to run as a Magisk / KernelSU / APatch module. It is written in modular Go and uses Alpine.js + Tailwind CSS for the frontend, all embedded inside a single binary (~10MB) with a tiny memory footprint.

## 2. Codebase Structure
- `main.go`: Entry point, command line flags, server setup.
- `internal/auth/`: Cookie-based session authentication manager.
- `internal/handlers/`: Clean request routers and controller logic (sysinfo, network, power, proxy, filemanager, hotspot, terminal).
- `internal/hotspot/`: SoftAP controls & connected clients ARP parsing.
- `internal/network/`: Persistent sysctl values, DNS options, dynamic SDK-aware TTL configurations.
- `internal/power/`: System power actions (`su -c reboot` wrapper).
- `internal/proxy/`: Clash/Mihomo configurations and watchdog loop logic.
- `internal/sysinfo/`: Hardware counters `/proc` reading utilities.
- `internal/terminal/`: WebSocket interactive PTY terminal.
- `web/`: Frontend resources (index.html, embedded assets).

## 3. Strict Routing & Verification Rules
- **No Direct UI Over-Editing**: The frontend uses a carefully calibrated Neo-Brutalist (AMOLED/Light) style. Follow the designer instructions strictly.
- **Cross-Compilation Checks**: Always verify modifications compile targeting `GOOS=android GOARCH=arm64` cleanly with zero syntax/type errors.
- **Tuning Config isolation**: Any system-level tweaks must be driven by variables defined within `tweaks.json` and resolved in `internal/network/config.go` dynamically. Never hardcode network config values or execute raw sysctl commands outside these properties.
- **Background specialist rules**:
  - Refactoring/implementation runs via **@fixer** (re-use session when possible).
  - Web research and references check runs via **@librarian** or **@explorer**.
