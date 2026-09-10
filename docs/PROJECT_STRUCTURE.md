# Struktur Proyek BFR-WEBUI-GO

Dokumen ini menjelaskan arsitektur folder dan struktur berkas repositori **BFR-WEBUI-GO**, mencakup komponen Backend Go, Frontend Modern Svelte 5, serta komponen Modul Android (Magisk/KernelSU/APatch).

---

## 📁 Pohon Direktori Utama

```text
BFR-WEBUI-GO/
├── main.go                      # Entry point aplikasi, inisialisasi server & flag CLI
├── module.prop                  # Metadata modul Android (id, name, version v1.2.2, author)
├── customize.sh                 # Script instalasi & setup izin saat di-flash via Root Manager
├── service.sh                   # Script startup otomatis saat booting perangkat Android
├── system.prop                  # System properties bawaan modul Android
├── tweaks.json                  # Konfigurasi bawaan untuk optimasi jaringan & kernel
│
├── .github/                     # GitHub Actions CI/CD Workflows
│   └── workflows/
│       ├── codeql.yml           # Static security analysis
│       ├── quality.yml          # Frontend check & Go unit test verification
│       └── release.yml          # Automated release & Magisk zip packager
│
├── docs/                        # Folder Dokumentasi Proyek
│   ├── INSTALLATION_ID.md       # Panduan Instalasi (Bahasa Indonesia)
│   ├── INSTALLATION_EN.md       # Panduan Instalasi (English)
│   ├── DEVELOPMENT.md           # Panduan Pengembangan & Kompilasi
│   ├── DESIGN.md                # Spesifikasi Desain UI/UX & Tema
│   ├── API_REFERENCE.md         # Referensi REST API & WebSocket
│   └── PROJECT_STRUCTURE.md     # Dokumen Struktur Proyek (Dokumen Ini)
│
├── internal/                    # Kode Sumber Backend (Go Internal Packages)
│   ├── auth/                    # Sesi login, Cookie SameSite=Strict, CSPRNG Token
│   ├── charger/                 # Kontrol sysfs pengisian daya baterai & auto-detect PMIC
│   ├── config/                  # Pengaturan Environment Variables terpusat (BFR_*)
│   ├── filemanager/             # Modul operasi berkas (ops, archive, search, perm)
│   ├── handlers/                # HTTP Routers & Middleware (body limit, security headers)
│   ├── hotspot/                 # Kontrol SoftAP hotspot, client ARP, & MAC filtering
│   ├── logger/                  # Stream log sistem & audit aktivitas real-time
│   ├── modem/                   # Kontrol modem seluler, kalkulasi sinyal RSRP/RSRQ/SINR
│   ├── modules/                 # Deteksi dan manajemen modul sistem Magisk/KSU/APatch
│   ├── nas/                     # Integrasi file sharing & WebDAV backup cloud sync
│   ├── network/                 # Tweaks sysctl, DNS resolver, RPS, MTU, & TTL dynamic
│   ├── power/                   # Eksekusi aksi daya (reboot, shutdown, bootloader, dll)
│   ├── proxy/                   # Kontrol daemon Clash/Mihomo & watchdog loop
│   ├── qos/                     # Manajemen Quality of Service & bandwidth shaping
│   ├── scrcpy/                  # Sesi Web Mirroring Layar Android & input touch
│   ├── smsviewer/               # Pembaca database SMS Android & inbox viewer
│   ├── ssh/                     # Kontrol daemon SSH / Dropbear
│   ├── sysinfo/                 # Polling hardware counters, CPU, RAM, Suhu, & Network
│   ├── telegram/                # Telegram bot remote management & command keyboards
│   ├── terminal/                # Interactive PTY Root Web Terminal via WebSocket
│   ├── tunnel/                  # Cloudflare / Cloudflared tunnel manager
│   └── vnstat/                  # Pemantauan penggunaan data lalu lintas jaringan proc
│
└── frontend/                    # Frontend Modern (Vite + Svelte 5 + TypeScript + Tailwind)
    ├── embed.go                 # Go embed.FS (menanamkan frontend/dist ke biner Go tunggal)
    ├── package.json             # Dependensi frontend & script kompilasi (v1.2.2)
    ├── vite.config.ts           # Konfigurasi bundler Vite & chunk splitting
    ├── svelte.config.js         # Konfigurasi compiler Svelte 5
    ├── tsconfig.json            # TypeScript configuration
    ├── index.html               # Entry point HTML Single Page Application
    │
    └── src/                     # Kode Sumber Frontend Svelte 5
        ├── App.svelte           # Root application shell & router view
        ├── main.ts              # Frontend mounting entry point
        │
        ├── assets/              # Aset Statis Frontend (qris.jpg, dll)
        │
        ├── api/                 # API Client layer
        │   └── client.ts        # Typed fetch client dengan interceptor & auth handler
        │
        ├── components/          # Komponen Svelte 5 Berbasis Kategori
        │   ├── layout/          # Header, Sidebar, BottomNav, Toasts, AppearanceModal
        │   ├── ui/              # Komponen Reusable (Card, Button, Badge, Modal, Input)
        │   └── tabs/            # Tab Views:
        │       ├── overview/    # Tab Overview & CasaOS Application Shortcuts
        │       ├── sysinfo/     # Tab System Info, Hardware Sensors, CPU Cores, Root Daemons
        │       ├── network/     # Tab Network Tweaks, DNS, TTL, MTU, Interfaces
        │       ├── vnstat/      # Tab Vnstat Bandwidth Accounting & Quota
        │       ├── proxy/       # Tab Clash/Mihomo Proxy Controller
        │       ├── modem/       # Tab Cellular Modem, Bands, & Signal Metrics
        │       ├── hotspot/     # Tab SoftAP Hotspot & Client Filtering
        │       ├── files/       # Modular Dual-Pane File Manager
        │       │   ├── types.ts            # Kontrak interface data murni
        │       │   ├── paneState.svelte.ts # Svelte 5 Runes State Manager per-panel
        │       │   ├── BookmarksBar.svelte # Quick-jump preset & pin kustom localStorage
        │       │   ├── FileDropzone.svelte # Overlay drag & drop upload terpadu
        │       │   ├── FileTable.svelte    # Tabel file responsif & aksi baris
        │       │   ├── FileModals.svelte   # Konsolidasi 12 modal dialog terpusat
        │       │   ├── FilePane.svelte     # Kontainer panel mandiri (breadcrumbs, toolbar)
        │       │   └── TabFileManager.svelte # Koordinator Single/Dual Pane & mobile tab switcher
        │       ├── terminal/    # Tab Web Terminal PTY xterm.js
        │       ├── scrcpy/      # Tab Android Screen Mirroring & Controls
        │       ├── charger/     # Tab Battery Charging Limiter & PMIC Controller
        │       ├── ssh/         # Tab Dropbear SSH Server Manager
        │       ├── sms/         # Tab SMS Inbox Reader & AT Commands
        │       ├── nas/         # Tab LAN Web Share & NAS File Server
        │       ├── tunnel/      # Tab Cloudflare / Cloudflared Tunnel
        │       ├── logs/        # Tab Live Daemon Logs & Logcat Stream
        │       ├── modules/     # Tab Magisk / KernelSU Module Manager Resmi
        │       ├── tools/       # Tab Peralatan Sistem & Snapshot Backup
        │       ├── telegram/    # Tab Bot Telegram Remote Management
        │       ├── speedtest/   # Tab Native Multi-Thread Speedtest
        │       ├── qos/         # Tab Smart Bandwidth Limiter & SQM CAKE
        │       └── about/       # Tab About, Key Features, & Donasi QRIS (Universal Themes)
        │
        ├── stores/              # Svelte 5 Rune Reactive Stores
        │   ├── auth.svelte.ts   # State otentikasi sesi & token
        │   ├── navigation.svelte.ts # State tab aktif, kategori, & hash routing
        │   ├── theme.svelte.ts  # AMOLED Dark / Light theme switcher
        │   └── toast.svelte.ts  # Global notification toast store
        │
        └── styles/              # Global Styles
            └── app.css          # Tailwind directives & Neo-Brutalist utility classes
```
