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

## 3. Fitur A — Editor Kapabilitas RRC (Gap 1)

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

## 4. Fitur B — Searcher Preferences (Gap 2)

Editor daftar per RAT (GSM/UMTS/LTE/NR), masing-masing: cell blacklist, channel blacklist, channel code whitelist, priority channel list (maks 6 entri; UMTS 1).

**File EFS:** `lte/rrc/efs/cell_selection_opt_params`, `lte/rrc/efs/cell_restrict_opt_params`, `lte/rrc/csp/cell_lock_list` (+ salinan `reg_files/…/csp/cell_lock_list`).

**Format pengepakan (persis dari dekripsi — ini yang membuat fitur ini layak jalan duluan):** `long[30]` per halaman; LTE cell = `(PCI<<32)|CID`; GSM cell = `(f1<<48)|(f2<<40)|(f3<<16)|f4`; NR membawa step SCS (`15 << nibble`); `-1` = terminator. Legenda: Channel = ARFCN|UARFCN|EARFCN|NR-ARFCN; Code = BSIC|PSC|PCI; Area = LAC|TAC; Cell = CID.

- **Backend:** `internal/efs/searcher.go` — pack/unpack murni (tanpa I/O) supaya bisa di-unit-test tanpa perangkat; pola pengepakan diuji terhadap contoh bit persis dari analisa.
- **Handler/route:** `GET/POST /api/efs/searcher` di `internal/handlers/searcher_handler.go`.
- **Frontend:** `QualcommSearcher.svelte` — 4 seksi RAT, 8 kolom, peringatan statis yang sama seperti aslinya: perubahan memicu modem power cycle; mengisi prioritas UMTS menggeser preferensi RAT ke UMTS-only.

## 5. Fitur C — Kunci ekstra di luar LTE (Gap 3)

Satu pola dengan cell lock yang sudah ada, file-file:

- GERAN: `feature_lock_arfcn_enabled`, `rr_efs_arfcn_list`
- WCDMA: `wcdma_rrc_freq_lock_item`
- `fake_bts_cell_barring_enabled`
- LTE CSP langsung: `lte/rrc/csp/pci_lock`, `lte/rrc/csp/earfcn_lock` (dibandingkan dulu dengan mekanisme cell lock QRTR yang sekarang — kalau duplikat fungsi, file CSP ini cukup dicatat, tidak diimplementasikan dua kali)

Backend di `internal/efs/searcher.go` atau file `locks.go` terpisah; UI menyatu di `QualcommSearcher.svelte` sebagai seksi "Kunci tambahan".

## 6. Fitur D — Switch Profil MBN / PDC (Gap 5)

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

## 7. Di luar cakupan

- **Laporan CA combos (Gap 4)** — butuh query kapabilitas QMI (protokol baru di `QRTRClient`); riset terpisah, bukan bagian spek ini.
- Perubahan apa pun di `internal/shannon`, `internal/samsung`, route `/api/shannon/*` dan `/api/samsung/*`, dan tab `shannon/` + `samsung/`.
- Flavor `nomodem` tidak berubah perilaku: tidak ada route/handler/komponen baru yang bocor ke build `nomodem` (build tag `!nomodem` + stub + `vendor_tabs.ts` dijaga).

## 8. Urutan pengerjaan

1. Fase 0 — verifikasi format file di perangkat (Fitur A).
2. Fitur B + C — format datanya paling pasti; pack/unpack + unit test dulu, baru I/O + UI.
3. Fitur A LTE — tulis per file yang formatnya sudah terkunci di Fase 0.
4. Fitur A NR — 4 file `nr5g/RRC/cap_*`.
5. Fitur D — fase riset R1–R3 bisa berjalan sejajar dengan langkah 3–4 (statis dulu); implementasi hanya sesudah mekanismenya terkunci.
6. Uji end-to-end di perangkat: tulis → SSR → baca balik → bandingkan dengan ModemCaps di perangkat yang sama.

## 9. Risiko & mitigasi

- **Format file salah tebak** → mitigasi Fase 0 + backup wajib sebelum tulis pertama per file; restore = tulis balik backup.
- **File tidak ada di perangkat tertentu** → `ReadCapability` toleran; UI menandai "tidak didukung perangkat ini" (mengikuti penjelasan footer ModemCaps: hardware/supplemental connectivity/airplane mode).
- **SSR mengganggu koneksi WebUI** → SSR selalu opsional dan eksplisit, tidak pernah otomatis.
