# CLAUDE.md - Project Rules & Guidelines for Claude Code

## 1. Project Overview
`BFR-WEBUI-GO` is an ultra-lightweight, 100% offline-ready Android System Control Panel & WebUI designed to run as a Magisk / KernelSU / APatch module. It is written in modular Go and uses Alpine.js + Tailwind CSS for the frontend, all embedded inside a single binary (~10MB) with a tiny memory footprint.

---

## 2. Codebase Structure
- `main.go`: Entry point, command line flags, server setup, background workers.
- `internal/auth/`: Cookie-based session authentication manager & password hashing.
- `internal/charger/`: Hardware-specific charging limitation controllers & sysfs scanners.
- `internal/filemanager/`: File navigation, download, upload, and archive management.
- `internal/handlers/`: Web controllers and API request routers.
- `internal/hotspot/`: SoftAP controls, MAC filtering & connected clients ARP parsing.
- `internal/logger/`: Centralized structured logging.
- `internal/modem/`: AT terminal & cellular band locking engine.
- `internal/modules/`: System module detection and command executions.
- `internal/nas/`: Local WebDAV & HTTP media sharing server.
- `internal/network/`: Persistent sysctl values, DNS options, dynamic SDK-aware TTL configurations.
- `internal/power/`: System power actions (`su -c reboot` wrapper).
- `internal/proxy/`: Clash/Mihomo configurations and watchdog loop logic.
- `internal/qos/`: Hybrid bandwidth control & traffic prioritization shaper (`tc HTB/IFB` + `iptables`).
- `internal/scrcpy/`: Scrcpy web integration, rendering and input event channels.
- `internal/smsviewer/`: AT-based/modem SMS parsing and retrieval handlers.
- `internal/speedtest/`: Multi-worker Go speedtest engine with ISP/Colo detection.
- `internal/ssh/`: Dropbear SSH client configuration and state manager.
- `internal/sysinfo/`: Hardware counters `/proc` reading utilities & per-core CPU usage.
- `internal/telegram/`: Bidirectional Telegram Bot daemon & SMS auto-forwarder.
- `internal/terminal/`: WebSocket interactive PTY terminal.
- `internal/tunnel/`: Persistent ARM64 daemon remote access tunnel (`cloudflared`, `tailscaled`, `zerotier-one`).
- `internal/vnstat/`: Vnstat bandwidth logging and analysis wrapper.
- `internal/worker/`: Periodic background task scheduler and cron events.
- `web/`: Frontend resources (`index.html`, `templates/`, `static/css/`, `static/js/`).

---

## 3. Strict Routing & Verification Rules
- **UI Architecture**: Supports modular UI paradigms (Neobrutal & Modern Clean). Follow design instructions strictly without breaking CSS scoping (`[data-ui-style="..."]`).
- **Cross-Compilation Verification**: Always verify code modifications compile cleanly targeting `GOOS=android GOARCH=arm64` with zero syntax/type errors (`go test ./...` and `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go test -c ./...`).
- **Tuning Config Isolation**: System-level network tweaks must be driven by variables defined within `tweaks.json` and resolved in `internal/network/config.go` dynamically. Never hardcode network config values or execute raw sysctl commands outside these properties.

---

## 4. Agent Behavior & Operational Constraints
- **Strict User Consent & Autonomy**: Adhere strictly to rules in `AGENTS.md` and `CLAUDE.md`. MUST NOT initiate any code modifications, refactoring, or file creations without explicit user requests.
- **Git Operations**: **DO NOT** run `git commit` or `git push` commands unless explicitly requested by the user.
- **No Arbitrary Code Editing**: Do not make arbitrary or unsolicited code modifications.
- **Language & Communication**:
  - Code, inline comments, commit messages, and configurations MUST be written in **English**.
  - ALL direct interaction and communication with the user MUST be conducted in **Indonesian (Bahasa Indonesia)**.
- **Code Length & Modularization**: If any single file contains code that is too long or complex, recommend splitting it into modular files rather than expanding it further.
- **Handling UI Feature Additions / Tabs**:
  - If the user requests a new UI feature or an addition to an existing tab, the agent **MUST** explicitly ask the user whether the feature should be created as a new dedicated tab or added to an existing tab.
  - Provide a recommendation with pros and cons for both options before the user decides.
- **Handling Inquiries, Analyses, and Feature Requests**:
  - If a user request involves analysis, proposes a new feature, ends with a "?" (indicating a question or exploratory intent), or has a tone representing an inquiry or questioning command, the agent **MUST NOT** directly modify the codebase.
  - Present clear recommendations or design candidates, detailing the pros, cons, and trade-offs of each option, and await explicit design/implementation approval from the user.
