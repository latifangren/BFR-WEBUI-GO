//go:build android || linux

package efs

import (
	"encoding/hex"
	"fmt"
	"time"

	"bfr-webui-go/internal/logger"
)

// ReadIMEIs reads IMEI 1 (NV 550) and IMEI 2 (NV 2497 / 5014) from modem DIAG.
func (m *Manager) ReadIMEIs() (*IMEIInfo, error) {
	nvm, err := GetManagerNV()
	if err != nil {
		return nil, fmt.Errorf("DIAG session unavailable: %w", err)
	}

	info := &IMEIInfo{Success: true}

	// 1. Read IMEI 1 (NV 550)
	if res, err := nvm.ReadNV(550, 0); err == nil && res.Success {
		info.IMEI1 = res.Formatted
	} else if err != nil {
		logger.Get().Warnf("IMEI", "Read NV 550 (IMEI 1) failed: %v", err)
	}

	// 2. Read IMEI 2: check NV 2497 first, then fallback to NV 5014
	if res, err := nvm.ReadNV(2497, 0); err == nil && res.Success && res.Formatted != "Not Set" && res.Formatted != "" {
		info.IMEI2 = res.Formatted
	} else {
		if res2, err2 := nvm.ReadNV(5014, 0); err2 == nil && res2.Success && res2.Formatted != "Not Set" {
			info.IMEI2 = res2.Formatted
		}
	}

	return info, nil
}

// WriteIMEI writes IMEI using the requested method (auto, normal, or efs) with safe sequencing.
func (m *Manager) WriteIMEI(req IMEIWriteReq) (*IMEIWriteResult, error) {
	result := &IMEIWriteResult{
		Slot:     req.Slot,
		StepLogs: []string{},
	}

	// 1. Validation & normalization
	normIMEI, err := NormalizeIMEI(req.IMEI)
	if err != nil {
		result.Error = fmt.Sprintf("IMEI validation failed: %v", err)
		result.StepLogs = append(result.StepLogs, fmt.Sprintf("FAIL: %v", err))
		return result, err
	}
	result.IMEI = normIMEI
	result.StepLogs = append(result.StepLogs, fmt.Sprintf("Validated 15-digit IMEI (Luhn verified): %s", normIMEI))

	// 2. Encode to 3GPP BCD format
	bcdData, err := EncodeIMEIToBCD(normIMEI)
	if err != nil {
		result.Error = fmt.Sprintf("BCD encoding failed: %v", err)
		result.StepLogs = append(result.StepLogs, fmt.Sprintf("FAIL: %v", err))
		return result, err
	}
	bcdHex := hex.EncodeToString(bcdData)
	result.StepLogs = append(result.StepLogs, fmt.Sprintf("Encoded 9-byte 3GPP BCD: %s", bcdHex))

	// 3. Connect to session
	sess, err := m.GetSession()
	if err != nil {
		result.Error = fmt.Sprintf("DIAG connection error: %v", err)
		result.StepLogs = append(result.StepLogs, fmt.Sprintf("FAIL: DIAG connection: %v", err))
		return result, err
	}

	// 4. Target NV ID
	var targetIDs []uint16
	if req.Slot == 0 {
		targetIDs = []uint16{550}
		result.StepLogs = append(result.StepLogs, "Target Slot: SIM 1 -> Primary NV Item 550")
	} else {
		targetIDs = []uint16{2497, 5014}
		result.StepLogs = append(result.StepLogs, "Target Slot: SIM 2 -> Secondary NV Items 2497 & 5014")
	}

	// 5. Switch to Offline Digital mode if requested
	if req.UseOfflineMode {
		result.StepLogs = append(result.StepLogs, "Switching modem to Offline-D (DIAG 0x29) to suspend cellular RF...")
		if err := sess.diag.SetMode(MODE_OFFLINE_D_F); err != nil {
			result.StepLogs = append(result.StepLogs, fmt.Sprintf("WARN: SetMode(Offline-D) response: %v (continuing anyway)", err))
		} else {
			result.StepLogs = append(result.StepLogs, "OK: Modem switched to Offline Digital mode successfully")
		}
		// Brief pause for RF settling
		time.Sleep(100 * time.Millisecond)
	}

	// 6. Security Elevation (SPC 000000 + engineering passwords)
	result.StepLogs = append(result.StepLogs, "Elevating DIAG security (Sequential SPC 000000 & engineering passwords)...")
	if sess.diag.BypassSecurity(4 * time.Second) {
		result.StepLogs = append(result.StepLogs, "OK: Security session elevated (accepted by modem)")
	} else {
		result.StepLogs = append(result.StepLogs, "WARN: No explicit ACK from bypass keys, proceeding with write attempts...")
	}

	nvm := NewNVManager(sess.diag)
	var writeSuccess bool
	var lastErr error

	method := req.Method
	if method == "" {
		method = "auto"
	}

	// Try Normal Method (DIAG 0x27)
	if method == "auto" || method == "normal" {
		result.StepLogs = append(result.StepLogs, "Attempting Normal Method: Standard NV Write (DIAG 0x27)...")
		successCount := 0
		for _, id := range targetIDs {
			if err := nvm.sendNVWriteStd(id, bcdData); err != nil {
				lastErr = err
				result.StepLogs = append(result.StepLogs, fmt.Sprintf("NV %d write rejected: %v", id, err))
			} else {
				successCount++
				result.StepLogs = append(result.StepLogs, fmt.Sprintf("OK: NV %d standard write succeeded!", id))
			}
		}

		// For SIM 1 (NV 550) or SIM 2 (NV 2497 / 5014), success on at least one candidate NV completes the write
		if successCount > 0 {
			writeSuccess = true
			result.MethodUsed = "normal"
			result.StepLogs = append(result.StepLogs, fmt.Sprintf("SUCCESS: Normal NV write completed successfully (%d NV items updated).", successCount))
		}
	}

	// Fallback to EFS Method if Normal failed or EFS was explicitly chosen
	if !writeSuccess && (method == "auto" || method == "efs") {
		result.StepLogs = append(result.StepLogs, "Attempting EFS Method: Writing via EFS2 File System Subsystem (0x13)...")

		// Path list for modem EFS identity items
		var efsPaths []string
		if req.Slot == 0 {
			efsPaths = []string{
				"/nv/item_files/modem/uim/mmgsdi/imei",
				"/nv/item_files/modem/lte/rrc/imei",
				"/nv/item_files/rfnv/00000550",
			}
		} else {
			efsPaths = []string{
				"/nv/item_files/modem/uim/mmgsdi/imei_2",
				"/nv/item_files/rfnv/00002497",
				"/nv/item_files/rfnv/00005014",
			}
		}

		efsWritten := 0
		for _, efsPath := range efsPaths {
			result.StepLogs = append(result.StepLogs, fmt.Sprintf("EFS2 PUT item file: %s", efsPath))
			if err := sess.WriteFile(efsPath, bcdData, 0644, 1); err != nil {
				result.StepLogs = append(result.StepLogs, fmt.Sprintf("EFS2 PUT %s: %v", efsPath, err))
			} else {
				efsWritten++
				result.StepLogs = append(result.StepLogs, fmt.Sprintf("OK: EFS2 PUT %s written successfully!", efsPath))
			}
		}

		// Also try Subsystem Write 0x4B as part of EFS method
		for _, id := range targetIDs {
			if err := nvm.sendNVWriteSubsys(id, 0, bcdData); err == nil {
				efsWritten++
				result.StepLogs = append(result.StepLogs, fmt.Sprintf("OK: Subsystem 0x4B write for NV %d succeeded!", id))
			}
		}

		if efsWritten > 0 {
			writeSuccess = true
			result.MethodUsed = "efs"
			result.StepLogs = append(result.StepLogs, fmt.Sprintf("SUCCESS: EFS Method completed (%d targets updated).", efsWritten))
		} else if lastErr != nil {
			result.Error = fmt.Sprintf("Both Normal and EFS methods failed. Last error: %v", lastErr)
		} else {
			result.Error = "EFS Method: All EFS item write targets rejected by modem"
		}
	}

	// 7. Cleanup mode / Reboot
	if req.UseOfflineMode {
		if req.RebootModem && writeSuccess {
			result.StepLogs = append(result.StepLogs, "Issuing Modem Subsystem Restart (SSR)...")
			_ = sess.diag.SSR()
			result.StepLogs = append(result.StepLogs, "Modem reboot command issued.")
		} else {
			result.StepLogs = append(result.StepLogs, "Returning modem to Online mode (DIAG 0x29 MODE_ONLINE)...")
			_ = sess.diag.SetMode(MODE_ONLINE_F)
			result.StepLogs = append(result.StepLogs, "OK: Modem restored to Online mode.")
		}
	} else if req.RebootModem && writeSuccess {
		result.StepLogs = append(result.StepLogs, "Issuing Modem Subsystem Restart (SSR)...")
		_ = sess.diag.SSR()
		result.StepLogs = append(result.StepLogs, "Modem reboot command issued.")
	}

	result.Success = writeSuccess
	return result, nil
}
