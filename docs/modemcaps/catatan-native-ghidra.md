# Catatan bedah native putaran lanjutan (Ghidra 12.1.4 headless, JDK Temurin 21)

Pseudocode penuh: `ghidra-out/liba-all.c` (119 fungsi), `libb-all.c` (208), `libc-all.c` (188).
JDK sempat hilang dari VM (reset) → diganti tarball Temurin 21.0.12 di ~/tools/jdk-21.0.12.1+1 (apt mirror macet total).

## liba.so — DUA peran, bukan satu

### A. Library JNI (peran lama)
- JNI_OnLoad @00104a00: baca `ro.build.version.sdk`; pilih tabel RegisterNatives SDK>25 vs ≤25 (17 method `a.am`). Nama method di tabel sebagian berupa data 1 char (DAT_0010173e dsb) — cocok dengan nama 1 huruf di Java.
- FUN_00104f00 (6.168 byte) = handler sesi/frame utama: tulis frame ke fd, baca balasan pakai helper baca-timeout FUN_00101df0(fd, buf, 0x100, timeout 200/60), parse field balasan di offset tetap, salin string keluar. Di fungsi yang sama ada jalur file lokal: open/read + `compress2(...,9)` di jalur tulis dan `uncompress` di jalur baca (FUN_001069fc, FUN_00106ecc, FUN_00107358 juga uncompress) — data file pertukaran `/data/tmp/*.bin` dikompresi zlib.
- Import socket lengkap dipakai juga di sisi JNI (socket() di FUN lain, line ±751 dump).

### B. Executable mandiri (BARU — `entry` @00103830, 1.480 byte)
liba.so punya entry point program: dijalankan sebagai proses anak (via linker), bicara protokol biner lewat stdin/stdout:
1. Baca 12 byte dari stdin = (domain, type, protocol) → `socket(domain,type,protocol)` → tulis balik nomor fd (4 byte) ke stdout.
2. Baca 8 byte flag. Bila flag1≠0 dan keluarga soket cocok ((6-domain)&7==3): bangun struct 0x54 byte berisi blob nama bawaan + field 0xbc3, panggil `ioctl(fd, 0x8010C305, struct)`; bila gagal, coba nilai 1, lalu loop 8 nilai dari tabel DAT_001013f4, lalu pasangan 0x100000001 — pola negosiasi/registrasi layanan.
3. Loop permintaan (sebanyak flag2): baca header 16 byte; dua jalur:
   - Jalur ioctl: `ioctl(fd, 0xC014C302, struct 0x124)` — ambil info layanan; kasus khusus mengulang dengan field=1.
   - Jalur lookup nama: `getsockname` → kirim pesan kontrol 20 byte bertipe 10 (lookup) ke alamat soket sendiri (node/port diisi -2/-1/-1/-1), kumpulkan balasan bertipe 4 ke tabel (maks dari struct), akhiri dengan pesan tipe 11.
4. Hasil ditulis ke stdout: jumlah entri, lalu per entri 4 dword (dengan pengemasan ulang field versi/instansi: 0x10000/0x30000 + bit geser).

Tafsir: ini **worker lookup layanan QRTR/QMI** — tugasnya memetakan nama layanan QMI ke (node, port, versi, instansi) untuk proses induk. Selaras string Java `QmiInit(data=...)` dan path Java `/d/msm_ipc_router/dump_servers`. Proses induknya hampir pasti dirutekan lewat rantai ProcessBuilder di class `a.z` (lihat laporan class pulihan).

Implikasi arsitektur: pembagian kerja aplikasi =
- proses anak root (liba executable / helper) untuk soket QRTR + ioctl,
- JNI in-process untuk frame diag/EFS + file,
- Java untuk state/UI.

## libb.so — peran TERBUKTI di level kode
FUN_0001269c (930) = implementasi listing direktori gaya `ls`: opendir/readdir loop, printf format per entri via string PIC-relatif, penanganan error per path, flag output. Konsisten dengan utilitas file multi-applet (toybox-style) dari putaran pertama (import chmod/chown/stat). Bukti naik kelas: dari "string+import" menjadi pseudocode.

## libc.so — tool SELinux, bukti tambahan
- Tabel data di ±0x8ebc berisi pointer ke string `/sys/fs/selinux/policy` (0x818) — struktur tabel path kebijakan dipakai program.
- `/sys/fs/selinux/load` tidak direferensikan pointer langsung (kemungkinan dibangun/dioffset dari tabel yang sama).
- Dekompilasi ARM32 Ghidra jelek untuk binary ini (entry "bad instruction data", literal pool tak ter-resolve jadi string) — alur main persisnya belum tertelusur; peran sebagai tool kebijakan SELinux tetap berdiri dari string+import putaran pertama + tabel pointer ini.

## Yang masih terbuka sesudah putaran ini
- Runtime tetap belum diuji (tidak ada perangkat).
- Pemetaan persis 17 method JNI → alamat fungsi (tabel RegisterNatives terbaca sebagian; nama method 1-huruf menyulitkan korelasi nama↔fungsi, tapi tanda tangan + urutan tabel konsisten).
- Makna numerik persis tipe pesan kontrol 10/11/4 ditafsirkan dari pola (lookup/jawaban/akhir), bukan dari header kernel.
