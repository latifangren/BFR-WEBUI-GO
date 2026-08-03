# Roadmap Pengembangan BFR-WEBUI-GO

Dokumen ini memetakan rencana peningkatan dan peta jalan pengembangan (*development roadmap*) untuk panel kontrol sistem Android **BFR-WEBUI-GO** (Magisk / KernelSU / APatch).

---

## 🌟 1. Fitur Utama Tingkat Lanjut (Proposed Go-Powered Advanced Features)

Berikut adalah 5 fitur tingkat lanjut berbasis kekuatan biner native Go yang ditempatkan pada urutan teratas peta jalan pengembangan:

* **[ ] 📡 Cellular Modem AT Controller & Band Locking (4G/5G)**: Interaksi langsung dengan port serial modem (`/dev/ttyUSB*`, `/dev/smd*`, `/dev/atcmd*`) untuk menampilkan metrik seluler presisi (*RSRP, RSRQ, SINR, Cell ID, Active Band*) serta fitur **Band Locking** (mengunci frekuensi LTE/5G tertentu langsung dari WebUI).
* **[ ] 💿 Dynamic USB Gadget Emulator (ISO Mount to PC & Virtual HID Input)**: Kontrol modul USB Gadget (`configfs`) dari WebUI untuk melakukan mount file `.iso` di HP sebagai bootable USB CD-ROM / Flashdisk ke PC (DriveDroid-style) serta emulasi keyboard/mouse USB virtual nirkabel.
* **[ ] 💬 Telegram Remote SMS OTP Auto-Forwarder & Dialer Listener**: Pemantauan database SMS SQLite sistem secara *event-driven* di latar belakang untuk mengekstrak kode OTP/token verifikasi dan meneruskannya otomatis ke Bot Telegram dalam waktu <1 detik dengan *0% wakelock* baterai.
* **[ ] 📈 Real-Time Traffic DPI (Deep Packet Inspection) & App Bandwidth Throttler**: Penyadapan paket data per-aplikasi (`nfqueue` / raw socket) dengan tampilan grafik pemakaian kuota real-time via WebSocket serta pembatasan kecepatan (*bandwidth limiter*) per-aplikasi/IP.
* **[ ] ⚡ Event-Driven Smart Thermal & System Governor Tuning**: Penyesuaian frekuensi/governor CPU dinamis berbasis deteksi sentuhan layar (`/dev/input/`) atau frame rate rendering (`surfaceflinger`) tanpa *polling loop* untuk menghemat baterai.

---

## 🛠️ 2. Peta Jalan Optimasi Teknis & Performa (v1.3.0+)

* **[ ] 🔴 Eliminasi Overhead Subshell Shell Root (`internal/charger/charger.go`)**: Mengganti eksekusi shell `su -c test -w` saat memindai izin charging dengan panggilan native Go `os.OpenFile` / `unix.Access` untuk menghemat ~10–20ms per pemanggilan dan mengeliminasi *context switch* CPU.
* **[ ] 🔴 Universal Panic Recovery Middleware (`internal/worker/worker.go`)**: Menambahkan `defer recover()` pada goroutine latar belakang (`worker.go`, `bot.go`, `ssh.go`) agar kesalahan format sysfs kernel HP tidak pernah memicu *crash* biner utama.
* **[ ] 🔴 In-Memory 750ms TTL Cache `/proc` (`internal/sysinfo/sysinfo.go`)**: Menambahkan in-memory TTL cache 750ms pada `GetStats()` untuk menghilangkan I/O berkas `/proc` berulang saat browser melakukan polling cepat.
* **[ ] 🟡 Pemindai Sysfs Charger Dinamis (`internal/charger/charger.go`)**: Pemindaian otomatis direktori `/sys/class/power_supply/*/` berdasarkan kata kunci charging jika jalur kandidat statis tidak ditemukan.
* **[ ] 🟡 Pencarian Dinamis Core Proxy `$PATH` & Modul (`internal/proxy/proxy.go`)**: Pencarian biner `mihomo`/`clash` secara dinamis via `exec.LookPath()` dan direktori modul Magisk/KSU.
* **[ ] 🟡 Daur Ulang Buffer `sync.Pool` Speedtest (`internal/speedtest/speedtest.go`)**: Menggunakan `sync.Pool` buffer 32KB pada worker loop speedtest untuk menghemat alokasi *Garbage Collector* (GC).

---

## ✅ 3. Fitur Terpasang & Catatan Rilis (v1.2.0 Completed Checklist)

* **[x] Direktori Storage Persisten (`/data/adb/bfr_webui_go/data/`)**: Penyimpanan seluruh berkas konfigurasi & data di direktori khusus yang 100% aman dari terhapus saat modul di-update.
* **[x] Enkripsi Password & API Ganti Password (`auth.json`)**: Penyimpanan password Salted SHA-256 Hash, endpoint REST `POST /api/auth/change-password`, `GET /api/auth/status`, dan badge quick-fill login adaptif.
* **[x] Navigasi Mobile 5-Kolom Grid & Slide-Up Bottom Sheet Modal**: Desain navigasi bawah 5-kolom responsif dengan panel bottom sheet melayang (`z-[999]`).
* **[x] Floating Navigation Toggle (`🧭 Nav`) & Mode Full-Screen Mobile**: Fitur menyembunyikan navbar bawah (`Hide ✕`) menjadi tombol floating pill ringkas untuk tampilan 100% full-screen.
* **[x] Tabbed Settings Modal 3-Tab**: Redesain modal Settings menjadi 3 Tab ringkas (📦 **Backup & Restore**, 🔐 **Security**, ☁️ **Cloud Sync**) lengkap dengan tombol `Close ✕` footer.
* **[x] URL Hash Tab Persistence (`#tab`) & Support Tombol Back Hardware HP**: Sinkronisasi URL fragment `#tab` sehingga refresh browser tetap di tab terakhir dan tombol *Back* fisik HP bernavigasi antar tab secara mulus.
* **[x] Dynamic Storage Usage Bar**: Indikator persentase memori dengan segmen warna *Used (Amber)* dan *Free (Hijau)* yang terisi secara presisi.
* **[x] Redesain Halaman Login Neo-Brutalist Glassmorphism**: Tampilan login card modern dengan badge info perangkat, toggle mata password, dan logo sosmed lokal 100% offline (Telegram, Facebook, GitHub).
* **[x] PWA Service Worker Cache Buster (`sw.js`)**: Pembaruan tembolok `bfr-webui-v1.2.0-b4` dengan strategi *Network-First* untuk dokumen HTML.
* **[x] Native Speedtest Engine & WebDAV Cloud Sync**: Pengujian kecepatan internet multi-thread Go murni dengan trace lokasi/ISP/Colo serta pengunggah otomatis WebDAV.
* **[x] LuCI OpenWrt Category Dropdown Navigation (Desktop)**: Navigasi header desktop 5 kategori (**Status ▾**, **System ▾**, **Services ▾**, **Network ▾**, **Extras ▾**).
* **[x] Universal Hardware Smart Charger Limiter**: Pemutus arus fisik Qualcomm PMIC (`force_main_fcc` 0 mA) pada Google Pixel 5 & HP Snapdragon.
* **[x] Modul Sistem Utama**: Native Root Web Terminal (PTY), Scrcpy Screen Mirroring, SoftAP Hotspot Controller, Vnstat Bandwidth Tracker, & Live Logcat Stream.

---

*Terakhir diperbarui: 4 Agustus 2026*
