# Project-Specific Routing & Codebase Rules

## 1. Project Overview
`BFR-WEBUI_GO` is an ultra-lightweight, 100% offline-ready Android System Control Panel & WebUI designed to run as a Magisk / KernelSU / APatch module. It is written in modular Go and uses Alpine.js + Tailwind CSS for the frontend, all embedded inside a single binary (~10MB) with a tiny memory footprint.

## 2. Codebase Structure
- `main.go`: Entry point, command line flags, server setup.
- `internal/auth/`: Cookie-based session authentication manager.
- `internal/charger/`: Hardware-specific charging limitation controllers.
- `internal/filemanager/`: File navigation, download, upload, and archive management.
- `internal/handlers/`: Web controllers and API request routers (auth, charger, filemanager, hotspot, layout, logs, modules, network, power, proxy, scrcpy, sms, ssh, sysinfo, terminal).
- `internal/hotspot/`: SoftAP controls & connected clients ARP parsing.
- `internal/logger/`: Centralized structured logging.
- `internal/modules/`: System module detection and command executions.
- `internal/network/`: Persistent sysctl values, DNS options, dynamic SDK-aware TTL configurations.
- `internal/power/`: System power actions (`su -c reboot` wrapper).
- `internal/proxy/`: Clash/Mihomo configurations and watchdog loop logic.
- `internal/scrcpy/`: Scrcpy web integration, rendering and input event channels.
- `internal/smsviewer/`: AT-based/modem SMS parsing and retrieval handlers.
- `internal/ssh/`: Dropbear SSH client configuration and state manager.
- `internal/sysinfo/`: Hardware counters `/proc` reading utilities.
- `internal/terminal/`: WebSocket interactive PTY terminal.
- `internal/vnstat/`: Vnstat bandwidth logging and analysis wrapper.
- `internal/worker/`: Periodic background task scheduler and cron events.
- `web/`: Frontend resources (index.html, embedded assets).

## 3. Strict Routing & Verification Rules
- **No Direct UI Over-Editing**: The frontend uses a carefully calibrated Neo-Brutalist (AMOLED/Light) style. Follow the designer instructions strictly.
- **Cross-Compilation Checks**: Always verify modifications compile targeting `GOOS=android GOARCH=arm64` cleanly with zero syntax/type errors.
- **Tuning Config isolation**: Any system-level tweaks must be driven by variables defined within `tweaks.json` and resolved in `internal/network/config.go` dynamically. Never hardcode network config values or execute raw sysctl commands outside these properties.
- **Background specialist rules**:
  - Refactoring/implementation runs via **@fixer** (re-use session when possible).
  - Web research and references check runs via **@librarian** or **@explorer**.

## 4. Agent Behavior & Operational Constraints
- **Strict User Consent & Autonomy**: The agent must strictly adhere to the rules in `AGENTS.md` and MUST NOT initiate any modifications, refactoring, or file creations without explicit user requests.
- **Git Operations**: Do NOT run git commit or push commands unless explicitly requested by the user.
- **No Arbitrary Code Editing**: Do not make arbitrary or unsolicited code modifications.
- **Language & Communication**: Code, inline comments, and configurations must be written in English. However, all direct interaction and communication with the user must be conducted in Indonesian (Bahasa Indonesia).
- **Code Length Limit**: If any single file contains code that is too long or complex, the agent must recommend splitting it into modular files rather than expanding it further.
- **Handling UI Feature Additions / Tabs**:
  - If the user requests a new UI feature or an addition to an existing tab, the agent **MUST** explicitly ask the user whether the feature should be created as a new dedicated tab or added to an existing tab.
  - The agent must provide a recommendation with pros and cons for both options before the user decides.
- **Handling Inquiries, Analyses, and Feature Requests**:
  - If a user request involves analysis, proposes a new feature, ends with a "?" (indicating a question or exploratory intent), or has a tone representing an inquiry or questioning command, the agent **MUST NOT** directly modify the codebase.
  - Instead, the agent must present clear recommendations or design candidates, detailing the pros, cons, and trade-offs of each option, and await explicit design/implementation approval from the user.
