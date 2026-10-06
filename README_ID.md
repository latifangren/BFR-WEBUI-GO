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

## ⚡ Fitur Unggulan Proyek

<details>
<summary><b>✨ Klik untuk melihat / menyembunyikan semua fitur lengkap proyek</b></summary>
<br>

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

### 🚀 Kontrol Kernel, Daya & Perangkat Keras
- **Dynamic Hardware Charge Limiter**: Pemindai otomatis node sysfs multi-vendor dengan bypass hardware PMIC Qualcomm (`force_main_fcc` 0 mA) serta dukungan override sysfs manual.
- **Jadwal Reboot Otomatis**: Dual-mode scheduler (`Uptime Interval` sejak boot atau `Waktu Spesifik Setiap Hari`) dengan countdown timer riil dan kontrol daya instan (`Reboot`, `Recovery`, `Bootloader`, `Power Off`).
- **SoC & Governor Tuner**: Penskalaan CPU multi-cluster, kebijakan governor, dan pemantau distribusi frekuensi inti prosesor secara langsung.
- **Telemetri Layanan Daemon Root**: Pemantauan PID riil, persentase CPU ternormalisasi, dan konsumsi RAM fisik (MB/KB) untuk `webui`, `mihomo`, `dropbear`, dan `adbd`.

### 📶 Modem Baseband, Carrier Aggregation & Cell Locking
- **Multi-Tier RAT Locking (Android 7–15)**: Penguncian mutlak **4G LTE Only** via binary bitmask native Android 11+ tanpa kebocoran degradasi ke 3G/GSM, mode **5G & 4G Dual Lock** khusus, serta fallback cerdas untuk Android versi lawas.
- **eNodeB Tower Cell Locking**: Penguncian menara BTS fisik tingkat hardware (`EARFCN` + `PCI`) dengan deteksi multi-CA Primary Component Carrier (PCC) dan penyimpanan persisten saat reboot (`cell_lock.json`).
- **Pemindai Menara Seluler (Cell Scanner)**: Penemuan serving cell, agregasi sekunder CA, dan sel tetangga secara real-time dengan metrik kekuatan sinyal radio (RSRP, RSRQ, RSSI, SINR).
- **Proprietary Vendor Baseband Engines**: Dukungan kompilasi penuh untuk Qualcomm QRTR/DIAG/EFS2, Samsung SecRIL (`band_manager.dex`), dan modem Google Tensor Shannon.

### 🛡️ Suite Proxy Box for Magisk (BFM)
- **Pengontrol Daemon Multi-Core**: Pengelola terpadu untuk `mihomo`, `sing-box`, dan `clash` dengan pelacakan memori RSS real-time, kontrol siklus hidup, dan pengawas otomatis watchdog.
- **Mode Routing Transparan**: Dukungan mode `TPROXY`, `REDIRECT`, `TUN`, dan `MIXED` dengan pengalihan transparan IPv6 serta pemblokir QUIC UDP 443.
- **Matriks Routing Per-App**: Pengatur kebijakan Blacklist vs Whitelist interaktif dengan pencarian instan pada seluruh aplikasi pihak ketiga yang terpasang di Android.
- **Proxying Hotspot / Tethering**: Pengalihan langsung lalu lintas klien hotspot Wi-Fi dan USB tethering melalui core proxy via `ap.list.cfg`.
- **Sinkronisasi Subscription & GeoX**: Pengunduh remote subscription instan, pembaruan otomatis GeoIP (`Country.mmdb`) dan GeoSite, lengkap dengan tautan rilis upstream resmi MetaCubeX.
- **Integrasi File Manager**: Pintasan 1-klik dari tab Proxy langsung menuju folder `/data/adb/box` di File Manager BFR.

### 🌐 Jaringan, DNS & Diagnostik
- **Setelan Jaringan Persisten**: Pengubah Dynamic TTL persisten (melewati batas kuota tethering operator) dan Custom DNS resolver (Cloudflare, Google, AdGuard, Quad9) tersimpan di `network_config.json`.
- **Reset DNS ke Default**: 1-klik tombol reset untuk membersihkan aturan NAT iptables kustom dan memulihkan DNS bawaan DHCP / operator seluler.
- **Mesin Diagnostik Ping Tangguh**: Engine ping multi-platform dengan batas waktu 7 detik, toleransi packet loss, dan fallback eksekusi ICMP root.
- **QoS Bandwidth Manager Router-Grade**: Profil pembentuk lalu lintas 1-klik (*Gaming Anti-Lag*, *Fair Share*, *Quota Saver*) melalui Linux TC HTB native dan ingress policing.
- **Pencatatan Bandwidth VnStat**: Pengukur kecepatan transfer antarmuka real-time, grafik konsumsi harian/bulanan, dan pelacak kuota siklus tagihan.

### 📺 Akses Jarak Jauh, Layar & Shell
- **Terminal Web Interaktif & Scrcpy Screen Mirror**: PTY terminal root penuh melalui WebSockets (`xterm.js`) dan pencerminan layar H.264 latensi rendah dengan injeksi sentuhan gestur.
- **Bundled Dropbear SSH**: Daemon ARM64 statis dengan pembuatan kunci host otomatis dan autentikasi root (`bfr`).
- **Bot Telegram Interaktif**: Perintah jarak jauh (`/stats`, `/charger`, `/ssh`, `/proxy`, `/reboot`), menu keyboard tetap, dan notifikasi keamanan instan (overheat, baterai, pergantian IP, login SSH).
- **Dukungan Donasi Interaktif**: Kartu mengambang bertema kartu Pokemon TCG dan modal inspeksi zoom QRIS beresolusi tinggi dengan tombol unduh gambar 1-klik.

</details>
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

## 📥 Panduan Pemasangan

1. Unduh file `BFR-WEBUI-Magisk-v1.2.7.zip` dari halaman [Releases](https://github.com/latifangren/BFR-WEBUI-GO-release/releases).
2. Buka aplikasi **Magisk / KernelSU / APatch** -> buka tab **Modul** -> pilih **Pasang dari penyimpanan (Install from storage)** -> pilih file ZIP yang telah diunduh -> tunggu hingga proses selesai -> **Reboot HP**.
3. Buka browser di perangkat lain yang terhubung ke jaringan Wi-Fi / hotspot yang sama dan akses:
   - **HTTP**: `http://<IP-HP>` (port standar 80 default)
   - **HTTPS**: `https://<IP-HP>` (port standar 443 default, langsung akses tanpa port tambahan)

---

## 📺 Fitur Scrcpy Remote Screen & Catatan HTTPS

> ⚠️ **PENTING: Wajib Menggunakan Konteks Aman (HTTPS) untuk Fitur Scrcpy**

- Fitur **Scrcpy Remote Screen Control** memanfaatkan teknologi **WebCodecs API** untuk decoding video H.264 berlatensi ultra-rendah langsung di browser.
- Browser modern (Google Chrome, Microsoft Edge, Mozilla Firefox) memberlakukan kebijakan keamanan ketat yang **memblokir WebCodecs pada koneksi HTTP biasa** (misal: `http://<IP-HP>`). Jika dibuka via HTTP biasa, browser memblokir hardware decoder video sehingga layar tampil **blank hitam**.
- **Solusi**:
  1. Akses WebUI via **HTTPS**: `https://<IP-HP>` (HTTPS berjalan otomatis di port standar 443, sehingga cukup ketikkan alamat URL secara langsung tanpa port).
  2. Karena BFR-WEBUI-GO menggunakan sertifikat SSL self-signed bawaan (`internal/tlsgen/`), browser akan menampilkan peringatan keamanan saat pertama kali dibuka. Klik **"Tingkat Lanjut (Advanced)"** lalu pilih **"Lanjutkan ke <IP-HP> (tidak aman)" / "Proceed"**.
  3. *(Alternatif Chromium)*: Masukkan origin `http://<IP-HP>` ke dalam pengaturan `chrome://flags/#unsafely-treat-insecure-origin-as-secure` kemudian restart browser.

---

## 📦 Kompilasi & Varian Rilis (Dual Build Flavors)

`BFR-WEBUI-GO` menyediakan 2 varian rilis resmi:
1. **Full Private Flavor (Branch `dev`)**: Termasuk seluruh perangkat baseband diagnostik hardware Qualcomm QRTR/DIAG/EFS2, Samsung One UI SecRIL, dan Google Tensor Shannon.
2. **Public Core Flavor (Branch `release/no-modem`)**: 100% bebas dari kode maupun string modem privat vendor, tetap menyertakan Telemetri Seluler & Radio universal (`dumpsys`, routing SIM, termal 5G) dan SMS.

### Prasyarat
- **Go**: Versi 1.22 atau lebih baru
- **Bun atau Node.js**: Bun (direkomendasikan) atau Node.js 18+ (pnpm/npm)

### Kompilasi & Packaging Satu Perintah
Skrip build secara otomatis mendeteksi branch Git aktif dan menerapkan build tag yang sesuai:
- Pada Windows:
  ```cmd
  :: Otomatis deteksi branch (atau berikan argumen 'full' / 'nomodem')
  build_zip.bat
  ```
- Pada Linux / macOS / Android Termux:
  ```bash
  chmod +x build.sh
  ./build.sh
  ```

### Kompilasi Biner Standalone (Manual)
- **Build Penuh (Private)**:
  ```bash
  cd frontend && bun run build && cd ..
  CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags "-s -w" -o webui .
  ```
- **Build Publik Non-Modem**:
  ```bash
  cd frontend && bun run build && cd ..
  CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -tags nomodem -ldflags "-s -w" -o webui .
  ```

Untuk dokumentasi spesifikasi arsitektur isolasi dan uji forensik string, silakan baca [`docs/MODEM-ISOLATION.md`](docs/MODEM-ISOLATION.md).
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
- **Repositori Proyek**: [https://github.com/latifangren/BFR-WEBUI-GO-release](https://github.com/latifangren/BFR-WEBUI-GO-release)
