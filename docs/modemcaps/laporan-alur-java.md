# Laporan Alur Java — Qct Modem Capabilities

**Paket:** `ru.fieldtest.modemcaps` · **Versi:** 2402-1 (versionCode 2402001) · **minSdk 16 / targetSdk 29**
**SHA-256 APK:** `cf20ddb3587139b54d009f2b0b9de040ff7ecb83db8ba596c9781607cb596f7a`
**Sumber bukti:** dekompilasi JADX di `jadx-src/sources/a/` (312 kelas, semua di package obfuscated `a`), resource di `jadx-src/resources/`, inspeksi paket REA/JADX di `rea-android.json`.
**Izin di manifest: tidak ada sama sekali** (`permissions: []` di `rea-android.json`). Semua kemampuan istimewa datang dari jalur native/root, bukan dari permission Android.

> Catatan metode: hampir semua string di kode terenkripsi dan didekripsi saat runtime lewat `a.a(long, c.f763a)` (543 kemunculan di `am.java`) atau `c.a(long)` (881 kemunculan di `am.java`). Laporan ini **tidak mendekripsi string apa pun**; mekanisme dekripsinya dijelaskan di Bagian 3. Beberapa metode raksasa gagal didekompilasi JADX dan tinggal stub (`am.java:1605-1610`, `am.java:2194-2199`), jadi sebagian alur direkonstruksi dari call site dan wrapper-nya.

---

## 0. Arsitektur keseluruhan

```
Launcher ──► a.am (Activity utama, 24.144 baris)
               │  statics am = seluruh state kapabilitas modem (matriks per-slot [5])
               │  17 method native (JNI) ke liba.so
               ├─► coroutine a.l  → System.loadLibrary("a")  (a/l.java:47)
               ├─► coroutine a.y  → startup: baca modem, isi state
               ├─► a.d            → serializer state am → Intent extras untuk editor
               └─► menu ─┬─► a.em (extends a.bd) — checklist band/RAT (expandable list)
                         ├─► a.ca (extends a.bd) — switch fitur LTE/NSA/NR
                         ├─► a.sc (extends a.bd) — editor blacklist/whitelist cell & channel
                         ├─► a.pc (extends a.bd) — picker prioritas RAT
                         └─► a.ba (Activity polos) — penampil teks read-only + cari + share
```

Pola desainnya konsisten:

- **State global di statics `am`.** Semua data kapabilitas (matriks `long[][]`, `byte[][]`, `String[]`, bitmask `f302a`) adalah field statis kelas `am`. Kelas lain (`a.m`, `a.d`, aktivitas editor) hanya membaca/menulis state itu lewat accessor.
- **Editor tidak pernah menyentuh modem langsung.** Aktivitas editor menerima salinan state lewat Intent extras, mengeditnya, lalu saat Back hasilnya ditulis kembali ke Intent yang sama (`bd.onBackPressed` → `setResult(a(), getIntent())`, `a/bd.java:56-61`) dan diproses `am` lewat coroutine `a.av` (`am.java:14421-14426`). Eksekusi tulis ke modem terjadi di sisi `am`/native.
- **Kunci extras satu karakter.** Konvensi di seluruh app: `"*"` = judul, `"#"` = jumlah halaman/slot, `"!"` = string sumber/subjudul, `":"` = array label halaman, `"T".."Z"`, `"b".."L"` = matriks data. String ≥ 16384 karakter tidak dikirim lewat Binder, melainkan disimpan di registry statis `am.f730a` (SparseArray) dan yang dikirim hanya `hashCode`-nya sebagai int (`a/a.java:321-329`); penerima mengambil **dan menghapusnya** (one-shot, `a/ba.java:87-101`).
- **UI utama dibangun programatik.** `am` tidak pernah memanggil `setContentView` (grep: 0 hit) — layout `res/layout/am.xml` praktis tidak dipakai; dua TextView dibuat dengan kode dan ditempel ke `android.R.id.content`.

---

## 1. Per-aktivitas

### 1.0 Kelas dasar editor: `a.bd` (dipakai em, ca, sc, pc)

`public abstract class bd extends Activity implements LayoutInflater.Factory2` — `a/bd.java:18`.

- Kontrak abstrak yang diimplementasikan tiap subclass: `int a()` (pengumpul hasil saat keluar), `View a(int, LayoutInflater, ViewGroup)` (pabrik view per halaman), `void a(int, View)` (binder/pengisi halaman), `void mo91a()` (hook init, `a/bd.java:30-43`).
- `onCreate` bersama (`a/bd.java:65-94`): membaca extras umum — `"*"` → judul, `"#"` → jumlah halaman/SIM (default 1), `"!"` → string sumber, `":"` → array label. Lalu:
  - 1 halaman: `setContentView` langsung ke view pabrik.
  - >1 halaman: `setContentView(R.layout.cc)` — FrameLayout berisi pager kustom `a.vp` (ViewPager hasil obfuscation AndroidX) dengan adapter `a.bf` (`a/bd.java:92-94`, `res/layout/cc.xml`). Satu halaman = satu slot/SIM.
- Menu dasar: satu item id `1048577` ikon close (`a/bd.java:143-150`); memilihnya = `setResult(0); finish()` alias batal (`a/bd.java:165-167`).
- Subclass dapat menambah item menu "delete" yang menghasilkan result code khas: em → 21, sc → 12 (lihat di bawah).

### 1.1 `a.em` — Checklist band/mode per RAT (expandable list)

`public final class em extends bd implements AdapterView.OnItemLongClickListener` — `a/em.java:26`. Layout: `res/layout/em.xml` (TextView monospace `@+id/info` + list kustom `<a.el>` `@+id/el`, `choiceMode="multipleChoice"`).

- **Data masuk (semua dari Intent, `em.java:376-580`):** `"b"` = `long[]` mask band tersedia; `"d".."j"` = `long[][]` Serializable tabel per-RAT; `"C"` = `long[]` pilihan saat ini; `"D".."L"` = matriks pilihan per grup; `"R"` = int (≤3 mengaktifkan urutan hint tutorial). Kunci RAT yang dipakai: 3, 5, 4, 7, 19, 20, 21, 22 (`em.java:455-580`). Extra wajib yang null → exception `cq("")`.
- **Binder (`em.java:645-921`):** label halaman dari `":"` (didekripsi per halaman, `em.java:673`); mask tersedia vs terpilih didekode dari daftar grup statis `[2,16,3,5,4,7,19,20,22,21]` (`em.java:73-74`) menjadi grup/anak di `ExpandableListView` kustom `a.el` (`a/el.java:16`) dengan adapter `a.bg`. Tap di sepertiga kiri item = expand/collapse grup; tap lain = toggle centang (di dalam `el.performItemClick`).
- **Long-press (`em.java:1390-1404`):** pada grup → set semua anak mengikuti status grup; pada anak → toggle semua anak segrup.
- **Tutorial/hint terjadwal** (hanya bila extra `"R"` mengizinkan, halaman 0): rantai `postDelayed` 500/1500/2500 ms (geser pager bolak-balik) dan ≥4500 ms (expand/collapse grup 0), diselingi Toast petunjuk (`em.java:82-148, 297-374`). Ukurannya membaca `SystemProperties.get(<string terenkripsi>)` — satu-satunya pembacaan properti sistem di kelima editor (`em.java:300`).
- **Hasil (`em.java:174-287`):** mask pilihan dihitung ulang dari `el.getCheckedItemIds()`, bit yang tidak tersedia dimask, lalu ditulis balik ke Intent yang sama: `"C"` (long[]) dan `"D".."L"` (Serializable). Return 1.
- **Menu delete** (id `1048593`, `em.java:1380-1412`): `setResult(21); finish()` — sinyal "reset/hapus pilihan", extras tidak diubah.
- Tidak memulai aktivitas lain, tidak ada dialog, tidak ada I/O file/native/prefs.

### 1.2 `a.ca` — Switch fitur LTE / NSA / NR (halaman kapabilitas)

`public final class ca extends bd` — `a/ca.java:19`. Layout: `res/layout/ca.xml` — ScrollView berisi ±20 `Switch` + CheckedTextView rantai Rx + kolom teks filter, dalam 3 seksi. Ini halaman "Capabilities" yang namanya jadi nama aplikasi.

Isi layout (id → label, `res/layout/ca.xml`):

- **LTE features:** `lte_ca_enabled` "LTE+LTE aggregation" (:28), `lte_ul256qam_enabled` "Uplink 256-QAM" (:37), `lte_ulca_enabled` "Uplink aggregation" (:41), `lte_dl256qam_enabled` "Downlink 256-QAM/1K-QAM" (:45), `lte_dlmimo_enabled` (:49), `lte_ca_filter_enabled` "Band combos filter" (:54) + `lte_ca_filter` kolom teks daftar combo (:64), `lte_voice_enabled` "Voice over LTE" (:75), CheckedTextView `lte_rx0..3_enabled` "Rx chain#0..3" (:79-92), `lte_rx0_d..lte_rx7_d` "D#0 (PRx) … D#7" (:95-131), `lte_ard_enabled` "Adaptive receive diversity" (:134).
- **NSA features:** `nsa_lte_enabled` "LTE+NSA aggregation" (:144), `nsa_lte_dsda` (:148), `nsa_ndds` "LTE+NSA during data SIM call" (:152), `nsa_lte_dss_enabled` "Dynamic spectrum sharing" (:156).
- **NR features:** `nr_ca_enabled` "NR+NR aggregation" (:168), `nr_nr_enabled` "NR+NR dual connectivity" (:176), `nr_nr_dsda` (:180), `nr_ul256qam_enabled` (:184), `nr_ulmimo_enabled` (:192), `nr_dl256qam_enabled` (:200), `nr_dlmimo_enabled` (:208), `nr_voice_enabled` "Voice over NR" (:216), `nr_redcap_enabled` "Reduced Capability (NR-Light)" (:220), `nr_rx0/1_enabled`, `nr_ard_enabled` (:228-236).
- Footer statis panjang di XML sendiri menjelaskan kenapa kontrol bisa abu-abu (tidak ada dukungan HW / tidak ada konektivitas supplemental / mode pesawat / fitur generik), daftar fitur **write-only** (LTE+LTE aggregation, LTE UL aggregation, DL 256-QAM, Rx chains, ARD), dan format combo band: `Band+Class+[Streams][Uplink Class]-...[Bandwidth Class Set]`.

Cara kerja kode (`a/ca.java`):

- **Sumber data = Intent extras berupa array string per halaman** dibaca di binder (`ca.java:467-2466`): string token-fitur per halaman (kunci terenkripsi `c.a(-193743384735216L)`), string filter combo (kunci `c.a(-193764859571696L)`), `"_"` = identitas jaringan, dan `int[]` gating (kunci `c.a(-228201907351024L)`) yang menentukan kontrol aktif/abu-abu via `de.a(...)`.
- **Status awal tiap switch** ditentukan dengan uji substring: `setChecked(gp.a(strToken, c.a(<token terenkripsi>), false))` — jadi seluruh state halaman dikodekan sebagai satu string token per halaman; tiap fitur punya token terenkripsinya sendiri (mis. `lte_ca_enabled` di `ca.java:683`).
- **Interaksi:** satu listener bersama untuk semua CheckedTextView (toggle, `ca.java:156-172`); satu `OnCheckedChangeListener` khusus di `lte_ca_enabled` yang hanya mengaktifkan `lte_ulca_enabled` bila string fitur mengandung token tertentu (`ca.java:745-753`); kontrol yang dinonaktifkan di halaman lain dicerminkan lewat listener `a.bc` yang menyalin status Checkable antar-halaman (`ca.java:686-695` dst.). Switch lain tidak punya listener — status dipanen saat keluar.
- **Hasil — serializer `ca.java:25-192`:** teks filter dinormalisasi (pisah `;`/newline, spasi→`-`, buang prefix, tambah suffix default bila tidak diakhiri salah satu dari 11 suffix terenkripsi, uppercase, gabung `;`), lalu string token-fitur dibangun ulang dengan menambahkan token terenkripsi untuk setiap kontrol yang tercentang, berurutan dari LTE → NSA → NR → Rx/ARD (`ca.java:45-150`), dan kedua array ditulis balik ke Intent (`ca.java:180-181`). Return = status per halaman.
- Tidak ada menu sendiri (hanya menu batal dari `bd`), tidak memulai aktivitas, tidak ada dialog/toast/IO.

### 1.3 `a.sc` — Editor blacklist/whitelist cell & channel per RAT

`public final class sc extends bd` — `a/sc.java:18`. Layout: `res/layout/sc.xml` — ScrollView berisi 8 `EditText` dalam 4 seksi (GSM, UMTS, LTE, NR) plus label peringatan kondisional (`req0..req10`, `lim0..lim3`, `*_support`, `*_default/_legacy/_ancient`) dan `@+id/output`.

Isi per seksi (label dari XML):

- **GSM:** `gsm_cells` "Cell blacklist (6 at most)", `gsm_channels` "Priority channel list (6 at most)".
- **UMTS:** `umts_channels` "Priority channel list (1 at most)" + peringatan "Changes RAT preference to UMTS-only if set".
- **LTE:** `lte_cells` "Channel code whitelist (6 at most)" / daftar prioritas, `lte_channels` daftar channel, `lte_smalls` "Channel blacklist (6 at most)".
- **NR:** `nr_cells`, `nr_channels` daftar kode/channel dengan format `channel step code band`.
- Hampir semua seksi membawa peringatan statis yang sama: "Modem power cycle will be toggled if changed", "MODEM OS WILL BE RESTARTED IF CHANGED (~1 minute) — may occasionally initiate phone reboot — may disconnect the application until restart", "Supplemental connectivity required", "No hardware support". Legenda format di footer: Channel = ARFCN/UARFCN/EARFCN/NR-ARFCN, Code = BSIC/PSC/PCI, Step = SCS, Area = LAC/TAC, Cell = CID.

Cara kerja kode (`a/sc.java`):

- **Binder (`sc.java:269-575`):** `output` diisi string `"!"` di halaman 0; hint numerik dari extras `long[]` (kunci terenkripsi + indeks halaman) diformat `dd.a(...)` ke `*_hint`; `int[]` gating (kunci yang sama dengan `ca`, `sc.java:331`) menurunkan ±20 boolean dari indeks 0..10 yang mengatur visibilitas label `req/lim/support/default/legacy/ancient`, termasuk ambang jumlah band 8/16/17 (LTE cells), 216 (LTE channels), 271/272 (NR).
- **Data nilai = extras `long[]` `"T".."Z"` dan `` "`" ``**, 6 slot per halaman (offset `i*6`), `-1` = terminator. Setiap nilai adalah bilangan 64-bit terpak: GSM cell = `(field1<<48)|(field2<<40)|(field3<<16)|field4` (`sc.java:438-462`); LTE cell = `(PCI<<32)|CID`; NR cell menyertakan step SCS `15 << nibble` (`sc.java:528-554`). Binder membongkarnya menjadi teks terformat; parser (`sc.java:34-212`) melakukan kebalikannya dari teks EditText (maks 6 entri per kolom).
- **Hasil (`sc.java:231-254`):** array `long[30]` diisi `-1`, salinan 35 elemen untuk `"T","U","`","V","W","X","Y","Z"` ditulis ke Intent, parser dijalankan per halaman.
- **Menu delete** (id `1048609`, `sc.java:581-598`): `setResult(12); finish()`.
- Tanpa listener klik sama sekali — murni edit teks; tanpa dialog/toast/aktivitas keluar.

### 1.4 `a.pc` — Picker prioritas (dua daftar, pilihan tunggal)

`public final class pc extends bd` — `a/pc.java:16`. **Tanpa layout XML**: UI dibangun programatik di pabrik view (`pc.java:66-101`) — dua `ListView` ditumpuk vertikal dalam satu LinearLayout: daftar atas (label biasa, adapter `a.h` di atas `android.R.layout.simple_list_item_1`) dan daftar bawah pilihan-tunggal (`simple_list_item_single_choice`) memakai widget kustom `a.i`.

- **Data masuk (`pc.java:103-140`, semua kunci terenkripsi):** `byte[][]` nilai kandidat, dua `String[]` label (untuk daftar atas dan bawah), `byte[][]` pilihan saat ini, dan dua `byte[][]` set referensi pembanding. Extra wajib null → exception.
- **Binder (`pc.java:146-200`):** tiap kandidat diberi flag bit di SparseIntArray: bit 1 = sama dengan pilihan saat ini, bit 2 = sama dengan set referensi C, bit 4 = sama dengan set referensi D; widget `a.i` merender glif status ⭐/⏳ (`pc.java:71-77`) dan menangani long-click sendiri (`a/i.java:20`).
- **Hasil (`pc.java:45-65`):** per halaman, posisi tercentang di daftar bawah menggantikan `f592b[page]` dengan kandidat terpilih dan array itu ditulis balik ke Intent (kunci terenkripsi yang sama dengan kunci token di `ca`). Return 3 bila semua halaman punya pilihan, -1 bila ada yang kosong.
- Dipakai `am` sebagai picker prioritas RAT — dijangkau pengguna lewat label menu **"PDC Preferences"** (dispatch bergeser +1, lihat 2.5); label "RAT Priority" sendiri justru membuka laporan Debug Info. Tanpa menu/dialog/aktivitas keluar.

### 1.5 `a.ba` — Penampil teks read-only + pencarian + share

`public final class ba extends Activity` — **tidak** extends `bd` — `a/ba.java:19`. Tanpa layout XML dan tanpa `setContentView`: seluruh UI programatik di `onCreate` (`ba.java:36-111`):

- Header TextView 12sp (teks terenkripsi), body TextView 12sp monospace yang bisa diseleksi, dibungkus ScrollView; `SearchView` di bagian bawah dengan listener `a.bb`; ketiganya ditempel ke `android.R.id.content` lewat loop tunggu `SystemClock.sleep(1)` sampai FrameLayout muncul (`ba.java:75-85`) — pola yang sama dengan `am`.
- Judul dari extra `"*"` (`ba.java:86`); isi dari extra `"!"` (`ba.java:87-101`): String langsung, atau Integer = kunci registry `am.f730a` yang diambil **dan dihapus** (konten besar >16K). Bila isi >16384 karakter, body awalnya diisi teks placeholder terenkripsi.
- Rendering asinkron: coroutine `a.az` mengubah string menjadi `Spannable` dengan pewarnaan glif status (lewat `a.a(String)`), menempelkannya ke body di UI thread, dan menyiapkan salinan lowercase untuk pencarian (`az` di `a/az.java`).
- **Pencarian (`a/bb.java:13`):** submit query (>1 char) mencari kemunculan berikutnya case-insensitive dari posisi kursor, menyeleksi teksnya di body; tidak ketemu → pelat SearchView diwarnai merah. Change query mewarnai ulang pelat.
- **Menu share** (`ba.java:112-138`): satu item ikon `ic_menu_share`; memilihnya membangun Intent kirim (action, MIME type, dan kunci extra semuanya string terenkripsi) berisi teks body dengan satu substring terenkripsi dibuang, lalu `startActivity(Intent.createChooser(...))`. Ini **satu-satunya implicit intent di seluruh sisi Java** — berbagi laporan lewat aplikasi lain.
- Dipakai `am` untuk menampilkan laporan: RFFE Components, EFS Misc Files, ICC Misc Files, LTE CA Combos (label→isi bergeser +1, lihat 2.5), dan laporan internal dari `m45a()`. Tidak mengembalikan result (Back = perilaku Activity biasa).

---

## 2. Aktivitas utama `a.am` (24.144 baris)

`public final class am extends Activity` — `a/am.java:49`. Bukan AppCompat. Import yang khas: `java.lang.reflect.Method`, `java.nio.ByteBuffer`, `java.util.concurrent.Semaphore`, `dalvik.annotation.optimization.{CriticalNative,FastNative}`. **Tidak ada** import `ExecutorService`, `Handler`, `TelephonyManager`, atau `java.io.File`.

### 2.1 Field & state

Hampir semua field adalah array primitif/string berukuran **[5]** (per slot/SIM), [3], [7], [8], [30], [35] — state kapabilitas modem, bukan state UI:

- Statis (`am.java:50-215`): `int[] A,B,w,x,y,z`; `SparseArray f730a` (registry spill-over Intent); `Object f272a` (lock); dua handle refleksi `Method f273a`, `Method b`; `HashMap f274a`; banyak `String`/`String[]`/`byte[][]` (diisi blok static-init `am.java:216-638`); `boolean f276d/f278e` (dua toggle opsi menu).
- Instans: hanya **dua view** — `volatile TextView f305a` (header) dan `f320b` (body, selectable, monospace). Sisanya data: `ByteBuffer[] f312a/f326b` ([5] × 3376 byte; slot 4 = 65536) dan `f335c` ([8] × 512B) sebagai buffer kerja modem; `long f302a` = **bitmask kapabilitas master** yang menggerbangi restore state, `onPrepareOptionsMenu`, dan `onDestroy`; `int f330c` = pemilih perintah/halaman saat ini; `int f338d` = indeks slot aktif; `Semaphore f307a` (1 permit) untuk serialisasi akses modem; `hn f303a` = handle Job coroutine utama.

### 2.2 onCreate (`am.java:14436-14862`)

1. Prioritas thread utama dinaikkan ke 10; `super.onCreate` (14444-14446).
2. ActionBar: sembunyikan ikon home pra-API 21; pasang `OnMenuVisibilityListener` yang hanya mencatat flag `f331c` (14447-14460).
3. `i = m43a()` — int diturunkan dari `getApplicationInfo().processName`; nilai ini dipakai sebagai kunci XOR/shift saat mendekode string menu di dispatcher (14461; helper di 10762+).
4. **Restore state** bila ada Bundle (14462-14746): ±50 kunci satu karakter (`"!"`, `"$"`, …, `"¡"`, `"\\"`) memulihkan kedua SparseIntArray, string, flag, dan — hanya bila `bundle.getInt("!") == Process.myPid()` — seluruh tabel kapabilitas termasuk salinan byte mentah ke ByteBuffer kerja.
5. **Jalur first-run** (hanya bila `f302a == 0`, 14747-14824):
   - `getSharedPreferences("g", 0)`; pref `"x"` dibaca ke `f390o` dan di-deobfuscate di tempat dengan XOR bergulir (`char ^= 19,20,21,…`, 14759-14788).
   - Thread mentah prioritas 1: `new df(ei).start()` — worker startup latar (`a/df.java` extends Thread, `run()` = invoke lambda yang memanggil `am.a(am, fp, fp, fp)`, 14795-14798).
   - Coroutine startup utama: `cs.a(..., new y(this, sharedPreferences, ...holders...))` (14801) — kelas `a/y` menerima activity + prefs + semua holder; error ditangkap dan ditambahkan ke `f390o`. Di dalam rantai coroutine inilah `a/l` menjalankan `System.loadLibrary("a")` (Bagian 3.2) dan pembacaan modem awal terjadi.
   - Tidak ada `setContentView`, tidak ada permintaan permission, tidak ada cek root/Magisk **di kelas ini** — semua di coroutine/native.
6. **Bangun UI programatik** (selalu, 14825-14862): dua TextView dibuat dengan kode — header 12sp; body 12sp selectable monospace dengan teks awal `f390o`, padding atas 21dp — lalu ditempel ke `findViewById(android.R.id.content)` dalam loop retry `SystemClock.sleep(1)`. Layout `R.layout.am` tidak pernah di-inflate oleh `am`.

### 2.3 Threading & konkurensi

- **Tanpa ExecutorService/AsyncTask/Handler.** Polanya: Kotlin coroutines (obfuscated `kotlinx.coroutines`, peluncur `cs.a`) + 1 Thread mentah + 1 Semaphore.
- 12 titik peluncuran coroutine: konstruktor (Job utama `a/l`, `am.java:1068`), startup (`a/y`), operasi baca/tulis modem (`a/u` ×3 — `am.java:4992, 5895, 7176`), hasil aktivitas (`a/av`), regenerasi laporan kapabilitas (`a/ax` → `m45a()`; cabang bernama `menu_ims` tapi dipicu label "RRC CA Combos"), restart MPSS (`a/ay` → `m42g()`; cabang bernama `menu_hw_rfc_report` tapi dipicu label "Restart MPSS"), dan dispatch ulang menu per slot (`a/aw` ×2).
- ±90 `runOnUiThread` di `m45a()` (routine render hasil raksasa, `am.java:10895-14123`): pekerjaan dilakukan di luar UI thread, lalu Toast/teks diposting lewat Runnable `a.g`; di banyak titik kode memeriksa `Looper.getMainLooper().getThread() == Thread.currentThread()` untuk memilih sentuh-UI-langsung vs posting.
- `Semaphore f307a`: dipakai di `onDestroy` — `tryAcquire(5 detik)` bila sesi modem aktif sebelum cleanup native (14879-14888).

### 2.4 Method native (JNI) — 17 deklarasi

Semua `static native`, beranotasi `@FastNative`/`@CriticalNative`:

| Method | Signature | Baris | Peran (dari pola pemakaian) |
|---|---|---|---|
| `d` | `int d(int, int, String, int, int)` | 7594 | perintah bernama per slot |
| `e` | `int e(int)` | 7677 | tutup/reset per indeks (dipakai `m.a(int)`) |
| `g` | `long g(long, long, long, long, long)` | 10492 | penggabung bitmask (dipanggil `m.a(long)` dengan konstanta `0,0,2^40,-1`) |
| `h` | `int h(int, int, String)` | 10715 | perintah bernama (private) |
| `i` | `int i(String, int)` | 10718 | operasi berbasis nama/path |
| `j` | `long j(int, int, int)` | 10722 | baca nilai long per slot |
| `k` | `int k(int, int, int, long)` | 10725 | tulis nilai long per slot |
| `l` | `int l(int)` | 10728 | buka/status per indeks |
| `m` | `ByteBuffer m(int, int, int)` | 10731 | ambil direct ByteBuffer data modem |
| `n` | `int n(String)` | 10735 | operasi atas nama/path (dipakai `m45a`, am.java:12017, argumen string terenkripsi) |
| `no` | `int no()` | 10738 | **getter last-error** — pola `return -no()` di wrapper |
| `o` | `int o(int, byte[], int, int, int, int)` | 10742 | **workhorse**: kirim frame perintah `byte[]` / terima respons per slot |
| `p` | `String p(int, int)` | 10746 | baca string per id |
| `r` | `int r(String, byte[])` | 10749 | **baca file by-name** ke buffer (dipakai di seluruh `a/m`) |
| `s` | `int s(long[], long[])` | 10752 | transformasi array long (dipakai `a/aq.java:95`) |
| `t` | `int t(int, int, int, int)` | 10755 | inisiasi sesi (dipanggil `m.a()` sebagai `am.t(1,5,0,400)`) |
| `u` | `String u()` | 10758 | baca string global (identitas perangkat) |

Pola pemakaian inti — pasangan `o()` + `no()` dibungkus wrapper privat (`am.java:1112-1195`): frame perintah kecil dikirim, mis. `{75,37,3,0,0,1}`, `{75,19,48,0,0,0,47,0}`, `{41,(byte)i,0}`, `{75,85,7,0,…}` (prefix `75`='K', `41`=')'), lalu `no()` dibaca sebagai kode error; satu wrapper melakukan polling dengan `SystemClock.sleep(50)` sampai timeout. Method lain sebagian besar **tidak dipanggil langsung di am.java**, melainkan lewat fasad `a.m` (Bagian 3.3) dan kelas coroutine. Di dalam `am`, native dicapai tidak langsung lewat method sesi pengembali `ck`: `b()` (2281), `c()` (3494), `d()` (7596), `e()` (7679), `f()` (7908), `m42g()` (10496), `m45a()` (10895) — dispatcher menu memilih berdasarkan `f330c` (2→`c()`, 12→`d()`, 3→`e()`, 4→`f()`; `am.java:23566-23573`).

### 2.5 Menu (`res/menu/am.xml`, 24 item) & dispatch-nya

`onCreateOptionsMenu` (14864-14872) meng-inflate `R.menu.am`. `onOptionsItemSelected` adalah monster 15554-23961 (±8.400 baris — mayoritas file ini). `onPrepareOptionsMenu` (23963-24051) mengatur visibilitas secara programatik **berdasarkan indeks item** dan state kapabilitas (`f302a`, isi string statis, `f319b`), bukan berdasarkan id.

> **KOREKSI 2026-10-21 — dispatch bergeser +1 (penting, menggantikan pembacaan awal tabel ini).** `onOptionsItemSelected` menghitung `int itemId = menuItem.getItemId() + 1;` SATU KALI di `am.java:15964` dan tidak pernah meng-assign ulang sampai akhir method — semua cabang `if (itemId == R.id.X)` membandingkan nilai yang sudah digeser. Akibatnya **label yang diklik pengguna selalu menjalankan aksi milik id resource BERIKUTNYA** (urutan id di `res/values/public.xml` 0x7f010030–0x7f01004e persis mengikuti urutan menu). Ini obfuskasi yang disengaja, bukan bug. Tabel di bawah adalah versi TERKOREKSI (label terlihat → aksi sebenarnya), diverifikasi silang `res/menu/am.xml`, `public.xml`, kode cabang, dan kamus string terdekripsi.

Struktur menu XML → perilaku kode (terkoreksi):

| Grup/Item (id, judul XML) | Visibel awal | Perilaku |
|---|---|---|
| `menu_config` "Config" | — | grup; diaktifkan hanya bila `f302a != 0` (`am.java:23970`) |
| `menu_rbp_config` "RAT/Band Preferences" | ya | → **editor checklist band `em.class`** via serializer `d.a(...)` (cabang `nr_ard_enabled`, `am.java:15965-15967`) |
| `menu_rat_prio` "RAT Priority" | **false** | **tidak membuka picker** — jatuh ke alur generik → laporan Debug Info. Picker `pc` dijangkau lewat label "PDC Preferences" (baris `menu_pdc_config`) |
| `menu_csp_config` "Searcher Preferences" | ya | → `sc.class` via serializer `d.b(...)` (cabang bernama `menu_debug`, `am.java:15968-15970`) — **judul menunya "Searcher", isinya editor cell/channel Bagian 1.3; satu-satunya item Config yang label dan isinya cocok** |
| `menu_dfp_config` "Data Preferences" | **false** | → **editor switch kapabilitas `ca.class`** via serializer `d.c(...)` (cabang `menu_efs_clear`, `am.java:15971-15973`) — judul window `ca` memang "Data Preferences" (`d.java:493`, konstanta terdekripsi persis). Tidak ada fitur data terpisah |
| `menu_pdc_config` "PDC Preferences" | **false** | → **picker RAT Priority `pc.class`**, extras dibangun inline: `"*"` judul, `"#"`=`f317b`, + 7 extra matriks (`byte[][]` f275c/f277d/f279e/f280f/f281g + `String[]` f284i/f287j) (cabang `menu_rat_prio`, `am.java:15974-15985`) — bukan halaman PDC/MBN |
| `menu_efs_clear` "Clear RMTEFS" | **false** | **tidak menghapus apa-apa** — id hasil gesernya (`menu_hw`, grup) tidak punya cabang; jatuh ke alur generik → laporan Debug Info. Editor `ca` yang dulu dikira milik id ini dibuka label "Data Preferences" |
| `menu_ims` "IMS" (grup) | **false** | grup; terlihat bila flag `f308a` (23963+) |
| `menu_ims1` / `menu_ims_status` "IMS Service Status" | false/ya | → **dialog laporan kapabilitas raksasa** (`f809a=-3`, blok `am.java:15989-23340`; cabang `menu_kacke`) — BUKAN penampil status IMS |
| `menu_hw` "HW Info" (grup) | **false** | grup info hardware |
| `menu_hw_rffe` "RFFE Components (deprecated)" | ya | → `ba.class`, judul "RFFE Components", isi string statis `c` (cabang `menu_hw_rrc_ca`, `am.java:23361-23366`); salah satu sumber string `c`: file modem `/rfc/rffe_scan/rffe_scan.dat` yang dibaca di `a.an` |
| `menu_hw_efs_misc` "EFS Misc. Files" | ya | → `ba.class`, judul "EFS Misc. Files", isi string statis `e` (cabang `menu_hw_icc_misc`, `am.java:23368-23373`) |
| `menu_hw_icc_misc` "ICC Misc. Files" | ya | → `ba.class` via serializer `d.d(...)`, `startActivityForResult` (cabang `menu_hw_lte_ca`, `am.java:23375-23380`) — isi dibangun on-demand: parser CA-combo dari blob biner `am.f285i` (`d.java:1068-2146`); class `ai` memformat bagian kartu [EF.IMSI_M]/[EF.KEYS] |
| `menu_hw_lte_ca` "LTE CA Combos (deprecated)" | ya | → `ba.class`, judul "LTE CA Combos", isi string statis `d` (cabang `menu_hw_reboot`, `am.java:23381-23387`) |
| `menu_hw_rrc_ca` "RRC CA Combos (deprecated)" | ya | → coroutine `a/ax` → `m45a()` (cabang `menu_ims`, `am.java:23389-23391`; `a/ax.java:25`) — meregenerasi laporan kapabilitas utama, bukan penampil combo tersendiri |
| `menu_hw_rfc_report` "RFC Report" | ya | **tidak ada implementasinya** — id hasil gesernya (`menu_hw_rffe`) tidak punya cabang; jatuh ke alur generik → laporan Debug Info |
| `menu_hw_reboot` "Restart MPSS" | ya | → coroutine `a/ay` → `m42g()` (cabang `menu_hw_rfc_report`, `am.java:23392-23395`; `a/ay.java:25`) — restart modem beneran ✓ label dan efek cocok |
| `menu_debug_status` "Debug" (grup) | **false** | terlihat bila teks debug `f292m` tidak kosong |
| `menu_debug` "Debug Info" | ya | → jalur generik cabang `menu_debug_status`: query modem `b()` — "system selection prefs" / "modes: 0x…" — BUKAN dump debug; laporan Debug Info justru jadi fall-through label lain ("RAT Priority", "Clear RMTEFS", "RFC Report") |
| `menu_options` "Options" (grup) | ya | grup — tidak bisa diklik; cabang `menu_options` (toggle statis `f278e`, `am.java:23403-23405`) tercapai dari label "Shitty kernel" |
| `menu_blitz` "Deny ⚡ connect" (checkable) | ya | **cuma toggle flag statis `f276d`** (cabang `menu_command1`, `am.java:23397-23401`) — tidak ada dialog sama sekali |
| `menu_kacke` "Shitty kernel" (checkable) | ya | **cuma toggle flag statis `f278e`** (cabang `menu_options`, `am.java:23403-23405`) — laporan kapabilitas raksasa `f809a=-3` yang dulu dikira milik item ini dipicu label "IMS Service Status" |
| `menu_about` "About" | ya | → dialog char-decode (`f809a=-2`, cabang `menu_blitz`, `am.java:23406+`) — dialog yang dulu dikira milik "Deny ⚡ connect" |
| `menu_commands` "Test Commands" (grup) | **false** | selalu dipaksa tak terlihat di `onPrepareOptionsMenu` (`am.java:23983`) |
| `menu_command1..5` "Test Commands 1–5" | (grup di atas) | **semua no-op** — klik TC1–TC5 mendarat di cabang `menu_command2..5` / `menu_commands` yang `return true` (`am.java:15986-15988`). Toggle `f276d` yang dulu dikira milik command1 dipicu label "Deny ⚡ connect" |
| (bukan item menu) `nr_ard_enabled` | — | cabang pertama dispatcher (→ `em.class` via `d.a`, `am.java:15965-15967`); id-nya tepat sesudah `menu_rbp_config`, jadi cabang ini tercapai justru dari klik label "RAT/Band Preferences" |

Catatan dispatch: **id item digeser `+1` satu kali** di `am.java:15964` — konsekuensinya seluruh tabel di atas (label → cabang id berikutnya); label yang id hasil gesernya tidak punya cabang ("RAT Priority", "Clear RMTEFS", "RFC Report") jatuh ke alur generik di ekor method yang menampilkan laporan Debug Info. Beberapa perintah juga tidak di-dispatch per id melainkan **posisional**: untuk kode perintah `10 <= f809a < 20`, dispatcher memanggil ulang `onOptionsItemSelected(null)` sekali per slot dengan `f330c` diganti, atau meluncurkan coroutine `a/aw` (`am.java:23658-23676, 23852`). Semua `startActivityForResult` memakai requestCode **0** dari satu titik bersama (`am.java:23492-23493`).

### 2.6 Dialog, clipboard, toast di `am`

- **AlertDialog** hanya dibangun di ekor dispatcher (dua blok nyaris identik, `am.java:23574-23612` dan `23732-23791`): judul terdekripsi; isi = pesan, atau daftar item dari string yang di-split `";"` dengan listener klik; tombol positif terdekripsi bila kode perintah `>= -2`; `show()` hanya bila dipicu dari menu dan kode `< 0`. Setelah tampil, TextView pesan diberi `Linkify.addLinks(...)` dengan skema terdekripsi.
- **Handler tombol dialog** `a(am, fp, DialogInterface, int)` (`am.java:1636-1690`): `which==0` → salin teks utama ke clipboard (string berbagi terformat: baris judul di-hashtag-kan + isi + flag `+`/`-` dari state); `which==1` → salin teks debug statis `f292m` ke clipboard; `which==2` → **perintah shell** (Bagian 5). Clipboard memakai API lama `ClipboardManager.setText` dengan nama service terdekripsi (`am.java:1652-1690`).
- **Toast:** 66 titik `Toast.makeText`, kebanyakan dari `m45a()` lewat Runnable `a.g` (hasil/error operasi modem, umumnya `LENGTH_LONG`).
- Tidak ada file picker / save dialog / implicit intent di `am.java` (grep `ACTION_VIEW`/`ACTION_SEND`/`Uri.parse`: 0 hit).

### 2.7 Lifecycle lain

- Ada: `onCreate`, `onCreateOptionsMenu`, `onPrepareOptionsMenu`, `onOptionsItemSelected`, `onActivityResult` (14421 — menyerahkan hasil ke coroutine `a/av`), `onBackPressed` (14428 — di API ≥31 memanggil `finish()` dulu), `onDestroy` (14875 — memanggil `onSaveInstanceState(null)` manual untuk flush prefs; bila finishing dan sesi modem aktif: semaphore `tryAcquire(5s)`, cleanup `m.g(...)` atas buffer sesi, `release()`), `onSaveInstanceState` (24053-24144 — menulis ±50 kunci Bundle yang dicerminkan restore di onCreate, **plus selalu** menulis 3 pref SharedPreferences `"g"`: `"t"`, `"z"`, `"y"`).
- **Tidak ada** `onResume/onPause/onStart/onStop/onNewIntent/onRequestPermissionsResult` — semua penyegaran digerakkan dari onCreate/menu/coroutine.

### 2.8 Yang TIDAK ada di `am.java` (negatif terverifikasi grep)

File I/O langsung (0 hit `java.io.File`/stream/`/sdcard`/`/data/`), `Runtime.exec`, literal `"su"`/`"sh"`, `TelephonyManager`/`SubscriptionManager`/`ContentResolver`, `Class.forName`/`getMethod`/`invoke` refleksi, `System.loadLibrary`. Semua itu didelegasikan: file lewat native `n/r/i(String…)`, refleksi disiapkan di `a/a.java` (Bagian 3.1), load library di `a/l.java` (Bagian 3.2).

---

## 3. Kelas helper penting

### 3.1 `a/a.java` (377 baris) — dekriptor string + util tema/UI

`public final class a implements Runnable` — `a/a.java:28`. Bukan hanya dekriptor:

- `a(int)` (`a.java:71-97`): membangun objek `ck` berisi frame protokol hardcoded (mis. `{0,0,0,47,0,126,0,16,1,0,0}`) — template perintah modem.
- `a(Activity)` (`a.java:99-252`): membangun `Resources.Theme` + deteksi dark mode; membaca `Settings.Secure.getInt`, `Settings.Global.getInt`, `Settings.System.getString` dengan **kunci terenkripsi** (`a.java:119-193`), dan menyiapkan handle refleksi (Bagian 5).
- `a(String)` (`a.java:254-297`): mewarnai ulang glif status dalam teks laporan (❌ ⬛ ☑ dll, penanda akhir char 9209) — dipakai penampil `ba`.
- `a(Intent,String,String)` (`a.java:321-329`): putExtra dengan spill-over registry untuk string ≥16384 (lihat Bagian 0).
- `a(Menu)` (`a.java:331-367`): merender token awal judul menu menjadi bitmap ikon item.
- Blok statis (`a.java:47-52`) memanggil dekriptor 4 kali dan membuang hasilnya — pemanasan/pemicu inisialisasi blob.

**Mekanisme dekripsi** — entry point `public static String a(long j, String[] strArr)` (`a.java:303-319`), murni Java, tanpa `Cipher`, tanpa panggilan native:

1. **Seed** = 32 bit rendah konstanta `long` di call site: `j2 = j & 0xFFFFFFFF` (`a.java:304`).
2. Seed di-hash gaya **SplitMix64**: `(j2 ^ (j2>>>33)) * 7109453100751455733L` (konstanta gamma SplitMix), lalu `(… ^ (…>>>28)) * -3808689974395783757L`, diambil 32 bit atasnya (`a.java:305-306`).
3. Hasilnya dilewatkan ke **mixer 16-bit** `a(long)` (`a.java:63-69`): memecah input jadi dua `short`, membentuk jumlah dan XOR keduanya, lalu menggabungkan rotasi/shift (`<<10|>>>22`, `<<9|>>>23`, `<<13|>>>19`, `^(s4<<5)`) menjadi state ±48 bit. Mixer ini dipakai berulang sebagai penggerak state (PRNG-like), bukan sebagai cipher standar.
4. **Offset blob** diturunkan dari 32 bit tinggi konstanta yang sama, dicampur dengan state: tiap konstanta call-site dengan demikian mengkodekan *seed + alamat* sekaligus (`a.java:307-309`).
5. **Pengalamatan blob datar per karakter:** karakter pertama dibaca di `strArr[offset/8191].charAt(offset%8191)` dan di-XOR-kan ke state (`a.java:310`).
6. **Panjang string** bukan disimpan di blob, melainkan diturunkan dari state: `(state>>>32) & 0xFFFF` (`a.java:311`).
7. **Loop autokey berantai** (`a.java:313-317`): untuk tiap posisi keluaran, state dimajukan dengan mixer lalu di-XOR dengan karakter blob berikutnya (offset+posisi+1), dan 16 bit atas state menjadi `char` keluaran berikutnya. Jadi blob berfungsi sebagai entropi per-langkah yang menggerakkan state — bukan ciphertext yang di-XOR keystream tetap.

Material kunci = (a) konstanta `long` per call site, (b) karakter blob itu sendiri. Tidak ada field kunci terpisah.

### 3.2 `a/c.java` (12 baris, 182 KB) — blob string terenkripsi

- Satu-satunya isi: `static final String[] f763a = { … }` di **satu baris** (`a/c.java:7`) + wrapper `public static String a(long j) { return a.a(j, f763a); }` (`a/c.java:9-11`).
- Bentuk terukur tanpa dekripsi: **8 elemen**, panjang `[8191, 8191, 8191, 8191, 8201, 8191, 8191, 3403]`, total **60.750 char**. Ukuran potongan 8191 persis pembagi yang dipakai dekriptor — array ini satu aliran char datar yang dipaginasi. Karakter mencakup code point 49-65.494 (padat glyph CJK/simbol): tiap `char` Java = satu kata data 16-bit blob.
- Skala pemakaian: referensi literal `c.f763a` — `m.java` ×1064, `d.java` ×508, `a.java` ×22, `l.java` ×10; panggilan `c.a(long)` di `m.java` saja ×338, di `am.java` ×881.

### 3.3 `a/l.java` (67 baris) — pemuat library native

Berbeda dari dugaan awal: `l` adalah **lambda coroutine Kotlin** (`final class l extends ef implements ew`, `a/l.java:6`) yang terikat ke activity `am`, diluncurkan sekali dari konstruktor `am` (`am.java:1068`). Isi suspend-nya:

- `Thread.currentThread().setPriority(10)` (`l.java:29`).
- Menghitung string ABI **hanya untuk pesan error**: `Build.CPU_ABI` (SDK<20), segmen terakhir `nativeLibraryDir` (SDK 20), atau `ApplicationInfo.primaryCpuAbi` (SDK>20) (`l.java:36-44`).
- **Satu-satunya jalur load, tanpa fallback:** `System.loadLibrary("a")` (`l.java:47`) — memuat `liba.so` dari direktori native APK. Tidak ada `System.load(path)`, tidak ada pemilihan ABI manual, tidak ada retry/ReLinker. Di seluruh source tree hanya ada satu panggilan load ini.
- Error ditelan dan dikemas: `catch (Throwable)` menyusun pesan dari pesan exception yang dipecah dengan 6 separator terenkripsi, atau pesan terdekripsi bila ABI kosong; hasil akhir dikembalikan sebagai `ck(Integer.valueOf(str.length()), str)` — sukses = `ck(0, "")`, gagal = `ck(panjang, pesan)` (`l.java:49-60`). Tidak ada static initializer di kelas ini.

### 3.4 `a/m.java` (6.748 baris) — fasad `am` + dekoder/formatter kapabilitas

`public final class m` — `a/m.java:16`, tanpa field, ±91 method statis. **Bukan** pemegang deklarasi native (grep `native`: 0 deklarasi):

- 6 method beranotasi `@FastNative/@CriticalNative` yang badannya murni delegasi Java ke native milik `am`: `a()` → `am.t(1,5,0,400)` (init sesi), `a(int)` → `am.e(i)`, `a(String,byte[])` → `am.r(...)` (**baca file by-name**, dipakai di mana-mana), `a(long)` → `am.g(j,0,0,2^40,-1)`, `m164a()` → `am.u()`, `b()` → `am.no()` (`m.java:99-117, 372-375, 3651-3654`).
- ±103 referensi accessor get/set atas field statis `am` dalam 9 grup huruf bergilir — inilah "API" yang dipakai kelas lain untuk menyentuh state.
- **11 dekoder ByteBuffer → teks** (`m.java:1581, 1677, 2455, 3569, 5364, 5474, 6180, 6539, 6592, 6631, 6678`): mengubah buffer mentah dari native `am.m(...)` menjadi laporan terbaca — record ≥372 byte, parsing nibble MCC/MNC, rendering status per byte, dll. Semua labelnya string terdekripsi.
- **Tabel numerik/label:** fungsi koefisien RF per ukuran blok 64/256/1024 (`m.java:19-97`); notasi sel dari bucket kuadrat 1..81 (`m.java:163-366`); unpack bitfield parameter sel (`m.java:368-371`); switch raksasa pemetaan **nomor band 1-8, 11-14, 17-21, 24-26, 28-53, 65-76, 85, 87, 88, 103, 106-108, 111, 252-256** → label terdekripsi (`m.java:383-562`). Hampir tidak ada literal plaintext semantik di file ini.
- **Identifikasi perangkat/model:** `b(String×6)` (`m.java:3766-5363`, ±1.600 baris) — kaskade switch `hashCode()` membandingkan 6 string input (identitas dari native/file) melawan string terdekripsi, digabung dengan probe file di bawah, menghasilkan string hasil terdekripsi.
- **Probe file sintetis:** `m182c()`, `m184d()`, `i()`, `m189i()` (`m.java:5612, 6249, 6690, 6729`) — loop atas daftar nama file terdekripsi, membaca tiap file lewat native `am.r(name, byte[])` dan menafsirkan byte-nya.
- **I/O file Java di m.java (lengkap):** satu-satunya tulis — `Files.write(Paths.get(<prefix path terdekripsi> + nama), bytes)` (`m.java:3630-3639`); empat probe eksistensi `new File(c.a(…)).exists()` dengan path terdekripsi (`m.java:4871, 4899, 4902, 4903`). Tanpa exec/refleksi/jaringan di file ini.

### 3.5 `a/d.java` (2.147 baris) — serializer Intent untuk editor

`public final class d`, tanpa field, 8 method: 4 formatter string privat + 4 exporter publik `X(am, Intent)` dengan total **52 `putExtra`** (`a/d.java:13-2146`):

- `d.a(am,intent)` → untuk `em`: lebih dulu memutasi bit flag di `f310a[]/f324b[]` milik am, lalu menulis extras `"*","#",":","b","d".."L","C","R"` berisi matriks-matriks kapabilitas (`d.java:40-242`).
- `d.b(am,intent)` → untuk `sc`: mem-parse blob teks `am.f/g/h` menjadi `TreeSet<Long>`/`long[][]` (pasangan dikemas `(j<<32)|value`), menulis `"!"` (gabungan counter `am.j..o`), `"T","U","`","V","W","X","Y","Z"`, dan extra per-indeks berkunci terenkripsi (`d.java:252-481`).
- `d.c(am,intent)` → untuk `ca`: 9 extra saja, beberapa kuncinya string terdekripsi penuh; nilai dari `f289k/f291l/f293m/f354f/f360g/f366h` + `int[]{f344e, f323b[0]}` (`d.java:492-503`).
- `d.d(am,intent)` → untuk `ba` (LTE CA combos): mengiterasi `byte[][] f285i` dan diakhiri `"*"`, `"#"`, `"!"` (string terbangun), `":"` = dua string terdekripsi (`d.java:1068-2146`).
- Tanpa I/O file/jaringan/refleksi/exec di file ini.

### 3.6 Kelas library (bukan logika aplikasi)

Sebagian dari 312 kelas adalah library yang ikut ter-obfuscate: `a/jd.java` = kotlinx.coroutines `JobSupport`, `a/vp.java` = AndroidX `ViewPager` (dipakai pager `bd`), `a/cs.java`/`a/hb.java`/`a/jp.java` = internal coroutines (dispatcher, ForkJoinPool, `availableProcessors`), `a/el.java`/`a/i.java`/`a/h.java`/`a/bg.java`/`a/bf.java`/`a/bc.java`/`a/bb.java`/`a/az.java` = widget/adapter/pembantu UI buatan aplikasi sendiri seperti diuraikan di Bagian 1.

---

## 4. Inventaris fitur (fitur → aktivitas → layout → bukti kode)

| # | Fitur yang terlihat pengguna | Aktivitas | Layout | Bukti kode |
|---|---|---|---|---|
| 1 | Laporan kapabilitas modem utama (header + teks hasil, bisa diseleksi/disalin) | `am` | programatik (layout `am.xml` tidak dipakai) | `am.java:14825-14862`, render `m45a()` 10895-14123 |
| 2 | RAT/Band Preferences — label menunya membuka editor checklist band (baris 3) | `em` via `am` | menu `menu_rbp_config` | `am.java:15965-15967` (cabang `nr_ard_enabled`, geser +1); `res/menu/am.xml` |
| 3 | Checklist band/RAT per grup (GSM/UMTS/LTE/NR…), centang massal via long-press, reset | `em` | `em.xml` + baris `el.xml` | `em.java:292-921, 1390-1412`; widget `a/el.java:16` |
| 4 | Switch kapabilitas LTE (CA, UL/DL 256-QAM, UL CA, DL MIMO, VoLTE, filter combo band, Rx chain 0-3, D#0-7, ARD) | `ca` | `ca.xml` seksi LTE | `ca.java:467-753, serializer 25-192` |
| 5 | Switch kapabilitas NSA (LTE+NSA, DSDA, NDDS, DSS) | `ca` | `ca.xml` seksi NSA | `ca.java:103-112` (serializer), layout :144-160 |
| 6 | Switch kapabilitas NR (NR CA, NR-DC, DSDA, UL/DL 256-QAM, UL/DL MIMO, VoNR, RedCap, Rx, ARD) | `ca` | `ca.xml` seksi NR | `ca.java:115-150`, layout :165-240 |
| 7 | Editor blacklist/whitelist cell & channel GSM/UMTS/LTE/NR (maks 6 entri, format ARFCN/BSIC/PSC/PCI/SCS/LAC/TAC/CID) | `sc` | `sc.xml` | `sc.java:269-575` binder, parser `34-212` |
| 8 | RAT Priority picker (dua daftar, pilihan tunggal, penanda status ⭐/⏳) — diakses via label menu "PDC Preferences" | `pc` | programatik | `pc.java:66-200`; widget `a/i.java:20`; pemicu `am.java:15974-15985` |
| 9 | Penampil laporan teks: RFFE Components, EFS Misc Files, ICC Misc Files, LTE CA Combos | `ba` | programatik | `ba.java:36-111`; pemicu `am.java:23361-23387` (label bergeser +1 dari nama cabangnya) |
| 10 | Pencarian teks di dalam laporan (highlight + loncati ke hasil) | `ba` | SearchView programatik | `a/bb.java:13`; `ba.java:62-69` |
| 11 | Bagikan laporan lewat share sheet Android | `ba` | menu share | `ba.java:112-138` |
| 12 | Salin laporan / teks debug ke clipboard dari dialog hasil | `am` | AlertDialog | `am.java:1636-1690, 23574-23612` |
| 13 | Restart MPSS (modem) | `am` → coroutine `a/ay` → `m42g()` | label menu `menu_hw_reboot` (cabang `menu_hw_rfc_report`) | `am.java:23392-23395`; `a/ay.java:25` |
| 14 | Restart modem power cycle otomatis saat preferensi searcher diubah (peringatan di UI) | `sc` | label statis `sc.xml` | `res/layout/sc.xml` teks peringatan; penulisan hasil `sc.java:231-254` → diproses `am`/`av` |
| 15 | Label "IMS Service Status" → dialog laporan kapabilitas raksasa (`f809a=-3`) — bukan penampil status IMS | `am` | menu `menu_ims_status` (cabang `menu_kacke`) | `am.java:15989-23340` |
| 16 | Label "Debug Info" → query modem `b()`: "system selection prefs" / "modes: 0x…" | `am` | menu `menu_debug` (jalur generik cabang `menu_debug_status`) | `am.java:2281` (`b()`) |
| 17 | Opsi "Deny ⚡ connect" (`menu_blitz`) dan "Shitty kernel" (`menu_kacke`), keduanya checkable — cuma toggle flag statis (`f276d` / `f278e`), tanpa dialog | `am` | grup `menu_options` | `am.java:23397-23405` (geser +1) |
| 18 | Test Commands 1–5 — semua no-op (`return true`) | `am` | grup `menu_commands` (visible=false) | `am.java:15986-15988, 23983` |
| 19 | Tutorial/hint animasi saat pertama membuka editor band | `em` | Toast terjadwal + auto-swipe pager | `em.java:82-148, 297-374` |
| 20 | Tema gelap/terang mengikuti Settings sistem | semua (via `bd`/`ba.getTheme`) | — | `a/a.java:99-252` (`a(Activity)`), `ba.java:31-34` |
| 21 | Simpan/pulihkan state antar rotasi & proses (Bundle ±50 kunci + SharedPreferences "g") | `am` | — | `am.java:14462-14746, 24053-24144` |
| 22 | Toast hasil/error operasi modem | `am` | — | 66 titik `Toast.makeText`, pola `am.java:10547+` |

---

## 5. Keamanan & privasi

### 5.1 Permukaan serangan / kemampuan istimewa

- **Manifest tanpa permission, tapi kemampuan setara root.** APK meminta **nol** permission Android. Seluruh akses istimewa (baca/tulis konfigurasi modem Qualcomm/QMI, baca file EFS, restart MPSS) berjalan lewat `liba.so` native (`am.r/n/i/o`) dan — untuk satu jalur — shell. Di perangkat tanpa jalur privilege itu, fungsi inti tidak akan bekerja; aplikasi memang alat field-test untuk perangkat yang sudah di-root/di-unlock.
- **Satu-satunya eksekusi shell di sisi Java** (`am.java:1642-1645`, handler tombol dialog `which==2` di `am.java:1636-1690`):
  ```java
  OutputStream out = new ProcessBuilder(c.a(-171362310155760L)).start().getOutputStream();
  out.write(str.getBytes(fi.b)); out.flush();
  ```
  Nama prosesnya satu string terenkripsi (konsisten dengan shell root seperti `su`, tidak dapat dipastikan tanpa dekripsi) dan perintah yang dibangun dari string statis `r` + substitusi ditulis ke **stdin** proses itu. Pemicunya adalah pilihan pengguna di dialog hasil — bukan input jaringan — tetapi ini berarti aplikasi dapat menjalankan perintah shell arbitrer yang sudah diprogram di dalamnya dengan hak proses tujuan.
- **Refleksi ke API tersembunyi (hidden API):** `a/a.java:245-249` (di dalam pembangun tema, dibungkus try/catch sunyi):
  ```java
  am.b     = Class.forName(a(<terenkripsi>)).getDeclaredMethod(a(<terenkripsi>), new Class[0]);
  am.f273a = Class.forName(a(<terenkripsi>)).getDeclaredMethod(a(<terenkripsi>), new Class[0]);
  ```
  Nama kelas dan method keduanya terenkripsi; `Method` hasilnya disimpan di field statis `am.b`/`am.f273a` untuk di-`invoke` dari tempat lain (tidak ada call site `invoke` untuk keduanya di dalam `am.java` — pemanggilnya kelas lain/native). Ini satu-satunya `Class.forName`/`getDeclaredMethod` di kode aplikasi (di luar internal library coroutines: `hb.java:85` ForkJoinPool, `cy.java:7` service-loader). Target pastinya tidak dapat disebutkan tanpa dekripsi — pola dua method tanpa-arg yang di-cache seperti ini lazimnya mengarah ke method internal framework. Karena targetSdk 29, pemanggilan hidden API semacam ini juga yang membuat aplikasi sensitif terhadap kebijakan greylist Android.
- **Tidak ada `setAccessible` di kode aplikasi** yang ditemukan; akses refleksi mengandalkan method yang memang dapat dijangkau.

### 5.2 Akses file & data

- **Baca file by-name lewat native:** `am.r(String, byte[])` adalah jalur baca utama; `a/m.java` memakainya untuk daftar nama file terdekripsi dalam identifikasi perangkat (`m.java:4871-4903, 5612+, 6249+, 6690+, 6729+`), ditambah 4 probe `File.exists()` berpath terdekripsi. Path konkret tidak dapat dicantumkan karena terenkripsi — dari konteks menunya, keluarga file yang ditangani adalah file-file **EFS modem** (menu "EFS Misc. Files", "Clear RMTEFS", "ICC Misc. Files") dan file identitas perangkat.
- **Satu-satunya tulis file Java:** `Files.write(Paths.get(<prefix terdekripsi> + nama))` di `m.java:3636`.
- **SharedPreferences `"g"`** adalah satu-satunya penyimpanan persisten Java: kunci `"x"` (string terobfuscate XOR bergulir, dibaca di onCreate), `"t"`, `"y"`, `"z"` (ditulis di onSaveInstanceState). Tidak ada database, tidak ada akses `/sdcard` dari Java (grep path literal `/…`: 0 hit di seluruh source).
- **Data yang keluar perangkat:** hanya lewat tindakan eksplisit pengguna — share sheet `ba` (Bagian 1.5) dan salin-ke-clipboard `am`. Isi yang dibagikan adalah laporan kapabilitas modem perangkat itu sendiri.
- **`SystemProperties.get`** dibaca sekali di `em.java:300` (kunci terenkripsi) untuk menentukan varian hint.

### 5.3 Jaringan

- **Tidak ada penggunaan jaringan sama sekali di sisi Java** — grep menyeluruh atas 312 kelas untuk `java.net`, `HttpURLConnection`, `Socket`, `URL(`: **0 hit**. Ditambah manifest tanpa permission `INTERNET`, aplikasi secara desain tidak dapat berkomunikasi lewat jaringan dari lapisan Android. (Yang dilakukan native `liba.so` terhadap soket lokal/modem berada di luar cakupan laporan Java ini.)

### 5.4 Obfuscation sebagai catatan keamanan

- **Enkripsi string menyeluruh** (blob 60.750 char di `a/c.java`, ±1.400+ call site dekripsi di `am` saja) menyembunyikan: nama file/path, nama service sistem, nama kelas/method refleksi, nama proses shell, token fitur, dan seluruh label semantik tabel band. Konsekuensi audit: permukaan perilaku aplikasi **tidak dapat diaudit penuh dari Java saja** — penilaian final atas path, perintah shell, dan target hidden API memerlukan dekripsi string atau analisis `liba.so`.
- Tidak ditemukan telemetri/analytics/crash-reporting SDK di daftar kelas; kelas non-aplikasi yang teridentifikasi semuanya kotlinx.coroutines + AndroidX ViewPager.
- Komponen terekspos: hanya `a.am` yang `exported=true` (launcher). Aktivitas lain tidak diekspor; `ba` menerima konten lewat extra `"!"` dari dalam aplikasi sendiri (registry hash-nya pun dihapus setelah sekali pakai).

---

## 6. Ringkasan alur ujung-ke-ujung

1. Pengguna membuka aplikasi → `am.onCreate` memulihkan state (Bundle/prefs) atau, saat pertama kali, meluncurkan Thread + coroutine startup.
2. Coroutine `a/l` memuat `liba.so`; bila gagal, pesan error terdekripsi ditampilkan sebagai teks body.
3. Coroutine startup membaca identitas & kapabilitas modem lewat fasad `a.m` → native `am` (`t` init sesi, `o` frame perintah, `m` ByteBuffer, `r` baca file, `p/u` baca string), mengisi statics `am`; dekoder `a.m` mengubah buffer mentah menjadi tabel band/fitur dan teks laporan yang dirender `m45a()` ke TextView body.
4. Menu utama menawarkan: editor band (`em`), halaman kapabilitas (`ca`), editor searcher (`sc`), RAT priority (`pc`), laporan-laporan (`ba`), IMS, debug, opsi blitz/kacke, restart MPSS.
5. Editor menerima salinan state lewat Intent (serializer `a.d`), mengedit tanpa menyentuh modem, dan mengembalikan hasil lewat result Intent saat Back (result khusus: em=21 reset, sc=12 reset, bd=0 batal).
6. `am.onActivityResult` menyerahkan hasil ke coroutine `a.av`, yang menerapkan perubahan ke statics dan — lewat method sesi/native — menuliskannya ke modem (termasuk power-cycle MPSS bila jenis perubahan mengharuskannya, seperti diperingatkan label di `sc.xml`).
7. Laporan apa pun dapat disalin ke clipboard, dibagikan lewat share sheet (`ba`), atau — untuk satu jalur dialog — diteruskan ke shell.

*Akhir laporan. Cakupan: lapisan Java/Kotlin hasil dekompilasi JADX saja; perilaku `liba.so` (JNI) dan dekripsi string aktual berada di luar cakupan sesuai batasan tugas.*
