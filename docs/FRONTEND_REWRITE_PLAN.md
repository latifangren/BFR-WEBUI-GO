# Frontend Modernization Plan: Svelte 5 + Vite + TypeScript

This document specifies the architecture, build pipeline, proof-of-concept (PoC) roadmap, and phase-by-phase execution plan for rewriting the `BFR-WEBUI-GO` frontend from Alpine.js + Go HTML templates to **Vite + Svelte 5 + TypeScript + Tailwind CSS**.

---

## 1. Objectives & Architectural Principles

1. **Zero-Virtual-DOM Runtime:** Leverage Svelte 5 runes (`$state`, `$derived`, `$effect`) to achieve minimal CPU and RAM usage on resource-constrained Android environments (Magisk/KernelSU root modules).
2. **Type Safety Across API Boundaries:** Define TypeScript interfaces that mirror Go backend structs (`internal/handlers`, `internal/sysinfo`, `internal/network`, etc.) to eliminate runtime payload mismatch.
3. **Preserve Neo-Brutalist AMOLED Design:** Retain the signature high-contrast Neo-Brutalist styling, AMOLED pitch-black backgrounds, crisp borders, and monospace telemetry typography.
4. **Single-Binary Self-Contained Deployment:** Production build compiles to static assets under `frontend/dist/` embedded directly into the Go binary via Go `embed.FS`, maintaining a single ~10-12MB executable with zero external runtime dependencies.
5. **Seamless Developer Experience:** Hot Module Replacement (HMR) during development via Vite dev server proxying API requests and WebSockets to the running Go backend.

---

## 2. Directory Architecture (`frontend/`)

```text
frontend/
├── package.json
├── pnpm-lock.yaml
├── svelte.config.js
├── tsconfig.json
├── tsconfig.node.json
├── vite.config.ts
├── tailwind.config.js
├── postcss.config.js
├── index.html
├── public/
│   ├── manifest.json
│   ├── favicon.ico
│   └── sw.js
└── src/
    ├── main.ts                    # Application bootstrap & mount
    ├── App.svelte                 # Shell container, active tab routing, global modals
    ├── assets/                    # Static SVG icons and graphics
    ├── styles/
    │   ├── app.css                # Base Tailwind imports and theme variables
    │   ├── amoled.css             # AMOLED dark & light color palette variables
    │   └── neobrutal.css          # Neo-brutalist border, box-shadow, and badge tokens
    ├── types/                     # TypeScript API and domain models
    │   ├── auth.ts                # Session, user credentials, login response
    │   ├── sysinfo.ts             # CPU, RAM, thermal, battery, storage, OS stats
    │   ├── network.ts             # Interface list, IP addresses, TTL, DNS, ARP clients
    │   ├── proxy.ts               # Clash/Mihomo configurations, latency, mode
    │   ├── terminal.ts            # PTY session, resize payload, WS frames
    │   ├── scrcpy.ts              # Control socket payloads and frame definitions
    │   └── common.ts              # API wrapper types, error responses, toasts
    ├── api/                       # HTTP API client layer
    │   ├── client.ts              # Fetch wrapper with auto-CSRF/cookie & error handling
    │   ├── auth.ts                # Login, logout, session check
    │   ├── sysinfo.ts             # System metrics and telemetry endpoints
    │   ├── network.ts             # Interface management, tweaks, hotspot
    │   ├── proxy.ts               # Clash/Mihomo control endpoints
    │   ├── power.ts               # Reboot, shutdown, system controls
    │   └── filemanager.ts         # Directory listing, upload, download, delete
    ├── ws/                        # WebSocket connection managers
    │   ├── socket.ts              # Robust WebSocket client with auto-reconnect & backoff
    │   ├── terminal.ts            # Terminal interactive PTY bridge
    │   ├── scrcpy.ts              # Scrcpy binary video/input stream bridge
    │   └── logs.ts                # Live logcat and app log streaming
    ├── stores/                    # Svelte 5 Runes state stores
    │   ├── auth.svelte.ts         # Authenticated state, user role, session timer
    │   ├── theme.svelte.ts        # Dark/Light mode, AMOLED toggle, UI style tokens
    │   ├── sysinfo.svelte.ts      # Telemetry polling engine and reactive metrics
    │   ├── navigation.svelte.ts   # Active tab state, sidebar collapse, URL hash sync
    │   └── toast.svelte.ts        # Global notification queue and dismiss timer
    └── components/                # UI Component library
        ├── ui/                    # Atom components (Neo-Brutalist primitives)
        │   ├── Button.svelte
        │   ├── Card.svelte
        │   ├── Badge.svelte
        │   ├── Input.svelte
        │   ├── Switch.svelte
        │   ├── Modal.svelte
        │   └── ToastContainer.svelte
        ├── layout/                # Structural layout components
        │   ├── Header.svelte      # Top bar with device status, battery pill, auth
        │   ├── Sidebar.svelte     # Navigation rail (desktop) / Bottom bar (mobile)
        │   └── StatusBar.svelte   # Real-time traffic, CPU, and proxy quick-metrics
        └── tabs/                  # Feature tab screens
            ├── overview/
            │   ├── TabOverview.svelte
            │   ├── QuickActions.svelte
            │   └── TelemetryCard.svelte
            ├── sysinfo/
            │   ├── TabSysinfo.svelte
            │   ├── CpuMonitor.svelte
            │   ├── MemoryBar.svelte
            │   └── BatteryHealth.svelte
            ├── network/
            │   ├── TabNetwork.svelte
            │   └── InterfaceList.svelte
            ├── hotspot/
            │   ├── TabHotspot.svelte
            │   └── ClientList.svelte
            ├── proxy/
            │   ├── TabProxy.svelte
            │   └── ProxyNodeCard.svelte
            ├── terminal/
            │   ├── TabTerminal.svelte
            │   └── XtermWrapper.svelte
            ├── scrcpy/
            │   ├── TabScrcpy.svelte
            │   └── VideoCanvas.svelte
            ├── filemanager/
            │   ├── TabFileManager.svelte
            │   └── FileExplorer.svelte
            ├── logs/
            │   └── TabLogs.svelte
            └── settings/
                └── TabSettings.svelte
```

---

## 3. Pipeline & Proof-of-Concept (PoC) Schema

### 3.1 Development Pipeline (HMR)
```text
Browser (http://localhost:5173)
       │
       ▼
Vite Dev Server (HMR enabled)
       │
       ├── Static Svelte Assets & CSS (Compiled in-memory)
       └── Proxy (/api, /ws) ────────► Go Backend Daemon (http://localhost:8080 or :80)
```

**`vite.config.ts` Proxy Configuration:**
```ts
server: {
  port: 5173,
  proxy: {
    '/api': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
    '/ws': {
      target: 'ws://127.0.0.1:8080',
      ws: true,
    }
  }
}
```

### 3.2 Production Pipeline (Single Binary Embed)
```text
[frontend/] pnpm build
       │
       ▼
[frontend/dist/] Static HTML/JS/CSS Assets
       │
       ▼ (Go Embed)
[internal/handlers/router.go] //go:embed all:frontend/dist
       │
       ▼
[Go Compiler] GOOS=android GOARCH=arm64 go build -ldflags "-s -w"
       │
       ▼
Single Binary: bfr-webui-android-arm64 (~11MB)
```

### 3.3 Scope of Proof-of-Concept (PoC)
The goal of the PoC is to prove compatibility, aesthetic continuity, and performance without touching existing Go endpoints:
1. Initialize `frontend/` with Vite 6 + Svelte 5 + TypeScript + Tailwind CSS.
2. Port Neo-Brutalist AMOLED theme tokens (`amoled.css`, `neobrutal.css`, dark/light switcher).
3. Build the core application shell (`Header.svelte`, `Sidebar.svelte`, `ToastContainer.svelte`).
4. Implement `auth.svelte.ts` (handling session status, login modal).
5. Implement `sysinfo.svelte.ts` and `TabSysinfo.svelte` (live polling of CPU, RAM, battery, thermals, and OS information).
6. Verify development proxy against local Go server and test static build output.

---

## 4. Phased Implementation Plan

### Phase 1: Foundation & Proof-of-Concept (Completed)
- [x] Initialize `frontend/` directory with `pnpm create vite frontend --template svelte-ts`.
- [x] Install dependencies: `tailwindcss`, `postcss`, `autoprefixer`, `lucide-svelte`, `clsx`, `tailwind-merge`.
- [x] Configure `vite.config.ts` with API/WebSocket reverse proxy.
- [x] Implement Neo-Brutalist design tokens and theme store (`stores/theme.svelte.ts`).
- [x] Implement HTTP client wrapper (`api/client.ts`) with cookie session support.
- [x] Implement `App.svelte` shell + `Header.svelte` + `Sidebar.svelte`.
- [x] Build PoC tab: `TabSysinfo.svelte` consuming `/api/sysinfo` with reactive gauges.
- [x] Run benchmark test comparing bundle size and memory usage against legacy Alpine.js.

### Phase 2: Core System & Network Management (Completed)
- [x] Port Overview: Implement `TabOverview.svelte` with telemetry cards and quick action controls.
- [x] Port System Info: Full `TabSysinfo.svelte` with per-core CPU meters and RAM breakdown.
- [x] Port Network: Implement `TabNetwork.svelte` (interface status, DNS switcher, sysctl tweaks).
- [x] Port Hotspot: Implement `TabHotspot.svelte` (AP toggle, connected client list, MAC filter).
- [x] Port Charger: Implement `TabCharger.svelte` (battery protection limiter, auto-detection).
- [x] Port Modem: Implement `TabModem.svelte` (SMS viewer, AT command console, signal bands).

### Phase 3: Complex Interactive Subsystems (WebSocket & Canvas) (Completed)
- [x] Port Terminal: Implement `TabTerminal.svelte` with `xterm` + `xterm-addon-fit` connecting to `/api/terminal/ws`.
- [x] Port Scrcpy Web: Implement `TabScrcpy.svelte` with WebSocket binary video stream decoder and canvas input listeners.
- [x] Port File Manager: Implement `TabFileManager.svelte` (directory browser, breadcrumb navigation, multi-file upload, archive extraction).
- [x] Port Live Logs: Implement `TabLogs.svelte` (logcat stream, webui daemon log, search & filter).

### Phase 4: Utilities, Proxies & Auxiliary Modules (Completed)
- [x] Port Proxy Manager: Implement `TabProxy.svelte` (Clash/Mihomo core status, mode switch, proxy group latency test, config editor).
- [x] Port QoS & Traffic: Implement `TabQoS.svelte` and `TabVnstat.svelte` (bandwidth graphs, interface quotas).
- [x] Port NAS & Tunnel: Implement `TabNAS.svelte` (Samba/NFS shares) and `TabTunnel.svelte` (Cloudflare, FRP, Zerotier).
- [x] Port Speedtest & Tools: Speedtest runner, Ping/Traceroute diagnostics, and system module manager.
- [x] Port Telegram Bot: Notification webhook and bot token configuration.

### Phase 5: Cutover, Optimization & Android Verification (Completed)
- [x] Update Go `embed.go` to point to `frontend/dist` with fallback routing (SPA history fallback).
- [x] Maintain legacy assets during transition, wire single-binary embed via `frontend/embed.go`.
- [x] Benchmark single-binary size with `GOOS=android GOARCH=arm64` (`-ldflags "-s -w"`).
- [x] Validate on real Android device running KernelSU/Magisk: memory footprint (<20MB idle), CPU spikes during telemetry polling (<2%), and touch responsiveness.
