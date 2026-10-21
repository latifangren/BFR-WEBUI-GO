# Peta Fitur ModemCaps ↔ BFR-WEBUI-GO

Status per fitur hasil menyilangkan dokumentasi ini dengan kode BFR `dev @4fe13824`. Rujukan implementasi: [`../modemcaps-qualcomm-spec.md`](../modemcaps-qualcomm-spec.md). Semua fitur ModemCaps bersifat Qualcomm — pemetaan hanya menyentuh jalur generic Qualcomm BFR (`internal/modem` + `internal/efs`), tidak pernah Samsung OneUI (`internal/samsung`) atau Pixel Tensor (`internal/shannon`).

| Fitur ModemCaps | Di BFR | Keterangan |
|---|---|---|
| Band lock LTE/NR | ✅ Sudah ada | QMI/QRTR `SetLTEBands`/`SetNRBands` + verifikasi; asal trah qcom-band-menu |
| Cell lock (PCI/EARFCN) persisten | ✅ Sudah ada | `cell_lock_config.go`, diterapkan ulang saat boot |
| Restart MPSS | ✅ Sudah ada | SSR via DIAG (`internal/efs`) |
| Browser EFS + editor NV generik | ✅ Sudah ada | EFS2/DIAG; tapi jalur tulisnya perlu fondasi (di bawah) |
| Info perangkat/SoC | ✅ Sudah ada | `internal/sysinfo` |
| Tangga supolicy/SELinux | ⛔ Bukan gap | BFR berjalan sebagai root (Magisk) |
| Staging file utuh → timpa → verifikasi | 🟡 **Fondasi spek** | BFR menimpa penuh tapi tanpa backup/verifikasi baca-balik/SSR terpasang; klasifikasi item file ada bug `force_item` |
| Halaman kapabilitas 38 token → 11 file EFS RRC | ❌ **Gap 1 → Fitur A** | Perlu Fase 0 verifikasi format byte |
| Searcher Preferences (blacklist/whitelist/priority) | ❌ **Gap 2 → Fitur B** | Format `long[30]` terdokumentasi persis |
| Kunci GERAN/WCDMA/fake-BTS + CSP locks | ❌ **Gap 3 → Fitur C** | Satu pola dengan cell lock |
| Laporan CA combos (LTE/RRC) | ❌ **Gap 4 → di luar cakupan spek** | Butuh query kapabilitas QMI (protokol baru) |
| Switch profil MBN / PDC | ❌ **Gap 5 → Fitur D** | Aktivasi belum tertelusur → fase riset R1–R3. Catatan koreksi 2026-10-21: label menu "PDC Preferences" ternyata membuka picker RAT Priority `pc` (dispatch +1) — pintu UI switch MBN/PDC bukan dari menu itu |
| RAT Priority (urutan akuisisi per slot) | ❌ **Gap 6 → Fitur E** | BFR hanya enum `PreferredRAT` kasar; picker `pc` dijangkau via label "PDC Preferences" — label "RAT Priority" sendiri membuka Debug Info |
| RAT/Band Preferences (matriks per slot) | ❌ **Gap 7 → Fitur E** | Label menunya membuka editor checklist band `em`; perlu keputusan sumber-kebenaran vs band lock |
| Band mask GSM/UMTS | ❌ **Gap 8 → Fitur F** | QRTR BFR hanya LTE/NR; jalurnya EFS/NV |
| IMS Service Status | ❌ **Gap 9 → Fitur F** | Read-only; nol `ims` di backend BFR. Label menu "IMS Service Status" di ModemCaps membuka dialog laporan kapabilitas raksasa, bukan penampil status khusus |
| Data Preferences | ✅ **= Gap 1 → Fitur A** | Terpecahkan 2026-10-21: "Data Preferences" adalah nama aplikasi untuk editor switch kapabilitas `ca` (dispatch +1, `am.java:15964`) — bukan fitur terpisah, tidak ada riset lanjutan |
| Laporan HW: ICC Misc, EFS Misc, RFC, RFFE | 🟡 **Sumber terpetakan (lampiran riset tertutup)** | RFFE: baca `/rfc/rffe_scan/rffe_scan.dat` → bisa ditiru ringan via EFS browser; ICC Misc: parser CA-combo on-demand dari blob biner `am.f285i`; EFS Misc/LTE CA Combos: string statis runtime; **RFC Report: tidak ada implementasi — jangan dikejar** |
| Editor tidak menyentuh modem (salinan state, tulis saat kembali) | 💡 Pola desain | Layak ditiru di UI spek: edit lokal → Apply eksplisit |
| Laporan ≥16 KB lewat registry one-shot (bukan Intent) | 💡 Pola desain | Relevan bila laporan CA combos jadi dibangun |

Versi naratif peta ini (dengan tingkat kesulitan per gap) tersimpan di luar repo: `~/workspace/your_files/ModemCaps-ke-BFR-Peta-Gap.md`.
