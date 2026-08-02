# Roadmap Pengembangan BFR-WEBUI-GO

Dokumen ini memetakan rencana peningkatan dan pengembangan untuk panel kontrol sistem Android `BFR-WEBUI-GO` yang terintegrasi (Magisk / KernelSU / APatch). Rencana ini berfokus pada efisiensi daya, performa, kemudahan penggunaan, dan keamanan.

---

## 1. Performa & Efisiensi Sumber Daya
* **[x] `sync.Pool` Memory Buffer Reuse**: Gunakan `sync.Pool` untuk pooling buffer `bytes.Buffer` atau slice byte pada penanganan upload/download file berukuran besar serta terminal WebSocket PTY untuk membatasi overhead Garbage Collector (GC) pada RAM Android rendah (2GB/3GB).
* **[x] Direct `/proc` & `/sys` Reading**: Hindari pemanggilan subprocess `su -c cat ...` pada berkas-berkas sysfs/procfs yang sebenarnya bisa dibaca langsung tanpa permission root khusus (misal: `/proc/meminfo`, `/proc/net/dev`) menggunakan `os.ReadFile` atau pembacaan `/proc/[pid]/comm`.
* **[x] Adaptive Frame Throttling untuk Scrcpy**: Lakukan pemantauan throughput WebSocket secara real-time pada stream Scrcpy screen capture, secara dinamis melompati frame (frame skipping) saat koneksi terdeteksi lambat atau jenuh.

---

## 2. Arsitektur & Desain Kode
* **[x] Root Command Executor Terpusat**: Sediakan helper `ExecSuContext(ctx, cmdStr)` dengan limit timeout default (maksimal 3-5 detik) menggunakan `context.WithTimeout` untuk mencegah penumpukan proses zombie/deadlock jika perintah sistem Android tidak merespons.
* **[x] Background Task Manager (Worker Pool)**: Implementasikan antrean tugas berbasis Go Channel untuk menangani operasi berat secara asinkron seperti kompresi/ekstraksi ZIP berukuran besar, speedtest, dan batch file operations.
* **[x] Isolasi Tweak Konfigurasi**: Pastikan semua perubahan sysctl dan kernel tweaks terisolasi ke file JSON tweaks eksternal tanpa ada penulisan baris perintah statis di kode handler.

---

## 3. UI/UX & Pengalaman Developer (DX)
* **[x] PWA (Progressive Web App) Support**: Tambahkan registrasi Service Worker (`sw.js`) dan berkas `manifest.json`. Memungkinkan pemasangan aplikasi panel kontrol ini langsung ke Android Home Screen dengan tampilan khas standalone app.
* **[x] Global Toast Notification Engine**: Ganti alert standar menggunakan sistem notifikasi toast dinamis berbasis Alpine.js store dengan transisi visual yang mulus.
* **[x] Embedded OpenAPI / Swagger UI**: Sediakan antarmuka dokumentasi API interaktif (Scalar UI atau Swagger UI) pada rute `/docs` untuk memudahkan integrasi script otomatisasi pihak ketiga.

---

## 4. Fitur & Integrasi Sistem Android
* **[x] Module Manager (Magisk / KernelSU / APatch)**: Tambahkan API `/api/modules` untuk mengelola modul root terpasang: melihat detail modul, mengaktifkan/menonaktifkan modul via berkas `disable`, serta upload file `.zip` modul untuk diinstal di latar belakang menggunakan `magisk --install-module`.
* **[x] Live Logcat Stream**: Sediakan streaming log sistem Android secara real-time menggunakan Server-Sent Events (SSE) atau WebSocket lengkap dengan opsi pencarian kata kunci dan filter log level (Debug, Info, Warn, Error).
* **[x] Thermal & CPU Governor Tweak**: Panel kontrol frekuensi CPU dan modul manager governor untuk mengubah mode scaling CPU secara instan (Performance, Schedutil, Powersave) langsung dari WebUI.
* **[x] Konfigurasi Backup & Restore**: Ekspor dan impor seluruh file konfigurasi (`charger_config.json`, `ssh_config.json`, tweaks, proxy rules) menjadi satu berkas kompresi `.tar.gz` portabel (diimplementasikan via kompresi JSON bundle).

---

## 5. Keamanan & Pengerasan Sistem (Security & Hardening)
* **Header-Based Anti-CSRF Defense**: Wajibkan verifikasi header khusus (`X-BFR-Request: 1`) pada seluruh request modifikasi (`POST`/`PUT`/`DELETE`). Skema ini mengeliminasi celah CSRF tanpa memerlukan manajemen token sesi yang rumit di sisi server.
* **Rate Limiting Middleware**: Batasi jumlah percobaan login pada rute `/api/auth/login` menggunakan algoritma *token bucket* per IP klien untuk menghadang upaya serangan *brute-force* pada antarmuka hotspot publik.
* **Optional HTTPS/TLS Support**: Dukungan opsional untuk mengaktifkan server HTTPS dengan memasukkan flag sertifikat TLS (`-cert`, `-key`) saat binary dijalankan.
* **Sistem Audit Log**: Catat aktivitas kritis perangkat (gagal login, eksekusi root command, perubahan file sistem, reboot) secara terpusat ke `/data/adb/modules/bfr_webui_go/security_audit.log` yang persisten lintas reboot.

---

## 6. Fitur Unggulan Native Go (Go-Powered Extensions)

### 🟢 Easy (Pengembangan Cepat & Tanpa Dependency Luar)
* **[ ] Native Speedtest & Bandwidth Stress Tester Engine**: Modul pengujian kecepatan internet (Download, Upload, Latency, Jitter) multi-thread berbasis HTTP Go murni tanpa membutuhkan biner eksternal (`curl`/`iperf3`).
* **[ ] Cloud Backup & Config Sync**: Pengunggahan otomatis berkas kompresi konfigurasi terenkripsi (`charger`, `ssh`, `telegram`, `tweaks`) ke server WebDAV/Cloud pribadi secara berkala.

### 🟡 Medium (Integrasi Package Go Standard & Procfs)
* **[ ] WebDAV Local Cloud / Network Drive**: Integrasi `golang.org/x/net/webdav` untuk menyajikan folder Android sebagai **Network Drive (Drive Z:)** di Windows File Explorer atau Mac Finder secara nirkabel via Wi-Fi (RAM < 5MB).
* **[ ] Real-Time Network Packet & App Traffic Analyzer**: Penguraian statistik pemakaian kuota internet per-antarmuka jaringan (`wlan0`/`rmnet`) dan monitoring alokasi data jaringan secara real-time.

---

## 7. Backlog Masa Depan (Future Enhancements)
* **[ ] Remote Device Hardware Telemetry & Thermal Graphing**: Streaming grafik sensor temperatur CPU/GPU, frekuensi per-core, dan penggunaan RAM secara live (interval 100ms) menggunakan WebSocket tanpa overhead CPU.
