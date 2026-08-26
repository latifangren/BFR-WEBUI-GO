# BFR-WEBUI-GO

> **Panel Kontrol Sistem Android & WebUI Ultra-Ringan**  
> Didesain khusus sebagai modul Magisk / KernelSU / APatch yang 100% offline-ready. Ditulis dalam bahasa Go modular yang tangguh, memanfaatkan Alpine.js pada frontend, serta dibalut antarmuka navigasi Kategori Dropdown ala LuCI OpenWrt Bootstrap bertema gelap AMOLED Neo-Brutalist menggunakan Tailwind CSS.

---

## 📊 Konsumsi Resource Riil di Android

Hasil pengukuran empiris langsung pada hardware Android target (Pixel 5 ARM64):

| Parameter Resource | Nilai Penggunaan Riil | Catatan Efisiensi |
| :--- | :--- | :--- |
| **Ukuran Biner di Disk** | **21 MB** | **100% Standalone** (Biner Go tunggal terkompilasi, 0 dependency Python/NodeJS) |
| **RAM Physical RSS (Total)** | **~30.8 MB** | **Sangat Hemat** (Berjalan mulus di HP Android RAM 2GB/3GB) |
| **RAM Dedicated (Private PSS)** | **~11.4 MB** | Konsumsi memori sangat bersih & minim |
| **Penggunaan CPU (Idle)** | **0.0% CPU** | Beban CPU 0% di latar belakang |
| **Penggunaan Swap Memory** | **0 KB** | 0% pengikisan memori flash internal |

---

## ⚡ Fitur Utama

- **Offline-Ready & Sangat Ringan**: Berjalan sebagai biner mandiri terkompilasi statis dengan konsumsi RAM yang sangat minim (~11MB PSS / ~30MB RSS).
- **Navigasi Kategori Dropdown Ala LuCI OpenWrt**: Antarmuka navigasi header baru berbasis 5 kategori dropdown (**Status ▾**, **System ▾**, **Services ▾**, **Network ▾**, **Extras ▾**) menghemat ruang hingga 60%.
- **Engine Speedtest Native (Trace Ala Ookla)**: Benchmark latensi & bandwidth multi-thread Go HTTP (Download/Upload/Ping/Jitter) lengkap dengan detail IP Public Klien, Nama Operator/ISP, Lokasi, dan Server Data Center (Kode IATA Colo).
- **WebDAV Cloud Backup & Auto-Sync**: Sinkronisasi otomatis latar belakang paket berkas terkompresi `.tar.gz` (`charger`, `ssh`, `telegram`, `tweaks`) ke server WebDAV cloud pribadi.
- **Telegram Bot Remote Management & Interactive Keyboards**:
  - Penuh kontrol jarak jauh via perintah `/start`, `/stats`, `/charger`, `/ssh`, `/proxy`, `/hotspot`, `/modules`, `/ip`, dan `/reboot`.
  - Menu tombol Reply Keyboard persisten 4 baris & Inline Action buttons interaktif untuk eksekusi 1-klik tanpa ketik manual.
  - Sakelar toggle notifikasi granular untuk battery guard, overheat alert (>45°C), status SSH, perubahan IP Publik, dan koneksi klien hotspot.
- **Bundled Static Dropbear SSH Daemon**: Biner statis `dropbear` ARM64 terintegrasi dengan pembuat host key otomatis, autentikasi kata sandi root (`bfr`), dan dukungan LAN penuh (`0.0.0.0:2222`).
- **Universal Hardware Smart Charger Limiter**: Auto-scanner sysfs multi-vendor dengan pemutus arus fisik Qualcomm PMIC (`force_main_fcc` 0 mA) serta dukungan custom path override.
- **Model Keamanan Ketat**: Sanitasi parameter input yang aman, proteksi celah CSRF dengan verifikasi Header Kustom, pembatasan laju IP (Rate Limiting), serta keamanan sesi SameSite Lax.
- **Pengontrol Core Proxy**: Manajemen penuh daemon Clash / Mihomo. Mendukung pertukaran mode proxy secara real-time (Rule, Global, Direct, Script) serta pemantauan log langsung via WebSockets.
- **Kustomisasi Optimasi Jaringan Android**:
  - Konfigurasi Sysctl dinamis (TCP Congestion BBRv2, alokasi memori buffer, dan parameter inti stack TCP).
  - Pengalihan DNS kustom terenkripsi/port standar via injeksi iptables DNAT.
  - Opsi penyamaran (spoofing) TTL & Hop Limit untuk IPv4 dan IPv6.
  - Tuning otomatis Receive Packet Steering (RPS) per interface untuk memaksimalkan jaringan seluler.
  - Pengendali dinamis nilai MTU dan panjang transmisi antrean antarmuka (TxQueueLen).
- **Manajer Modul Magisk / KernelSU / APatch**: Panel visual untuk melihat daftar modul terpasang, detail metadata (`module.prop`), menonaktifkan/mengaktifkan modul (via trigger file `disable`), serta instalasi/flash modul baru langsung melalui pengunggahan file `.zip`.
- **Governor CPU & Pemantau Suhu**: Grafik beban frekuensi per-core CPU, info thermal zone perangkat, serta pemilih governor dinamis (Performance, Schedutil, Powersave).
- **Logcat Android Real-Time**: Aliran baris log sistem Android langsung via koneksi WebSocket dengan filter level log (Debug, Info, Warn, Error) dan kolom pencarian dinamis.
- **Cadangan & Pulihkan Konfigurasi**: Ekspor unduhan 1-klik dan impor (restore) seluruh file konfigurasi sistem (tweak, smart charger, clash rules, SSH) dalam bentuk berkas paket data terenkripsi aman.
- **Kontrol SoftAP & Hotspot**: Kustomisasi hotspot Wi-Fi, pemantauan status client terhubung (tabel DHCP/ARP leases), dan opsi blokir perangkat.
- **Web Terminal Root (PTY)**: Konsol interaktif shell root penuh langsung pada browser yang ditenagai oleh pty Go dan WebSockets.
- **Manajer File Tangguh**: Operasi manajemen file lengkap (baca, tulis, salin, potong-tempel clipboard, modifikasi perizinan `chmod`/`chown` rekursif, pencarian cepat, serta pengarsip ZIP aman dari eksploitasi ZipSlip).
- **Dukungan PWA**: Antarmuka web dapat dipasang di Home Screen perangkat dengan Service Worker offline caching.

---

## 🌐 Dokumentasi Lengkap

Panduan langkah demi langkah cara pemasangan, opsi kustomisasi, serta penggunaan terdokumentasi dalam dua bahasa:

- 🇬🇧 [English Installation & Operation Guide](./docs/INSTALLATION_EN.md)
- 🇮🇩 [Panduan Instalasi Bahasa Indonesia](./docs/INSTALLATION_ID.md)

---

## ⚙️ Variabel Lingkungan (Environment Overrides)

BFR-WEBUI-GO dapat dikustomisasi sepenuhnya via variabel lingkungan. Pengaturan bawaan di bawah ini akan digunakan secara otomatis jika variabel tidak diset.

Templat acuan lengkap dapat dilihat di berkas [`env.example`](./env.example):

| Variabel | Nilai Bawaan | Deskripsi |
|---|---|---|
| `PORT` | `80` | Port HTTP layanan WebUI |
| `BFR_PASSWORD` | `bfr` | Kata sandi masuk WebUI |
| `BFR_SU_BIN` | `su` | Jalur biner root executor (seperti `su`, `ksu`, `apatch`) |
| `BFR_MODULE_DIR` | `/data/adb/modules/bfr_webui_go` | Direktori instalasi modul |
| `BFR_BOX_BASE` | `/data/adb/box` | Jalur basis framework Box/Proxy |
| `BFR_CLASH_API` | `http://127.0.0.1:9090` | Endpoint pengontrol API Clash |
| `BFR_LEASES_FILE` | `/data/misc/dhcp/dnsmasq.leases`| Berkas riwayat sewa DHCP SoftAP |
| `BFR_ALLOWED_DIRS` | `/sdcard,/storage,/data/adb...` | Batasan folder kerja FileManager |

---

## 🛠️ Melakukan Kompilasi Manual

Untuk membangun biner target Android ARM64 secara manual dari source code:

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o webui main.go
```

---

## 📄 Lisensi

Proyek ini menggunakan lisensi resmi MIT.
