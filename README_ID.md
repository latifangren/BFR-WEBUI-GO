# BFR-WEBUI-GO

> **Panel Kontrol Sistem Android & WebUI Ultra-Ringan**  
> Didesain khusus sebagai modul Magisk / KernelSU / APatch yang 100% offline-ready. Ditulis dalam bahasa Go modular yang tangguh, memanfaatkan Alpine.js pada frontend, serta dibalut antarmuka bertema gelap AMOLED Neo-Brutalist menggunakan Tailwind CSS.

---

## ⚡ Fitur Utama

- **Offline-Ready & Sangat Ringan**: Berjalan sebagai biner mandiri (~10MB) terkompilasi statis dengan konsumsi RAM yang sangat minim (~15MB RAM RSS).
- **Model Keamanan Ketat**: Sanitasi parameter input yang aman, proteksi celah CSRF dengan verifikasi Header Kustom, pembatasan laju IP (Rate Limiting), serta keamanan sesi SameSite Strict.
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
- **Smart Battery Charger**: Kontrol pengisian daya otomatis berbasis ambang batas kapasitas baterai melalui sysfs charging switch emulator.
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
| `PORT` | `8080` | Port HTTP layanan WebUI |
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
