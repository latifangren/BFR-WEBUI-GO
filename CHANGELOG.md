# Changelog

All notable changes to the **BFR-WEBUI-GO** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

## [1.2.6] - 2026-10-03
### Changed & Refactored
- **Domain-Modularized Backend Routing (`internal/handlers/`)**:
  - Decomposed central monolithic `router.go` into dedicated domain routing registrars (`routes_system.go`, `routes_network.go`, `routes_modem.go`, `routes_services.go`), reducing `router.go` from ~400 lines to 176 lines while maintaining exact API endpoint parity.
- **Dynamic Frontend Tab Registry (`frontend/src/`)**:
  - Replaced 27 manual component imports and 60-line `if-else` ladder in `App.svelte` with a unified dynamic component registry (`frontend/src/components/tabs/registry.ts`), powered by Svelte 5 reactive derived runes.
- **Frontend Toolchain Migration to Bun**:
  - Transitioned package manager, local build scripts, and CI workflows from `pnpm` to `bun` (`bun.lock`, `oven-sh/setup-bun@v2`), delivering 2–3x faster compilation and check benchmarks.
- **TypeScript & A11y Verification**:
  - Resolved `SysinfoStats` property contract mismatches (`resolution`, `density`) in `TabSysinfo.svelte`.
  - Fixed `Button` variant typing and accessible form label tags in `QualcommNV.svelte`, achieving 0 errors and 0 warnings in `svelte-check`.

### Added
- **Mobile Layout Density & Bento Grid Overhaul (`TabOverview.svelte`, `TabSysinfo.svelte`, `TabModem.svelte`, `TabPower.svelte`, `TabModules.svelte`, `TabSpeedtest.svelte`, `TabNetwork.svelte`, `TabHotspot.svelte`, `TabAbout.svelte`, `Card.svelte`, `App.svelte`)**:
  - **Responsive 2-Column Mobile Grids**: Replaced single-column vertical stacks with compact 2-column bento grids across mobile viewports for quick telemetry gauges, hardware specification metrics, sysctl tweaks, and device action cards.
  - **TabOverview & TabSysinfo Density**: 5 quick gauges, quick action shortcuts, device summary, and device hardware specifications formatted in a balanced 2-column mobile presentation to eliminate excessive vertical scrolling.
  - **TabModem Optimization**: Target architecture spec boxes converted to 2-column layout and vendor band navigation buttons converted to dense horizontal pill buttons.
  - **TabPower Reboot Action Matrix**: 5 large stacked reboot cards transformed into a dense 2-column action grid (`grid-cols-2 lg:grid-cols-4`), cutting card height by 65%.
  - **TabModules Mobile Priority Reorder**: Reordered card sequence on mobile via flex/grid order (`order-1` for installed modules, `order-2` for ZIP installer) so active modules are immediately visible without scrolling past the dropzone.
  - **TabSpeedtest Dual-Gauge Layout**: Download and upload speed gauges positioned side-by-side on mobile with a full-width ping & jitter status bar underneath.
  - **TabNetwork Sysctl Tweak Grid**: 9 kernel and network optimization toggles formatted into a 2-column grid instead of 9 individual vertical rows.
  - **TabHotspot Mobile Client Cards**: Eliminated horizontal table scrolling on mobile viewports by rendering connected client ARP bindings as clean, stacked cards while retaining full tabular view on tablet and desktop.
  - **Global Spacing & Padding Tightening**: Reduced global card padding from `p-5` to `p-3 sm:p-5`, inner header spacing from `mb-4` to `mb-3 sm:mb-4`, and container gaps across 28 tabs from `gap-6`/`space-y-6` to `gap-3 sm:gap-6`/`space-y-3 sm:space-y-6`.

- **ProjectX Dark Theme & Neobrutalism Fine-Tuning (`themes.css`, `theme.svelte.ts`, `AppearanceModal.svelte`, `common.ts`)**:
  - **`projectx` Theme Preset**: Introduced Retro-Pop Neobrutalist design tokens ported from ProjectX Go backend featuring deep purple-toned backgrounds (`#18151e`, `#24202b`, `#302b38`, `#0d0b10`), bright accent yellow (`#ffe14a`), and warm cream foregrounds (`#fffaf4`).
  - **Neobrutalism Extended Tuning Controls**: Sliders and controls in Appearance modal to customize shadow offset (0–8px), border width (1–4px), border radius (0–14px), button click depth (1–4px), and canvas texture overlays (Dots, Grid, None) with 1-click Reset to Defaults.

- **Glassmorphism Style Paradigm & Wallpaper Gallery System (`internal/handlers/theme_handler.go`, `AppearanceModal.svelte`, `App.svelte`)**:
  - **Glassmorphism UI Mode**: Added `[data-ui-style="glass"]` with translucent frosted-glass panels, backdrop blur filters, and customizable card opacity.
  - **Real-Time Glass Tuning**: Independent sliders for Backdrop Blur (0–40px), Card Opacity (10%–100%), and Wallpaper Dimming (0%–80%).
  - **Per-Device Wallpaper Gallery**: Built-in curated presets (Abstract Dark, Cyber Grid, Deep Space, Sunset Gradient) plus custom image uploads (`POST /api/theme/wallpapers`), custom image serving (`GET /api/theme/wallpapers/file`), and deletion (`DELETE /api/theme/wallpapers/delete`).
  - **Persistent Root Module Storage**: Uploaded wallpapers are stored in `/data/adb/bfr_webui_go/data/wallpapers/` outside the Magisk module directory, surviving module updates and uninstalls. Active wallpaper and sliders persist in client `localStorage` for per-device personalization.

- **Qualcomm Baseband NV Item Manager & Security Bypass (`internal/efs/`, `internal/handlers/nv_handler.go`, `QualcommNV.svelte`)**:
  - **3-Tier NV Read/Write Fallback**: Automatically cascades through Standard NV (command `0x26`/`0x27`), Subsystem NV (command `0x4B`), and Extended NV (command `0x75`/`0x76`) for broad chipset compatibility.
  - **Field-Verified Security Bypass**: Single-click `BypassSecurity()` firing all 11 known Qualcomm engineering service programming codes (SPC) and diag passwords (`000000`, `FFFFFFFF`, standard manufacturer keys) using exact HDLC payloads.
  - **NV Item Database & Browser**: Database of common NV items with descriptions, data lengths, read-only protection flags, and categorizations (RF, LTE, 5G, IMS, SIM, System).
  - **Read, Write, Backup & Restore**: REST API and interactive WebUI for reading/writing individual items by ID in hex format, downloading full or ranged NV backups as JSON, and restoring backups to baseband.

- **Shannon 5G NR & EN-DC UE Capability Visualizer (`internal/shannon/nr_*.go`, `ShannonCAEditor.svelte`)**:
  - **100% Native Go Protobuf Engine for `ShannonNrUECap`**: Custom binary protobuf decoder for the Shannon modem 5G NR / EN-DC capability format (`.binarypb`) targeting Google Tensor G1–G5 (Pixel 6–10) without any external dependencies.
  - **EN-DC, NR-CA, and NR-DC Combo Classification**: Automatically classifies each decoded combination as EN-DC (LTE anchor + NR secondary), NR-CA (pure NR carrier aggregation), or NR-DC (FR1+FR2 dual connectivity) from raw component vectors.
  - **Full Component Decoding**: Resolves bandwidth class (A–G), MIMO streams (DL/UL), modulation order (QAM64/QAM256), subcarrier spacing (15/30/120 kHz), max bandwidth (MHz), and 90 MHz support flag for each LTE and NR component carrier.
  - **Pixel 6–10 Scope Unified with LTE CA Editor**: Both 5G NR and 4G LTE CA editors are merged into a single `Shannon CA & UE Capability` tab with an inline mode switcher supporting device generations G1 through G5.
  - **Load Demo Dataset (Pixel Sub-6)**: Built-in `sub6.binarypb` fixture embedded at compile-time via `go:embed` for instant offline demo of 515 EN-DC / NR-CA combinations (Pixel 6 series dataset) without requiring a physical device file.
  - **Import `.binarypb` from Browser**: Upload any 5G NR capability file directly from browser storage for analysis and inspection.
  - **Export / Download**: Streams the active decoded document back to the browser as `.binarypb` for archiving or comparison.
  - **Combo Count Badges on Tab Switcher**: Mode switcher buttons ([5G NR & EN-DC] / [4G LTE CA]) show real-time loaded combo counts as inline badge indicators.

- **Unload / Clear Dataset Feature (`internal/shannon/nr_manager.go`, `internal/shannon/ca_manager.go`)**:
  - **`POST /api/shannon/nr/clear`**: New endpoint to unload the active 5G NR document from server memory without restarting the daemon, resetting all combo counts and active file path.
  - **`POST /api/shannon/ca/clear`**: New endpoint to unload the active LTE CA document from server memory.
  - **Frontend Unload Button**: Contextual `[Unload]` button (Trash2 icon) appears in the 5G NR card action bar only when a dataset is currently loaded; clicking it calls the clear endpoint and resets all local Svelte state with a toast notification.

### Fixed
- **Tab Switcher 5G NR ↔ 4G LTE Unresponsive After Navigation**: Mode switcher buttons (`viewMode = 'nr'` / `viewMode = 'lte'`) were unresponsive after switching to LTE view due to click events being absorbed by child elements (icon and span nodes). Fixed by adding `pointer-events-none` to all child elements and `e.preventDefault()` with explicit `z-10 select-none` to the button containers, ensuring event targets always resolve to the button itself.
- **Null Array Serialization in 5G NR JSON Output**: `ParseShannonNrUECap` and `resolveRawCombo` now initialize all combo and component slices with `make([]T, 0)` instead of `nil`, ensuring JSON always serializes as `[]` instead of `null` — preventing frontend null-dereference errors when iterating `components_lte` or `components_nr`.
- **Null-Safe Frontend Rendering**: Added `|| []` guards in `filteredNrCombos`, `filteredLteCombos`, and all `{#if !combo.components_lte || ...}` table row guards in `ShannonCAEditor.svelte` to prevent runtime errors after unloading datasets.

- **Google Tensor Shannon LTE CA Combo Editor (`internal/shannon/`, `frontend/src/components/tabs/shannon/ShannonCAEditor.svelte`)**:

  - **100% Native Go Protobuf Engine for Shannon Baseband**: Native decoder and encoder for `ShannonLteUECap` protocol buffers (`.binarypb`) eliminating Python desktop dependencies.
  - **Pixel 9 & Pixel 10 Dedicated Scope**: Tailored for Google Tensor G4 (Pixel 9) and Tensor G5 (Pixel 10) basebands with prominent device compatibility warnings explaining why Pixel 6, 7, and 8 use internal legacy NV tables instead.
  - **Interactive Carrier Aggregation Explorer**: Searchable combo table rendering band combinations (e.g. `1A4[A]+3C4+8A2`), bandwidth classes (A–F), MIMO streams, and PLMN operator `conf_id` mappings.
  - **Auto Combo Generator & Fallback Subsets**: Generates subset combinations from large multi-band configurations up to 5CC with automated spatial stream nerfing (limiting total DL streams to 24) and uplink assignments (FDD/TDD A+A).
  - **Combo Pruning (128 Combo Cap)**: Filters essential frequency bands and excludes redundant intra-band configurations to fit within cellular network `requestedFrequencyBands-r11` constraints.
  - **Hardware Rule Validation Engine**: Pre-save verification guarding against invalid low-band combinations (only `20+28` allowed), low-band 4x4 MIMO violations, uplink assignments on SDL bands (B29/B32/B75), and RF duplexer conflicts (B7+B38 UL).
  - **Direct Device File IO & Browser Import/Export**: Load/save directly to `/vendor/firmware/.../uecap.binarypb` on device or import/export `.binarypb` files via web browser.

- **Qualcomm Snapdragon EFS2 & NV Memory Browser (`internal/efs/`, `frontend/src/components/tabs/qualcomm/QualcommEFS.svelte`)**:
  - **100% Native Go Baseband DIAG Engine**: Direct Linux kernel `AF_QIPCRTR` IPC packet router communicating directly with Qualcomm DIAG service `4097` (`SUBSYS_FS` opcode `0x4B 0x13`) on Snapdragon devices without external C daemons (`qcom-efsd`).
  - **Full EFS2 Filesystem Navigation**: Recursive tree traversal, directory listing (`opendir`/`readdir`), file statistics (`stat`), storage capacity gauge (`statfs`), mkdir, rename, unlink, and chmod permission modification.
  - **EFS Item Files Protocol Support**: Native support for EFS item files (`/nv/item_files/...`) via atomic opcode `0x26`/`0x27` (`EFS2_GET`/`EFS2_PUT`), bypassing POSIX `open/read` limitations.
  - **Direct NV Item Inspector**: Hex and text editor for reading and writing 128-byte Non-Volatile (NV) RAM memory structures (legacy opcode `0x26`/`0x27` & indexed opcode `0x4B 0x30`).
  - **Integrated SPC Security Elevation**: Interactive modal to authenticate Service Programming Code (SPC, default `000000`, opcode `0x41`) to escalate baseband write privileges.
  - **Modem Subsystem Restart (SSR)**: Direct modem baseband reboot trigger via DIAG transport (`0x25` cmd `3`) without requiring full Android system reboot.
  - **Modem Filesystem Image Backup**: On-the-fly modem-generated TAR archive extraction (`EFS2_FS_IMAGE`, opcode `0x17`) streamed directly to user browser for safe EFS partitioning backup.
  - **Safety Lock & Read-Only Protection**: Read-Only mode enforced by default with double-confirmation dialogs to prevent accidental baseband partition corruption.
  - **Dedicated Sub-Tab in Qualcomm Snapdragon (`TabQualcomm.svelte`)**: Seamlessly integrated into `TabQualcomm.svelte` as sub-tab #4 alongside Band Locking, Cell & Tower Lock, and Hardware & Profiles.

- **Multi-Vendor Cellular & Modem Control Overhaul (`TabModem.svelte`, `TabQualcomm.svelte`, `TabShannon.svelte`, `TabSamsung.svelte`)**:
  - **Modular Vendor Architecture**: Separated modem controls into dedicated hardware tabs: Universal Cellular Telemetry, Qualcomm Snapdragon (QMI/QRTR), Google Tensor (Shannon), and Samsung One UI (SecRIL).
  - **Qualcomm Snapdragon QRTR Baseband Engine (`internal/modem/qualcomm_qrtr.go`, `TabQualcomm.svelte`)**: Direct Linux kernel `AF_QIPCRTR` IPC socket interface communicating directly with the Qualcomm baseband modem.
  - **Atomic QMI Band Setters & Verification**: Ground-truth band verification via QMI `0x0034` (`NAS_GET_SYS_INFO`) eliminating false positive band locking feedback.
  - **Tower Cell Scanning & Live Discovery (`QualcommCellScanner.svelte`, `internal/modem/qualcomm_qrtr.go`)**: Real-time discovery of serving, neighbor, and secondary Carrier Aggregation (CA) cell towers with live signal power metrics (RSRP, RSRQ, RSSI, SINR).
  - **eNodeB Tower Cell Locking (`QualcommCellLock.svelte`, `internal/modem/qualcomm_qrtr.go`)**: Hardware-level cell tower locking (`EARFCN` + `PCI`) via baseband QRTR to prevent frequent signal hopping.
  - **Quick Band Presets & Operator Profiles (`QualcommProfiles.svelte`)**: One-click frequency band presets for Indonesian telcos (Telkomsel, XL Axiata, Indosat Ooredoo, Smartfren, Tri), Speed Aggregation (B3+B1), and Wide Coverage (B8).
  - **Google Tensor Shannon Modem Suite (`internal/shannon/`, `TabShannon.svelte`)**:
    - **Full Baseband Feature Port**: Ported engine and logic from `shannon-band-menu` for Google Pixel 6, 7, 8, and 9 series devices (Google Tensor GS101, GS201, G3, G4).
    - **Comprehensive Multi-RAT Band Selection**: Independent multi-band selection for LTE (B1–B71), 5G NR-SA (n1–n261), 5G NR-NSA, 3G WCDMA (B1–B19), and 2G GSM (850, 900, 1800, 1900) directly manipulating baseband NV items (`!SAEL3.Manual.Band.Select`, `!NRRRC.MANUAL_NR_BAND_LIST`, `!LTE.NR Manual Band List`, `UL3.Etc.disabled_band`, `GL3.Edge_Band_Config`).
    - **Independent RAT Locking & Presets**: 1-click radio presets (`5G Only`, `4G Only`, `LTE+5G`, `3G Only`, `2G Only`, `Auto`) and 5G NR mode switching (`both`, `sa`, `nsa`, `lte_only`) with dummy LTE Band 255 protection to prevent modem panics when LTE is disabled.
    - **VoNR (Voice over 5G) Control**: Dedicated toggle for `OEM_GFEATURE_QMS_VONR_SAVE_AP_VALUE` and `OMC.BASED.VONR.PLMN.ENABLE` (`Carrier Default`, `Always ON`, `Always OFF`).
    - **Zero-Drop Instant Reload via DM Interface (`/dev/umts_dm0`)**: Diagnostic port keypress injection into Shannon baseband menu to reload radio NV without disruptive `AT+CFUN` connection drops.
    - **Modular Neo-Brutalist UI**: 3 sub-tabs matching Qualcomm's modular interface: *Band Lock (All RATs)*, *RAT Lock & 5G Mode*, and *VoNR & Modem State* with live NV memory inspection and NSG conflict detection.
  - **Samsung One UI SecRIL Band Locking (`TabSamsung.svelte`, `internal/samsung/`)**: Refocused interface for Samsung devices utilizing `ISemTelephony` (`0x02` OEM RIL).
  - **High-Fidelity Multi-SIM & eSIM Telemetry (`internal/modem/sim.go`)**: Accurate slot index mapping and eSIM/pSIM profile distinction parsing `dumpsys isub` internal state.
- **NetMonster Core Telemetry Engine Port (`internal/modem/netmonster_parser.go`, `internal/modem/netmonster_bands.go`, `internal/modem/netmonster_mcc.go`)**:
  - **Timing Advance (TA) & Tower Distance Estimation**: Ported 3GPP Timing Advance formula (`1 TA = 78.12m`) with Samsung dummy-zero filtering, exposing live distance calculations (e.g. `~312 m (TA: 4)`) directly in the WebUI and cell scanner.
  - **5G NR NSA & EN-DC State Detection (`DetectNrNsaState`)**: Ported `NrNsaStateParser.kt` to accurately detect 5G Non-Standalone (LTE Anchor + NR Secondary) even when Android 12–16 OS privacy masks `mNrState=****` or `NOT_RESTRICTED`.
  - **Carrier Aggregation (CA) False-Positive Fix**: Eliminated neighbor cell band pollution from the CA aggregation chain; SCells are now strictly validated against `PhysicalChannelInfo` (`SecondaryServing`).
  - **5G NR Frequency Raster & Band Hint Disambiguation**: Ported `BandTableNr.kt` 5000 kHz raster check `(startFreq - freq) % 5000 == 0` and `mBands` hints to correctly resolve overlapping NR bands (e.g. n78 vs n77, n1 vs n65, n38 vs n41).
  - **MCC to ISO Alpha-2 & Offline PLMN Database (`netmonster_mcc.go`)**: Added 200+ global MCC-to-country mappings and Indonesian PLMN operator resolver (Telkomsel, XL Axiata, Indosat Ooredoo, Smartfren, Tri) for offline fallback.
  - **Vendor Quirks Postprocessors**: Integrated Pixel Tensor GSM-as-LTE neighbor remapping and vendor negative signal inversion correction.
  - **Dumpsys Buffer Expansion**: Expanded dumpsys line parsing limit from 120/140 to 350 lines to prevent premature truncation on Dual SIM and high-density radio environments.
- **Native RFC 4918 WebDAV Server & LAN Clipboard (`TabNAS.svelte`, `internal/nas/`)**:
  - **Native WebDAV Server**: Fully native Go RFC 4918 WebDAV engine running on dynamic port 85 with cross-platform OS mounting support (Windows Map Network Drive, macOS Finder, Linux DAVFS, mobile file managers).
  - **Flexible Authentication Modes**: Configurable Basic Authentication or anonymous read/write access with customizable storage roots.
  - **Real-Time LAN Clipboard Synchronizer**: Instant clipboard sharing across local Wi-Fi network for text, URLs, and code snippets without external cloud services.
- **Smart Battery & Charging Limiter Overhaul (`TabCharger.svelte`, `internal/charger/`)**:
  - **ACC Modular Architecture**: Complete rewrite inspired by Advanced Charging Controller architecture with automated sysfs switch probing (`charging_enabled`, `battery_charging_enabled`, `input_suspend`, etc.).
  - **Live Battery Telemetry**: Real-time gauges for charging current (mA), voltage (mV), battery temperature (°C), capacity (%), and charge counter (mAh).
  - **Configurable Charging Thresholds**: Automated daemon to pause charging at user-defined upper percentage and resume at lower threshold.
- **Modem Native Go Elimination of External Binaries (`internal/modem/`)**:
  - Replaced legacy binary scripts (`bin/qm-5g-thermal`, `bin/sim_switcher`, `bin/band_manager`) with 100% native Go implementations.
  - Native Go sysfs thermal sensor reader (`thermal.go`) tracking Sub-6GHz Power Amps, RF PMIC, Modem Skin, and AP SoC temperatures.
  - Direct DEX invocation via `app_process` for seamless AOSP `ISub` SIM routing.

### Fixed & Optimized
- **Cellular Signal UI Refinement (`TabModem.svelte`)**:
  - Handled missing SINR gracefully by displaying `N/A` with informative "No SNR broadcast" subtitle instead of raw sentinel `-999`.
  - Formatted signal metrics with proper unit labels (`dBm` for RSRP/RSSI, `dB` for RSRQ/SINR).
  - Split EARFCN and PCI into distinct, dedicated fields in the Cell Identity card.
  - Handled Android 12–16 OS-level location privacy redactions gracefully with an `Android 16 Masked` badge instead of ambiguous raw placeholder strings.

## [1.2.5] - 2026-09-23

### Added
- **Vnstat Bandwidth Accounting Overhaul (`TabVnstat.svelte`, `internal/vnstat/`)**:
  - **Weekly Period Accounting (`weekly`)**: Track bandwidth per ISO week (`YYYY-Www`) with automatic Monday 00:00 rollover.
  - **Yesterday Telemetry (`yesterday`)**: Capture and preserve previous day traffic totals upon daily rollover for direct day-to-day comparison.
  - **7-Day Historical Trend (`daily_history`)**: Store up to 7 days of daily bandwidth history persisted in `vnstat_data.json`.
  - **Visual 7-Day Stacked Bar Chart (`VnstatHistoryChart.svelte`)**: Visual chart comparing peak volume percentages and download/upload split.
  - **Interface Category Filters & Period Switcher (`VnstatInterfaceTable.svelte`)**: Quick interface filters for `All`, `Cellular (rmnet*)`, `Wi-Fi (wlan*)`, `SoftAP (ap*)`, and `Other`, plus multi-period selection (`Today`, `Yesterday`, `This Week`, `This Month`).
  - **Database Reset Safety Modal**: Added safety confirmation modal for traffic counter reset (*Strictly Non-Silent*).
- **Android Partition & Backup Manager (`TabPartitions.svelte`, `internal/partition/`)**:
  - **Group Backup Single `.ZIP` Packaging**: Package Radio/EFS and Boot Core group backups into compressed `.zip` archives with embedded JSON metadata manifest and SHA256 checksums.
  - **Browser Auto-Download Trigger**: Automatic browser download initiation upon quick group backup completion.
  - **Checklist Multi-Selection & Batch Deletion (`PartitionBackupList.svelte`)**: Table selection checkboxes, master toggle, `Delete Selected`, and `Delete All` with explicit warning modal.
  - **Safety Whitelist & Zero-Byte Verification**: Guardrails against zero-byte partition dumps and strict image-size/block-size validation before flashing.
- **PTU/Terminal Interactive Shell Enhancements (`TabTerminal.svelte`, `internal/terminal/`)**:
  - **Environment Injections**: Standard Android root shell environment variables (`$HOME=/data/adb`, `$TERM=xterm-256color`, `$PATH`).
  - **PTU Resize Synchronization**: WebSocket window dimension resize signaling (`TIOCSWINSZ`) for responsive terminal sizing.
  - **Responsive Virtual Key Bar**: Dedicated mobile virtual keys with auto-hiding on desktop viewports and toolbar toggle.

### Fixed & Optimized
- **File Manager Multi-File & Path Upload Bugfix (`internal/handlers/file_handler.go`)**:
  - Supported both `files` and `file` multipart form keys for drag-and-drop and standard file uploads.
  - Fixed target upload directory resolution using query parameter fallback.
- **Partition Category Navigation Bar**:
  - Preserved persistent partition category filter tabs when navigating into Saved Backups.

## [1.2.4] - 2026-09-20

### Added
- **Scrcpy Remote Screen Control (`TabScrcpy.svelte`, `internal/scrcpy/`)**: Real-time hardware video streaming using embedded `scrcpy-server.jar` and low-latency WebCodecs Annex-B H.264 decoding.
- **Hardware & Navigation Controls**: Remote Android key injection for Power, Home, Back, Recents, Volume Up/Down, and physical Screen Power toggle (`screen_power`).
- **Power Auto-Reboot Scheduler (`internal/power/scheduler.go`, `internal/handlers/power_handler.go`, `TabPower.svelte`)**: Interval-based and daily clock auto-reboot scheduler with Telegram bot broadcast notification.
- **Built-in Self-Signed TLS Generator (`internal/tlsgen/`)**: Automated ECDSA P-256 certificate provisioning for direct HTTPS on standard port 443 without port configuration.

### Fixed & Optimized
- **Scrcpy Low-Latency Touch & Stream Pipeline**: Restored video-resolution coordinate mapping to align with `scrcpy-server` hardware scaler and zero-delay touch injection.
- **Compact Mobile Navigation Dock**: Refined navigation button spacing and touch targets for mobile Chrome viewports.
- **Retro Deck Theme Contrast**: Resolved button and active pill blackout inside tone cards.

### Browser Security & HTTPS Notice
- Modern browsers (Chrome, Edge, Firefox) enforce Secure Context (`window.isSecureContext`) restrictions on the WebCodecs API (`VideoDecoder`). Accessing via plain HTTP (`http://<ip>`) results in a blank black screen because the browser blocks hardware video decoders. Users should access the WebUI directly via HTTPS (`https://<ip>`) without specifying ports (standard port 443).

## [1.2.3] - 2026-09-17

### Added
- **Dedicated Samsung RIL & SecRIL Telephony Control Tab (`TabSamsung.svelte` & `internal/samsung/`)**:
  - Direct integration with Samsung One UI baseband telephony via `ISemTelephony` (`0x02` OEM RIL Band Selection).
  - Deployed low-level `band_manager` shell bridge and `band_manager.dex` Dalvik payload running under radio UID 1001.
  - Interactive frequency band lock matrix supporting LTE/NR bands (B1, B3, B5, B8, B40, etc.).
  - Automatic frequency band restoration (`band_manager auto`) and live carrier status telemetry (`AUTOMATIC` vs `LOCKED`).
  - Pre-configured quick selection presets for major Indonesian cellular operators (Telkomsel, Indosat Ooredoo Hutchison, XL Axiata, Smartfren).
  - Instant Dual-SIM Mobile Data Switcher (`sim_switcher.dex`) leveraging `ISub` Android IPC to toggle active cellular data between SIM 1 and SIM 2.
  - 5G NR Thermal Throttling & EN-DC Restriction Diagnostic (`qm-5g-thermal`) with automated root-cause culprit detection and real-time thermistor monitoring.
  - Real-time Samsung device environment verification (manufacturer, model, One UI / SEP version, RIL tool status).
  - Strict parameter boundary validation (SIM slot 0/1, valid band range 1-1000, 64-band safety limit, secure directory permissions `770`).
- **Seamless Neo-Brutalist & Modern UI Style Integration**:
  - Registered `samsung` tab across all navigation layout modes (Modern Sidebar, Classic Topbar, Bottom Popover, Bottom Nav).
  - Fully reactive design system support adhering to all 9 themes (AMOLED, Dark, Light, Dracula, Nord, Cyberpunk, Emerald, Sunset, Retro).

## [1.2.2] - 2026-09-09

### Added
- **Modular File Manager Architecture & Dual Pane / Dual Commander (`frontend/src/components/tabs/files/`)**:
  - Refactored monolithic File Manager into 7 isolated Svelte 5 Runes modules (`types.ts`, `paneState.svelte.ts`, `BookmarksBar.svelte`, `FileDropzone.svelte`, `FileTable.svelte`, `FileModals.svelte`, `FilePane.svelte`).
  - Implemented Dual Pane / Dual Commander mode with responsive layout (side-by-side 2-column on desktop/tablet, touch-friendly tab pill switcher `[ Panel A | Panel B ]` on mobile).
  - Implemented one-click cross-pane file transfer (`Copy to Other Pane` and `Move to Other Pane`) leveraging `/api/files/batch`.
  - Added Quick Bookmarks Bar with Android/Magisk system presets (`/`, `/sdcard`, `/data/adb`, `/data/adb/modules`, `/data/local/tmp`) and persistent user custom bookmarks via `localStorage`.
  - Implemented drag-and-drop file upload overlay dropzone (`FileDropzone.svelte`) with instant multi-file upload.
- **Root Daemon Services Resource Telemetry (`internal/sysinfo/sysinfo.go` & `TabSysinfo.svelte`)**:
  - Added real-time PID, normalized multi-core CPU utilization percentage, and resident RAM usage (MB/KB) tracking via direct `/proc/[pid]/statm` and `/proc/[pid]/stat` inspection without subprocess overhead.
  - Formatted ultra-lightweight daemon memory gracefully (e.g. Dropbear SSH displaying `51 KB` instead of truncating to `0 MB`).
- **Comprehensive Tab About Restoration (`TabAbout.svelte`)**:
  - Restored BFR WEBUI PRO branding hero with dynamic specifications grid.
  - Restored Magisk Module Key Features showcase grid (BBR2, Tweaks, Dynamic Charging Limiter, SMS OTP Scanner, Web Terminal & Scrcpy).
  - Restored Support & Donation Hub with high-resolution QRIS payment showcase (`qris.jpg`) and interactive donor confirmation form with direct Telegram and Facebook integration.
- **Tri-Engine Customization Matrix & Compact Appearance Studio**: Integrated Navigation Layout Engine (`topbar` vs `sidebar`), UI Style Paradigm Engine (`neobrutal` vs `modern`), and 9 Color Palettes (including **Retro Deck**) into a compact, minimalist Appearance Studio (`AppearanceModal.svelte`).
- **Modular Navigation Engine (`docs/MODULAR_NAVIGATION_PLAN.md`)**:
  - **Classic Top Bar (Full-Width)**: Desktop header category pills with hover flyouts and 100% full-width content cards; mobile 5-column bottom navigation with touch-friendly upward popovers.
  - **Modern Sidebar**: Desktop collapsible accordion groups per category (Core, Network, System, Tools) with state memory and pulsing active-tab dot indicators; mobile clean slide-over drawer without duplicate bars.
- **Dedicated Hero Login Stage (`LoginCard.svelte`)**: Restored full visual and functional parity with the legacy WebUI, featuring glowing squircle 'B' monogram, amber `PRO` badge, dynamic device model & Android version telemetry, password visibility toggle (eye icon), 1-tap `Quick Access: Default: bfr (Tap to fill)` auto-fill button, pre-login Appearance Studio access, and `CONNECT & COMMUNITY` footer with direct Telegram, Facebook, and GitHub links.
- **Svelte 5 + Vite + TypeScript Frontend Rewrite**: Complete modern rewrite of the WebUI frontend replacing legacy Alpine.js with Svelte 5 Runes (`$state`, `$derived`, `$props`), full TypeScript strict typing, and modular component hierarchy (`frontend/src/`).
- **Embedded Single-Binary Distribution (`frontend/embed.go`)**: Go `embed.FS` integration compiling the Vite production bundle (`frontend/dist`) directly into the single Go binary (~10-12MB), maintaining 100% offline-ready Magisk / KernelSU / APatch module operation.
- **Reactive Navigation & Layout Architecture**: Client-side tab navigation (`stores/navigation.svelte.ts`) with hash routing, category grouping (System, Network, Services, Connectivity, Diagnostics, About), desktop sidebar, and mobile drawer.
- **Integrated xterm.js Web Terminal**: Native interactive PTY terminal (`TabTerminal.svelte`) over WebSocket with auto-reconnect, dynamic resize fitting, theme synchronisation, and quick-command bar.
- **Scrcpy Web Stream Mirroring**: Canvas-based H.264 video rendering and touch event forwarding (`TabScrcpy.svelte`) for low-latency Android screen control.
- **Real-Time Telemetry Streaming**: Optimized WebSocket bridge for continuous hardware sensor updates (CPU per-core stats, memory, thermals, network rates).
- **Dual-Style Theme System**: AMOLED dark, light, and 6 additional palettes (Dracula, Nord, Cyberpunk, Emerald, Sunset, Dark Navy) with instant Neobrutalism and Modern Clean style switching.

### Changed
- **Modern Build Toolchain**: Transitioned from loose static assets to Vite + Svelte 5 + Tailwind CSS + pnpm build pipeline with strict compilation verification (`svelte-check` and `tsc`).
- **Vnstat Telemetry Contract**: Aligned frontend traffic accounting with Go backend `StatsResponse` (Daily/Monthly period statistics and per-interface telemetry).
- **Proxy Core Controller**: Unified proxy mode changes (`rule`, `global`, `direct`, `script`) and service daemon actions (`start`, `stop`, `restart`) through `/api/proxy/control`.
- **Charger Limitation Controller**: Updated UI to handle unified PMIC hardware status and battery protection threshold toggling.

### Fixed
- **Backend-Frontend API Contract Mismatches**:
  - **Tab NAS**: Resolved status field mismatch (`active` vs `running`), ensuring server state and input controls reflect accurately.
  - **Tab Tunnel**: Resolved status field mismatch (`active` vs `running`), restoring instant public URL and provider status detection.
  - **Tab Telegram**: Added flexible parsing for `chat_id` string/number into `allowed_chat_ids: []int64`, supported empty string `""` reset, and flattened config response.
  - **Tab Logs**: Supported both `time` and `timestamp` fields, eliminating empty dashes in the daemon logs table and widening column to `w-36`.
  - **Tab Network**: Populated active DNS (`primary`/`secondary`) and dynamic TTL spoofing status (`current_ttl`).
  - **Anti-Nil String Error Normalization**: Replaced `fmt.Sprintf("%v", err)` returning `"<nil>"` with `errString(err)` helper returning JSON `null` when operations succeed.
  - **Dynamic TTL Spoofing Detection on Android 11+**: Updated `GetTTLSpoofStatus()` in `tweaks.go` to dynamically inspect `-j TTL --ttl-set` via `iptables -t mangle -S POSTROUTING` (compatible with TTL 65).
  - **Static Media Gzip Content-Length Mismatch**: Fixed broken QRIS image loading caused by GzipMiddleware preserving uncompressed `Content-Length` headers; stripped header on compression and bypassed gzip for media files (`.jpg`, `.png`, `.webp`, `.svg`, `.zip`) and SSE logs.
- **Header Top Bar & Appearance Modal Simplification**:
  - Removed redundant style toggle and theme dropdown buttons from top bar, unifying controls into a single `[ 🎨 Theme ]` button (Option 1).
  - Streamlined `AppearanceModal.svelte` by eliminating the redundant Live Preview Sandbox, converting options to compact segmented toggles, and fitting within standard viewports without heavy scrolling.
- **Tab Tools Redundancy Removal**:
  - Eliminated duplicate Magisk / KernelSU modules table from `TabTools.svelte`, replacing it with an elegant direct shortcut to `Core ➔ Modules`.
- **Overview Telemetry Capacity Display**:
  - Enhanced Storage and Memory (RAM) cards in `TabOverview.svelte` to display full capacity metrics (`used / total` and `free`).
- **Silent Failure on API Error Responses**: Corrected client-side handling where `{ success: false }` was silently ignored instead of throwing proper error feedback.
- **Connection Teardown & Lifecycle Leaks**: Added proper disposal for WebSocket subscriptions, debounce timers (`onDestroy`), and xterm instances during tab navigation transitions.
- **API Endpoint & Contract Mismatches**: Resolved payload discrepancies across Proxy, Charger, VnStat, and Modem modules identified in cross-layer audit.

---

## [1.2.1] - 2026-08-26

### Added
- **CasaOS-Style Application Dashboard Shortcuts**: Added persistent external application link launcher on Overview tab supporting custom titles, URLs, and icons backed by `shortcuts.json` storage surviving Magisk module updates.
- **Dedicated VnStat Bandwidth & Quota Tab**: Relocated VnStat quota cards and interface statistics table from Overview to a dedicated tab (`vnstat`) under the Network category navigation.
- **CPU Per-Core Load Modal**: Relocated detailed per-core CPU frequency and usage breakdown into an on-demand modal popup (`showCpuModal`) triggered by clicking the CPU Load card.
- **UI/UX Design Specification (`docs/DESIGN.md`)**: Extracted detailed UI/UX design architecture specifications covering Neobrutalism vs Modern Clean paradigms, 7 preset themes, layout structure, navigation system, component anatomy, and z-index governance.
- **Modem Signal Quality Metrics & Percentage Calculations**: Added EARFCN parsing, signal percentages (`RSRPPct`, `RSRQPct`, `SINRPct`), and quality rating indicators (`QualityRSRP`, `QualityRSRQ`, `QualitySINR`) to cellular modem status engine.

### Changed
- **Default HTTP Server Port Change**: Changed default WebUI server HTTP bind port from `8080` to `80` across server flag fallbacks, sysinfo checkers, tunnel forwarders, shell setup scripts, and system documentation.
- **Overview Tab Clean Redesain**: Streamlined Overview tab layout by replacing cluttered statistics cards with sleek CasaOS-style application shortcuts and on-demand modal popups.

### Fixed
- **Navbar Dropdown Submenu Stacking Context in Modern Clean Style**: Fixed desktop navigation category dropdown submenu clipping where main content cards overlay dropdown menus due to missing z-index rules in precompiled Tailwind build. Added CSS z-index governance rules in `base.css` (`header` at `z-index: 1000`, `nav div.absolute` at `z-index: 1050`, `.nav-dropdown` at `z-index: 1060`) and static utility classes `.z-60` to `.z-999`.
- **Dropdown Hover Disappear Fix**: Eliminated `button:hover` `transform: translateY(-1px)` positional shift on category trigger buttons in Modern Clean style to prevent premature Alpine `@mouseleave` events.
- **SQM / CAKE Engine Badge Contrast**: Fixed dark mode text contrast on SQM / CAKE status badges in QoS tab.
- **English Translation Standardization**: Standardized all UI texts, modal dialogs, and error messages to English.
- **Multi-Preset Color Theme Architecture & Appearance Options Modal**: Added support for 7 preset color themes (**Dark**, **Light**, **Dracula**, **Nord**, **Cyberpunk**, **Emerald**, **Sunset**) managed via `data-theme="..."` CSS variables on `<html>`. Replaced separate theme/style buttons in Header with a unified **🎨 Appearance** modal dialog featuring real-time component previews.
- **Dynamic System Properties via `tweaks.json`**: Converted `system.prop` into a clean placeholder file to prevent Magisk/KernelSU from forcing static boot properties (`ro.telephony.default_network`, `dalvik.vm.*`, `net.tcp.buffersize.*`) when network tweaks are set to `false`. All optimizations are now dynamically driven 100% by `tweaks.json` & WebUI toggles.

### Added
- **Modular UI Theme Architecture (Neobrutal vs Modern Clean)**:
  - Added support for 2 UI design paradigms: **⚡ Neobrutal** (bold 2px borders, 4px hard offset shadows, sharp corners) and **✨ Modern Clean** (soft drop shadows, smooth `rounded-xl` corners, subtle 1px borders, glassmorphism backdrop blur).
  - Split CSS into modular structure (`web/static/css/base.css`, `web/static/css/styles/neobrutal.css`, `web/static/css/styles/modern.css`, and `web/static/css/style.css`).
  - Added UI Style switcher toggle in Header (`web/templates/layout/header.html`) and persistent `localStorage.setItem('uiStyle', ...)` state manager in `web/static/js/modules/common/common.js`.
  - Added anti-FOUC inline script in `web/index.html` to prevent theme flicker on page load.
- **1-Click Auto Install Daemon Binary Remote Tunnel**: Penambahan fungsi pengunduhan biner persisten ARM64 resmi (`cloudflared`, `tailscaled`, `zerotier-one`) langsung dari WebUI dengan hak akses eksekusi `0755` serta UI indikator status biner interaktif.
- **Dedicated Remote Access Tunnel Tab (`internal/tunnel`)**: Dukungan Hybrid Tunnel (Cloudflare Quick Tunnel / Token, Tailscale, ZeroTier) dengan deteksi biner dinamis, status URL publik real-time, dan kontrol via API `/api/tunnel/*`.
- **Dedicated Local NAS Lite & Media Share Tab (`internal/nas`)**: Native Go WebDAV & HTTP File Sharing Server dengan opsi Basic Auth, Mode Read-Only, kalkulasi kapasitas memori internal/SD Card/USB OTG, dan kontrol via API `/api/nas/*`.
- **Telegram SMS Auto-Forwarder (OTP & Kuota)**: Pilihan auto-forward SMS masuk ke chat Telegram, perintah bot `/sms`, filter OTP-only (`\b\d{4,8}\b`), dan keyword custom filter.
- **Hotspot MAC Address Filtering & Access Rules**: Mode Whitelist & Blacklist menggunakan `iptables -m mac --mac-source` dengan aksi instant block dari tabel client Hotspot.
- **Comprehensive Unit Test Suite**: Added 20 unit test files covering 100% of internal packages (`auth`, `charger`, `config`, `handlers`, `hotspot`, `modem`, `modules`, `network`, `power`, `qos`, `scrcpy`, `smsviewer`, `sysinfo`, `telegram`, `terminal`, `vnstat`). Fully verified with 100% pass rate and Android `arm64` cross-compilation compatibility.
- **Dedicated QoS Bandwidth Control & Traffic Prioritization Tab (`internal/qos`)**:
  - Implemented dedicated **QoS Control** UI tab with Neo-Brutalist / AMOLED Dark styling.
  - Built Hybrid QoS Engine with high-precision `tc HTB/IFB` qdisc bandwidth shaper and automatic fallback to `iptables Mangle / MARK`.
  - Added per-client bandwidth limits with quick presets (`Gaming` 5M, `Stream` 25M, `Browsing` 10M, `Strict` 2M, `Unlimited`) and custom Download/Upload Mbps caps.
  - Implemented DSCP/TOS traffic classification for Low-Latency Gaming (ICMP/UDP) and VoIP/Streaming.
  - Added Time-based Bandwidth Scheduler for automated peak-hours throttling.
  - Added persistent state saving (`qos.json`) and automatic background QoS rule application during server boot in `main.go`.
- **Dedicated Cellular Modem & Band Locking Tab (`internal/modem`)**:
  - Implemented dedicated **Modem & Band Lock** UI tab featuring real-time signal gauges and a Neo-Brutalist AT terminal interface.
  - Built Hybrid Multi-Engine band locking supporting Universal Android Framework (`cmd phone lte-set-band-mode`), Qualcomm Snapdragon AT Serial Direct (`/dev/smd11`, `/dev/ttyUSB*`, `/dev/atcmd*`), and Vendor Secret Code intent hooks.
  - Built Hex Bitmask Auto-Calculator for LTE & 5G NR band combinations (`0x8000000005`, etc.) with manual Hex override and copy functionality.
  - Built Interactive AT Command Console with live serial response stream (`OK`/`ERROR`) and built-in AT presets (`AT+CPSI?`, `AT+QENG="servingcell"`, `AT+QCAINFO`, `AT^SYSCONFIG`, `AT+EGMR=1,7,"..."`).
  - Added real-time cellular signal metrics parser (`RSRP`, `RSRQ`, `SINR`, `RSSI`, Operator, Band, Cell ID, TAC, PCI, EARFCN).
- **Default Network Tweaks Reset**: Updated `internal/network/tweaks.json` to disable all initial network optimization flags by default.
- **Authentication IP Rate Limiting**: Implemented per-IP rate limiter in `Manager.Authenticate()` allowing a maximum of 5 failed login attempts per minute window to prevent brute-force attacks (`HTTP 429`).
- **Sysctl Initial Defaults Backup & Restoration**: Added `BackupSysctlDefaults()` and `RestoreSysctlDefaults()` in `internal/network/tweaks.go` (`POST /api/network/tweaks/restore` and `♻️ Restore Original Sysctl Defaults` button) to restore initial pre-tune sysctl parameters and reset persistent `tweaks.json` state.
- **Per-Core CPU Frequency & Usage Diagnostics**: Added per-core CPU frequency reading from sysfs (`scaling_cur_freq`) and real-time CPU core usage badge grid rendering in `tab_sysinfo.html`.

### Improved
- **HTTP Gzip Response Compression & Asset Caching**: Added Gzip response writer middleware with lazy initialization and `Flush()` support for SSE streams, alongside `Cache-Control` static asset headers.
- **Non-blocking Sysinfo Background Ticker & Static Hardware Caching**: Implemented 1.5s background ticker loop updating `cachedStats` with non-blocking `RLock()` reads, cached static hardware info (`wm size`, `wm density`, model, SDK/kernel versions) via `sync.Once`, and replaced `df` subprocesses with native `syscall.Statfs`.
- **WebSocket Terminal Security & Token Verification**: Replaced presence-only cookie checks in `HandleWebsocket()` with full session token validation (`ValidateSession`) and Origin host verification (`EqualFold`).
- **Network Tweaks Batching & Dynamic RAM TCP Buffer Scaling**: Batched all root shell executions in `ApplyAllTweaks()` into a single `su -c` command chain and scaled TCP max buffers based on physical RAM capacity (`<4GB` RAM capped at 16MB max buffer).
- **Subshell Elimination & Dynamic Sysfs Scanner** (`internal/charger/charger.go`): Replaced wasteful `su test -w` subshells with native Go `os.OpenFile` checks and added dynamic fallback sysfs scanner (`/sys/class/power_supply/*/`) matching keywords (`charging`, `suspend`, `limit`, `fcc`, `store_mode`, `switch`).
- **Universal Worker Panic Recovery** (`internal/worker/worker.go`): Added panic recovery handler to background worker pool and fallback goroutines.
- **In-Memory TTL Cache for `/proc` Reads** (`internal/sysinfo/sysinfo.go`): Implemented 750ms TTL in-memory cache for `GetStats()` using `sync.RWMutex` to eliminate unnecessary disk and `/proc` I/O.
- **Dynamic `$PATH` & Module Binary Scanner** (`internal/proxy/proxy.go`): Added dynamic `$PATH` lookup (`exec.LookPath`) and Magisk/KSU module directory scanner (`/data/adb/modules/*/bin/{mihomo,clash}`) alongside static core paths.
- **Speedtest Engine Buffer Recycling** (`internal/speedtest/speedtest.go`): Implemented `sync.Pool` for 32KB buffer recycling in download/upload multi-worker loops to eliminate GC allocations.
- **Service Panic Safety** (`internal/telegram/bot.go`, `internal/ssh/ssh.go`): Added `defer recover()` panic recovery handlers to Telegram polling, notification workers, and SSH daemon control loops.

## [1.2.0] - 2026-08-03

### Added
- **Mobile Responsive Navigation & Slide-Up Bottom Sheet System**:
  - Implemented 5-column grid Fixed Bottom Navigation Category Bar (`grid grid-cols-5`) and Slide-Up Bottom Sheet Modal for mobile devices.
  - Added Collapsible Navigation Toggle with Floating Pill Button (`🧭 Nav`) for 100% full-screen mobile viewing.
- **Tabbed Settings Modal (3-Tab System)**:
  - Redesigned monolithic Settings Modal into a compact 3-tab dialog (📦 **Backup & Restore**, 🔐 **Security**, ☁️ **Cloud Sync**) with a bottom `Close ✕` exit button, preventing content clipping on mobile and desktop screens.
- **Persistent Storage Directory (`/data/adb/bfr_webui_go/data/`)**:
  - Migrated storage of all module configs (`auth.json`, `speedtest_history.json`, `telegram_config.json`, `charger_config.json`, `ssh_config.json`, `cloud_config.json`, `tweaks_config.json`, `vnstat_data.json`) to persistent directory outside module path (`/data/adb/bfr_webui_go/data/`) to prevent data loss on module updates with automatic legacy file migration.
- **Encrypted Password Management & Change Password API**:
  - Implemented SHA-256 salted password hashing, `POST /api/auth/change-password`, `GET /api/auth/status` with `is_default_pass`, and dynamic Quick-Fill badge adaptation on the login page (`Default: bfr` vs `🔒 Password Kustom Aktif`).
- **URL Hash Tab Persistence & Hardware Back Button Support**:
  - Enabled `#tab` hash navigation sync (`#files`, `#terminal`, `#proxy`, etc.) so browser refreshes stay on the active tab and Android hardware Back button navigates between previous tabs smoothly.
- **File Manager & Storage Usage Fixes**:
  - Fixed `storageInfo.used_pct` mapping for 100% accurate amber/green storage usage bar, added responsive table horizontal scroll container, fixed action button text wrapping.
- **Login Page Redesign (Candidate 1)**:
  - Neo-Brutalist Glassmorphism login card with device badge, show/hide password eye toggle, and local offline inline SVG social links (Telegram, Facebook, GitHub).
- **SMS Viewer Mobile Card Layout**:
  - Redesigned SMS message items into mobile-friendly stacked cards with clean word wrapping.
- **Persistent Storage Directory & Password Hashing Management**:
  - Added central persistent data directory helper (`GetPersistentDataDir` & `GetPersistentFilePath`) targeting `/data/adb/bfr_webui_go/data` with automatic legacy config migration across `auth.go`, `charger.go`, `ssh.go`, `bot.go`, `cloud.go`, `speedtest.go`, and `vnstat.go`.
  - Added secure SHA-256 + Salt password hashing and persistent `auth.json` configuration in `internal/auth/auth.go`.
  - Implemented `IsDefaultPassword()` indicator and `ChangePassword(currentPass, newPass)` backend manager.
  - Added REST API endpoints `GET /api/auth/status` (returning `authenticated` and `is_default_pass`) and `POST /api/auth/change-password` (`/api/auth/change-password`).
  - Implemented multi-threaded Go network speedtest engine (`internal/speedtest/speedtest.go`) supporting concurrent Ping/Jitter probes, multi-worker HTTP GET download, and multi-worker HTTP POST upload speed testing with live progress tracking (`/api/speedtest/start`, `/api/speedtest/status`, `/api/speedtest/stop`, `/api/speedtest/history`).
  - Added Ookla-style Client IP, ISP/Carrier name, Location, and Server Data Center / IATA Colo mapping (`fetchClientAndISPInfo`) parsing Cloudflare `/cdn-cgi/trace`, `ip-api.com`, and `ipinfo.io` with fallback handlers.
  - Added `getSpeedtestHTTPClient` with custom `net.Resolver` fallback to public DNS (`1.1.1.1:53`, `8.8.8.8:53`, `9.9.9.9:53`) and `InsecureSkipVerify: true` to bypass Android IPv6 loopback (`[::1]:53`) DNS failures and TLS verification blocks.
  - Added fallback speedtest endpoints across Cloudflare (`https://1.1.1.1/cdn-cgi/trace`, `https://speed.cloudflare.com/__down?bytes=0`, `http://1.1.1.1/`), OVH (`http://proof.ovh.net/files/10Mb.dat`), and Hetzner (`http://speed.hetzner.de/10MB.bin`).
  - Added `HistoryEntry` storage in `speedtest.go` (max 20 entries) and fixed frontend JSON field mapping (`progress_pct`, `ping_ms`, `jitter_ms`, `download_mbps`, `upload_mbps`, `client_ip`, `isp`, `location`, `server_colo`, `server_name`) with auto-polling timer initialization on test start.
  - Implemented WebDAV Cloud Backup manager (`internal/backup/cloud.go`) supporting compressed `.tar.gz` archive generation and automated periodic PUT sync with WebDAV servers (`/api/backup/cloud/config`, `/api/backup/cloud/sync`).
  - Redesigned top navigation bar (`web/templates/layout/sidebar.html` & `web/templates/layout/header.html`) into 5 categorized dropdown menus: **Status ▾** (`overview`, `sysinfo`, `logs`), **System ▾** (`files`, `terminal`, `tools`), **Services ▾** (`telegram`, `charger`, `ssh`, `scrcpy`), **Network ▾** (`network`, `proxy`), and **Extras ▾** (`sms`, `about`).
  - Added smooth Alpine.js dropdown interaction (`x-data="{ open: false }"`, `@mouseenter`, `@mouseleave`, `@click.outside`) with visual active indicators matching AMOLED dark theme aesthetic.
  - Saved over 60% header space for clean layout scaling while maintaining 100% compatibility with existing tab IDs and backend APIs.
- **Universal Hardware Smart Charger Limiter & Custom Path Override**:
  - Expanded sysfs auto-scanner (`internal/charger/charger.go`) to detect Qualcomm PMIC hardware charge cutoff nodes (`/sys/class/power_supply/main/force_main_fcc`, `force_main_icl`), successfully cutting off charging current (`0 mA`) on devices like Google Pixel 5.
  - Added support for Google Pixel 5 / Tensor `charge_limit` percentage threshold nodes and expanded candidate sysfs paths across Samsung, Xiaomi, OnePlus, OPPO, Realme, ASUS ROG, and MediaTek devices.
  - Added `custom_path` override configuration in backend and UI to allow manual sysfs node specification if auto-scan misses vendor-specific nodes.
  - Added `EXPERIMENTAL` badge and kernel hardware dependency note in WebUI charger control card (`web/templates/charger/tab_charger.html`).
- **Telegram Bot Remote Management & System Alerts**:
  - Implemented bidirectional Telegram Bot daemon (`internal/telegram/bot.go`) with `/start`, `/stats`, `/charger`, `/ssh`, `/proxy`, `/hotspot`, `/modules`, `/ip`, `/reboot`, `/tweak`, and `/cmd` commands.
  - Added persistent custom Reply Keyboard menu (`ReplyKeyboardMarkup`) for single-tap navigation and interactive Inline Keyboards (`InlineKeyboardMarkup`) with callback queries for quick actions (`/charger`, `/ssh`, `/proxy`, `/reboot`).
  - Added granular notification toggles (`battery_guard`, `battery_overheat`, `ssh_status`, `ip_change`, `hotspot_client`) in `NotificationConfig` and background notification tickers for real-time alerts.
  - Added notification settings checkboxes under Telegram Bot card in WebUI Tools tab (`web/templates/tools/tab_tools.html`) and frontend state manager (`web/static/js/modules/telegram/telegram.js`).
  - Added custom Go `net.Resolver` fallback to public DNS (`1.1.1.1:53`, `8.8.8.8:53`, `9.9.9.9:53`) to bypass Android IPv6 loopback (`[::1]:53`) DNS lookup failures.
  - Added automated Telegram startup notifications on boot.
- **Bundled Static Dropbear SSH Daemon**:
  - Bundled standalone static `dropbear` and `dropbearkey` ARM64 binaries (`bin/arm64/`), removing reliance on external Termux or stock ROM SSH binaries.
  - Added automated host key generation (`/data/ssh/dropbear_ecdsa_host_key`).
  - Added root password authentication support (`bfr`) via automated `/data/ssh/passwd` mount bind to `/etc/passwd` with valid MD5 crypt hash and `/bin/sh` shell setup.
  - Added full LAN binding (`0.0.0.0:2222`) support and auto-start SSH daemon on server boot if enabled.

### Fixed & Improved
- **Module Config Persistence Across OTA Updates**:
  - Updated `customize.sh` to preserve all `*.json` configuration files (`telegram_config.json`, `ssh_config.json`, `charger_config.json`, `tweaks.json`) during Magisk/KernelSU module updates.
  - Added UTF-8 BOM (`\xef\xbb\xbf`) prefix stripping across JSON loaders to prevent parsing errors.
  - Fixed LAN API authentication issues by setting session cookies to `SameSite=Lax`.

## [1.1.0] - 2026-08-02

### Added
- **Magisk / KernelSU / APatch Module Manager**:
  - Full module scanning API (`/api/modules`) parsing properties (`module.prop`) and handling enabled/disabled/removed states.
  - POST endpoints for toggling modules (via touch/rm `/disable`) and installing uploaded module ZIP packages (`magisk --install-module` / `ksud` / `apatch`).
  - Dark AMOLED module manager UI dropdown view in Tools with ZIP flashing dropzone.
- **CPU Governor & Thermal Control**:
  - GET `/api/sysinfo/governor` listing available CPU scaling governors, current CPU frequencies per core, and GPU/Thermal zone temperatures.
  - POST `/api/sysinfo/governor` applying selected scaling governors to all CPU cores with timeout limits and regex safety.
  - Segmented control UI panel inside System Info.
- **Live Android Logcat Stream**:
  - Live logcat tail WebSocket handler (`/api/logs/logcat/stream`) executing `logcat` subprocess and streaming data in real-time under request context.
  - Real-time logging terminal component in Logs with severity level buttons and text search filters.
- **Config Backup & Restore**:
  - Implemented `/api/backup/export` and `/api/backup/import` APIs to securely back up and restore vital system settings (charger, ssh, tweaks, proxy).
  - Gear button (`⚙️`) settings overlay and Backup/Restore dropzone modal.
- **PWA (Progressive Web App) Support**:
  - Added `web/manifest.json` with standalone display configuration, AMOLED theme colors, and icons.
  - Added `web/sw.js` (Service Worker) with offline caching strategy for static assets and network-first strategy for `/api/*`.
  - Registered Service Worker and theme color meta tags in `web/index.html`.
- **Global Toast Notification Engine**:
  - Implemented Alpine.js store-driven floating Toast notification system (`$dispatch('notify', {type, title, message})`).
- **Embedded OpenAPI 3.0 & Interactive Docs Viewer**:
  - Created complete OpenAPI 3.0 specification (`web/openapi.json`).
  - Added `/docs` endpoint serving embedded Scalar API Reference documentation viewer.
- **Root Command Executor (`ExecSuContext`)**:
  - Added `config.ExecSuContext` and `config.ExecSuTimeout` helpers with `context.WithTimeout` to prevent process deadlocks when running root commands.
- **Background Worker Task Pool**:
  - Added lightweight 2-worker goroutine queue (`internal/worker/worker.go`) for executing CPU and disk-intensive operations asynchronously.
- **Development Roadmap**:
  - Added `roadmap.md` documenting future improvements across Performance, Architecture, UX, System Integrations, and Security.

### Performance & Memory
- **Procfs Direct Parsing**:
  - Replaced subprocess `pidof` and `cat /proc` shell pipes with direct Go procfs reading (`/proc/[pid]/comm` and `/proc/[pid]/status`) to reduce CPU overhead and latency.
- **`sync.Pool` Memory Buffer Reuse**:
  - Implemented `internal/bufferpool` reusing `bytes.Buffer` and bucketed byte slices (1KB, 2KB, 32KB) across terminal PTY, Scrcpy JPEG encoding, and file copy operations.
- **Adaptive Frame Throttling for Scrcpy**:
  - Implemented `writeMux.TryLock()` frame skipping to prevent buffer accumulation when streaming over slow Wi-Fi.

---

## [1.0.0] - 2026-08-02

### Added
- **Enhanced File Manager**:
  - Modular backend refactoring (`filemanager.go`, `ops.go`, `permissions.go`, `archive.go`, `search.go`).
  - Single & batch file operations (`CopyPath`, `MovePath`, `BatchDelete`, `BatchCopy`, `BatchMove`).
  - File permissions & ownership editor (`chmod` octal modes & `chown` owner/group with recursive toggle).
  - ZIP archive compression (`CompressZip`) and extraction (`ExtractZip`) with Zip Slip protection.
  - Case-insensitive bounded file search (`SearchFiles`).
  - Partition storage usage metrics (`GetStorageUsage`).
  - Interactive UI components: Storage Usage Bar, Cut/Copy/Paste Clipboard status pill, Multi-select checkboxes, Image preview modal with keyboard navigation, and custom Octal-to-Symbolic `rwx` permission editor matrix.
- **Dynamic Gateway & DNS Detection**:
  - Active gateway detection via `ip route get 8.8.8.8` (Tier 1) with fallbacks for Android 8–16+.
  - Multi-source DNS detection and transparent UI status labeling (`Active`, `Active / Wi-Fi`, `Active / Cellular`, `Wi-Fi Config`, `Cellular Config`).
  - Upgraded DNS Switcher supporting `ndc resolver` and dual-protocol TCP/UDP port 53 `iptables` DNAT redirection.
- **Centralized Environment Configuration (`internal/config`)**:
  - Added support for `BFR_SU_BIN`, `BFR_MODULE_DIR`, `BFR_CLASH_API`, `BFR_SMS_DB`, `BFR_LEASES_FILE`, and `BFR_ALLOWED_DIRS` environment variable overrides.
- **Application Event Logger**:
  - Implemented global `LogWriter` streaming standard Go logs into `logger.Get()`.
  - Added real-time audit event logging across Auth, Power, Network, Proxy, SoftAP Hotspot, File Manager, Terminal, SSH, Charger, and Scrcpy handlers.
- **Feature-Based Modular Web Structure**:
  - Reorganized frontend templates into `web/templates/<feature>/` subdirectories.
  - Reorganized JavaScript modules into `web/static/js/modules/<feature>/` subdirectories.
  - Refactored Go HTML template loader in `router.go` to parse templates recursively using `fs.WalkDir`.
- **Documentation Suite**:
  - Added `docs/PROJECT_STRUCTURE.md`, `docs/API_REFERENCE.md`, `docs/DEVELOPMENT.md`, `docs/SECURITY.md`, and `env.example`.

### Fixed
- **Deadlock in SSH Manager**: Fixed `sync.RWMutex` re-entrancy deadlock in `ssh.go` (`Start()` and `Stop()`).
- **Security Audit Findings (Group A, Medium, Low)**:
  - Resolved command injection vulnerabilities in `SetSysctl`, `SetDNS`, `SetInterfaceConfig`, `ConfigureRPS`, and `resolveDeviceName` via strict regex and IP validation.
  - Enforced `SanitizePath` with symlink resolution across file operations.
  - Hardened authentication tokens (no timestamp fallback) and updated session cookies to `SameSite=Strict`.
  - Enforced 1MB request body limit on POST handlers and added HTTP security headers (`nosniff`, `DENY`, `no-referrer`).
  - Set 512KB WebSocket ReadLimit for terminal sessions and added 5s timeout to Clash API HTTP client.
  - Fixed temporary SMS DB file permissions (`chmod 600`).
  - Added graceful shutdown handling for `SIGINT`/`SIGTERM` in `main.go`.

---

## [0.4.0] - 2026-07-31
- Initial public release of BFR-WEBUI-GO as a Magisk / KernelSU module.
