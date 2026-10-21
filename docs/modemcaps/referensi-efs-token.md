# Referensi Cepat — Path EFS, Token, dan Format Data ModemCaps

Lembar rujukan implementor, dipadatkan dari `dekripsi-string.md` dan `analisis-utama.md`. Path relatif terhadap `/nv/item_files/modem/` kecuali ditulis penuh.

## 1. Path EFS kapabilitas (21 path persis dari lapisan Java)

**LTE RRC / fitur**
- `lte/rrc/efs/lte_feature_enable` · `lte/rrc/efs/lte_feature_disable`
- `lte/rrc/efs/cell_selection_opt_params` · `lte/rrc/efs/cell_restrict_opt_params`
- `lte/rrc/efs/cell_lock_list` (+ salinan `reg_files/…/csp/cell_lock_list`)
- `lte/rrc/efs/max_num_mimo_lyr`

**Penguncian (CSP)**
- `lte/rrc/csp/pci_lock` · `lte/rrc/csp/earfcn_lock`
- GERAN: `feature_lock_arfcn_enabled`, `rr_efs_arfcn_list`, `fake_bts_cell_barring_enabled`
- WCDMA: `wcdma_rrc_freq_lock_item`

**CA / RF**
- `lte/common/ca_disable` · `lte/ML1/ard_config` (adaptive Rx diversity) · `lte/ML1/rx_select` (Rx chain)
- `mcs/trm/ant_div_switch_master`

**NR**
- `nr5g/RRC/cap_control_dl_256qam` · `nr5g/RRC/cap_disable_ul_256qam` · `nr5g/RRC/cap_limit_rf_mimo` · `nr5g/RRC/cap_nrca_downgrade_1cc`

**MCFG / PDC (profil MBN)**
- `/nv/item_files/mcfg/mcfg_autoselect_by_uim` · `…/mcfg_autoselect_by_uim_Subscription01`
- Direktori modem: `/mcfg_sw/generic` · `/mcfg_hw/generic`
- String UI: `PDC Preferences`, `PDC profile`, `_pdc_sw_listview`, `_pdc_hw_listview`, `with MBN update only`, `Select MCFG on SIM change`

**Staging & node diagnostik**
- Pertukaran Java↔native: `/data/tmp/modemcaps_*.bin`, `/data/tmp/00028874` (zlib level 9)
- Diag/SoC: `/dev/diag`, `/d/diag/{status,bridge,mhiinfo}`, `/sys/class/diag/diag/dev`, `/sys/devices/soc0/{soc_id,raw_id,chip_name,machine,images}`, `/sys/kernel/debug/qcom_socinfo/mpss/name`, `/d/msm_ipc_router/dump_servers`
- Log dukungan: `/sdcard/modemcaps_osinfo.log` (dump `getprop` disaring — baris `imei`/`imsi`/`esn`/`serialno` dibuang)

## 2. Token halaman kapabilitas (38 token)

State satu halaman dikodekan sebagai **satu string token**; switch aktif ⇔ token ada sebagai substring. Urutan bangun ulang: LTE → NSA → NR → Rx/ARD.

- **LTE:** `[LTE_CA]` `[LTE_CA_FILTER]` `[LTE_UL_CA]` `[LTE_DL_256Q]` `[LTE_UL_256Q]` `[LTE_DL_MIMO]` `[LTE_HORXD]` (adaptive Rx diversity) `[LTE_RX0]`–`[LTE_RX3]` `[LTE_VOICE]`
- **NSA:** `[NSA_LTE]` `[NSA_LTE_DSS]` `[NSA_NDDS]` (NSA saat data-call SIM lain) `[NSA_LTE_DSDA]`
- **NR:** `[NR_CA]` `[NR_DL_256Q]` `[NR_UL_256Q]` `[NR_DL_MIMO]` `[NR_UL_MIMO]` `[NR_REDCAP]` `[NR_VOICE]` `[NR_NR_DSDA]`
- Pemetaan diversity: D#0 = PRx, D#1–D#3 = DRx, D#6 = ARD.
- **Write-only** (tidak bisa dibaca balik, dari footer UI asli): LTE+LTE aggregation, UL aggregation, DL 256-QAM, Rx chain, ARD. Switch bisa abu-abu karena: hardware tidak mendukung, butuh supplemental connectivity, mode pesawat, atau fitur generik (tidak tergantung SIM).

## 3. Tata bahasa filter band-combo (`[LTE_CA_FILTER]`)

```
Band+Class+[Streams][Uplink Class]-...[Bandwidth Class Set]
Class: A|C|D|E|F · Streams: 2|4 · BW Class Set: 0|1|2|4|m3|m7|mAll
```
Teks dinormalisasi: pisah `;`, uppercase, suffix default bila tidak ditulis.

## 4. Format pengepakan Searcher Preferences

Per halaman, nilai terpak dalam `long[30]`; 4 seksi RAT (GSM/UMTS/LTE/NR), tiap seksi: cell blacklist, channel blacklist, channel code whitelist, priority channel list — maks 6 entri (UMTS 1).

- GSM cell = `(f1<<48)|(f2<<40)|(f3<<16)|f4`
- LTE cell = `(PCI<<32)|CID`
- NR menyertakan step SCS (`15 << nibble`)
- `-1` = terminator
- Legenda: Channel = ARFCN|UARFCN|EARFCN|NR-ARFCN · Code = BSIC|PSC|PCI · Step = SCS · Area = LAC|TAC · Cell = CID
- Efek samping resmi (tertulis di halaman aslinya): mengisi prioritas UMTS ⇒ preferensi RAT jadi UMTS-only; perubahan memicu modem power cycle / restart OS modem (±1 menit, kadang reboot ponsel).

## 5. Editor band

Grup RAT pada checklist (kunci internal): 3, 4, 5, 7, 19, 20, 21, 22 — mencakup GSM/UMTS/LTE/NR. Mask band `long[]`; tabel label nomor band 1–256 (LTE/NR) ada di fasad `a.m`. Long-press = centang massal satu grup. Result code khas: editor band `21` = reset, searcher `12` = reset, `0` = batal.

## 6. Identitas perangkat (laporan)

26 system property dibaca untuk label perangkat: `ro.product.{model,device,board,marketname,…}`, `ro.baseband`, `ro.board.platform`, `ro.boot.{cid,mid}`, varian vendor Sony (`ro.semc.*`/`ro.somc.*`), ASUS, Oppo/OPlus.
