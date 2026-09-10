# BFR-WEBUI-GO Frontend (Svelte 5 + TypeScript + Vite)

Frontend Single Page Application (SPA) modern untuk **BFR-WEBUI-GO**, dibangun menggunakan **Svelte 5 Runes**, **TypeScript**, **Tailwind CSS**, dan **Vite**, dikompilasi ke dalam biner tunggal Go melalui `embed.FS` (`frontend/embed.go`).

---

## 🛠️ Stack Teknologi
- **Framework**: Svelte 5 (`$state`, `$derived`, `$props`, `$effect`)
- **Language**: TypeScript (Strict type checking via `svelte-check` & `tsc`)
- **Styling**: Tailwind CSS + Neo-Brutalist design tokens (AMOLED dark / light modes)
- **Bundler**: Vite dengan asset chunk splitting teroptimasi
- **Icons**: Lucide Svelte (`@lucide/svelte`)
- **Terminal**: xterm.js + WebLinks + Fit Addon (`@xterm/xterm`, `@xterm/addon-fit`)

---

## 🚀 Perintah Pengembangan

```bash
# Instalasi dependensi
pnpm install

# Menjalankan dev server dengan Hot Module Replacement (HMR)
pnpm dev

# Type-checking & Svelte verification
pnpm check

# Build distribusi produksi ke frontend/dist/
pnpm build
```

---

## 📁 Struktur Direktori
- `src/App.svelte`: Shell aplikasi utama dengan responsive layout.
- `src/components/layout/`: Header (Navbar ringkas Opsi 1), Sidebar, Bottom Navigation, Toast notifications, AppearanceModal (Studio Kustomisasi Kompak).
- `src/components/ui/`: Komponen UI reusable (Card ber-tone semantik, Button, Badge, Modal, Input, Toggle).
- `src/components/tabs/`:
  - `overview/`: Tab Overview dengan CasaOS App Shortcuts dan Card Telemetri Lengkap (CPU, RAM used/total, Storage used/total).
  - `sysinfo/`: Tab System Info, Hardware Sensors, CPU Cores, dan Root Daemon Services Telemetry (PID, CPU %, RAM MB/KB).
  - `files/`: Modular Dual-Pane File Manager (`types.ts`, `paneState.svelte.ts`, `BookmarksBar`, `FileDropzone`, `FileTable`, `FileModals`, `FilePane`, `TabFileManager`).
  - `about/`: Tab About dengan branding BFR PRO, Magisk Key Features, dan Hub Donasi QRIS interaktif (kompatibel 9 tema).
  - Tab views lainnya: `network`, `vnstat`, `proxy`, `modem`, `hotspot`, `terminal`, `scrcpy`, `charger`, `ssh`, `sms`, `logs`, `modules`, `tools`, `telegram`, `speedtest`, `qos`.
- `src/stores/`: Reactive Svelte 5 rune stores (`auth`, `navigation`, `theme`, `toast`, `sysinfo`).
- `src/assets/`: Aset statis frontend (`qris.jpg`).
- `src/api/client.ts`: Typed fetch API client dengan error handling terpusat.
- `embed.go`: Ekspor `DistFS` untuk penanaman aset Vite langsung ke biner Go tunggal.
