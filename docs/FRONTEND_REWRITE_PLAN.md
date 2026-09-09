# Frontend Modernization Plan: Svelte 5 + Vite + TypeScript

This document specifies the architecture, build pipeline, comprehensive audit findings, dual-style design system, multi-palette theming engine, and phased execution roadmap for modernizing the `BFR-WEBUI-GO` frontend from legacy Alpine.js + Go HTML templates (`web/`) to **Vite + Svelte 5 (Runes) + TypeScript + Tailwind CSS** (`frontend/`).

---

## 1. Objectives & Architectural Principles

1. **Zero-Virtual-DOM Runtime (Svelte 5 Runes):**
   - Leverage Svelte 5 reactive primitives (`$state`, `$derived`, `$effect`, `$props`) to eliminate Virtual DOM overhead and achieve minimal CPU and RAM usage on resource-constrained Android root environments (KernelSU / Magisk / APatch).
2. **Strict API Contract & Type Safety:**
   - Define TypeScript interfaces that strictly mirror Go backend structs (`internal/handlers`, `internal/sysinfo`, `internal/network`, `internal/hotspot`, etc.) to eliminate runtime payload mismatches and silent data corruption.
3. **Dual UI Style Paradigms & Multi-Color Theme System:**
   - **Two UI Style Paradigms:** Authentic **Neobrutalism** (2px solid borders, 4px hard offset box-shadows, 4px crisp radii, monospace telemetry) and **Modern Clean** (1px subtle border, 16px soft radii, glassmorphism blur, soft diffused shadows).
   - **Eight Color Theme Presets:** `dark` (default navy/dark), `light` (high-contrast daylight), `amoled` (pure pitch-black `#000000`), `dracula` (vampire purple), `nord` (arctic cyan), `cyberpunk` (neon yellow), `emerald` (forest green), and `sunset` (warm dusk orange).
4. **Mobile-First Android Ergonomics:**
   - Native Android handheld ergonomics with a thumb-accessible **Mobile Bottom Navigation Bar** (`md:hidden`) dividing 20+ tabs into compact functional categories (Overview, Network, System, Tools, More Sheet) instead of top-corner hamburger menus.
5. **Full Feature & Subsystem Parity:**
   - Restore all capabilities from legacy `web/templates/` including Dropbear SSH manager, Magisk/KSU module ZIP flasher, CasaOS-style local app shortcuts, live SVG performance telemetry, and full archive/file manipulation.
6. **Self-Contained Single Binary Deployment:**
   - Compile to static assets under `frontend/dist/` embedded into the Go binary via Go `embed.FS` (`frontend/embed.go`), preserving a compact single executable (~11-12MB) with zero external runtime dependencies.

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
│   ├── favicon.svg
│   └── icons.svg
└── src/
    ├── main.ts                          # Bootstrap, theme initialization, root mount
    ├── App.svelte                       # Root shell, active tab router, global modals
    ├── styles/
    │   ├── app.css                      # Tailwind base, components, utilities
    │   ├── themes.css                   # 8 Color palettes CSS variables + 2 UI styles
    │   ├── neobrutal.css                # Neobrutalist borders, hard shadows, grid pattern
    │   └── modern.css                   # Modern clean glassmorphism, soft shadows
    ├── types/                           # Strict TypeScript models mirroring Go structs
    │   ├── auth.ts                      # Auth status, session, login payload
    │   ├── sysinfo.ts                   # CPU, RAM, thermal, battery, storage, OS stats
    │   ├── network.ts                   # Interfaces, IP addresses, DNS, tweaks, TTL
    │   ├── hotspot.ts                   # SoftAP status, connected clients, MAC filter
    │   ├── proxy.ts                     # Clash/Mihomo configurations, mode, latency
    │   ├── files.ts                     # File items, breadcrumb, batch ops, storage
    │   ├── terminal.ts                  # PTY session, resize frames
    │   ├── scrcpy.ts                    # Binary video frame & input event payloads
    │   ├── ssh.ts                       # Dropbear status, config, key management
    │   ├── shortcuts.ts                 # CasaOS-style app shortcuts CRUD
    │   ├── modules.ts                   # Magisk/KSU module list, toggle, zip install
    │   └── common.ts                    # API wrapper, Toast, Modal, Error models
    ├── api/                             # Centralized HTTP API client layer
    │   ├── client.ts                    # Fetch client with 401 interceptor & credentials
    │   ├── auth.ts                      # Login, logout, session check, password change
    │   ├── sysinfo.ts                   # Telemetry, metrics, CPU governor
    │   ├── network.ts                   # Tweaks, DNS, TTL, RPS, ping
    │   ├── hotspot.ts                   # SoftAP toggle, client list, MAC filter
    │   ├── proxy.ts                     # Proxy status, node switch, logs, config, delay
    │   ├── power.ts                     # System reboot, shutdown, recovery
    │   ├── charger.ts                   # Battery charging limiter toggle & config
    │   ├── files.ts                     # List, read, save, upload, download, ops
    │   ├── modem.ts                     # AT command, signal, bands, reset
    │   ├── ssh.ts                       # Dropbear SSH status, config, control
    │   ├── shortcuts.ts                 # App shortcuts list, save, delete
    │   ├── modules.ts                   # Modules list, toggle, zip flash upload
    │   ├── backup.ts                    # Export/import settings, cloud sync
    │   ├── telegram.ts                  # Bot notification config & control
    │   ├── qos.ts                       # Bandwidth limiter & packet priority
    │   ├── vnstat.ts                    # Traffic database & stats
    │   └── speedtest.ts                 # Speedtest start, stop, status, history
    ├── ws/                              # WebSocket client managers
    │   ├── socket.ts                    # Robust WS client (exponential backoff & ping)
    │   ├── terminal.ts                  # Interactive terminal PTY socket bridge
    │   ├── scrcpy.ts                    # Scrcpy binary video stream & touch events
    │   └── logs.ts                      # Real-time logcat and system daemon stream
    ├── stores/                          # Svelte 5 Runes global stores
    │   ├── auth.svelte.ts               # Session state, login modal trigger, role
    │   ├── theme.svelte.ts              # UI Style (neobrutal/modern) & 8 Color Themes
    │   ├── sysinfo.svelte.ts            # Resource-aware polling engine & hardware stats
    │   ├── navigation.svelte.ts         # Active tab, mobile bottom bar, hash sync
    │   └── toast.svelte.ts              # Global notification queue and dismiss timer
    └── components/
        ├── ui/                          # Style-agnostic atomic primitives
        │   ├── Badge.svelte
        │   ├── Button.svelte
        │   ├── Card.svelte
        │   ├── Input.svelte
        │   ├── Switch.svelte
        │   ├── Modal.svelte
        │   └── ToastContainer.svelte
        ├── layout/                      # Application shell components
        │   ├── Header.svelte            # Telemetry pills (battery, temp, RAM, uptime)
        │   ├── Sidebar.svelte           # Desktop navigation rail (20+ tabs categorized)
        │   ├── BottomNav.svelte         # Mobile bottom bar with thumb-friendly sheet
        │   └── AppearanceModal.svelte   # UI Style & 8 Color Themes switcher
        └── tabs/                        # Tab feature screens (22 tabs)
            ├── overview/                # Bento grid, CasaOS shortcuts, active services
            ├── sysinfo/                 # Per-core gauges, memory breakdown, CPU governor
            ├── network/                 # Interface status, DNS switcher, sysctl tweaks
            ├── hotspot/                 # AP toggle, clients ARP list, MAC filter
            ├── proxy/                   # Clash/Mihomo node latency, group mode, watchdog
            ├── modem/                   # Signal metrics, AT console, band lock, reset
            ├── sms/                     # SMS inbox viewer, OTP quick-copy (Restored Tab)
            ├── power/                   # Power actions, reboot styles, uptime counter
            ├── charger/                 # Battery protection limiter, bypass charging
            ├── terminal/                # Xterm.js PTY shell with robust WS client
            ├── scrcpy/                  # Low-latency canvas screen streaming & touch
            ├── files/                   # File manager, text editor, zip compress/extract
            ├── logs/                    # Live logcat stream, app daemon logs, filter
            ├── qos/                     # Bandwidth limiter & queuing rules
            ├── vnstat/                  # Monthly/daily traffic charts & interface quota
            ├── nas/                     # Samba / NFS share service manager
            ├── tunnel/                  # Cloudflare tunnel, Tailscale, Zerotier
            ├── speedtest/               # Speedtest runner & historic latency records
            ├── ssh/                     # Dropbear SSH server manager (Restored Tab)
            ├── modules/                 # Magisk/KSU module manager & ZIP flasher (Restored Tab)
            ├── tools/                   # System diagnostics, ping, traceroute, shortcuts
            ├── telegram/                # Telegram alert bot configuration
            └── about/                   # Version info, changelog, developer links
```

---

## 3. Comprehensive Audit Findings & Gap Analysis

An exhaustive technical comparison between Go backend handlers (`internal/handlers/router.go`, `internal/`) and Svelte 5 frontend (`frontend/src/`) revealed the following issues:

### 3.1 Endpoint Coverage & Ghost Endpoints

1. **Ghost Endpoint (404 Not Found):**
   - `/api/proxy/delay` is called by `TabProxy.svelte:76`, but **does not exist** in Go router (`internal/handlers/proxy_handler.go`).
   - *Fix:* Implement `HandleProxyDelay` in backend or route delay tests directly to the local Clash/Mihomo REST controller (`127.0.0.1:9090/proxies/{name}/delay`).
2. **Backend Endpoints Without Frontend Integration (34 Routes):**
   - **SSH Daemon (3 routes):** `/api/ssh/status`, `/api/ssh/config`, `/api/ssh/control` (completely absent in Svelte).
   - **App Shortcuts (3 routes):** `/api/shortcuts/list`, `/api/shortcuts/save`, `/api/shortcuts/delete` (completely absent in Svelte).
   - **Backup & Cloud Sync (3 routes):** `/api/backup/import`, `/api/backup/cloud/config`, `/api/backup/cloud/sync`.
   - **Advanced File Operations (9 routes):** `/api/files/read` (text editor), `/api/files/copy`, `/api/files/move`, `/api/files/batch`, `/api/files/permissions`, `/api/files/compress`, `/api/files/extract`, `/api/files/search`, `/api/files/storage`.
   - **Modem Extensions (2 routes):** `/api/modem/bands` (LTE/NR band locking), `/api/modem/reset`.
   - **System Root (3 routes):** `/api/modules/install` (ZIP flash), `/api/sysinfo/governor` (CPU governor), `/api/network/rps` (packet steering).
   - **Other Missing Calls:** `/api/auth/change-password`, `/api/speedtest/history`, `/api/proxy/logs`, `/api/proxy/watchdog`, `/api/proxy/config`, `/api/tunnel/upload`.

---

### 3.2 Payload, Parameter & Type Mismatches

| Subsystem | File & Line | Backend Expectation (Go) | Frontend Implementation (Svelte) | Impact |
| :--- | :--- | :--- | :--- | :--- |
| **Network DNS** | `TabNetwork.svelte:141`<br>`network_handler.go:29` | `struct { DNS1 string json:"dns1"; DNS2 string json:"dns2" }` | Sends `{ primary: p, secondary: s }` | **FATAL**: Decodes as empty strings. Sets empty DNS, breaking internet resolution on device. |
| **Hotspot MAC Filter** | `TabHotspot.svelte:106`<br>`mac_filter.go:18` | `struct { Mode string json:"mode"; BlockedMACs []string json:"blocked_macs"; AllowedMACs []string json:"allowed_macs" }` | Sends `{ mode: macFilterMode, list: nextList }` | **FATAL**: Decodes `BlockedMACs` as `nil`. Erases all blocked/allowed MAC addresses. |
| **Network Tweaks** | `TabNetwork.svelte:22`<br>`network/config.go:16` | Keys: `bbr2_congestion_control`, `dalvik_responsiveness`, `tcp_buffer_optimization`, etc. | Keys: `tcp_bbr_enabled`, `low_latency_mode`, `fastopen_enabled`, `ipv6_disable` | **Silent Failure**: Tweaks are ignored by backend and not saved to `tweaks.json`. |
| **File Manager List** | `TabFileManager.svelte:63`<br>`file_handler.go:68` | Returns `{ "path": string, "files": []FileInfo }` | Reads `res.current_path` | `current_path` is undefined, causing breadcrumb navigation drift. |
| **File Text Reader** | `TabFileManager.svelte:92`<br>`file_handler.go:73` | Dedicated text read endpoint `/api/files/read?path=...` | Calls `/api/files/download?path=...` | Download endpoint triggers attachment stream, corrupting non-UTF8 buffers. |
| **Tunnel Config** | `TabTunnel.svelte:19`<br>`tunnel/tunnel.go:21` | Expects `cloudflare_token`, engine `cloudflare`, `tailscale`, `zerotier` | Sends `{ token, server, port }`, engine includes `frp` | Token remains empty string; tunnel fails to start; `frp` is unsupported. |
| **Telegram Control** | `TabTelegram.svelte:83`<br>`telegram_handler.go:102`| Supports actions: `start`, `stop`, `restart` | Sends `{ action: 'test' }` | Backend returns HTTP 400 `Unknown control action`. |
| **Sysinfo Stats** | `TabOverview.svelte:268`<br>`sysinfo.go:73, 79` | `AndroidVer json:"android_version"`, `Services map[string]ServiceStatus` | Reads `stats?.android_ver`, calls `.filter()` on `active_services` | UI shows `Android N/A` and `0 Running` services permanently. |

---

### 3.3 WebSocket & Vite Development Proxy Issues

1. **Broken Vite Dev Proxy (`frontend/vite.config.ts:10-18`):**
   - Backend routes WebSockets under `/api/` (`/api/terminal/ws`, `/api/scrcpy/ws`, `/api/logs/logcat/stream`).
   - Vite config proxies `/ws` (which does not exist in backend) and proxies `/api` **without `ws: true`**.
   - *Result:* During `pnpm dev`, WebSocket handshakes fail to upgrade (HTTP 101).
2. **Unutilized WebSocket Manager (`frontend/src/ws/socket.ts`):**
   - `socket.ts` contains robust exponential backoff reconnection and ping-pong timeout logic.
   - `TabTerminal.svelte`, `TabLogs.svelte`, and `TabScrcpy.svelte` bypass `socket.ts` and instantiate raw `new WebSocket()` with naive reconnection loops that cause CPU spikes on Android.

---

### 3.4 Auth & Session Resilience Gap

- `frontend/src/api/client.ts` has no HTTP 401 Unauthorized interceptor. When a session cookie expires or backend restarts, buttons throw red toast errors without redirecting the user to the login modal until page refresh.

---

## 4. UI Style Paradigms & Multi-Palette Theming Engine

To restore the authentic BFR-WEBUI visual experience, the frontend must support **2 distinct UI Style Paradigms** and **8 Color Theme Presets**, selectable in real time via an `AppearanceModal.svelte`.

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                       BFR-WEBUI THEME ARCHITECTURE                          │
│                                                                             │
│   UI Style Paradigm [data-ui-style]       Color Theme Preset [data-theme]   │
│   ├── "neobrutal" (Industrial Retro)      ├── "dark"      (Default Navy)    │
│   └── "modern"    (Clean Glassmorphism)   ├── "light"     (High-Contrast)   │
│                                           ├── "amoled"    (OLED Pitch Black)│
│                                           ├── "dracula"   (Vampire Purple)  │
│                                           ├── "nord"      (Arctic Cyan)     │
│                                           ├── "cyberpunk" (Neon Yellow)     │
│                                           ├── "emerald"   (Forest Green)    │
│                                           └── "sunset"    (Warm Dusk Orange)│
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.1 Specification: Two UI Style Paradigms

#### Paradigm A: Neobrutalism (`data-ui-style="neobrutal"`)
- **Philosophy:** Bold, unapologetic, retro-industrial command center aesthetic.
- **Borders:** `border: 2px solid var(--neo-border) !important` on all cards, buttons, inputs, and modals.
- **Hard Drop Shadows:** `box-shadow: 4px 4px 0px 0px var(--neo-shadow) !important` (buttons translate `translate-x-[2px] translate-y-[2px]` on active).
- **Corner Radii:** `border-radius: 4px !important` (crisp, sharp industrial corners; strictly eliminate `rounded-lg` and `rounded-xl`).
- **Surface Texture:** Subtle micro-dot matrix overlay (`radial-gradient(var(--neo-border) 1px, transparent 1px)` with 24px spacing, 3% opacity).
- **Typography:** Bold monospace labels for headers, telemetry data, and status pills.

#### Paradigm B: Modern Clean (`data-ui-style="modern"`)
- **Philosophy:** Smooth, premium glassmorphism with soft diffused lighting.
- **Borders:** Subtle `border: 1px solid var(--neo-border)` with low opacity.
- **Diffused Shadows:** Soft multi-layer depth (`box-shadow: 0 10px 30px -5px rgba(0,0,0,0.15), 0 4px 10px -2px rgba(0,0,0,0.05)`).
- **Corner Radii:** `border-radius: 16px` for cards, `12px` for buttons, `8px` for inputs.
- **Glassmorphism:** `backdrop-filter: blur(12px)` on cards, headers, and floating toolbars.

---

### 4.2 Specification: Eight Color Theme Presets

All themes are defined through standardized CSS custom properties in `frontend/src/styles/themes.css`:

```css
:root {
  --neo-bg: #090d16;
  --neo-card: #131927;
  --neo-card-sub: #1e293b;
  --neo-text: #f8fafc;
  --neo-muted: #94a3b8;
  --neo-border: #334155;
  --neo-accent: #7c3aed;
  --neo-shadow: #1e293b;
  --neo-success: #10b981;
  --neo-warning: #f59e0b;
  --neo-danger: #ef4444;
}
```

| Theme Preset | Identifier | Background (`--neo-bg`) | Card (`--neo-card`) | Accent (`--neo-accent`) | Border (`--neo-border`) | Shadow Tone (`--neo-shadow`) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Dark (Default)** | `dark` | `#090d16` (Deep Navy) | `#131927` | `#7c3aed` (Violet) | `#334155` | `#1e293b` |
| **Light** | `light` | `#f4f6fa` (Cool Gray) | `#ffffff` | `#2563eb` (Royal Blue) | `#0f172a` (Neobrutal)<br>`#cbd5e1` (Modern) | `#0f172a` (Solid Black)<br>`#94a3b8` (Modern) |
| **AMOLED** | `amoled`| `#000000` (Pure Black) | `#0a0a0a` | `#10b981` (Emerald) | `#262626` | `#1c1c1c` *(Contrast shadow, prevents collapse)* |
| **Dracula** | `dracula`| `#1e1f29` (Dracula Navy) | `#282a36` | `#bd93f9` (Lavender) | `#44475a` | `#191a21` |
| **Nord** | `nord` | `#242933` (Polar Night) | `#2e3440` | `#88c0d0` (Frost Cyan) | `#434c5e` | `#1b1f27` |
| **Cyberpunk** | `cyberpunk`| `#0d0f18` (Matrix Dark) | `#181b28` | `#facc15` (Neon Yellow)| `#facc15` | `#facc15` |
| **Emerald** | `emerald`| `#042f2e` (Deep Jungle) | `#064e3b` | `#10b981` (Mint Green) | `#115e59` | `#022c22` |
| **Sunset** | `sunset` | `#120b18` (Dusk Purple) | `#1f1329` | `#f97316` (Vibrant Orange)| `#4c1d95` | `#0a050e` |

---

### 4.3 Elimination of Hardcoded Tailwind Colors

Components must **never** hardcode `text-emerald-400` or `text-amber-400` directly on text, which causes WCAG AA contrast failures (< 2.0:1) on white cards in Light Mode. 

All alerts, badges, and status labels must use semantic tokens:
- Green / Online: `text-[var(--neo-success)]` (Light: `#059669`, Dark: `#10b981`)
- Amber / Warning: `text-[var(--neo-warning)]` (Light: `#d97706`, Dark: `#f59e0b`)
- Red / Danger: `text-[var(--neo-danger)]` (Light: `#dc2626`, Dark: `#ef4444`)
- Accent / Info: `text-[var(--neo-accent)]`

---

### 4.4 Mobile Handheld Ergonomics: Bottom Navigation Bar

On Android handheld screens (aspect ratios 19.5:9 to 21:9), top-header hamburger buttons are ergonomically inaccessible with one thumb.

1. **Mobile Bottom Navigation (`components/layout/BottomNav.svelte`):**
   - Fixed at viewport bottom (`fixed bottom-0 inset-x-0 z-40 md:hidden`).
   - 4 Primary Action Items:
     - 📊 **Overview** (`overview`)
     - 🌐 **Network** (`network`)
     - ⚡ **System** (`sysinfo`)
     - 🛠️ **Tools** (`tools`)
   - 1 Action Sheet Button:
     - 📑 **More / Menu** (opens a clean bottom-sheet drawer with search filter to instantly jump to any of the 22 tabs).
2. **Minimum Touch Targets:**
   - All interactive icons, tabs, and toggles must satisfy `min-h-[44px] min-w-[44px]` touch targets.

---

## 5. Feature Parity Matrix (`web/` vs `frontend/`)

| Feature / Tab | Legacy Web (`web/`) | Svelte 5 Current (`frontend/`) | Target Svelte 5 Action |
| :--- | :--- | :--- | :--- |
| **SSH Daemon Manager** | Dedicated tab (`tab_ssh.html`) | **Missing** | Create `TabSSH.svelte` + route `/api/ssh/*` |
| **Magisk/KSU Module Manager**| Dedicated tab (`tab_modules.html`)| Compressed in Tools; **Flash ZIP missing** | Create dedicated `TabModules.svelte` with drag & drop ZIP flash |
| **SMS Inbox & OTP Finder** | Dedicated tab (`tab_sms.html`) | Sub-view inside Modem tab | Split into dedicated `TabSMS.svelte` with OTP copy button |
| **CasaOS App Shortcuts** | Overview card (`tab_overview.html`) | **Missing** | Restore Shortcuts Manager on Overview with CRUD `/api/shortcuts/*` |
| **Interactive Telemetry Modals**| CPU/Battery/Network modals | **Missing** (redirects tab instead) | Implement popup modals for voltage mV, per-core clocks, thermal zones |
| **Header Live Metrics** | Battery, Temp, RAM, Uptime | Only Battery & Temp | Restore free RAM percentage & System Uptime pills in Header |
| **Live Performance SVG Graphs**| Real-time CPU/Traffic SVG | Static gauges only | Port SVG dynamic waveform poller to Svelte 5 canvas/SVG |
| **Advanced File Explorer** | Compress, Extract, Chmod, Search | Basic list, download, upload only | Add Archive, Chmod, Copy/Move, Search modals to `TabFileManager` |
| **CPU Scaling Governor** | Present in Sysinfo | **Missing** | Add governor dropdown (`/api/sysinfo/governor`) to `TabSysinfo` |
| **Cloud Backup & Sync** | Export/Import, WebDAV/Rclone | **Missing** | Add Backup & Cloud Sync modal in Header/Settings |

---

## 6. Revised Phased Implementation Roadmap

```text
  Phase 1: Foundation & Theming System (Dual-Style + 8 Colors + Dev Proxy)
     │
     ▼
  Phase 2: Critical API Contract Harmonization & Fatal Bug Fixes
     │
     ▼
  Phase 3: WebSocket Architecture & Mobile Ergonomics (Bottom Nav)
     │
     ▼
  Phase 4: Missing Tab & Feature Parity (SSH, Modules ZIP, SMS, Shortcuts)
     │
     ▼
  Phase 5: Advanced File Manager & Auxiliary Endpoints Integration
     │
     ▼
  Phase 6: Android Resource Optimization, Verification & Binary Build
```

---

### Phase 1: Foundation & Theming Engine (Dual-Style + 8 Color Palettes)
- [x] **Fix Vite Proxy Configuration:**
  - Update `frontend/vite.config.ts` to add `ws: true` to the `/api` proxy rule and remove the non-existent `/ws` proxy rule.
- [x] **Implement Complete Theming CSS Tokens:**
  - Update `frontend/src/styles/themes.css` with 8 color theme palettes (`dark`, `light`, `amoled`, `dracula`, `nord`, `cyberpunk`, `emerald`, `sunset`).
  - Create `frontend/src/styles/neobrutal.css` (2px borders, 4px sharp radii, 4px/4px offset shadows, micro-dot pattern).
  - Create `frontend/src/styles/modern.css` (1px subtle borders, 16px soft radii, glassmorphism blur, diffused shadows).
  - Define semantic variables (`--neo-success`, `--neo-warning`, `--neo-danger`, `--neo-accent`).
- [x] **Upgrade Theme Store:**
  - Refactor `frontend/src/stores/theme.svelte.ts` to manage:
    - `uiStyle`: `'neobrutal' | 'modern'` (persisted to `localStorage.getItem('uiStyle')`).
    - `colorTheme`: `'dark' | 'light' | 'amoled' | 'dracula' | 'nord' | 'cyberpunk' | 'emerald' | 'sunset'`.
  - Add inline anti-FOUC script in `frontend/index.html` to prevent theme flashing on reload.
- [x] **Build Appearance Modal:**
  - Implement `frontend/src/components/layout/AppearanceModal.svelte` with style paradigm cards, 8-palette grid preview, and real-time active badge sample.
  - Wire appearance button (🎨 icon) in `Header.svelte`.

---

### Phase 2: Critical API Contract Harmonization & Fatal Bug Fixes
- [x] **Fix Network DNS Payload (Fatal):**
  - Update `frontend/src/components/tabs/network/TabNetwork.svelte` to POST `{ dns1: p, dns2: s }` matching `dnsRequest` struct in `internal/handlers/network_handler.go`.
  - Fix GET DNS logic to properly read preset servers list and active DNS.
- [x] **Fix Hotspot MAC Filter Payload (Fatal):**
  - Update `frontend/src/components/tabs/hotspot/TabHotspot.svelte` to POST `{ mode: macFilterMode, blocked_macs: [...], allowed_macs: [...] }` matching `MACFilterConfig`.
  - Update GET handler to unwrap nested `res.config.mode` and `res.config.blocked_macs`.
- [x] **Harmonize Network Tweaks Payload:**
  - Map `TabNetwork.svelte` state keys to exact `TweaksConfig` struct keys (`bbr2_congestion_control`, `dalvik_responsiveness`, `tcp_buffer_optimization`, etc.).
- [x] **Fix Proxy Delay Endpoint (Ghost 404):**
  - Add `HandleProxyDelay` in `internal/handlers/proxy_handler.go` OR point `TabProxy.svelte` directly to local Clash/Mihomo REST controller (`http://127.0.0.1:9090/proxies/{name}/delay`).
- [x] **Harmonize Sysinfo & Overview Properties:**
  - Fix `TabOverview.svelte` to read `stats.android_version` instead of `stats.android_ver`.
  - Fix active services count by inspecting the `stats.services` map (`map[string]ServiceStatus`).
- [x] **Implement 401 Session Interceptor:**
  - Update `frontend/src/api/client.ts` to detect HTTP 401 and set `authStore.authenticated = false`, triggering the login modal cleanly.

---

### Phase 3: WebSocket Architecture & Mobile Ergonomics
- [x] **Universal WebSocket Client Adoption:**
  - Refactor `TabTerminal.svelte`, `TabLogs.svelte`, and `TabScrcpy.svelte` to use the unified `frontend/src/ws/socket.ts` client.
  - Ensure binary packets for Scrcpy canvas and text packets for PTY terminal are handled seamlessly with auto-reconnect backoff.
- [x] **Mobile Bottom Navigation Bar:**
  - Create `frontend/src/components/layout/BottomNav.svelte` with 4 thumb-zone items (`Overview`, `Network`, `System`, `Tools`) + `More Sheet` button for viewport `< 768px` (`md:hidden`).
  - Implement full-screen or bottom-sheet tab drawer with instant search filter.
- [x] **Header Quick Metrics Restoration:**
  - Update `frontend/src/components/layout/Header.svelte` to include:
    - Free RAM badge (% and GB).
    - System Uptime counter.
    - Quick Power action button modal trigger.

---

### Phase 4: Missing Tab & Feature Parity Restoration
- [x] **Restore Dedicated SSH Tab:**
  - Create `frontend/src/components/tabs/ssh/TabSSH.svelte`.
  - Register `ssh` in `navigation.svelte.ts` and `App.svelte`.
  - Implement Dropbear service toggle, port configuration, and authorized keys viewer consuming `/api/ssh/*`.
- [x] **Restore Dedicated Magisk / KSU Modules Tab:**
  - Create `frontend/src/components/tabs/modules/TabModules.svelte`.
  - Implement drag-and-drop ZIP flash installer consuming `/api/modules/install`.
  - Implement module list, enable/disable toggle, and description cards consuming `/api/modules`.
- [x] **Restore Dedicated SMS Tab:**
  - Create `frontend/src/components/tabs/sms/TabSMS.svelte`.
  - Implement SMS inbox table, search filter, and quick OTP 1-click copy button consuming `/api/sms/inbox`.
- [x] **Restore CasaOS App Shortcuts:**
  - Implement shortcuts grid component on `TabOverview.svelte`.
  - Connect to `/api/shortcuts/list`, `/api/shortcuts/save`, `/api/shortcuts/delete` with Add/Edit/Delete link modals.
- [x] **Restore Telemetry Detail Modals:**
  - Implement modal popup on Overview CPU card: per-core frequency, governor, thermal zones.
  - Implement modal popup on Overview Battery card: voltage (mV), current (mA), health status, temperature.

---

### Phase 5: Advanced File Manager & Auxiliary Endpoints Integration
- [x] **Advanced File Explorer Capabilities (`TabFileManager.svelte`):**
  - Implement Archive creation (ZIP compress via `/api/files/compress`).
  - Implement Archive extraction (Unzip via `/api/files/extract`).
  - Implement File Permissions editor (chmod octal via `/api/files/permissions`).
  - Implement Batch operations (multi-select delete, copy, move via `/api/files/batch`).
  - Implement File Search modal (`/api/files/search`).
  - Implement Storage breakdown bar (`/api/files/storage`).
  - Route text file viewing to `/api/files/read` instead of `/api/files/download`.
- [x] **Auxiliary System Endpoints:**
  - Integrate CPU scaling governor selector in `TabSysinfo.svelte` (`/api/sysinfo/governor`).
  - Integrate Speedtest history table in `TabSpeedtest.svelte` (`/api/speedtest/history`).
  - Integrate Proxy watchdog status & config switcher in `TabProxy.svelte` (`/api/proxy/watchdog`, `/api/proxy/config`).
  - Implement Backup export, import, and cloud sync modal (`/api/backup/*`).

---

### Phase 6: Android Resource Optimization, Verification & Embedded Binary Build
- [x] **Resource-Aware Polling Throttling:**
  - Modify `frontend/src/stores/sysinfo.svelte.ts`:
    - Poll at 2000ms when `activeTab === 'overview' || activeTab === 'sysinfo'`.
    - Drop poll rate to 10000ms (10 seconds) when user is on background tabs (Terminal, Files, Proxy, Scrcpy) to save battery and reduce thermal load.
- [x] **TypeScript Build & Lint Verification:**
  - Run `pnpm run check` and `pnpm run build` in `frontend/` to verify zero type errors.
  - Verify static assets in `frontend/dist/` are under ~2.5MB total uncompressed.
- [x] **Cross-Compilation & Binary Size Benchmark:**
  - Compile single binary: `GOOS=android GOARCH=arm64 go build -ldflags "-s -w"`
  - Verify embedded binary size is within target ~10-12MB.
- [x] **Physical Android Target Validation (KernelSU/Magisk):**
  - Test on physical Android device:
    - Idle RAM usage < 20MB.
    - CPU usage during active telemetry < 2%.
    - One-handed mobile bottom navigation responsiveness.
    - Zero console error logs across all 22 tabs and both UI style paradigms.

---

## 7. Verification Checklist

Before closing the rewrite, verify:
- [x] Neobrutalist mode has 2px solid borders, 4px sharp radii, and 4px offset hard drop-shadows across all cards.
- [x] Modern Clean mode has 1px subtle borders, 16px soft radii, and smooth blur backdrop.
- [x] All 8 color themes switch instantly without page refresh, and persist across browser reloads.
- [x] AMOLED pitch black (`#000000`) exhibits visible elevation shadows without clipping.
- [x] Light mode text passes WCAG AA contrast standards (> 4.5:1) with zero illegible light-green/amber text.
- [x] Saving DNS (`/api/network/dns`) applies DNS1 and DNS2 correctly without corrupting system resolver.
- [x] Saving MAC Filter (`/api/hotspot/mac-filter`) correctly preserves blocked and allowed MAC lists.
- [x] Terminal, Scrcpy, and Logcat WebSockets connect cleanly via both `pnpm dev` and embedded production binary.
- [x] All 22 tabs are fully functional with zero ghost 404 endpoints.
