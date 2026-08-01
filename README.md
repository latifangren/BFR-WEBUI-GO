# BFR-WEBUI-GO

> **Panel Kontrol System Android & WebUI Ultra-Ringan**  
> Didesain khusus sebagai modul Magisk / KernelSU / APatch yang 100% offline-ready, ditulis dalam bahasa Go modular dengan antarmuka Alpine.js & Tailwind CSS.

---

## ⚡ Fitur Utama

- **Offline-Ready & Ringan**: Binary tunggal (~10MB) terkompilasi statis dengan konsumsi RAM sangat rendah.
- **Manajemen Proxy & Core**: Kontrol daemon Clash / Mihomo, pergantian mode (Rule, Global, Direct, Script), serta pemantauan log real-time via WebSocket.
- **Pengaturan Jaringan & Tweaks**:
  - Konfigurasi Sysctl (TCP Congestion BBR2, Buffer optimization, dll).
  - Pengaturan DNS kustom & DNS Spoofing.
  - Pengaturan TTL & HL Spoofing (IPv4 & IPv6).
  - Tuning Packet Steering / RPS (Receive Packet Steering) per interface.
  - Pengaturan MTU dan TxQueueLen dinamis.
- **SoftAP & Hotspot Controller**: Manajemen hotspot Android, pemantauan perangkat terhubung (ARP table), dan pembatasan client.
- **Pengontrol Pengisian Daya (Charger Control)**: Manajemen ambang pengisian daya baterai & kontrol sysfs otomatis.
- **SMS Viewer**: Membaca SMS masuk (termasuk kode OTP/2FA) langsung dari dashboard.
- **Terminal Web (PTY)**: Terminal interaktif penuh berbasis PTY & WebSockets dengan hak akses root.
- **Manajer File Terintegrasi**: Akses pembacaan, pengunggahan, pengeditan, dan pengunduhan file sistem.
- **Informasi Sistem & Hardware**: Pemantauan beban CPU, penggunaan RAM, status baterai, suhu thermal, dan statistik jaringan.

---

## 🌐 Dokumentasi & Panduan Instalasi

Dokumentasi lengkap panduan instalasi dapat dilihat pada folder [`docs/`](./docs/):

- 🇮🇩 [Panduan Instalasi Bahasa Indonesia](./docs/INSTALLATION_ID.md)
- 🇬🇧 [English Installation Guide](./docs/INSTALLATION_EN.md)

---

## ⚙️ Konfigurasi Variabel Lingkungan (Environment Variables)

BFR-WEBUI-GO dapat dikustomisasi menggunakan variabel lingkungan. Pengaturan default otomatis digunakan jika variabel tidak diset.

Lihat contoh lengkap di file [`env.example`](./env.example):

| Variabel | Default | Deskripsi |
|---|---|---|
| `PORT` | `8080` | Port HTTP tempat WebUI berjalan |
| `BFR_PASSWORD` | `bfr` | Password autentikasi login WebUI |
| `BFR_SU_BIN` | `su` | Perintah/binary root (misal `su`, `ksu`, `apatch`) |
| `BFR_MODULE_DIR` | `/data/adb/modules/bfr_webui_go` | Direktori tempat modul terinstall |
| `BFR_BOX_BASE` | `/data/adb/box` | Path basis untuk modul Box |
| `BFR_CLASH_BASE` | `/data/adb/clash` | Path basis untuk modul Clash |
| `BFR_CLASH_API` | `http://127.0.0.1:9090` | Endpoint REST API Clash / Mihomo |
| `BFR_SMS_DB` | *(Auto-scan)* | Path kustom database SMS SQLite |
| `BFR_LEASES_FILE` | `/data/misc/dhcp/dnsmasq.leases` | Path file DHCP leases SoftAP |
| `BFR_ALLOWED_DIRS` | `/sdcard,/storage,/data/adb,...` | Batas direktori yang diizinkan untuk File Manager |

---

## 🛠️ Kompilasi dari Source

Untuk melakukan kompilasi manual targeting Android arm64:

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bfr_webui_go main.go
```

---

## 📄 Lisensi

Proyek ini dirilis di bawah lisensi MIT.
