# Struktur Proyek BFR-WEBUI-GO

Dokumen ini menjelaskan arsitektur folder dan struktur berkas repositori **BFR-WEBUI-GO**, mencakup komponen Backend Go, Frontend Web (HTML/CSS/JS), serta komponen Modul Android (Magisk/KernelSU/APatch).

---

## 📁 Pohon Direktori Utama

```text
BFR-WEBUI-GO/
├── main.go                      # Entry point aplikasi, inisialisasi server & flag CLI
├── build.sh                     # Script kompilasi Linux/Bash untuk target Android arm64
├── build.bat                    # Script kompilasi Windows/CMD untuk target Android arm64
├── env.example                  # Templat opsional untuk konfigurasi Environment Variables
├── module.prop                  # Metadata modul Android (id, name, version, author)
├── customize.sh                 # Script instalasi & setup izin saat di-flash via Root Manager
├── service.sh                   # Script startup otomatis saat booting perangkat Android
├── system.prop                  # System properties bawaan modul Android
├── tweaks.json                  # Konfigurasi bawaan untuk optimasi jaringan & kernel
│
├── docs/                        # Folder Dokumentasi Proyek
│   ├── INSTALLATION_ID.md       # Panduan Instalasi (Bahasa Indonesia)
│   ├── INSTALLATION_EN.md       # Panduan Instalasi (English)
│   └── PROJECT_STRUCTURE.md     # Dokumen Struktur Proyek (Dokumen Ini)
│
├── internal/                    # Kode Sumber Backend (Go Internal Packages)
│   ├── auth/                    # Sesi login, Cookie SameSite=Strict, CSPRNG Token
│   ├── charger/                 # Kontrol sysfs pengisian daya baterai
│   ├── config/                  # Pengaturan Environment Variables terpusat (BFR_*)
│   ├── filemanager/             # Modul operasi berkas terpisah (ops, archive, search, perm)
│   ├── handlers/                # HTTP Routers & Middleware (body limit, security headers)
│   ├── hotspot/                 # Kontrol SoftAP hotspot & parsing client ARP/leases
│   ├── logger/                  # Stream log sistem & audit aktivitas real-time
│   ├── network/                 # Tweaks sysctl, DNS resolver, RPS, MTU, & TTL spoofing
│   ├── power/                   # Eksekusi aksi daya (reboot, shutdown, bootloader, dll)
│   ├── proxy/                   # Kontrol daemon Clash/Mihomo & watchdog loop
│   ├── scrcpy/                  # Sesi Web Mirroring Layar Android & input touch
│   ├── smsviewer/               # Pembaca database SMS Android (mmssms.db / telephony.db)
│   ├── ssh/                     # Kontrol daemon SSH / Dropbear
│   ├── sysinfo/                 # Polling hardware counters, CPU, RAM, Suhu, & Network Detail
│   ├── terminal/                # Interactive PTY Root Web Terminal via WebSocket
│   └── vnstat/                  # Pemantauan penggunaan data lalu lintas jaringan proc
│
└── web/                         # Berkas Antarmuka Web (Embedded Frontend)
    ├── embed.go                 # Directives //go:embed untuk membungkus aset statis
    ├── index.html               # Halaman utama (Single Page Dashboard + Alpine.js data)
    │
    ├── templates/               # Modul Templat HTML (Terorganisir Berdasarkan Fitur)
    │   ├── layout/              # Header, Sidebar, & Modal Dialogs Global
    │   ├── overview/            # Tab Dashboard Utama & Ringkasan Sistem
    │   ├── network/             # Tab Pengaturan Jaringan & DNS Switcher
    │   ├── proxy/               # Tab Kontrol Proxy Clash & Mihomo
    │   ├── filemanager/         # Tab Manajer Berkas (Baru: Storage Bar & Checkbox)
    │   ├── terminal/            # Tab Terminal PTY Root Interaktif
    │   ├── tools/               # Tab Fitur Alat (SSH Control, Charger Control)
    │   ├── sms/                 # Tab Pembaca SMS Masuk
    │   ├── scrcpy/              # Tab Remote Screen Mirroring & Touch Input
    │   ├── hotspot/             # Tab Manajemen SoftAP Hotspot
    │   ├── sysinfo/             # Tab Informasi Sistem Lengkap & Grafik
    │   ├── logs/                # Tab Log Sistem & Audit Aktivitas
    │   └── about/               # Tab Informasi Aplikasi & Donasi
    │
    └── static/                  # Berkas Aset Statis Web
        ├── css/                 # Stylesheets (Tailwind CSS, Xterm.css, Custom Neo-Brutalist)
        └── js/                  # JavaScript Modules & Libraries
            ├── alpine.min.js    # Framework Reactive Alpine.js
            ├── xterm.js         # Library Terminal Emulation
            ├── app.js           # Main App Initializer & Store Registration
            └── modules/         # Modul JS Berdasarkan Fitur
                ├── common/      # Fungsi helper umum, toast, & modal donasi
                ├── sysinfo/     # Logika polling & pembaruan grafik hardware
                ├── network/     # Logika DNS switcher, sysctl, & RPS
                ├── filemanager/ # Logika manager file (Cut, Copy, Paste, Zip, Chmod, Image View)
                ├── proxy/       # Logika status & mode proxy Clash/Mihomo
                ├── terminal/    # Logika WebSocket PTY terminal
                ├── hotspot/     # Logika SoftAP client & toggle hotspot
                ├── vnstat/      # Logika pemantauan traffic data
                ├── charger/     # Logika pengontrol pengisian daya
                ├── sms/         # Logika pembaca SMS
                ├── scrcpy/      # Logika remote screen mirroring & input
                ├── ssh/         # Logika pengatur daemon SSH
                └── logs/        # Logika pembacaan & penyaringan log sistem
```

---

## 🧩 Penjelasan Komponen Utama

### 1. Root & Script Deployment
- **`main.go`**: Memulai server HTTP `net/http` pada port yang dikonfigurasi, menginisialisasi router middleware, dan mendengarkan signal OS (`SIGINT`/`SIGTERM`) untuk *graceful shutdown*.
- **`module.prop` & `customize.sh`**: Menyediakan metadata dan script instalasi otomatis saat modul di-flash di Magisk/KernelSU/APatch.
- **`env.example`**: Menyediakan templat kustomisasi environment variables seperti port kustom (`PORT`), password (`BFR_PASSWORD`), lokasi root binary (`BFR_SU_BIN`), atau path modul (`BFR_MODULE_DIR`).

### 2. Backend Subsystems (`internal/`)
- **`filemanager/`**: Didesain modular menjadi `filemanager.go` (dasar & read), `ops.go` (copy/move/delete massal), `permissions.go` (chmod/chown), `archive.go` (zip/unzip), dan `search.go` (pencarian & info storage).
- **`handlers/`**: Mengatur rute REST API dan WebSocket, dilengkapi middleware pembatas body (`MaxBytesReader`), header keamanan (`X-Frame-Options`, `nosniff`), serta verifikasi autentikasi cookie `SameSite=Strict`.
- **`sysinfo/`**: Menggunakan pembacaan sistem langsung dari `/proc` dan `/sys` tanpa ketergantungan eksternal, dilengkapi parser deteksi Gateway & DNS resolver dinamis untuk Android 8-16+.
- **`logger/`**: Menyediakan stream log audit aktivitas sistem real-time yang menangkap event keamanan, login, jaringan, dan aksi root.

### 3. Frontend Architecture (`web/`)
- **Single Page Application (SPA)** dengan **Alpine.js** & **Tailwind CSS**.
- Seluruh file HTML dan JS disusun secara **modular berbasis fitur (Feature-Based Directory Structure)** di dalam folder `web/templates/<fitur>/` dan `web/static/js/modules/<fitur>/`.
- Seluruh aset web dibungkus langsung ke dalam binary executable tunggal Go menggunakan `embed.FS`, sehingga aplikasi berjalan **100% offline tanpa membutuhkan koneksi internet**.

---

## 🛠️ Alur Kompilasi (Build Flow)

1. **Pemindaian & Embedding**: Go mengumpulkan `web/index.html`, `web/templates/**/*`, dan `web/static/**/*` menggunakan `embed.FS`.
2. **Kompilasi Binary**:
   ```bash
   GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o webui .
   ```
3. **Pengemasan Modul Zip**:
   File `webui`, `customize.sh`, `service.sh`, `module.prop`, `system.prop`, `tweaks.json`, dan `env.example` dikompresi menjadi `BFR-WEBUI-Magisk-vX.Y.Z.zip` siap flash.
