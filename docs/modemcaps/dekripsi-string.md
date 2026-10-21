# Hasil Dekripsi String — Qct Modem Capabilities

**Target:** `modemcaps-2402-1d_0_it0k.apk` (`ru.fieldtest.modemcaps` v2402-1) · **Tanggal:** 21 Oktober 2026 · **Melengkapi:** `ModemCaps-Dokumentasi-Analisis.md` Bagian 7–9

---

## 1. Metode dan hasil

Mekanisme dekriptor `a.a(long, String[])` yang terpetakan di putaran pertama diimplementasikan ulang **persis** (Java, semantik overflow/shift/`short` identik dengan kode asli), lalu dijalankan terhadap semua konstanta call-site yang dikumpulkan dari 293 class hasil dekompilasi.

Satu koreksi penting soal bahan baku: blob `c.f763a` **tidak** diambil dari rendering source JADX (hasilnya meleset 115 karakter), melainkan diekstrak langsung dari `classes.dex` — urutan `const-string` di `<clinit>` class `c`, diverifikasi silang dengan panjang string di DEX string pool (UTF-16). Hasil yang konsisten dari dua jalur independen:

- Blob: 8 potongan — 7 × 8.191 + 1 × 3.298 = **60.635 karakter** (angka 60.750 di dokumentasi utama putaran pertama adalah salah hitung dan sudah dikoreksi).
- Call-site terkumpul: **4.172 referensi** di seluruh kode, berisi **2.113 konstanta unik**.
- Berhasil didekripsi terbaca: **2.104 string**. Sisa 9 konstanta adalah false positive penangkapan pola (konstanta numerik biasa, bukan string terenkripsi) — jadi cakupan dekripsi atas string asli adalah **100%**, tanpa satu pun call-site yang tertinggal.

Daftar lengkapnya ada di `ModemCaps-Kamus-String.tsv` (konstanta → plaintext, 2.113 baris).

## 2. Tiga tanda tanya putaran pertama — terjawab

**a) Proses shell di `am.java:1642` memang `su`.**
Konstanta di `ProcessBuilder` terdekripsi menjadi persis `su`. Yang ditulis aplikasi ke stdin shell root itu adalah skrip pengumpulan log dukungan: daftar isi `/dev` dan sysfs SoC, `chmod` pada node-node tertentu, sentuh socket `qmux_radio`, lalu dump `getprop` yang **disaring membuang** baris berisi `imei`, `imsi`, `esn`, `serialno`, `sno`, `hostname`, `tcp`, dan `dalvik` — hasil akhirnya ditulis ke `/sdcard/modemcaps_osinfo.log`. Jadi jalur shell ini adalah fitur "kumpulkan info perangkat untuk support", bukan mekanisme tersembunyi lain — tetap berjalan sebagai root, dan tetap hanya terpicu dari dialog atas tindakan pengguna.

**b) Target refleksi di `a/a.java:246-247` ternyata jinak.**
Bukan hidden-API framework sama sekali:

| Cache | Class | Method |
|---|---|---|
| `am.b` | `android.app.Activity` | `invalidateOptionsMenu()` |
| `am.f273a` | `java.io.File` | `lastModified()` |

Keduanya method publik biasa. Aplikasi mengenkripsi namanya semata karena *semua* string dienkripsi tanpa kecuali — bukan karena menyembunyikan akses terlarang. Catatan "sensitif greylist hidden-API" di putaran pertama dengan ini **gugur**.

**c) Tool SELinux bundel (`libc.so`) terkonfirmasi dari sisi Java.**
Dua string payload `supolicy --live` muncul di kode: varian utama berisi aturan terarah (domain aplikasi diberi akses baca debugfs/qti_debugfs/proc/sysfs, baca-tulis `system_data_file` dan `unlabeled`, akses penuh ke `diag_device`, socket, dan `qipcrtr_socket`), dan varian fallback yang jauh lebih longgar — domain ditandai `mlstrustedsubject` dan diberi aturan izinkan-semua `* * {*}`. Urutan varian mana yang dicoba lebih dulu dan lewat proses apa `supolicy` dipanggil tidak terlihat dari lapisan Java (kemungkinan dari native atau binary pendamping), tetapi keberadaan dan isi payload-nya kini fakta, bukan simpulan dari string table saja: aplikasi secara aktif melonggarkan kebijakan SELinux perangkat agar domainnya bisa menjangkau node diag dan EFS modem.

## 3. Temuan baru dari string plaintext

**Path EFS konkret di lapisan Java (21 path).** Sebelumnya hanya awalan path dari `.rodata` native yang diketahui; sekarang nama file persisnya terbaca, dan cocok satu-satu dengan fitur yang didokumentasikan:

- LTE RRC: `lte/rrc/efs/lte_feature_enable`, `lte_feature_disable`, `cell_selection_opt_params`, `cell_restrict_opt_params`, `cell_lock_list` (+ salinan `reg_files/.../csp/cell_lock_list`), `max_num_mimo_lyr`
- Penguncian: `lte/rrc/csp/pci_lock`, `lte/rrc/csp/earfcn_lock`, GERAN `feature_lock_arfcn_enabled`, `rr_efs_arfcn_list`, `fake_bts_cell_barring_enabled`, WCDMA `wcdma_rrc_freq_lock_item`
- CA / RF: `lte/common/ca_disable`, `lte/ML1/ard_config`, `lte/ML1/rx_select`, `mcs/trm/ant_div_switch_master`
- NR: `nr5g/RRC/cap_control_dl_256qam`, `cap_disable_ul_256qam`, `cap_limit_rf_mimo`, `cap_nrca_downgrade_1cc`

**Token fitur halaman kapabilitas (38 token).** Seluruh kosakata token terenkripsi untuk state halaman `a.ca` kini terbaca — format kurung siku: `[LTE_CA]`, `[LTE_CA_FILTER]`, `[LTE_UL_CA]`, `[LTE_DL_256Q]`, `[LTE_UL_256Q]`, `[LTE_DL_MIMO]`, `[LTE_HORXD]`, `[LTE_RX0]`–`[LTE_RX3]`, `[LTE_VOICE]`, `[NSA_LTE]`, `[NSA_LTE_DSS]`, `[NSA_NDDS]`, `[NSA_LTE_DSDA]`, `[NR_CA]`, `[NR_DL_256Q]`, `[NR_UL_256Q]`, `[NR_DL_MIMO]`, `[NR_UL_MIMO]`, `[NR_REDCAP]`, `[NR_VOICE]`, `[NR_NR_DSDA]`, dan lain-lain.

**Jalur diagnostik perangkat.** String-string node diag dan SoC: `/dev/diag`, `/d/diag/status`, `/d/diag/bridge`, `/d/diag/mhiinfo`, `/sys/class/diag/diag/dev`, `/sys/devices/soc0/{soc_id,raw_id,chip_name,machine,images}`, `/sys/kernel/debug/qcom_socinfo/mpss/name`, `/d/msm_ipc_router/dump_servers` — konsisten dengan halaman laporan modem/SoC dan pembacaan identitas chipset.

**Identitas perangkat dibaca dari system property (26 properti).** `ro.product.{model,device,board,marketname,...}`, `ro.baseband`, `ro.board.platform`, `ro.boot.cid/mid`, termasuk varian vendor (Sony `ro.semc.*`/`ro.somc.*`, ASUS, Oppo/OPlus) — dipakai merangkai label nama perangkat di laporan.

**Satu-satunya URL di seluruh aplikasi:** grup dukungan Telegram pengembang, `t.me/ru_fieldtest_modemcaps` (ditampilkan di teks About/bantuan). Tidak ada endpoint jaringan lain — menguatkan kesimpulan putaran pertama: nol telemetri, nol server.

**File kerja sementara:** `/data/tmp/modemcaps_*.bin` dan `/data/tmp/` — pola nama file pertukaran data modem di sisi Java, selaras dengan `/data/tmp/00028874` yang terlihat di native.

## 4. Yang berubah dari kesimpulan putaran pertama

1. Blob string: **60.635** karakter (bukan 60.750).
2. Shell `am.java:1642`: dari "konsisten dengan `su`, belum terkonfirmasi" menjadi **terkonfirmasi `su`**, dengan isi skrip stdin yang sudah terbaca penuh (pengumpulan log support ke `/sdcard/modemcaps_osinfo.log`, dengan penyaringan field identitas dari dump `getprop`).
3. Refleksi `a/a.java:246-247`: dari "target tidak diketahui, pola hidden-API" menjadi **dua method publik biasa** (`Activity.invalidateOptionsMenu`, `File.lastModified`) — kekhawatiran greylist dicabut.
4. Peran tool SELinux: dari simpulan string-table menjadi **terkonfirmasi payload `supolicy --live` di kode Java**, termasuk varian fallback izinkan-semua.
5. Batasan analisis nomor 1 di dokumentasi utama ("string belum didekripsi") **tertutup**.

Yang masih terbuka sama seperti sebelumnya: perilaku runtime belum diuji di perangkat, 19 class belum ter-decompile, dan urutan pemanggilan persis `supolicy` dari lapisan native belum ditelusuri.

---

## Addendum (putaran pendalaman kedua, 21 Oktober 2026)

Klaim cakupan di dokumen ini perlu diluruskan: angka 2.104 string / "100%" berlaku untuk call-site di 293 class yang ter-decompile JADX. Setelah 20 class sisanya berhasil didisassembly (lihat `ModemCaps-Pendalaman-2.md`), ditemukan **990 konstanta tambahan** — total kamus menjadi **3.103 konstanta unik / 3.092 string plaintext**, dan `ModemCaps-Kamus-String.tsv` sudah diperbarui ke angka itu. Ditemukan juga lapisan obfuscation kedua di `a.z` (balik-urutan + geser karakter berkunci digit runtime) yang melindungi nama file helper native (`/liba.so`, `/libb.so`).
