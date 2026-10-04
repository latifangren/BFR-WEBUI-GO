# Google Pixel 5 (Redfin) Modem, EFS & Partition Architecture Documentation

This document records the in-depth architectural analysis, Qualcomm baseband partition structure, EFS (Embedded File System) v2 synchronization mechanisms, and direct hardware experiment results on the **Google Pixel 5 (`redfin` - Qualcomm Snapdragon 765G / SM7250)**.

---

## 1. Hardware Specifications & Modem Architecture

- **Device:** Google Pixel 5 (Codename: `redfin`)
- **Chipset / SoC:** Qualcomm Snapdragon 765G 5G (SM7250)
- **Modem Subsystem:** Qualcomm MPSS (Modem Peripheral Subsystem) / AMSS Baseband
- **Internal Storage:** UFS 2.1
- **Linux Block Partition Paths:**
  - `/dev/block/by-name/` (Modern Android standard path)
  - `/dev/block/bootdevice/by-name/` (Legacy compatibility symlink)
  - `/dev/block/platform/soc/1d84000.ufshc/by-name/` (Physical UFS Controller)

---

## 2. Modem, EFS & Telephony Partition Map

Based on backup analysis from `F:\GITHUB\PIXEL5\EFS_MODEM_BACKUP\EFS_MODEM_BACKUP`:

| Partition Name | File Size | Type / Format | Function & Characteristics |
| :--- | :--- | :--- | :--- |
| **`fsg`** | 2.5 MB (2,621,440 B) | EFS2 Binary Image | **File System Golden (Master Image):** Factory reference master. Read by modem only when `modemst1/st2` are corrupt or empty. |
| **`modemst1`** | 2.5 MB (2,621,440 B) | EFS2 Raw Block | **EFS Active Runtime 1:** Primary active modem filesystem partition. Contains NV Item 550 (IMEI), RF calibration, and cellular provisioning. |
| **`modemst2`** | 2.5 MB (2,621,440 B) | EFS2 Raw Block | **EFS Active Runtime 2:** Direct backup mirror / redundancy of `modemst1`. |
| **`fsc`** | 128 KB (131,072 B) | Cookie / Counter | **File System Cookie:** Integrity token and generation sync counter between `modemst1` and `modemst2`. |
| **`persist`** | 32 MB (33,554,432 B) | Ext4 Filesystem | **Vendor Persistence:** Hardware calibration (camera, gyroscope/accelerometer sensors), Wi-Fi/BT MAC addresses, and vendor radio flags. |
| **`modem`** | 134.2 MB (134,217,728 B) | Fat16 / VFAT Image | **Modem Firmware (NON-HLOS):** Compiled Qualcomm baseband binaries (`mba.mbn`, `qdsp6sw.mbn`, etc.). Read-only firmware. |

---

## 3. Test Results: Direct Partition Flashing vs Modem DIAG

Direct raw block flashing on `/dev/block/by-name/` vs Qualcomm DIAG port commands:

### A. Raw Block Flashing via Linux (`dd` / Android Root)
- **Command:**
  ```bash
  dd if=/sdcard/modemst1.bin of=/dev/block/by-name/modemst1
  dd if=/sdcard/modemst2.bin of=/dev/block/by-name/modemst2
  dd if=/sdcard/fsg.bin of=/dev/block/by-name/fsg
  dd if=/sdcard/fsc.bin of=/dev/block/by-name/fsc
  sync
  ```
- **Result:** IMEI and cellular signal immediately return to 100% normal without requiring Windows Qualcomm tools (QPST, QFIL, QCN Writer) or DIAG port rituals.
- **Root Cause:** Raw block writes bypass the baseband OS layer, ignoring all security locks, SPC `000000`, and NV Item 550 read-only restrictions.

---

### B. Auto-Recovery via `fsg` (Factory Golden Master)
- **Test:** Zeroing `modemst1` and `modemst2` while leaving valid data in `fsg`.
  ```bash
  dd if=/dev/zero of=/dev/block/by-name/modemst1
  dd if=/dev/zero of=/dev/block/by-name/modemst2
  sync
  reboot
  ```
- **Result:** Phone boots normally, signal is active, and IMEI remains intact.
- **Root Cause:** Detecting corrupted or blank `modemst1/st2`, the Qualcomm modem reads `fsg`. Because `fsg` contains a valid image, the modem automatically clones data from `fsg` into `modemst1/st2`.

---

## 4. Empirical Experiments & Technical Findings

### Experiment 1: Standard Wipe via Android Root
- Running `dd if=/dev/zero` while Android is running.
- **Observation:** `rmt_storage` daemon immediately syncs cached RAM blocks back to disk on shutdown, restoring partitions before power-off.

### Experiment 2: Partition Lock via Mount
- Partitions `/dev/block/by-name/modemst*` are raw block devices, not mounted filesystems. Attempting `umount` fails (`not mounted`).

### Experiment 3: Erasing via Fastboot
- Fastboot commands `fastboot erase modemst1` and `fastboot erase modemst2` are blocked or restricted on production Google bootloaders.

### Experiment 4: Successful Zeroing via Service Freeze + Instant Hardware Reset (`sysrq`)
- **Objective:** Force the modem into a pure blank state from Android without TWRP/Fastboot and without wiping user data.
- **Procedure:**
  1. Stop all modem, storage, and RIL daemons so none lock or flush from RAM:
     ```bash
     stop vendor.rmt_storage
     stop vendor.tftp_server
     stop modem_svc
     stop vendor.qcrild
     stop vendor.qcrild2
     stop vendor.google.radioext@1.0
     ```
  2. Clear active radio databases in `/data`:
     ```bash
     rm -f /data/vendor/radio/qcril*
     rm -f /data/vendor/radio/copy_complete
     rm -f /data/vendor/radio/db_check_done
     ```
  3. Zero all 4 modem partitions (`fsg`, `fsc`, `modemst1`, `modemst2`) until *No space left on device*.
  4. Run `sync`, then trigger an **Instant Hardware Reset** via Kernel SysRq:
     ```bash
     echo b > /proc/sysrq-trigger
     ```
     *(Bypasses graceful shutdown, preventing Android from spawning daemons to flush RAM).*
- **Post-Reboot Diagnostic Results:**
  - `gsm.version.baseband`: **BLANK / NULL**.
  - `gsm.network.type`: `[Unknown,Unknown]`.
  - Service `iphonesubinfo`: `Can't find service` (Telephony disabled).
  - Hex `modemst1`: Headers `IMGEFS1@` / `IMGEFS1D` **COMPLETELY ABSENT**.
- **Critical Finding: Wi-Fi Subsystem Failure!**
  - **Symptom:** When EFS is completely empty, in addition to lost cellular signal and null IMEI, **Wi-Fi functionality also crashes / fails to enable**.
  - **Snapdragon 765G Architectural Cause:**
    1. **QMI / IPA Subsystem Dependency:** The Qualcomm Wi-Fi chip (WCN3990) communicates tightly with the MPSS modem via QMI (Qualcomm Management Interface) protocols and IPA (IP Accelerator).
    2. **WLAN Calibration Failure:** Without baseband modem init, Wi-Fi firmware handshake fails to initialize RF calibration tables.

---

## 5. Why Does IMEI Get Lost During Factory Reset?

```
┌─────────────────────────────────────────────────────────────┐
│                       FACTORY RESET                         │
│               (Erases entire /data partition)               │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
        Directory /data/vendor/radio/ wiped clean
        (Database qcril.db & EFS cache deleted)
                               │
                               ▼
                    Device Reboots / Booting
                               │
                               ▼
     Modem queries master fallback reference partition: FSG
                               │
                               ▼
   ┌───────────────────────────┴───────────────────────────┐
   │                                                       │
[Condition 1: fsg is ZERO / EMPTY]           [Condition 2: fsg has VALID BACKUP]
   │                                                       │
   ▼                                                       ▼
Modem has no data source.                    Modem successfully clones fsg to modemst1.
IMEI becomes Null / 00499...                 IMEI & Signal REMAIN 100% NORMAL
Cellular signal LOST (No Service).           (Factory Reset Proof).
```

---

## 6. Recommendations & Best Practices

### A. Making IMEI Permanent & Factory Reset-Proof
Ensure the `fsg` partition is populated with a valid image backup:
```bash
su
dd if=/sdcard/EFS_MODEM_BACKUP/fsg.bin of=/dev/block/by-name/fsg && sync
```

### B. Safe Daily Minimal Restore Procedure
If cellular signal degrades or drops, restore only these 4 baseband partitions without touching hardware calibration (`persist`):
```bash
su
dd if=/sdcard/EFS_MODEM_BACKUP/fsg.bin of=/dev/block/by-name/fsg
dd if=/sdcard/EFS_MODEM_BACKUP/fsc.bin of=/dev/block/by-name/fsc
dd if=/sdcard/EFS_MODEM_BACKUP/modemst1.bin of=/dev/block/by-name/modemst1
dd if=/sdcard/EFS_MODEM_BACKUP/modemst2.bin of=/dev/block/by-name/modemst2
sync
reboot
```

### C. Partition Safety Invariant
- **Never wipe `persist` without backup:** `persist` contains camera sensor calibration and sensor flags that cannot be recovered from modem partitions alone.
- **Always take pre-write snapshots** of `fsg`, `fsc`, `modemst1`, `modemst2` before testing baseband modifications.
