# Spek Implementasi — Fitur ModemCaps untuk Jalur Qualcomm

**Branch:** `dev-muse` (basis `dev @4fe13824`) · **Status:** spek, belum ada kode · **Tanggal:** 21 Oktober 2026
**Sumber data:** analisa penuh ModemCaps v2402-1 (dokumentasi + dekripsi 3.092 string + pendalaman native) dan peta gap `ModemCaps-ke-BFR-Peta-Gap.md`.

---

## 1. Batas tegas — tiga stack, jangan dicampur

Repo ini punya tiga jalur modem yang **terpisah total** dan spek ini hanya menyentuh jalur pertama:

| Stack | Paket | Route | Transport | Status di spek ini |
|---|---|---|---|---|
| Generic Qualcomm | `internal/modem`, `internal/efs` | `/api/modem/*`, `/api/qualcomm/*`, `/api/efs/*` | QMI/QRTR + DIAG/EFS2 | **SATU-SATUNYA yang diubah** |
| Samsung OneUI | `internal/samsung` | `/api/samsung/*` | jalur OneUI sendiri | TIDAK disentuh |
| Pixel Tensor | `internal/shannon` | `/api/shannon/*` | protobuf Shannon (CA/NR doc) | TIDAK disentuh — CA editor Tensor sudah ada dari trah Shannon-lte-editor |

ModemCaps adalah aplikasi Qualcomm murni (EFS modem Qualcomm). Tidak ada satu pun konsepnya yang dipindahkan ke Shannon/Samsung: format file, token, dan pengepakan datanya spesifik EFS Qualcomm.

## 2. Yang sudah tersedia (dipakai ulang, bukan dibangun)

- Sesi EFS2/DIAG: `internal/efs` `Manager` — connect, baca/tulis file EFS (whitelist prefix `/nv/item_files/`), mode READ-ONLY yang memblokir penulisan & SSR.
- Backup/restore NV: pola `nv/backup` + `nv/restore` — pola yang sama dipakai untuk backup file kapabilitas sebelum tulis.
- SSR: `Manager.RestartModem()` + endpoint `/api/efs/ssr` — dipakai sebagai langkah opsional sesudah tulis (pola sama seperti alur IMEI `RebootModem`).
- UI: `TabQualcomm.svelte` + komponen `frontend/src/components/tabs/qualcomm/` (EFS, NV, IMEI, Bands, CellLock, CellScanner, Profiles) — fitur baru masuk sebagai sub-tab di sini, mengikuti pola modal & state yang ada.

## 3. Fondasi — Perbaikan EFS Write (prasyarat semua fitur)

Semua fitur di spek ini menulis file EFS, dan jalur tulis yang ada sekarang (`EfsSession.WriteFile`, `internal/efs/efs2_linux.go:496`, lewat `POST /api/efs/write`) punya kelemahan terverifikasi dari pembacaan kode. Pola acuan dari QPST dan ModemCaps: **susun file utuh di luar modem → timpa sekali jadi → verifikasi → restart modem supaya file dikonsumsi**. BFR sudah menimpa penuh (`O_TRUNC` di kedua jalur; item ≤6.144 byte malah satu PUT atomik), tapi pendampingnya belum ada. Yang diperbaiki di fondasi ini:

1. **Klasifikasi item file jangan percaya frontend.** Sekarang `QualcommEFS.svelte` mengirim `force_item: 1` hanya bila path mengandung `/nv/item_files/`; path item sah lain (`/nv/reg_files/…`, `/cgps/nv/item_files/…` — keduanya dikenali `IsItemPath` di server) terkirim `0`, dan di `WriteFile` nilai `0` *memaksa* mode reguler tanpa `O_ITEMFILE`. Perbaikan: default server = auto (`-1`, putuskan dari `IsItemPath`); field JSON yang tidak dikirim tidak boleh terbaca sebagai `0` (pakai pointer/sentinel); frontend cukup mengirim auto kecuali user menimpa manual.
2. **Backup otomatis sebelum timpa.** Setiap tulis ke path yang sudah ada wajib menyimpan salinan lama lebih dulu (pola backup NV yang sudah ada di repo), lengkap dengan path + timestamp, dan tersedia endpoint/UI restore per file. File yang sudah ter-TRUNC tidak boleh tertinggal setengah jadi tanpa jalan pulang.
3. **Verifikasi baca-balik.** Sesudah tulis: baca ulang file dari modem, bandingkan byte/hash dengan yang dikirim; mismatch = error eksplisit ke UI, bukan sukses semu. Error pada CLOSE tidak boleh lagi diabaikan, dan urutan yang benar adalah CLOSE → sync (sekarang sync dipanggil sebelum close selesai). Kegagalan jalur PUT harus tercatat di hasil, bukan jatuh diam-diam ke jalur OPEN/WRITE.
4. **Opsi "tulis + SSR" satu aksi.** Flag opsional pada request tulis (pola `RebootModem` di alur IMEI): sesudah verifikasi lolos, jalankan `RestartModem()`. Default mati, tidak pernah otomatis — tapi user tidak perlu lagi berburu tombol SSR terpisah supaya editnya "berpengaruh".
5. **Catatan item file besar.** Item file >6.144 byte jatuh ke jalur OPEN/WRITE ber-chunk; sebagian firmware menolak tulis parsial pada item file. Perilaku per perangkat dicatat di Fase 0 Fitur A; bila terbukti menolak, strategi alternatifnya dirumuskan di sana (bukan ditebak sekarang).

**Kriteria selesai fondasi:** unit test klasifikasi path (item vs reguler, termasuk `reg_files`/`cgps`); di perangkat uji: tulis → baca-balik identik; tulis sengaja digagalkan di tengah → backup bisa di-restore dan file kembali utuh; tulis + SSR membuat perubahan yang tadinya "tidak berpengaruh" terlihat efeknya.

## 4. Fitur A — Editor Kapabilitas RRC (Gap 1)

Halaman switch setara halaman kapabilitas ModemCaps: 38 token dalam 3 seksi.

**LTE:** LTE+LTE aggregation `[LTE_CA]`, UL aggregation `[LTE_UL_CA]`, UL 256-QAM `[LTE_UL_256Q]`, DL 256-QAM/1K-QAM `[LTE_DL_256Q]`, DL MIMO `[LTE_DL_MIMO]`, filter band-combo `[LTE_CA_FILTER]` (+ teks combo format `Band+Class+[Streams][Uplink Class]-…[BW Class Set]`, Class A|C|D|E|F, Streams 2|4, BW Set 0|1|2|4|m3|m7|mAll), VoLTE `[LTE_VOICE]`, Rx chain #0–#3 `[LTE_RX0..3]`, diversity D#0 (PRx), D#1–D#3 (DRx), D#6 (ARD), adaptive Rx diversity `[LTE_HORXD]`.
**NSA:** LTE+NSA `[NSA_LTE]`, NSA DSDA `[NSA_LTE_DSDA]`, NSA saat data-call SIM lain `[NSA_NDDS]`, DSS `[NSA_LTE_DSS]`.
**NR:** NR CA `[NR_CA]`, UL/DL 256-QAM `[NR_UL_256Q]`/`[NR_DL_256Q]`, UL/DL MIMO, VoNR `[NR_VOICE]`, RedCap `[NR_REDCAP]`, NR DSDA `[NR_NR_DSDA]`, Rx chain, ARD.

**Pemetaan token → file EFS** (path persis hasil dekripsi; relatif terhadap `/nv/item_files/modem/`):

| File | Isi |
|---|---|
| `lte/rrc/efs/lte_feature_enable`, `lte/rrc/efs/lte_feature_disable` | set fitur LTE aktif/nonaktif |
| `lte/common/ca_disable` | penonaktifan CA |
| `lte/ML1/ard_config` | adaptive receive diversity |
| `lte/ML1/rx_select` | pemilihan Rx chain |
| `mcs/trm/ant_div_switch_master` | master switch antena diversity |
| `lte/rrc/efs/max_num_mimo_lyr` | batas layer MIMO |
| `nr5g/RRC/cap_control_dl_256qam` | kontrol DL 256-QAM NR |
| `nr5g/RRC/cap_disable_ul_256qam` | penonaktifan UL 256-QAM NR |
| `nr5g/RRC/cap_limit_rf_mimo` | batas RF MIMO NR |
| `nr5g/RRC/cap_nrca_downgrade_1cc` | downgrade NR CA ke 1CC |

**Sifat write-only (dari footer UI ModemCaps, wajib ditampilkan di UI):** LTE+LTE aggregation, UL aggregation, DL 256-QAM, Rx chain, ARD tidak bisa dibaca balik — statusnya ditampilkan sebagai "tidak terbaca", bukan ditebak.

### Desain

- **Backend:** file baru `internal/efs/capability.go` + `capability_types.go` (di atas sesi `Manager` yang ada; TIDAK di `internal/modem` — itu wilayah QMI/QRTR). Mengikuti aturan isolasi dual-flavor repo: semua file baru yang menyentuh baseband proprietary wajib ber-tag `//go:build !nomodem`, route hanya didaftarkan di `routes_vendor_modem.go`, dan komponen frontend vendor hanya lewat `vendor_tabs.ts` dengan gate `HAS_VENDOR_MODEM` — supaya flavor `nomodem` tetap nol kode vendor. Satu struct `CapabilityState` per seksi; fungsi `ReadCapability()` (baca semua file yang readable, toleran file tidak ada = fitur tidak didukung) dan `ApplyCapability(state, opts{Backup bool, SSR bool})` (backup dulu → tulis per file → SSR opsional; berhenti di kegagalan pertama dan laporkan file mana).
- **Handler:** `internal/handlers/capability_handler.go`, route baru di seksi "Qualcomm Snapdragon EFS2 & DIAG" `routes_vendor_modem.go`:
  - `GET /api/efs/capability` — state saat ini + daftar file yang ada/tidak ada per perangkat
  - `POST /api/efs/capability/apply` — tulis dari state yang dikirim UI
  - Menghormati mode READ-ONLY EFS persis seperti endpoint tulis lain.
- **Frontend:** `frontend/src/components/tabs/qualcomm/QualcommCapability.svelte`, sub-tab baru di `TabQualcomm.svelte`. *(Keputusan terkunci 21 Okt 2026: sub-tab di dalam TabQualcomm, bukan tab vendor baru.)* Switch per token, kolom teks filter combo dengan validasi format, penanda write-only, tombol Apply dengan konfirmasi + opsi SSR.

### Fase 0 (WAJIB sebelum kode tulis)

Layout byte persis tiap file di atas adalah format sisi modem yang disimpulkan dari pola pemakaian ModemCaps, bukan spec resmi. Sebelum `ApplyCapability` ditulis:
1. Dump isi file-file ini dari perangkat Qualcomm uji lewat `/api/efs/read` yang sudah ada.
2. Bandingkan dengan keadaan fitur yang diketahui (mis. perangkat dengan RedCap aktif vs tidak).
3. Kunci format per file di dokumen ini (endianness, satuan, nilai enable/disable) — satu file, satu baris kesimpulan + hex dump contoh.

## 5. Fitur B — Searcher Preferences (Gap 2)

Editor daftar per RAT (GSM/UMTS/LTE/NR), masing-masing: cell blacklist, channel blacklist, channel code whitelist, priority channel list (maks 6 entri; UMTS 1).

**File EFS:** `lte/rrc/efs/cell_selection_opt_params`, `lte/rrc/efs/cell_restrict_opt_params`, `lte/rrc/csp/cell_lock_list` (+ salinan `reg_files/…/csp/cell_lock_list`).

**Format pengepakan (persis dari dekripsi — ini yang membuat fitur ini layak jalan duluan):** `long[30]` per halaman; LTE cell = `(PCI<<32)|CID`; GSM cell = `(f1<<48)|(f2<<40)|(f3<<16)|f4`; NR membawa step SCS (`15 << nibble`); `-1` = terminator. Legenda: Channel = ARFCN|UARFCN|EARFCN|NR-ARFCN; Code = BSIC|PSC|PCI; Area = LAC|TAC; Cell = CID.

- **Backend:** `internal/efs/searcher.go` — pack/unpack murni (tanpa I/O) supaya bisa di-unit-test tanpa perangkat; pola pengepakan diuji terhadap contoh bit persis dari analisa.
- **Handler/route:** `GET/POST /api/efs/searcher` di `internal/handlers/searcher_handler.go`.
- **Frontend:** `QualcommSearcher.svelte` — 4 seksi RAT, 8 kolom, peringatan statis yang sama seperti aslinya: perubahan memicu modem power cycle; mengisi prioritas UMTS menggeser preferensi RAT ke UMTS-only.

## 6. Fitur C — Kunci ekstra di luar LTE (Gap 3)

Satu pola dengan cell lock yang sudah ada, file-file:

- GERAN: `feature_lock_arfcn_enabled`, `rr_efs_arfcn_list`
- WCDMA: `wcdma_rrc_freq_lock_item`
- `fake_bts_cell_barring_enabled`
- LTE CSP langsung: `lte/rrc/csp/pci_lock`, `lte/rrc/csp/earfcn_lock` (dibandingkan dulu dengan mekanisme cell lock QRTR yang sekarang — kalau duplikat fungsi, file CSP ini cukup dicatat, tidak diimplementasikan dua kali)

Backend di `internal/efs/searcher.go` atau file `locks.go` terpisah; UI menyatu di `QualcommSearcher.svelte` sebagai seksi "Kunci tambahan".

## 7. Fitur D — Switch Profil MBN / PDC (Gap 5)

Padanan halaman **PDC Preferences** ModemCaps — fitur yang di BFR belum ada sama sekali (tidak ada `mbn`/`mcfg`/`pdc` di backend maupun frontend; `QualcommProfiles` yang ada adalah profil band lock, bukan MBN). Permukaan fitur dari kamus string hasil dekripsi:

- Dua daftar konfigurasi MBN yang terpasang di modem: **SW configs** (`_pdc_sw_listview`) dan **HW configs** (`_pdc_hw_listview`), direktori modem `/mcfg_sw/generic` dan `/mcfg_hw/generic`
- Profil PDC aktif saat ini ("PDC profile", "PDC configs: …") dan pergantiannya
- Toggle **autoselect on SIM change**: file EFS `/nv/item_files/mcfg/mcfg_autoselect_by_uim` + varian `mcfg_autoselect_by_uim_Subscription01`
- Mode pembaruan "with MBN update only"

**Yang belum diketahui (jujur):** jalur aktivasi persisnya di ModemCaps belum tertelusur ujung-ke-ujung — fitur ini tertangkap dari dekripsi string, bukan bedah alur. Di ekosistem Qualcomm, pengelolaan konfigurasi MBN umumnya berjalan lewat layanan QMI MCFG/PDC (list/activate/deactivate) dan/atau file EFS `mcfg`, dan ModemCaps kemungkinan menggerakkannya lewat kanal frame native-nya. Karena itu Fitur D punya fase riset sendiri:

- **Fase R1 (statis):** telusuri pemakaian layar PDC di smali `a.z`/`a.an` dan laporan alur Java — frame apa yang dikirim saat profil diganti.
- **Fase R2 (perangkat):** di perangkat uji, ganti profil lewat ModemCaps lalu diff isi EFS sebelum/sesudah memakai EFS browser BFR yang sudah ada; catat file & byte yang berubah.
- **Fase R3 (protokol):** kalau jalurnya QMI, petakan service/message MCFG-PDC dari referensi publik dan cocokkan dengan `QRTRClient` yang ada.

### Desain (sesudah fase riset mengunci mekanisme)

- **Backend:** pembacaan daftar + aktivasi di sisi QMI masuk `internal/modem` (file baru `qualcomm_pdc.go`, sejajar `qualcomm_qrtr.go` — wilayah QMI/QRTR); toggle autoselect adalah file EFS sehingga ditulis lewat sesi `internal/efs`. File baru mana pun yang menyentuh baseband wajib `//go:build !nomodem`.
- **Handler:** `internal/handlers/mbn_handler.go`; route di seksi Qualcomm `routes_vendor_modem.go`:
  - `GET /api/qualcomm/mbn` — daftar SW + HW configs, profil aktif, status autoselect
  - `POST /api/qualcomm/mbn/activate` — ganti profil aktif (selalu catat profil sebelumnya; SSR sesudahnya opsional-eksplisit seperti pola IMEI)
  - `POST /api/qualcomm/mbn/autoselect` — toggle autoselect by UIM per subscription
- **Frontend:** `frontend/src/components/tabs/qualcomm/QualcommMBN.svelte`, sub-tab di `TabQualcomm.svelte` (keputusan terkunci yang sama dengan Fitur A/B).
- **Pengaman:** aktivasi profil yang salah bisa membuat modem gagal registrasi sampai dikembalikan — UI wajib menampilkan profil aktif sebelumnya dan menyediakan jalan pulang satu klik; tidak ada aktivasi tanpa konfirmasi.

## 8. Fitur E — RAT Priority & RAT/Band Preferences (Gap 6 + 7)

Dua editor per slot dari menu utama ModemCaps yang belum ada padanannya di BFR:

- **RAT Priority** (`menu_rat_prio` → editor `pc`): daftar prioritas RAT *berurutan* per slot — kandidat diberi penanda status dari perbandingan bit dengan pilihan saat ini. BFR hanya punya enum kasar `PreferredRAT` (`5g_only`/`4g_only`/`hybrid`/`3g_only`) di `BandConfig`; itu bukan urutan akuisisi.
- **RAT/Band Preferences** (`menu_rbp_config`): matriks preferensi band per slot untuk semua RAT, dengan reset tersendiri ("🗑 RAT/Band preferences"). Berbeda sifat dari band lock QMI BFR (mengunci) — ini preferensi yang disimpan sebagai matriks `byte[][]` per slot di state ModemCaps.

**Titik keputusan desain (wajib diputus sebelum implementasi):** kalau dua-duanya hidup, mana sumber kebenaran — matriks preferensi atau `BandConfig` band lock? Rekomendasi spek: dataset terpisah, UI memberi peringatan bila band lock sedang aktif saat preferensi diubah, dan keduanya tidak menulis sasaran yang sama tanpa sepengetahuan user.

**Yang belum terkunci:** jalur tulis persis kedua editor ini (alur perintah generik lewat sesi frame di ModemCaps). Verifikasinya mengikuti pola Fase 0/Fase R Fitur D: telusur smali pemakaian menunya + diff state EFS/NV di perangkat sebelum/sesudah. Implementasi tidak dimulai sebelum mekanismenya terbukti.

### Desain (sesudah mekanisme terkunci)

- **Backend:** mengikuti tempat mekanismenya terbukti tinggal — data matriks per slot kemungkinan besar berbasis EFS/NV sehingga di `internal/efs` (file baru `ratprefs.go`, tag `!nomodem`); bila terbukti QMI, pindah ke `internal/modem`.
- **Handler/route:** `GET/POST /api/qualcomm/rat-priority` dan `GET/POST /api/qualcomm/band-preferences` di `routes_vendor_modem.go`, handler `internal/handlers/ratprefs_handler.go`.
- **Frontend:** `QualcommRatPriority.svelte` (dua daftar bertumpuk seperti aslinya: kandidat di atas, pilihan berurutan di bawah) + seksi matriks preferensi di komponen yang sama; sub-tab TabQualcomm.

## 9. Fitur F — Band GSM/UMTS & Status IMS (Gap 8 + 9)

- **Band mask GERAN/UTRAN (Gap 8).** Editor band ModemCaps mencakup grup GSM/UMTS; `QRTRClient` BFR hanya menyetel LTE/NR (`SetLTEBands`/`SetNRBands`) — band GSM/WCDMA di BFR sebatas info hardware. Di ModemCaps penyetelannya lewat data EFS/NV (bukan QMI band lock), jadi backend-nya di `internal/efs` dan nama file/datanya ditambahkan ke daftar verifikasi Fase 0. UI: perluasan `QualcommBands.svelte` dengan seksi GSM/WCDMA (bukan komponen baru).
- **IMS Service Status (Gap 9).** Penampil status layanan IMS ModemCaps (grup menu `menu_ims`; laporan utamanya juga memuat baris Voice over LTE / Voice over Wi-Fi). Di BFR belum ada sama sekali (nol `ims` di `internal/modem`, `internal/efs`, handlers). Sifatnya read-only. Mekanisme query statusnya (kanal frame/DIAG) ditelusuri bersama riset Fitur E. Desain: `GET /api/qualcomm/ims/status`, panel kecil `QualcommIMS.svelte` sebagai sub-tab TabQualcomm — tanpa aksi tulis apa pun di versi pertama.

## 10. Lampiran riset — belum komitmen implementasi

Dua temuan yang keberadaannya pasti tapi isinya belum terpetakan; dicatat supaya tidak hilang, dikerjakan sebagai riset statis (tidak butuh perangkat) kapan saja:

- **Data Preferences (Gap 10).** Menu `menu_dfp_config` + string "Data Preferences" ada di ModemCaps, tapi tidak ada editor khusus — alur perintah generik. Target riset: telusuri dispatch posisional menunya di smali `am`/`a.z` sampai ketemu data apa yang dibaca/ditulis; sesudah itu baru dinilai layak jadi fitur atau tidak.
- **Keluarga laporan HW (perluasan Gap 4).** Selain CA combos (LTE CA Combos, RRC CA Combos — sudah di luar cakupan), ModemCaps punya **ICC Misc Files**, **EFS Misc Files**, **RFC Report** (info kalibrasi RF), dan **RFFE Components**. Target riset: petakan sumber data tiap laporan — yang berbasis baca file EFS/ICC berpotensi ringan (BFR sudah punya EFS browser + kanal baca by-name), yang berbasis query protokol tetap berat dan mengikuti nasib Gap 4.

## 11. Di luar cakupan

- **Laporan CA combos (Gap 4)** — butuh query kapabilitas QMI (protokol baru di `QRTRClient`); riset terpisah, bukan bagian spek ini.
- Perubahan apa pun di `internal/shannon`, `internal/samsung`, route `/api/shannon/*` dan `/api/samsung/*`, dan tab `shannon/` + `samsung/`.
- Flavor `nomodem` tidak berubah perilaku: tidak ada route/handler/komponen baru yang bocor ke build `nomodem` (build tag `!nomodem` + stub + `vendor_tabs.ts` dijaga).

## 12. Urutan pengerjaan

0. Fondasi EFS Write (Seksi 3) — kode + unit test bisa jalan tanpa perangkat; verifikasi baca-balik & restore diuji begitu ada perangkat.
1. Fase 0 — verifikasi format file di perangkat (Fitur A).
2. Fitur B + C — format datanya paling pasti; pack/unpack + unit test dulu, baru I/O + UI.
3. Fitur A LTE — tulis per file yang formatnya sudah terkunci di Fase 0.
4. Fitur A NR — 4 file `nr5g/RRC/cap_*`.
5. Fitur D — fase riset R1–R3 bisa berjalan sejajar dengan langkah 3–4 (statis dulu); implementasi hanya sesudah mekanismenya terkunci.
6. Fitur E — sesudah verifikasi mekanismenya (pola Fase 0/R); titik keputusan sumber-kebenaran vs band lock diputus user dulu.
7. Fitur F — band GSM/UMTS mengikuti daftar verifikasi Fase 0; status IMS read-only bisa menyusul kapan saja sesudah query-nya terbukti.
8. Lampiran riset (Data Preferences, laporan HW) — pekerjaan statis, boleh dicicil kapan saja tanpa perangkat; hasilnya menentukan apakah naik jadi fitur.
9. Uji end-to-end di perangkat: tulis → SSR → baca balik → bandingkan dengan ModemCaps di perangkat yang sama.

## 13. Risiko & mitigasi

- **Format file salah tebak** → mitigasi Fase 0 + backup wajib sebelum tulis pertama per file; restore = tulis balik backup.
- **File tidak ada di perangkat tertentu** → `ReadCapability` toleran; UI menandai "tidak didukung perangkat ini" (mengikuti penjelasan footer ModemCaps: hardware/supplemental connectivity/airplane mode).
- **SSR mengganggu koneksi WebUI** → SSR selalu opsional dan eksplisit, tidak pernah otomatis.
