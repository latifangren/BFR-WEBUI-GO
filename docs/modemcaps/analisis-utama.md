# Dokumentasi Analisis — Qct Modem Capabilities

**Target:** `modemcaps-2402-1d_0_it0k.apk` · **Metode:** analisis statis (REA + JADX + Ghidra), tanpa menjalankan aplikasi · **Tanggal:** 21 Oktober 2026

---

## 1. Ringkasan eksekutif

Qct Modem Capabilities adalah **alat field-test untuk membaca dan mengubah kapabilitas modem Qualcomm** langsung di level file konfigurasi modem (EFS/NV), bukan aplikasi Android biasa. Ia menampilkan dan mengedit: band yang didukung per teknologi (GSM/UMTS/LTE/NR), kombinasi agregasi (CA combos), fitur LTE/NSA/NR (MIMO, 256-QAM, VoLTE/VoNR, DSS, RedCap), rantai RF (Rx chains), preferensi pencarian sel (blacklist/whitelist cell & channel), prioritas RAT, dan status IMS — lalu dapat me-restart subsistem modem (MPSS) agar perubahan berlaku.

Aplikasi ini **tidak meminta satu pun permission Android**. Seluruh kemampuan istimewanya datang dari lapisan native dan jalur shell: `liba.so` berbicara ke modem lewat soket dan file EFS, sebuah binary utilitas file dan sebuah tool kebijakan SELinux ikut dibundel dengan nama samaran `.so`. Kode Java-nya dilindungi obfuscation menyeluruh (semua class bernama 1–2 huruf) dan **enkripsi string penuh** (blob 60.635 karakter, 4.172 titik dekripsi di seluruh kode), sehingga label, path file, dan perintahnya tidak terbaca dari dekompilasi biasa. *Putaran lanjutan (dekripsi massal) sudah dikerjakan — hasilnya di `ModemCaps-Dekripsi-String.md`; seluruh 2.113 string berhasil dipulihkan.*

**Konsekuensi praktis:** aplikasi hanya berfungsi penuh di perangkat Qualcomm yang sudah dibuka jalurnya (root/engineering build). Di ponsel ritel biasa, fungsi intinya tidak akan berjalan — dan memang begitu desainnya.

---

## 2. Identitas dan komposisi paket

| Aspek | Nilai |
|---|---|
| Nama aplikasi | Qct Modem Capabilities |
| Package | `ru.fieldtest.modemcaps` |
| Versi | versionName `2402-1`, versionCode `2402001` |
| SDK | compileSdk 29 (Android 10), targetSdk 29, minSdk 16 (Android 4.1) |
| Ukuran / SHA-256 | 529 KB · `cf20ddb3587139b54d009f2b0b9de040ff7ecb83db8ba596c9781607cb596f7a` |
| Permission | **Tidak ada sama sekali** |
| Komponen | 6 activity; tanpa service, receiver, atau provider |
| Tema UI | `Theme.DeviceDefault` (bawaan sistem, tanpa Material/AppCompat) |
| Isi APK | 1 `classes.dex` (928 KB, 312 class), 6 file `.so` (3 nama × 2 ABI), `resources.arsc` + resource bernama acak (obfuscation gaya AndResGuard) |
| Library pihak ketiga | Hanya kotlinx.coroutines dan AndroidX ViewPager — tanpa SDK iklan, analytics, atau crash reporting |

**Enam activity** (semua di package obfuscated `a`):

| Class | Peran |
|---|---|
| `a.am` | Activity utama/launcher (24.144 baris) — pemegang seluruh state modem dan satu-satunya yang menyentuh native |
| `a.em` | Editor checklist band per RAT |
| `a.ca` | Halaman switch kapabilitas LTE/NSA/NR (asal nama aplikasinya) |
| `a.sc` | Editor blacklist/whitelist cell & channel (Searcher Preferences) |
| `a.pc` | Picker prioritas RAT |
| `a.ba` | Penampil laporan teks read-only + pencarian + share |

---

## 3. Arsitektur

```
Launcher ──► a.am  (state global modem di field statis, per slot/SIM [5])
               │ 17 method native (JNI) ──► liba.so ──► soket + file EFS modem
               ├─ coroutine a.l  → System.loadLibrary("a")
               ├─ coroutine a.y  → startup: baca modem, isi state, render laporan
               ├─ serializer a.d → Intent extras untuk membuka editor
               └─ menu ─┬─ a.em  checklist band/RAT
                        ├─ a.ca  switch kapabilitas
                        ├─ a.sc  editor searcher (cell/channel)
                        ├─ a.pc  prioritas RAT
                        └─ a.ba  penampil laporan (RFFE, EFS Misc, ICC, LTE CA)

Binary pendamping (di dalam APK, menyamar sebagai .so):
  libb.so  = utilitas file gaya busybox/toybox (stat/ls/chmod/chown)
  libc.so  = tool kebijakan SELinux (bukan libc sistem)
```

Tiga prinsip desain yang konsisten di seluruh kode:

1. **State global di statics `am`.** Semua data kapabilitas (matriks `long[][]`, `byte[][]`, `String[]`, bitmask master) adalah field statis activity utama. Kelas lain hanya membaca/menulis state itu lewat accessor di fasad `a.m`.
2. **Editor tidak pernah menyentuh modem.** Editor menerima *salinan* state lewat Intent, mengeditnya, dan mengembalikannya saat tombol Back ditekan. Penulisan ke modem selalu dilakukan `am` setelah hasil kembali.
3. **UI utama programatik.** Activity utama tidak memakai layout XML — hanya dua `TextView` (header + badan laporan monospace yang bisa diseleksi) yang dibuat dengan kode.

---

## 4. Alur kerja ujung-ke-ujung

1. **Aplikasi dibuka** → `am.onCreate` menaikkan prioritas thread utama, lalu memulihkan state dari Bundle (rotasi/proses) atau SharedPreferences `"g"`.
2. **Library dimuat** → coroutine `a.l` menjalankan satu-satunya jalur load di seluruh aplikasi: `System.loadLibrary("a")`, tanpa fallback. Gagal muat = pesan error ditampilkan sebagai teks laporan.
3. **Sesi modem dibuka** → fasad `a.m` memanggil native `am.t(1, 5, 0, 400)` (init sesi), lalu membaca identitas dan kapabilitas modem lewat:
   - `o(slot, byte[], …)` — workhorse pengirim *frame perintah* modem (frame kecil berprefix byte `75`/`41` terlihat di kode), dengan `no()` sebagai getter kode error terakhir;
   - `m(slot, …)` — mengambil direct `ByteBuffer` berisi data mentah modem;
   - `r(nama, byte[])` — membaca **file by-name** (jalur baca file EFS);
   - `p(id)` / `u()` — membaca string per-id / identitas perangkat global.
4. **Data diterjemahkan** → ±11 dekoder di `a.m` mengubah ByteBuffer mentah menjadi tabel band, token fitur, dan teks laporan; tabel pemetaan nomor band 1–256 (LTE/NR) ke label juga ada di kelas ini.
5. **Laporan dirender** oleh routine raksasa `m45a()` (±3.200 baris) ke TextView badan, dengan Toast untuk hasil/error operasi (66 titik Toast).
6. **Pengguna memilih menu** → dispatcher `onOptionsItemSelected` (±8.400 baris, sebagian dispatch-nya *posisional* per slot, bukan per id menu) membuka editor atau laporan yang sesuai. Satu jebakan besar: id item digeser **+1** sebelum dibandingkan (`am.java:15964`), jadi label menu ≠ aksi — mis. label "Data Preferences" membuka editor kapabilitas, label "RAT Priority" membuka Debug Info (tabel terkoreksi: laporan alur §2.5).
7. **Editor bekerja** di atas salinan state (lihat Bagian 5), lalu saat Back: hasil ditulis kembali ke Intent yang sama dan dikembalikan dengan result code khas (0 = batal, 21 = reset dari `em`, 12 = reset dari `sc`).
8. **Perubahan diterapkan** → `onActivityResult` menyerahkan hasil ke coroutine `a.av`, yang memperbarui statics dan menulisnya ke modem lewat native. Untuk jenis perubahan tertentu (preferensi searcher), modem di-*power cycle*/restart MPSS — persis seperti peringatan di UI (±1 menit, kadang memicu reboot ponsel, aplikasi terputus sampai restart).
9. **Laporan dapat dibagikan** — disalin ke clipboard dari dialog hasil, atau dibagikan lewat share sheet Android dari penampil `ba`. Ini satu-satunya jalur data keluar perangkat, dan hanya atas tindakan eksplisit pengguna.

Penyimpanan persisten di sisi Java hanya SharedPreferences `"g"` (kunci `x` terobfuscate XOR bergulir; `t`, `y`, `z` ditulis saat `onSaveInstanceState`). Tidak ada database.

---

## 5. Cara kerja tiap proses / fitur

### 5.1 Activity utama `a.am` — pusat kendali

- **Peran:** satu-satunya pintu ke modem; memegang state per slot (array berukuran [5]); merender laporan kapabilitas; menampung dispatcher menu 24 item.
- **Threading:** tanpa Executor/Handler — Kotlin coroutines (12 titik peluncuran) + 1 thread mentah startup + 1 `Semaphore` (1 permit) untuk serialisasi akses modem. `onDestroy` menunggu semaphore sampai 5 detik sebelum cleanup buffer sesi native.
- **Menu dan jebakan penamaannya** (terverifikasi dari kode, bukan dari judul menu):
  - dispatcher menggeser id item **+1** sebelum membandingkan (`am.java:15964`) — obfuskasi disengaja; pemetaan label di bawah ini adalah hasil koreksi 2026-10-21 (rincian + bukti: laporan alur §2.5);
  - label *"Data Preferences"* (`menu_dfp_config`) sebenarnya **membuka halaman switch kapabilitas `ca`** — judul window `ca` memang "Data Preferences"; cabang kodenya bernama `menu_efs_clear`/"Clear RMTEFS";
  - label *"Clear RMTEFS"* sendiri tidak menghapus apa-apa (jatuh ke alur generik → laporan Debug Info); label *"RFC Report"* juga tanpa implementasi;
  - label *"RAT Priority"* membuka laporan Debug Info; picker prioritas RAT `pc` dijangkau lewat label *"PDC Preferences"*; label *"RAT/Band Preferences"* membuka editor checklist band `em`;
  - `menu_csp_config` berjudul *"Searcher Preferences"* membuka editor `sc` (cabang yang dipakai bernama `menu_debug`);
  - grup *Test Commands* selalu dipaksa tidak terlihat; command 1–5 semuanya no-op;
  - item *About* membuka dialog char-decode (`f809a=-2`) — dialog yang semula dikira milik "Deny ⚡ connect"; kedua opsi checkable ("Deny ⚡ connect", "Shitty kernel") ternyata cuma toggle flag statis, tanpa dialog.
- **Dialog hasil:** tombol dialog dapat (0) menyalin laporan ke clipboard, (1) menyalin teks debug ke clipboard, (2) **menjalankan jalur shell** (Bagian 8).

### 5.2 Editor band `a.em` — checklist band per RAT

- Expandable list kustom (`a.el`) berisi grup teknologi (kunci RAT 3, 4, 5, 7, 19, 20, 21, 22) dengan anak-anak band; data masuk berupa mask `long[]` band tersedia dan matriks pilihan per grup.
- Tap di sepertiga kiri item = expand/collapse; tap lain = toggle centang. **Long-press** = centang massal satu grup.
- Saat pertama dibuka, ada tutorial terjadwal: pager bergeser otomatis (500/1500/2500 ms) dan Toast petunjuk muncul berurutan.
- Tombol menu delete menghasilkan result **21** (reset pilihan).

### 5.3 Halaman kapabilitas `a.ca` — switch LTE / NSA / NR

±20 switch dalam 3 seksi (label persis dari layout):

- **LTE:** LTE+LTE aggregation, Uplink 256-QAM, Uplink aggregation, Downlink 256-QAM/1K-QAM, DL MIMO, **Band combos filter** (+ kolom teks daftar combo), Voice over LTE, Rx chain #0–#3, pemetaan diversity **D#0 (PRx), D#1–D#3 (DRx), D#6 (ARD)**, Adaptive receive diversity.
- **NSA:** LTE+NSA aggregation, LTE+NSA untuk DSDA, LTE+NSA during data SIM call, Dynamic spectrum sharing.
- **NR:** NR+NR aggregation, NR+NR dual connectivity, NR DSDA, UL/DL 256-QAM, UL/DL MIMO, Voice over NR, Reduced Capability (NR-Light), Rx chain, ARD.

Mekanisme uniknya: **seluruh state satu halaman dikodekan sebagai satu string token**; status tiap switch ditentukan dengan uji substring token terenkripsi, dan saat keluar string token dibangun ulang dari switch yang tercentang (urutan LTE → NSA → NR → Rx/ARD). Teks filter combo dinormalisasi (pisah `;`, uppercase, suffix default) mengikuti format resmi aplikasi:

```
Band+Class+[Streams][Uplink Class]-...[Bandwidth Class Set]
Class: A|C|D|E|F · Streams: 2|4 · BW Class Set: 0|1|2|4|m3|m7|mAll
```

Footer halaman ini — ditulis pengembangnya sendiri di XML — menjelaskan kontrol bisa abu-abu karena: tidak ada dukungan hardware, butuh *supplemental connectivity*, mode pesawat, atau fitur generik (tidak tergantung SIM); dan bahwa sebagian pengaturan bersifat **write-only** (tidak bisa dibaca balik): LTE+LTE aggregation, UL aggregation, DL 256-QAM, Rx chains, ARD.

### 5.4 Editor searcher `a.sc` — blacklist/whitelist cell & channel

- 8 kolom teks dalam 4 seksi (GSM, UMTS, LTE, NR), masing-masing maksimal 6 entri (UMTS hanya 1): cell blacklist, channel blacklist, channel code whitelist, priority channel list.
- Nilai disimpan terpak rapi dalam `long[30]` per halaman: GSM cell = `(field1<<48)|(field2<<40)|(field3<<16)|field4`; LTE cell = `(PCI<<32)|CID`; NR menyertakan step SCS (`15 << nibble`). `-1` adalah terminator.
- Legenda format resmi: Channel = ARFCN | UARFCN | EARFCN | NR-ARFCN; Code = BSIC | PSC | PCI; Step = SCS; Area = LAC | TAC; Cell = CID.
- Peringatan statis di halaman: mengisi prioritas UMTS mengubah preferensi RAT menjadi UMTS-only; perubahan memicu modem power cycle / restart OS modem.
- Menu delete menghasilkan result **12** (reset).

### 5.5 Picker prioritas RAT `a.pc`

- Dua daftar bertumpuk (programatik, tanpa XML): kandidat di atas, pilihan tunggal di bawah. Kandidat diberi penanda status ⭐/⏳ berdasarkan perbandingan bit dengan pilihan saat ini dan dua set referensi.
- Mengembalikan pilihan per halaman; return 3 bila semua halaman terisi, −1 bila ada yang kosong.

### 5.6 Penampil laporan `a.ba`

- Menampilkan laporan besar (RFFE Components, EFS Misc. Files, ICC Misc. Files, LTE CA Combos — label→isi bergeser +1, lihat laporan alur §2.5) sebagai teks monospace yang bisa diseleksi.
- String ≥ 16.384 karakter tidak dikirim lewat Intent/Binder, melainkan disimpan di registry statis `am` dan hanya hash-nya yang dikirim; penerima mengambil **dan menghapusnya** (one-shot).
- Dilengkapi pencarian teks (lompat ke hasil berikutnya, plat pencarian memerah bila tidak ketemu) dan tombol share — **satu-satunya implicit intent di seluruh aplikasi**.

### 5.7 Kelas pendukung penting

| Class | Isi |
|---|---|
| `a.m` (6.748 baris) | Fasad native + ±103 accessor state + 11 dekoder ByteBuffer→teks + tabel band 1–256 + identifikasi perangkat (±1.600 baris kaskade perbandingan string) + probe file identitas |
| `a.d` (2.147 baris) | Serializer Intent: 52 `putExtra` yang mengemas state `am` untuk keempat editor/penampil |
| `a.bd` | Kelas dasar editor: pager per slot/SIM, kontrak hasil saat Back, menu batal bersama |
| `a.a` (377 baris) | Dekriptor string + pembangun tema + pewarna glif status laporan |
| `a.c` (182 KB, 12 baris) | Blob string terenkripsi: 8 string, total 60.635 char |
| `a.l` (67 baris) | Lambda coroutine pemuat `liba.so` |

---

## 6. Lapisan native

### 6.1 `liba.so` — jembatan JNI ke modem (36 KB arm64 / 15,6 KB arm32)

- Stripped; hanya mengekspor `JNI_OnLoad`, yang membaca `ro.build.version.sdk` lalu me-register **17 method native** milik `a.am` dengan salah satu dari dua tabel signature (SDK > 25 vs ≤ 25; tabel lama pointer string-nya bergeser 1 byte).
- Kemampuan dari import-nya: soket lengkap (`socket/bind/connect/send/recv/recvfrom/sendto/setsockopt/ioctl`), file (`open/read/write/mmap`), thread (`pthread_create`), kompresi zlib (`compress2/uncompress` — untuk blob terkompresi), `__system_property_get`.
- String plaintext di `.rodata` membocorkan sasaran kerjanya — path di dalam **EFS modem Qualcomm**:

```
/nv/item_files/modem/lte/     rrc/efs/     rrc/cap/
../nr5g/RRC/                  common/      /data/tmp/     /data/tmp/00028874
```

Artinya unit kerja liba adalah **file-file kapabilitas RRC LTE dan NR5G** (band, CA combo, fitur) di filesystem internal modem, dengan `/data/tmp` sebagai area staging di sisi Android.

### 6.2 `libb.so` — utilitas file yang menyamar (29.988 byte)

File yang sama persis (md5 identik) di kedua folder ABI, dan selalu ELF 32-bit. Tidak mengekspor fungsi; string format di dalamnya milik utilitas `stat`/`ls` GNU/toybox (`"ID: %-8i Namelen: %-7l Type: %T"`, `"%n %i %l %t %s %b %f…"`, `"'%s' -> '%s'"`), dan ia mengimpor `chmod`, `chown`, `getpwuid`, `opendir`. Kesimpulan: **binary utilitas file bergaya busybox/toybox yang dinamai `.so`** — pola binary PIE yang dijalankan lewat `/system/bin/linker` sebagai proses pembantu, bukan library untuk di-`dlopen`.

### 6.3 `libc.so` bundel — tool SELinux yang menyamar (32.952 byte)

Juga identik di kedua ABI dan 32-bit. Isinya bukan libc: ada string `/sys/fs/selinux/policy`, `/sys/fs/selinux/load`, `SE Linux Module`, `typeattribute`, `attradd`, `object_r` — ciri **tool kompilasi/pemuatan kebijakan SELinux** (keluarga sepolicy-inject/libsepol). Fungsinya dalam ekosistem aplikasi ini: membuka akses SELinux agar proses dapat menjangkau soket/file milik modem di perangkat yang enforcing. Nama `libc.so` adalah kamuflase (dan tabrakan nama dengan libc sistem diselesaikan oleh jalur pemuatannya sendiri).

> Anomali yang disengaja: kedua file 32-bit itu ikut berada di folder `arm64-v8a`, jadi di perangkat 64-bit murni keduanya tidak bisa di-load sebagai library 64-bit — konsisten dengan kesimpulan keduanya dijalankan sebagai executable 32-bit, bukan di-load.

---

## 7. Mekanisme enkripsi string — dan hasil dekripsinya

Dekriptor `a.a(long, String[])` murni Java, tanpa `Cipher` dan tanpa bantuan native:

1. **Seed** = 32 bit rendah konstanta `long` di setiap call site; **offset blob** dikodekan di 32 bit tingginya — satu konstanta membawa alamat + kunci sekaligus.
2. Seed di-hash gaya **SplitMix64** (dua perkalian konstanta gamma SplitMix + xor-shift).
3. State digerakkan **mixer 16-bit** khusus (pecah short, jumlah + XOR, rotasi 10/9/13 bit) yang dipakai berulang sebagai PRNG.
4. Karakter pertama dibaca dari blob datar di `blob[offset/8191].charAt(offset%8191)` (blob = `c.f763a`: 7 potongan 8.191 char + 1 potongan 3.298 char, total **60.635 char** berisi glyph 16-bit padat — diverifikasi dari `classes.dex`, bukan dari rendering JADX).
5. Panjang string diturunkan dari state (`(state>>>32) & 0xFFFF`), bukan disimpan.
6. Loop **autokey berantai**: tiap karakter keluaran = 16 bit atas state setelah state dimajukan mixer dan di-XOR dengan karakter blob berikutnya.

Material kuncinya hanya dua: konstanta per call site dan blob itu sendiri; tidak ada field kunci terpisah.

**Hasil dekripsi massal (putaran lanjutan):** dekriptor diimplementasikan ulang persis dan dijalankan ke seluruh 4.172 referensi call-site (2.113 konstanta unik) — **2.104 string pulih terbaca, cakupan 100%** atas string asli (9 konstanta sisa adalah false positive numerik). Temuan lengkapnya di `ModemCaps-Dekripsi-String.md`, kamus lengkapnya di `ModemCaps-Kamus-String.tsv`. Tiga hal yang sebelumnya tertutup kini terkonfirmasi: proses shell-nya persis `su`; payload `supolicy --live` tool SELinux ada di kode Java (termasuk varian fallback izinkan-semua); dan 21 path file EFS konkret di lapisan Java terbaca nama persisnya.

---

## 8. Keamanan dan privasi

- **Nol permission, kemampuan setara root.** Semua akses istimewa berjalan di luar sistem permission Android (native + shell). Di perangkat tanpa jalur privilege, fungsi inti mati — aplikasi memang ditujukan untuk perangkat field-test.
- **Jalur shell di Java ternyata tiga** (dua ditemukan di putaran pendalaman kedua): (1) handler tombol dialog di `am.java:1642` menjalankan `ProcessBuilder("su")` — **terkonfirmasi hasil dekripsi** — dan menulis skrip pengumpulan log dukungan ke stdin-nya (inventaris `/dev` + sysfs SoC, dump `getprop` yang disaring membuang baris `imei`/`imsi`/`esn`/`serialno`, keluaran ke `/sdcard/modemcaps_osinfo.log`); (2) coroutine `a.z` memelihara proses `su` **persisten** untuk tangga pelonggaran SELinux (`setenforce 0` → `supolicy` terarah → fallback allow-all via `libc.so` bundel) dan untuk transport frame biner ke helper native; (3) coroutine `a.ao` membangun `ProcessBuilder` sendiri saat serialisasi state. Semuanya terpicu alur internal aplikasi, bukan input eksternal. Rincian di `ModemCaps-Pendalaman-2.md` Bagian 4.
- **Refleksi — ternyata bukan hidden-API:** `a/a.java:246-249` menyiapkan dua `Class.forName(...).getDeclaredMethod(...)` yang setelah didekripsi terbaca sebagai method publik biasa: `android.app.Activity.invalidateOptionsMenu()` (di-cache di `am.b`) dan `java.io.File.lastModified()` (di-cache di `am.f273a`). Namanya terenkripsi hanya karena semua string dienkripsi. Kekhawatiran greylist hidden-API dari putaran pertama dengan ini gugur.
- **Jaringan: tidak ada.** Grep menyeluruh 312 class untuk `java.net`/Socket/URL: 0 hit, dan tidak ada permission INTERNET. Satu-satunya data keluar adalah share sheet dan clipboard atas perintah pengguna — isinya laporan kapabilitas modem perangkat itu sendiri.
- **Komponen terekspos:** hanya activity launcher yang `exported=true`. Konten besar antar-activity lewat registry hash one-shot yang langsung dihapus setelah diambil.
- **Tidak ditemukan** telemetri, analytics, iklan, atau crash reporting.

---

## 9. Batasan analisis (yang belum terverifikasi)

1. ~~**String belum didekripsi.**~~ **TERTUTUP di putaran lanjutan** — seluruh string berhasil dipulihkan (lihat Bagian 7 dan `ModemCaps-Dekripsi-String.md`). Yang masih belum diketahui dari lapisan string: urutan persis pemanggilan `supolicy` dari lapisan native.
2. ~~**19 dari 312 class tidak ter-decompile**~~ **TERTUTUP di putaran pendalaman kedua** — ke-20 class riil yang hilang (dari total 355 class defs di DEX) berhasil didisassembly smali penuh via androguard, termasuk dua method raksasa di `a.z` (59.472 instruksi) dan `a.an` (28.240 instruksi) yang membuat JADX macet. `a.z` teridentifikasi sebagai hub operasi shell/sistem dan satu-satunya pemanggil template `supolicy`; `a.an` sebagai coroutine baca/tulis state kapabilitas. Rinciannya di `ModemCaps-Pendalaman-2.md`.
3. **Perilaku runtime tidak diuji** — tidak ada perangkat/emulator yang menjalankan APK ini; urutan persis frame perintah modem (`o()`) dan peran tiap method native disimpulkan dari pola pemakaian dan pseudocode Ghidra, bukan dari observasi langsung.
4. **Peran `libb.so`/`libc.so`** — naik bukti di putaran kedua tanpa menjalankan prosesnya: pseudocode Ghidra `libb.so` memuat implementasi listing direktori gaya `ls` yang utuh, dan `libc.so` terbukti menyimpan tabel pointer ke `/sys/fs/selinux/policy`. Temuan native terbesar putaran kedua: **`liba.so` juga executable mandiri** — entry point-nya menjalankan worker lookup layanan QRTR/QMI berprotokol pipa stdin/stdout untuk proses induk (dirinci di `ModemCaps-Pendalaman-2.md`). Alur `main` persis `libc.so` masih belum tertelusur (dekompilasi ARM32-nya buruk), dan pemetaan 17 method JNI ↔ alamat fungsi masih sebagian.
5. Tanda tangan APK tidak diverifikasi terhadap sertifikat penerbit mana pun (`signature_verification: not_performed` dari REA).
6. ~~**Peta dispatch menu salah satu langkah.**~~ **TERTUTUP 2026-10-21** — putaran koreksi menemukan `onOptionsItemSelected` menggeser id item +1 (`am.java:15964`); tabel label→aksi di laporan alur §2.5 sudah diganti versi terkoreksi (diverifikasi silang menu XML, `public.xml`, kode cabang, dan kamus terdekripsi). Efek sampingnya: "Data Preferences" terpecahkan (= editor kapabilitas `ca`, padanan Fitur A di spek) dan sumber laporan HW terpetakan (RFFE dari `/rfc/rffe_scan/rffe_scan.dat`; ICC Misc parser on-demand; RFC Report tidak ada). Yang tetap terbuka dari koreksi ini: baris pembangun spesifik string statis laporan `c`/`d`/`e` belum dipetakan, dan referensi `/rfc/rfdevice_report.dat` belum ketemu di kode ter-decompile.

---

## 10. Bukti dan rujukan

- REA `inspect-android-package` (JADX headless 0.7.1): Evidence `ev_47695c98989d95c5693f54f0f3cfb11581b4d19105b7d353f495d58cd67dbc53`
- REA `analyze` liba.so via Ghidra 12.1.4: Evidence `ev_b2be1d269e273e8ba8ddb918a92f67f37cc8125b505bcaff7fc85ba529473bef`
- REA `function` JNI_OnLoad: Evidence `ev_d59cad882da83ed25038906b7acc5bfc2de564df1fd76504f3afe684973d1e1b`
- Artefak kerja: `~/workspace/modemcaps-analysis/` — `jadx-src/` (hasil dekompilasi), `java-flow-report.md` (laporan alur Java rinci dengan sitasi file:line), `rea-*.json` (Evidence mentah), `native/` (library hasil ekstrak)
- Sitasi file:line lengkap per klaim ada di `java-flow-report.md` Bagian 1–5
