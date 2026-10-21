# Pendalaman Putaran Kedua — Class Hilang, Lapisan String Baru, dan Native

**Target:** `modemcaps-2402-1d_0_it0k.apk` (`ru.fieldtest.modemcaps` v2402-1) · **Tanggal:** 21 Oktober 2026 · **Melengkapi:** `ModemCaps-Dokumentasi-Analisis.md` dan `ModemCaps-Dekripsi-String.md`

Putaran ini menutup tiga celah yang diakui terbuka di dokumentasi awal: 19–20 class yang gagal didecompile, bedah native per-fungsi, dan jalur pemanggilan `supolicy`. Satu klaim putaran sebelumnya juga **dikoreksi** di sini (lihat Bagian 2).

---

## 1. Dua puluh class yang hilang — berhasil dipulihkan

Jalur JADX per-class ternyata tidak menolong: mode `--single-class` tetap memuat seluruh DEX dan tersangkut di fase loading yang sama. Yang berhasil adalah **disassembly smali via androguard** — linear, tanpa analisis graf — selesai untuk semua class dalam hitungan detik, termasuk dua method raksasa yang membuat JADX menyerah.

Koreksi hitungan: DEX sebenarnya memuat **355 class defs**; angka "312" dari inspeksi paket adalah hitungan class non-sintetis versi tool tersebut. Dari 355 itu, 293 ter-decompile JADX, 20 class riil hilang (sisanya lambda sintetis yang sepele). Ke-20 class itu sekarang terdisassembly penuh di `~/workspace/modemcaps-analysis/disasm/`.

| Class | Peran (terverifikasi dari invoke + string terdekripsi) |
|---|---|
| `a.y` | Pembungkus tipis: hanya membuat `a.z` dengan 11 nilai tangkapan yang sama (am, SharedPreferences, holder proses/state) lalu menjalankannya |
| `a.z` | **Hub operasi sistem/shell + coroutine inisialisasi utama.** 29 method, satu method berisi 59.472 instruksi. Menjalankan proses (`su` persisten, `sh`, helper native), membaca `SystemProperties`, memeriksa file, memanggil init sesi native `am.t`, frame `am.o` (×7), getter error `am.no` (×17) — dan menjadi satu-satunya pemanggil template `supolicy` |
| `a.aa`–`a.ah` | Lambda coroutine *di dalam* `z`: pembangun baris laporan per-item dari hasil fasad native + satu string terdekripsi (`'16'`, `'19'`, `'20'`, `' #'`); `aa` mengatur prioritas thread lebih dulu, `ad` adalah langkah yang menunggu (`SystemClock.sleep`) dan menghitung bit (`Integer.bitCount`, token `[LTE_CA]`) |
| `a.ai` | Pemformat laporan kartu/ICC: string `[ST#`, `[EF.IMSI_M]`, `[EF.KEYS]`, konstruksi record `a.ck` berulang, dekoder byte `a.f` |
| `a.aj` | Worker induk: mem-parse `ByteBuffer` (`getLong`/`getShort`), memformat tanggal, memanggil method reflektif ter-cache, dan **menginstansiasi `a.ak`, `a.al`, dan `a.an`** |
| `a.ak` | Pembangun string token kapabilitas: set token default lengkap LTE+NR (`[LTE_CA_FILTER]`, `[LTE_DL_MIMO][LTE_ARD_4RX]`, `[NR_UL_256Q][NR_UL_MIMO][NR_DL_MIMO]`, …), label `mcc `/`mnc ` |
| `a.al` | Pemformat satu seksi laporan (helper string `a.gp`, `String.format`, lookup `SparseIntArray`) |
| `a.an` | **Coroutine baca/tulis state kapabilitas.** Method 28.577 instruksi; merujuk seluruh 21 path EFS dan token fitur — inilah mesin muat/simpan konfigurasi modemnya |
| `a.ao` | Serialisasi state: 49× `ByteBuffer.putLong`, lalu simpan ke SharedPreferences (21× `putInt`, 4× `putLong`). **Penting: `ao` juga membangun `ProcessBuilder` sendiri** — pemakaian shell tidak eksklusif di `z`/`am` |
| `a.do`, `a.jw`, `a.kl`, `a.jz`, `a.if` | Infrastruktur library coroutines, bukan logika aplikasi: `do` interface CoroutineContext, `jw` holder `Object[]`, `kl` pola TimeSource abstrak, `jz` enum switch-map sintetis `{1,2,3,4,5}`, `if` pembungkus lambda sebagai completion handler |

## 2. Koreksi klaim dekripsi: cakupannya sekarang benar-benar penuh

Laporan dekripsi putaran pertama menyatakan cakupan 100% — itu benar **untuk call-site di 293 class yang ter-decompile**, tetapi 20 class yang hilang ternyata menyimpan konstanta mereka sendiri. Dari disassembly ke-20 class itu dipanen **990 konstanta tambahan**, 988 di antaranya string terbaca (2 false positive numerik).

**Total kamus sekarang: 3.103 konstanta unik → 3.092 string plaintext** (`ModemCaps-Kamus-String.tsv` sudah diperbarui). Isi baru yang paling menonjol: kosakata laporan DIAG/FTM yang lengkap (mode/subsystem/command, panjang request/respons, pesan error `FTM Error :`, `UNIQ Error :`, `NOT IMPLEMENTED`), yang menunjukkan halaman laporan RFC/ICC aplikasi ini merender hasil perintah FTM mentah per subsistem.

## 3. Lapisan obfuscation kedua di `a.z`

Sebagian kecil string paling sensitif di `z` dilindungi lapisan kedua di atas enkripsi blob: hasil dekripsi blob masih berupa teks acak yang **dibalik urutan karakternya lalu digeser per karakter** dengan kunci satu digit yang diturunkan saat runtime (digit dari teks state di UI, atau nilai `am.a()`). Contoh yang terverifikasi dengan kunci 3:

- `fh>rv1delo2` → **`/liba.so`**
- `#rv1eelo2` → **`/libb.so`**
- `#orrw` → **`tool`**

Yang dilindungi lapisan ini persis hal yang paling ingin disembunyikan penulisnya: nama file helper native-nya sendiri.

## 4. Arsitektur proses — lingkaran tertutup

Potongan-potongan dari semua putaran sekarang menyatu menjadi satu rantai yang terverifikasi di setiap sambungannya:

1. `am.onCreate` → coroutine `a.y` → coroutine `a.z`.
2. `z` membuka `ProcessBuilder("su")` (tiga lokasi terkonfirmasi di disassembly), memperoleh ketiga stream prosesnya.
3. `z` merangkai path `<nativeLibraryDir>/liba.so` (nama file dibuka dari lapisan obfuscation kedua) dan menjalankannya sebagai program di dalam shell root itu.
4. Program `liba.so` (entry point-nya sendiri, lihat Bagian 5) berbicara protokol biner lewat stdin/stdout dengan proses induk — persis stream yang dipegang `z`. Helper kedua `z` menulis path itu ke stdin `su`, diikuti **frame biner mentah** dari ByteBuffer, lalu menunggu balasan biner dengan polling `InputStream.available()` + `sleep` — transport frame lewat shell root ke binary pembantu.
5. **Tangga pelonggaran SELinux — terpecahkan penuh.** Template `supolicy` dari `a.a()`/`a.b()` hanya dipanggil dari `z`, dan placeholder-nya kini diketahui persis: `{r}` diganti kata `allow`; `{d}` diganti **domain SELinux aplikasi sendiri**, dibaca dari `/proc/self/attr/current` lewat fasad native (bawaan bila gagal baca: `untrusted_app`, awalan `u:r:` dibuang). Tiga perintah dicoba berjenjang ke stdin proses `su` yang persisten, masing-masing dibaca balasannya dan diuji substring-nya (`'eni'` dari *denied*, `'siv'` dari *Permissive*):
   1. `setenforce 0 && echo Permissive` — matikan enforcing global lebih dulu;
   2. template terarah `a.a()` (aturan per-tipe untuk domain aplikasi) + ekor `&& echo Submissive`;
   3. fallback allow-all `a.b()` — dengan token `supolicy` **diganti path `<nativeLibraryDir>/libc.so`**: binary bundel itulah yang menjalankan aturan pada jenjang ini, bukan `supolicy` sistem.
   
   Jadi peran `libc.so` kini pasti dari dua arah: ia adalah **binary `supolicy` kompatibel milik aplikasi sendiri**, dipakai saat jalur sistem tidak tersedia/gagal.

## 5. Native per-fungsi (Ghidra headless, dekompilasi penuh: liba 119 fungsi, libb 208, libc 188)

**`liba.so` punya dua peran.** Selain library JNI (17 method `a.am`, dua tabel RegisterNatives menurut SDK), ia adalah **executable mandiri** dengan entry point sendiri (`entry` @00103830) yang menjalankan protokol pipa biner:

1. Baca 12 byte dari stdin = parameter `socket(domain, type, protocol)`; buat soket; tulis nomor fd ke stdout.
2. Baca flag; lakukan `ioctl(fd, 0x8010C305, …)` dengan struct 84 byte berisi nama bawaan — dengan tangga percobaan ulang (nilai 1, lalu 8 nilai dari tabel, lalu pasangan tetap) khas negosiasi/registrasi layanan.
3. Loop permintaan: jalur `ioctl(fd, 0xC014C302, …)` untuk info layanan, dan jalur lookup nama — kirim pesan kontrol tipe 10, kumpulkan jawaban tipe 4 ke tabel, tutup dengan tipe 11 — lalu tulis hasilnya (node/port/versi/instansi per layanan) ke stdout.

Ini **worker lookup layanan QRTR/QMI**: memetakan nama layanan QMI ke endpoint-nya untuk proses induk — selaras string Java `QmiInit(data=…)` dan `/d/msm_ipc_router/dump_servers`.

Di sisi JNI, fungsi terbesar `FUN_00104f00` (6.168 byte) adalah handler sesi frame diag: tulis frame ke fd, baca balasan dengan helper timeout, parse field di offset tetap; jalur file lokalnya memakai `compress2` level 9 saat menulis dan `uncompress` saat membaca — file pertukaran `/data/tmp/*.bin` disimpan terkompresi zlib.

**`libb.so` — peran terbukti di level kode** (naik kelas dari simpulan string): `FUN_0001269c` adalah implementasi listing direktori gaya `ls` (loop `opendir`/`readdir`, pencetakan format per entri, penanganan error per path) — konsisten dengan utilitas file multi-applet pemilik import `chmod`/`chown`/`stat`.

**`libc.so` — identitas terkonfirmasi dari sisi pemanggil:** tabel data di ±0x8ebc berisi pointer ke string `/sys/fs/selinux/policy`, dan dari sisi Java (Bagian 4) kini terbukti binary ini dipanggil dengan sintaks aturan `supolicy --live` yang sama persis, sebagai jenjang fallback tangga SELinux. Dekompilasi ARM32 Ghidra untuk binary ini buruk (literal pool tidak ter-resolve) sehingga alur `main` internalnya tetap belum tertelusur — tetapi peran dan cara pemanggilannya tidak lagi diragukan.

## 6. Penilaian kelengkapan sesudah putaran ini

- **String: tuntas.** 3.092 plaintext, termasuk yang dilindungi dua lapisan.
- **Java: tuntas secara struktur.** Seluruh 355 class defs terpetakan perannya; 293 ter-decompile Java penuh, 20 terdisassembly smali penuh (dua method raksasa `z`/`an` terbaca di level instruksi, bukan parafrase baris-per-baris).
- **Native: terpetakan secara struktur dan protokol.** Peran ganda `liba.so`, protokol pipanya, dan handler frame diag-nya terbaca; pemetaan persis 17 method JNI ↔ alamat fungsi masih sebagian (nama method 1 huruf + tabel terpotong menyulitkan korelasi nama).
- **Yang tetap belum bisa dari analisis statis:** perilaku runtime di perangkat sungguhan — urutan frame terhadap modem asli dan efek tulis EFS-nya hanya bisa dibuktikan dengan menjalankan APK-nya di perangkat Qualcomm root.
