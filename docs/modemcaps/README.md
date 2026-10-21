# Dokumentasi ModemCaps — Hasil Bongkar APK `ru.fieldtest.modemcaps` v2402-1

Folder ini berisi **seluruh hasil analisa reverse-engineering** aplikasi **Qct Modem Capabilities** (`modemcaps-2402-1d_0_it0k.apk`, 529 KB, SHA-256 `cf20ddb3…596f7a`), dikerjakan 20–21 Oktober 2026 dengan analisis statis penuh: REA + JADX + disassembly smali (androguard) + Ghidra headless, **tanpa menjalankan aplikasi di perangkat**.

Dokumentasi ini adalah bahan mentah di balik spek implementasi [`../modemcaps-qualcomm-spec.md`](../modemcaps-qualcomm-spec.md): setiap path EFS, token fitur, dan format data yang dirujuk spek itu berasal dari sini.

## Isi folder

| File | Isi |
|---|---|
| `analisis-utama.md` | Dokumentasi utama: identitas paket, arsitektur, alur ujung-ke-ujung, cara kerja tiap fitur (editor band, halaman kapabilitas, searcher, prioritas RAT, penampil laporan), lapisan native, mekanisme enkripsi, keamanan/privasi, batasan analisis |
| `dekripsi-string.md` | Hasil dekripsi massal string: mekanisme SplitMix64 + mixer 16-bit autokey, 3.092 string plaintext pulih, 21 path EFS persis, 38 token fitur, payload `supolicy --live`, jalur shell `su` terkonfirmasi |
| `pendalaman-native-shell.md` | Putaran kedua: 20 class yang gagal di-decompile JADX berhasil didisassembly penuh, lapisan obfuscation kedua di `a.z`, arsitektur proses shell/native, bedah per-fungsi `liba.so`/`libb.so`/`libc.so` (Ghidra) |
| `kamus-string.tsv` | Kamus lengkap string hasil dekripsi (3.103 konstanta unik → 3.092 plaintext): kolom ID konstanta + string. Bahan rujukan saat menelusuri perilaku yang belum terpetakan |
| `referensi-efs-token.md` | Lembar rujukan cepat buat implementor: daftar path EFS, token kapabilitas, format pengepakan searcher, tata bahasa filter band-combo, path MCFG/PDC |
| `peta-ke-bfr.md` | Pemetaan fitur ModemCaps ↔ status di BFR (sudah ada / Gap 1–10 / fitur spek A–F / bukan gap) |

## Fakta kunci (satu layar)

- Editor kapabilitas modem **Qualcomm** di level file EFS/NV: band per RAT, CA combos & fitur LTE/NSA/NR, Rx chain/diversity, searcher preferences, prioritas RAT, profil PDC/MBN, status IMS — plus restart MPSS.
- **Nol permission Android**; kemampuan istimewa dari native (`liba.so`, 17 method JNI + mode executable worker QRTR/QMI) dan shell `su` persisten dengan tangga pelonggaran SELinux (`setenforce 0` → `supolicy --live` terarah → fallback allow-all).
- File pertukaran Java↔native di-**staging di `/data/tmp/*.bin`** (terkompresi zlib): konten disusun utuh dulu, baru ditimpakan ke EFS — pola yang menjadi acuan Fondasi EFS Write di spek.
- Obfuscation total: 312 class bernama 1–2 huruf + enkripsi string penuh (blob 60.635 karakter) — sudah terbuka 100% lewat dekripsi; tidak ada telemetri/server; satu-satunya URL adalah grup Telegram pengembang.

## Yang masih belum terverifikasi (jujur)

Perilaku runtime tidak pernah diuji di perangkat; urutan persis frame perintah modem disimpulkan dari pola pemakaian + pseudocode; jalur aktivasi PDC/MBN dan isi Data Preferences belum tertelusur ujung-ke-ujung. Rincian di Bagian 9 `analisis-utama.md` dan Seksi 10 spek (lampiran riset).
