# Branch Architecture & Build Flavor Isolation Specification

This document defines the strict isolation model between the **Private Full Build (`dev`)** and the **Public Non-Modem Build (`release/no-modem`)**.

---

## 1. Branch Hierarchy & Target Matrix

```
                        ┌──────────────────────────────────────┐
                        │             dev (Main)               │
                        │ Full Private Build (Everything)      │
                        └──────────────────┬───────────────────┘
                                           │
                                           │ git merge dev (Fast, 0 Conflict)
                                           ▼
                        ┌──────────────────────────────────────┐
                        │          release/no-modem            │
                        │ Public Distribution (Zero Vendor IP) │
                        └──────────────────────────────────────┘
```

| Dimension | `dev` (Full Private) | `release/no-modem` (Public Clean) |
| :--- | :--- | :--- |
| **Target Audience** | Development, internal test rigs | Public releases, open distributions |
| **Cellular Telemetry** | Active (`dumpsys`, signal, SIM, thermal) | Active (`dumpsys`, signal, SIM, thermal) |
| **Vendor Baseband IP** | Qualcomm QRTR, Samsung SecRIL, Tensor Shannon, EFS2/IMEI | **0% Included (Zero Trace)** |
| **Go Build Tag** | Default (no tag needed) | `-tags nomodem` |
| **Frontend Flag** | `HAS_VENDOR_MODEM = true` | `HAS_VENDOR_MODEM = false` |
| **Vendor UI Tabs** | Loaded via `vendor_tabs.ts` | Tree-shaken out (`VENDOR_TABS = {}`) |
| **Hardware Band Locking Card**| Direct vendor tab switcher buttons | Displays `PRIVATE FEATURE ONLY` badge |

---

## 2. Technical Isolation Seams

The codebase enforces isolation through **two architectural seams**:

### A. Backend Seam: `registerVendorModemRoutes`
- Located in `internal/handlers/routes_modem.go`.
- Delegates all vendor modem registration to `registerVendorModemRoutes(rc)`.
- **Implementation**:
  - `internal/handlers/routes_vendor_modem.go` (`//go:build !nomodem`): Registers real handlers.
  - `internal/handlers/routes_vendor_stub.go` (`//go:build nomodem`): Clean no-op.
- **Compiler Dead-Code Elimination**:
  When built with `-tags nomodem`, Go compiler strictly excludes `routes_vendor_modem.go`, `efs_handler.go`, `imei_handler.go`, `samsung_handler.go`, `shannon_handler.go`, and the underlying `internal/efs`, `internal/samsung`, `internal/shannon` packages.

### B. Frontend Seam: `features.ts` & `vendor_tabs.ts`
- `frontend/src/config/features.ts`:
  - `export const HAS_VENDOR_MODEM = true` (in `dev`)
  - `export const HAS_VENDOR_MODEM = false` (in `release/no-modem`)
- `frontend/src/components/tabs/vendor_tabs.ts`:
  - Exports vendor components in `dev`.
  - Exports `{}` in `release/no-modem`.
- **Bundler Tree-Shaking**:
  When `VENDOR_TABS` is empty, Vite completely eliminates `TabQualcomm.svelte`, `TabSamsung.svelte`, and `TabShannon.svelte` from the production bundle `index-*.js`.

---

## 3. Binaries & Packaging Isolation

- **`bin/sim_switcher.dex`**: Permanently deprecated and removed. SIM switching runs via native Android AOSP IPC in Go.
- **`bin/band_manager.dex`**: Excluded from `release/no-modem`.
- **Build Scripts**:
  - `build.bat` & `build.sh`: Automatically detect branch name. If on `release/no-modem`, automatically passes `-tags nomodem`. Alternatively accepts `./build.sh nomodem`.
  - `build_zip.bat`: Packages Magisk zip with `-nomodem.zip` suffix when on `release/no-modem`.

---

## 4. Verification Check Before Distribution

Before releasing any artifact from `release/no-modem`, run this forensic string scan:

```bash
python -c "
with open('bfr-webui-android-arm64', 'rb') as f:
    data = f.read()

targets = [b'internal/efs', b'internal/shannon', b'internal/samsung', b'QualcommIMEI', b'/api/efs', b'/api/qualcomm']
for t in targets:
    assert data.count(t) == 0, f'LEAK DETECTED: {t}'
print('PASS: Zero vendor modem code or strings in binary!')
"
```
