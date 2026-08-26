# Panduan Instalasi BFR-WEBUI-GO (Bahasa Indonesia)

Dokumen ini berisi panduan langkah demi langkah untuk menginstal dan menjalankan **BFR-WEBUI-GO** di perangkat Android yang telah di-root (menggunakan Magisk, KernelSU, atau APatch).

---

## 📋 Persyaratan Sistem

1. Perangkat Android ter-root (Android 8.0+ direkomendasikan).
2. Salah satu Root Manager berikut:
   - **Magisk** v20.4+
   - **KernelSU** v0.6.0+
   - **APatch** v10.7+
3. Ruang penyimpanan minimal ~15MB.

---

## 🚀 Cara Instalasi

### Metode 1: Instalasi via Zip Modul (Direkomendasikan)

1. Unduh rilis file zip modul terbaru `bfr-webui-go-vX.Y.Z.zip` dari halaman Releases.
2. Buka aplikasi Root Manager Anda (**Magisk App**, **KernelSU App**, atau **APatch App**).
3. Masuk ke menu **Modules** / **Modul**.
4. Pilih **Install from storage** / **Pasang dari penyimpanan** dan pilih file zip yang telah diunduh.
5. Tunggu proses flashing selesai, lalu lakukan **Reboot / Muat Ulang Perangkat**.

---

### Metode 2: Instalasi Manual via Terminal / Shell Root

Jika Anda ingin memasang atau menguji binary secara manual tanpa membuat paket modul zip:

1. Salin binary `bfr_webui_go` ke direktori modul:
   ```bash
   su
   mkdir -p /data/adb/modules/bfr_webui_go
   cp /sdcard/Download/bfr_webui_go /data/adb/modules/bfr_webui_go/
   chmod 755 /data/adb/modules/bfr_webui_go/bfr_webui_go
   ```

2. Buat file `module.prop` dasar:
   ```bash
   cat << 'EOF' > /data/adb/modules/bfr_webui_go/module.prop
   id=bfr_webui_go
   name=BFR WebUI Go
   version=v1.2.0
   versionCode=120
   author=BFR
   description=Android System Control Panel & WebUI
   EOF
   ```

3. Buat script startup `service.sh`:
   ```bash
   cat << 'EOF' > /data/adb/modules/bfr_webui_go/service.sh
   #!/system/bin/sh
   MODDIR=${0%/*}
   until [ "$(getprop sys.boot_completed)" = "1" ]; do
       sleep 2
   done
   sleep 5
   $MODDIR/bfr_webui_go > /dev/null 2>&1 &
   EOF
   chmod 755 /data/adb/modules/bfr_webui_go/service.sh
   ```

4. Jalankan service secara langsung atau reboot perangkat:
   ```bash
   /data/adb/modules/bfr_webui_go/service.sh &
   ```

---

## 🌐 Cara Mengakses WebUI

1. Pastikan perangkat terhubung ke Wi-Fi atau gunakan IP lokal perangkat.
2. Buka browser (Chrome, Kiwi, Firefox, dll) di perangkat Android Anda atau perangkat lain di jaringan lokal yang sama.
3. Masuk ke URL:
   ```text
   http://localhost:80
   ```
   *Atau dari perangkat lain di Wi-Fi yang sama:*
   ```text
   http://<IP_ANDROID_ANDA>:80
   ```
4. Masukkan password default saat login:
   ```text
   bfr
   ```

---

## ⚙️ Kustomisasi Konfigurasi (Optional)

Anda dapat merubah password, port, atau path default dengan menyetel **Environment Variables** sebelum layanan berjalan. 

Contoh menambahkan password kustom di `/data/adb/modules/bfr_webui_go/service.sh`:

```bash
export PORT=80
export BFR_PASSWORD="PasswordSangatAman123"
export BFR_SU_BIN="ksu"  # jika menggunakan KernelSU
$MODDIR/bfr_webui_go > /dev/null 2>&1 &
```

---

## ❓ Troubleshooting & Pertanyaan Umum

- **WebUI tidak bisa dibuka:**
  Cek apakah proses berjalan menggunakan terminal root:
  ```bash
  su -c "pgrep -f bfr_webui_go"
  ```
- **Lupa Password:**
  Ubah variabel `BFR_PASSWORD` di `service.sh` atau restart service setelah menyetel env variable baru.
