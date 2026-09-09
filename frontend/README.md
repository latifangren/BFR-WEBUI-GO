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
- `src/components/layout/`: Header, Sidebar, Bottom Navigation, Toast notifications.
- `src/components/ui/`: Komponen UI reusable (Card, Button, Badge, Modal, Input, Toggle).
- `src/components/tabs/`: Tab views (Overview, System, Network, Vnstat, Proxy, Modem, Hotspot, Filemanager, Terminal, Scrcpy, Charger, SSH, SMS, Logs, About).
- `src/stores/`: Reactive Svelte 5 rune stores (`auth`, `navigation`, `theme`, `toast`).
- `src/api/client.ts`: Typed fetch API client dengan error handling.
- `embed.go`: Ekspor `DistFS` untuk penanaman aset ke Go binary.
