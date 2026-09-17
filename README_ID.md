# BFR-WEBUI-GO

> **Panel Kontrol Sistem Android & WebUI Ultra-Ringan Berkinerja Tinggi**  
> Didesain khusus sebagai modul Magisk / KernelSU / APatch yang 100% offline-ready. Dibangun menggunakan backend Go native modular dan frontend modern Svelte 5 + TypeScript + Tailwind CSS yang tertanam langsung ke dalam satu binary mandiri (~10-12MB).

---

## 📊 Konsumsi Resource Riil di Android (Pixel 5 ARM64)

Hasil pengukuran empiris langsung pada hardware Android target:

| Parameter Resource | Nilai Pengukuran | Keunggulan Arsitektur |
| :--- | :--- | :--- |
| **Ukuran Biner di Disk** | **~16 MB** | **100% Standalone** (Biner Go tunggal terkompilasi, tanpa dependensi runtime Python/Node.js) |
| **RAM Fisik RSS** | **~21 MB** | **Sangat Efisien** (Berjalan mulus di HP Android low-end RAM 2GB/3GB) |
| **RAM Khusus (Private PSS)**| **~11 MB** | Jejak memori privat sangat rendah |
| **Penggunaan CPU (Idle)** | **0.0% - 0.2%** | Beban CPU 0% di latar belakang, tanpa polling berlebih |
| **Penggunaan Swap / Storage**| **0 KB** | Nol pengikisan memori flash, melindungi masa pakai eMMC/UFS |

---

## ⚡ Fitur Unggulan

### 📁 File Manager Modular Dual-Pane (Dual Commander)
- **Tampilan Split Dual-Pane**: Tampilan dua panel berdampingan (side-by-side) pada desktop/tablet, dan tab pill switcher responsif (`[ Panel A | Panel B ]`) pada layar HP.
- **Transfer Silang Cepat**: Tombol instan `Copy to Other Pane` dan `Move to Other Pane` yang ditenagai endpoint batch `/api/files/batch` dengan proteksi fallback lintas partisi (*cross-device link*).
- **Quick Bookmarks Bar**: Preset folder sistem Android & Magisk bawaan (`/`, `/sdcard`, `/data/adb`, `/data/adb/modules`, `/data/local/tmp`) serta dukungan pin bookmark kustom yang tersimpan di `localStorage`.
- **Upload Drag & Drop**: Overlay dropzone layar penuh yang otomatis mendeteksi seretan file dan mengunggah langsung ke direktori aktif.
- **Peralatan Berkas Lengkap**: Editor kode monospasi, preset izin chmod oktal (`0755`, `0644`, `0777`), kompresi dan ekstraksi ZIP/TAR.

### 🎨 Matriks Kustomisasi (Appearance Studio)
- **Appearance Studio Ringkas**: Modal pengaturan tampilan yang kompak dan minimalis tanpa ruang berlebih.
- **9 Palet Warna Pilihan**: `Dark Navy`, `Pure AMOLED`, `Clean Light`, `Dracula`, `Nordic Frost`, `Cyberpunk 2077`, `Matrix Emerald`, `Retro Sunset`, dan **`Retro Deck`** (warna pastel ice dengan border berkarakter).
- **2 Gaya Paradigma Visual**:
  - **Neobrutalism**: Border tegas 2px, bayangan offset mekanis, sudut 4px, dan badge taktil.
  - **Modern Clean**: Border halus 1px, lekukan membulat lembut, dan bayangan ambient.
- **2 Tata Letak Navigasi**:
  - **Classic Top Bar**: Header pills kategori dengan dropdown flyout di desktop; bilah navigasi 5-kolom di HP dengan popover ramah sentuhan.
  - **Modern Sidebar**: Accordion drawer yang dapat diciutkan (Core, Network, System, Tools) di desktop; slide-over drawer di HP.

### 🚀 Optimasi Kernel & Sistem
- **BBR2 TCP Congestion Control**: Kontrol kongesti otomatis untuk jaringan nirkabel latensi rendah dan bandwidth tinggi.
- **System Optimizer Tweaks**: Tuning sysctl kernel permanen, TCP FastOpen, alokasi batas antrean, dan spoofing TTL cerdas (kompatibel Android 11+).
- **Dynamic Hardware Charge Limiter**: Pemindai sysfs otomatis dengan pemutus arus Qualcomm PMIC (`force_main_fcc` 0 mA) untuk menjaga keawetan baterai.
- **Telemetri Resource Root Daemon**: Pelacakan riil PID, persentase CPU ternormalisasi multi-core, dan konsumsi RAM (MB/KB) untuk `webui`, `mihomo`, `dropbear`, dan `adbd`.

### 🌐 Jaringan & Konektivitas
- **Modem Seluler & Penguncian Band**: Penguncian frekuensi multi-engine via AT serial Qualcomm (`/dev/smd11`, `/dev/ttyUSB*`), `cmd phone`, dan kode rahasia. Metrik sinyal instan RSRP, RSRQ, SINR, dan EARFCN.
- **Kontroler Core Proxy**: Pengelola daemon Clash / Mihomo dengan log live streaming, editor konfigurasi, watchdog, dan pergantian mode rule/global/direct.
- **Pencatatan Kuota VnStat**: Pengukur bandwidth real-time, grafik konsumsi harian/bulanan, dan monitor batas kuota siklus tagihan.
- **Terminal Web & Scrcpy Screen Mirroring**: Root terminal interaktif berbasis WebSocket (`xterm.js`) dan mirroring layar H.264 latensi rendah dengan kontrol gestur sentuh.

### 🤖 Manajemen Jarak Jauh & Sinkronisasi Cloud
- **Bot Telegram Interaktif**: Perintah kendali jarak jauh (`/stats`, `/charger`, `/ssh`, `/proxy`, `/reboot`), menu keyboard 4 baris, dan notifikasi keamanan otomatis (overheat, baterai, pergantian IP, login SSH).
- **Backup Cloud WebDAV**: Kompresi otomatis dan sinkronisasi berkas konfigurasi (`charger`, `ssh`, `telegram`, `tweaks`) ke server cloud WebDAV pribadi.
- **Dropbear SSH Terintegrasi**: Daemon static ARM64 bawaan dengan pembuatan kunci host otomatis dan autentikasi root (`bfr`).
- **Pusat Dukungan & Donasi**: Barcode pembayaran QRIS (`qris.jpg`) dan konfirmasi donatur instan via Telegram dan Facebook developer.

---

## 🛠️ Arsitektur & Tech Stack

```
Arsitektur BFR-WEBUI-GO
├── Frontend (Embedded SPA dalam biner Go tunggal)
│   ├── Svelte 5 (Runes: $state, $derived, $props)
│   ├── TypeScript (Pengetikan ketat di semua modul)
│   ├── Tailwind CSS (Token Desain Neobrutalism & Modern)
│   └── Vite (Pipeline bundling aset produksi)
│
└── Backend (Modul Go Native)
    ├── Go 1.22+ (Konkurensi native & beban memori sangat hemat)
    ├── Standard Library HTTP Mux & Middleware (Gzip, CSRF, Rate Limiter)
    ├── Kontroler Subsistem (charger, network, proxy, terminal, vnstat, modem)
    └── Runtime Modul Magisk / KernelSU / APatch (/data/adb/modules/bfr_webui_go)
```

---

## 📦 Kompilasi & Pembuatan Paket

### Prasyarat
- **Go**: Versi 1.22 atau lebih baru
- **Node.js & pnpm/npm**: Node.js 18+

### Membuat Paket ZIP Modul Magisk
Pada Windows:
```powershell
.\build_zip.bat
```
Pada Linux / macOS:
```bash
chmod +x build.sh
./build.sh
```
Paket ZIP modul Magisk (`BFR-WEBUI-Magisk-v1.2.3-local.zip`) akan dihasilkan di root proyek, siap diflash melalui Magisk, KernelSU, atau APatch manager.

### Kompilasi Biner Standalone (Cross-Compile)
Untuk mengompilasi biner Android ARM64 langsung:
```powershell
cd frontend
npm run build
cd ..
$env:CGO_ENABLED="0"; $env:GOOS="android"; $env:GOARCH="arm64"; go build -ldflags "-s -w" -o webui .
```

---

## 🔒 Arsitektur Keamanan

- **Autentikasi Sesi**: Berbasis cookie dengan proteksi `HttpOnly`, `SameSite=Lax`, dan token CSRF double-submit.
- **Proteksi Brute-Force**: Pembatas laju IP mengunci maksimal 5 kali percobaan gagal per menit (`HTTP 429`).
- **Sanitasi Jalur Input**: Validasi ketat terhadap path traversal pada operasi berkas (`CleanPath` & isolasi `AllowedDirs`).
- **Pengiriman Media yang Mulus**: Pengecualian media kompresi pada middleware Gzip untuk mencegah galat `Content-Length`.

---

## 📄 Lisensi & Pembuat

- **Pengembang / Pemelihara**: [latifangren](https://github.com/latifangren)
- **Lisensi**: MIT Open Source License
- **Repositori Proyek**: [https://github.com/latifangren/BFR-WEBUI-GO](https://github.com/latifangren/BFR-WEBUI-GO)
