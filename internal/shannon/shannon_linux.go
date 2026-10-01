//go:build linux || android

package shannon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

const (
	RouterPath     = "/dev/umts_router"
	DMPath         = "/dev/umts_dm0"
	atTimeout      = 3 * time.Second
	MenuKeyGapMs   = 50 * time.Millisecond
	MenuVerifyGap  = 100 * time.Millisecond
	MenuBackKey    = 0x5C
	LteDummyBand   = 255
)

var (
	mu sync.Mutex

	// Default supported bands for Shannon modem
	defaultSupportedLTE   = []int{1, 2, 3, 4, 5, 7, 8, 12, 13, 14, 17, 18, 19, 20, 25, 26, 28, 29, 30, 32, 38, 39, 40, 41, 46, 48, 66, 71}
	defaultSupportedWCDMA = []int{1, 2, 4, 5, 8}
	defaultSupportedGSM   = []int{850, 900, 1800, 1900}
	defaultSupportedSA    = []int{1, 2, 3, 5, 7, 8, 12, 20, 25, 26, 28, 29, 30, 38, 40, 41, 48, 66, 70, 71, 75, 77, 78, 79, 257, 258, 260, 261}
)

// IsSupported returns true if the Shannon modem router interface is present.
func IsSupported() bool {
	if _, err := os.Stat(RouterPath); err == nil {
		return true
	}
	// Check if running on Google Tensor SoC
	soc, _ := config.ExecSuTimeout(time.Second, "getprop ro.soc.model 2>/dev/null || true")
	socStr := strings.ToLower(strings.TrimSpace(string(soc)))
	if strings.Contains(socStr, "gs101") || strings.Contains(socStr, "gs201") ||
		strings.Contains(socStr, "zuma") || strings.Contains(socStr, "tensor") {
		return true
	}
	return false
}

// GetStatus checks hardware capability, NV items, RAT activity, and VoNR status.
func GetStatus() DeviceStatus {
	mu.Lock()
	defer mu.Unlock()

	st := DeviceStatus{
		IsShannon:      IsSupported(),
		ActiveSlot:     0,
		Mode:           "both",
		VoNRState:      -2,
		SupportedLTE:   defaultSupportedLTE,
		SupportedSA:    defaultSupportedSA,
		SupportedWCDMA: defaultSupportedWCDMA,
		SupportedGSM:   defaultSupportedGSM,
		ActiveLTEBands: make([]int, 0),
		ActiveSABands:  make([]int, 0),
		ActiveNSABands: make([]int, 0),
		ActiveWCDMABands: make([]int, 0),
		ActiveGSMBands: make([]int, 0),
		ActiveRATs:     make([]string, 0),
	}

	if _, err := os.Stat(RouterPath); err == nil {
		st.RouterAvailable = true
	}
	if _, err := os.Stat(DMPath); err == nil {
		st.DMAvailable = true
	}

	// Model & SoC
	model, _ := config.ExecSuTimeout(time.Second, "getprop ro.product.model 2>/dev/null || true")
	st.DeviceModel = strings.TrimSpace(string(model))
	soc, _ := config.ExecSuTimeout(time.Second, "getprop ro.soc.model 2>/dev/null || true")
	st.SoC = strings.TrimSpace(string(soc))
	if st.SoC == "" {
		socMan, _ := config.ExecSuTimeout(time.Second, "getprop ro.soc.manufacturer 2>/dev/null || true")
		st.SoC = strings.TrimSpace(string(socMan))
	}

	// Classify modem type
	socUpper := strings.ToUpper(st.SoC)
	if st.IsShannon {
		st.ModemType = "Samsung Shannon (Google Tensor)"
	} else if strings.HasPrefix(socUpper, "SM") || strings.Contains(socUpper, "SNAPDRAGON") || strings.Contains(socUpper, "QUALCOMM") {
		st.ModemType = "Qualcomm Snapdragon Baseband"
	} else if strings.Contains(socUpper, "EXYNOS") {
		st.ModemType = "Samsung Exynos Baseband"
	} else if strings.HasPrefix(socUpper, "MT") || strings.Contains(socUpper, "DIMENSITY") {
		st.ModemType = "MediaTek Baseband"
	} else {
		st.ModemType = "Non-Shannon Modem"
	}

	// NSG detection: only relevant for Shannon hardware and strictly check the real NSG Android package
	// to avoid false positives with standard system daemons named 'bridge'.
	if st.IsShannon {
		nsgCheck, _ := config.ExecSuTimeout(time.Second, "pgrep -f com.qtrun.QuickTest 2>/dev/null || true")
		if strings.TrimSpace(string(nsgCheck)) != "" {
			st.NSGRunning = true
		}
	} else {
		st.NSGRunning = false
	}

	if !st.RouterAvailable {
		st.StatusMessage = "Shannon /dev/umts_router interface unavailable on this device."
		return st
	}

	// 1. Query Hardware Supported SA bands from NV
	suppNRRaw, err := googGetNVArray("!NRRRC.SUPPORTED_NR_BAND_LIST")
	if err == nil && len(suppNRRaw) > 0 {
		var bands []int
		for _, b := range suppNRRaw {
			if len(b) >= 2 {
				val := int(b[0]) | (int(b[1]) << 8)
				if val >= 1 && val <= 2048 {
					bands = append(bands, val)
				}
			}
		}
		if len(bands) > 0 {
			st.SupportedSA = bands
		}
	}

	// 2. Query NR.CONFIG.MODE
	modeBytes, err := googGetNV("NR.CONFIG.MODE", 0)
	if err == nil && len(modeBytes) >= 1 {
		switch modeBytes[0] {
		case 0x11:
			st.Mode = "both"
		case 0x01:
			st.Mode = "nsa"
		case 0x10:
			st.Mode = "sa"
		case 0x00:
			st.Mode = "lte_only"
		default:
			st.Mode = "both"
		}
	}

	// 3. Query LTE manual bands
	lteEnbBytes, err := googGetNV("!SAEL3.Manual.Band.Select Enb/ Dis", 0)
	if err == nil && len(lteEnbBytes) > 0 && lteEnbBytes[0] == 1 {
		st.LTEManualEnabled = true
		lteBitmaps, err := googGetNVArray("!SAEL3.Manual.Enabled.RFBands.BitMap")
		if err == nil && len(lteBitmaps) > 0 {
			fullBits := make([]byte, 32)
			for i, chunk := range lteBitmaps {
				if i < 4 && len(chunk) >= 8 {
					copy(fullBits[i*8:], chunk[:8])
				}
			}
			st.ActiveLTEBands = bitmapToBands(fullBits)
		}
	} else {
		st.LTEManualEnabled = false
		st.ActiveLTEBands = st.SupportedLTE
	}

	// 4. Query 5G NR SA manual bands
	if st.Mode != "nsa" && st.Mode != "lte_only" {
		cntBytes, err := googGetNV("!NRRRC.NUM_MANUAL_NR_BAND_LIST", 0)
		if err == nil && len(cntBytes) >= 2 {
			count := int(cntBytes[0]) | (int(cntBytes[1]) << 8)
			if count > 0 {
				st.SAManualEnabled = true
				saList, err := googGetNVArray("!NRRRC.MANUAL_NR_BAND_LIST")
				if err == nil {
					for i, b := range saList {
						if i >= count {
							break
						}
						if len(b) >= 2 {
							val := int(b[0]) | (int(b[1]) << 8)
							if val > 0 && val <= 2048 {
								st.ActiveSABands = append(st.ActiveSABands, val)
							}
						}
					}
				}
			}
		}
	}
	if len(st.ActiveSABands) == 0 {
		st.ActiveSABands = st.SupportedSA
	}

	// 5. Query 5G NR NSA manual bands
	if st.Mode != "sa" && st.Mode != "lte_only" {
		nsaEnbBytes, err := googGetNV("!LTE.NR Manual Band Enable/Disable", 0)
		if err == nil && len(nsaEnbBytes) > 0 && nsaEnbBytes[0] != 0 {
			st.NSAManualEnabled = true
			nsaList, err := googGetNVArray("!LTE.NR Manual Band List")
			if err == nil {
				for _, b := range nsaList {
					if len(b) >= 2 {
						val := int(b[0]) | (int(b[1]) << 8)
						if val == 0 {
							break
						}
						if val > 0 && val <= 2048 {
							st.ActiveNSABands = append(st.ActiveNSABands, val)
						}
					}
				}
			}
		}
	}
	if len(st.ActiveNSABands) == 0 {
		st.ActiveNSABands = st.SupportedSA
	}

	// 6. Query WCDMA 3G
	wcdmaDis, err := googGetNV("UL3.Etc.disabled_band", 0)
	if err == nil && len(wcdmaDis) >= 4 {
		if wcdmaDis[0] == 0xFF && wcdmaDis[1] == 0xFF && wcdmaDis[2] == 0xFF && wcdmaDis[3] == 0xFF {
			st.WCDMAManualEnabled = true
			st.ActiveWCDMABands = []int{} // disabled completely
		} else {
			maxB, _ := googGetNV("UL3.Etc.max_band", 0)
			if len(maxB) > 0 && maxB[0] >= 1 && maxB[0] <= 19 {
				st.WCDMAManualEnabled = true
				camped, err := googGetNVArray("UL3.Etc.Storing Last Camped Bands")
				if err == nil {
					for i, b := range camped {
						if i >= int(maxB[0]) {
							break
						}
						if len(b) >= 1 && b[0] >= 1 && b[0] <= 19 {
							st.ActiveWCDMABands = append(st.ActiveWCDMABands, int(b[0]))
						}
					}
				}
			} else {
				st.WCDMAManualEnabled = false
				st.ActiveWCDMABands = st.SupportedWCDMA
			}
		}
	} else {
		st.ActiveWCDMABands = st.SupportedWCDMA
	}

	// 7. Query GSM 2G
	gsmCfg, err := googGetNV("GL3.Edge_Band_Config", 0)
	if err == nil && len(gsmCfg) >= 1 {
		val := gsmCfg[0]
		st.ActiveGSMBands = make([]int, 0)
		if val&0x01 != 0 {
			st.ActiveGSMBands = append(st.ActiveGSMBands, 850)
		}
		if val&0x02 != 0 {
			st.ActiveGSMBands = append(st.ActiveGSMBands, 900)
		}
		if val&0x04 != 0 {
			st.ActiveGSMBands = append(st.ActiveGSMBands, 1800)
		}
		if val&0x08 != 0 {
			st.ActiveGSMBands = append(st.ActiveGSMBands, 1900)
		}
		st.GSMManualEnabled = val != 0x0F
	} else {
		st.ActiveGSMBands = st.SupportedGSM
	}

	// 8. VoNR state
	vonrBytes, err := googGetNV("OEM_GFEATURE_QMS_VONR_SAVE_AP_VALUE", 0)
	if err == nil && len(vonrBytes) >= 1 {
		if vonrBytes[0] == 0x01 {
			st.VoNRState = 1
		} else if vonrBytes[0] == 0x00 {
			st.VoNRState = 0
		} else if vonrBytes[0] == 0xFF {
			st.VoNRState = -1
		}
	}

	// 9. Active RAT list
	if len(st.ActiveGSMBands) > 0 {
		st.ActiveRATs = append(st.ActiveRATs, "GSM")
	}
	if len(st.ActiveWCDMABands) > 0 {
		st.ActiveRATs = append(st.ActiveRATs, "WCDMA")
	}
	if len(st.ActiveLTEBands) > 0 {
		st.ActiveRATs = append(st.ActiveRATs, "LTE")
	}
	if st.Mode != "lte_only" && (len(st.ActiveSABands) > 0 || len(st.ActiveNSABands) > 0) {
		st.ActiveRATs = append(st.ActiveRATs, "NR")
	}

	return st
}

// ApplyShannonSettings applies comprehensive band, mode, RAT, and VoNR settings via NV and instant DM menu reload.
func ApplyShannonSettings(req LockRequest) error {
	mu.Lock()
	defer mu.Unlock()

	if _, err := os.Stat(RouterPath); err != nil {
		return fmt.Errorf("shannon router %s not found on this device", RouterPath)
	}

	hasChanges := false
	applyMode := 0 // 0: SA+NSA, 1: NSA, 2: SA, 3: LTE only

	// 1. LTE Bands
	if req.ApplyLTE {
		if req.LTEAll {
			// Auto / Default
			if err := googSetNV("!SAEL3.Manual.Band.Select Enb/ Dis", 0, []byte{0}); err != nil {
				return fmt.Errorf("failed to reset LTE bands to auto: %w", err)
			}
		} else {
			bitmap := bandsToBitmap256(req.LTEBands)
			// Clear dummy band 255 from user selection
			bitmap[31] &^= (1 << 7)
			if len(req.LTEBands) == 0 {
				// LTE off: lock to dummy band 255 so modem does not panic
				bitmap[31] |= (1 << 7)
			}
			for i := 0; i < 4; i++ {
				if err := googSetNV("!SAEL3.Manual.Enabled.RFBands.BitMap", i, bitmap[i*8:(i+1)*8]); err != nil {
					return fmt.Errorf("failed to set LTE band bitmap index %d: %w", i, err)
				}
			}
			if err := googSetNV("!SAEL3.Manual.Band.Select Enb/ Dis", 0, []byte{1}); err != nil {
				return fmt.Errorf("failed to enable manual LTE bands: %w", err)
			}
		}
		hasChanges = true
	}

	// 2. 5G NR SA Bands
	if req.ApplySA && !req.SAAll && len(req.SABands) > 0 {
		count := len(req.SABands)
		if count > 60 {
			count = 60
		}
		for i := 0; i < count; i++ {
			b := req.SABands[i]
			val := []byte{byte(b & 0xFF), byte((b >> 8) & 0xFF)}
			_ = googSetNV("!NRRRC.MANUAL_NR_BAND_LIST", i, val)
		}
		// Clear leftover
		for i := count; i < 60; i++ {
			_ = googSetNV("!NRRRC.MANUAL_NR_BAND_LIST", i, []byte{0, 0})
		}
		cntBytes := []byte{byte(count & 0xFF), byte((count >> 8) & 0xFF)}
		_ = googSetNV("!NRRRC.NUM_MANUAL_NR_BAND_LIST", 0, cntBytes)
		hasChanges = true
	} else if req.ApplySA && req.SAAll {
		_ = googSetNV("!NRRRC.NUM_MANUAL_NR_BAND_LIST", 0, []byte{0, 0})
		hasChanges = true
	}

	// 3. 5G NR NSA Bands
	if req.ApplyNSA && !req.NSAAll && len(req.NSABands) > 0 {
		count := len(req.NSABands)
		if count > 60 {
			count = 60
		}
		for i := 0; i < count; i++ {
			b := req.NSABands[i]
			val := []byte{byte(b & 0xFF), byte((b >> 8) & 0xFF)}
			_ = googSetNV("!LTE.NR Manual Band List", i, val)
		}
		// Null terminate
		_ = googSetNV("!LTE.NR Manual Band List", count, []byte{0, 0})
		_ = googSetNV("!LTE.NR Manual Band Enable/Disable", 0, []byte{1})
		hasChanges = true
	} else if req.ApplyNSA && req.NSAAll {
		_ = googSetNV("!LTE.NR Manual Band Enable/Disable", 0, []byte{0})
		hasChanges = true
	}

	// 4. WCDMA 3G Bands
	if req.ApplyWCDMA {
		if req.WCDMAAll {
			_ = googSetNV("UL3.Etc.disabled_band", 0, []byte{0, 0, 0, 0})
			_ = googSetNV("UL3.Etc.max_band", 0, []byte{0xFF})
		} else if len(req.WCDMABands) == 0 {
			// Disable WCDMA
			_ = googSetNV("UL3.Etc.disabled_band", 0, []byte{0xFF, 0xFF, 0xFF, 0xFF})
		} else {
			_ = googSetNV("UL3.Etc.disabled_band", 0, []byte{0xFF, 0xFF, 0xFF, 0xFF})
			for idx, b := range req.WCDMABands {
				_ = googSetNV("UL3.Etc.Storing Last Camped Bands", idx, []byte{byte(b)})
			}
			_ = googSetNV("UL3.Etc.max_band", 0, []byte{byte(len(req.WCDMABands))})
			_ = googSetNV("UL3.Etc.disabled_band", 0, []byte{0, 0, 0, 0})
		}
		hasChanges = true
	}

	// 5. GSM 2G Bands
	if req.ApplyGSM {
		var val byte
		if req.GSMAll {
			val = 0x0F
		} else {
			for _, b := range req.GSMBands {
				switch b {
				case 850:
					val |= 0x01
				case 900:
					val |= 0x02
				case 1800:
					val |= 0x04
				case 1900:
					val |= 0x08
				}
			}
		}
		_ = googSetNV("GL3.Edge_Band_Config", 0, []byte{val})
		_ = googSetNV("GL3.Operator Specific Band", 0, []byte{val})
		hasChanges = true
	}

	// 6. VoNR
	if req.ApplyVoNR {
		apVal := byte(0xFF)
		plmnVal := byte(0x00)
		if req.VoNRState == 1 {
			apVal = 0x01
			plmnVal = 0x01
		} else if req.VoNRState == 0 {
			apVal = 0x00
			plmnVal = 0x00
		}
		_ = googSetNV("OEM_GFEATURE_QMS_VONR_SAVE_AP_VALUE", 0, []byte{apVal})
		_ = googSetNV("OMC.BASED.VONR.PLMN.ENABLE", 0, []byte{plmnVal})
		hasChanges = true
	}

	// 7. RAT Lock toggle
	if req.ApplyRAT && len(req.RATs) > 0 {
		hasGSM := sliceContains(req.RATs, "GSM")
		hasWCDMA := sliceContains(req.RATs, "WCDMA")
		hasLTE := sliceContains(req.RATs, "LTE")
		hasNR := sliceContains(req.RATs, "NR")

		if !hasGSM {
			_ = googSetNV("GL3.Edge_Band_Config", 0, []byte{0x00})
			_ = googSetNV("GL3.Operator Specific Band", 0, []byte{0x00})
		}
		if !hasWCDMA {
			_ = googSetNV("UL3.Etc.disabled_band", 0, []byte{0xFF, 0xFF, 0xFF, 0xFF})
		}
		if !hasLTE {
			// Dummy band 255
			dummyBitmap := make([]byte, 32)
			dummyBitmap[31] |= (1 << 7)
			for i := 0; i < 4; i++ {
				_ = googSetNV("!SAEL3.Manual.Enabled.RFBands.BitMap", i, dummyBitmap[i*8:(i+1)*8])
			}
			_ = googSetNV("!SAEL3.Manual.Band.Select Enb/ Dis", 0, []byte{1})
		}
		if !hasNR {
			applyMode = 3 // LTE only
		}
		hasChanges = true
	}

	// 8. NR Mode
	if req.ApplyMode {
		switch req.Mode {
		case "nsa":
			applyMode = 1
		case "sa":
			applyMode = 2
		case "lte_only":
			applyMode = 3
		default:
			applyMode = 0
		}
		hasChanges = true
	}

	// 9. Instant CFUN-Free Modem Menu Apply via /dev/umts_dm0
	if hasChanges {
		if _, err := os.Stat(DMPath); err == nil {
			if err := menuSetNRMode(applyMode); err != nil {
				logger.Get().Warnf("Shannon", "DM modem menu trigger returned error: %v, falling back to AT+CFUN cycle", err)
				_, _ = executeATCmd("AT+CFUN=0")
				time.Sleep(500 * time.Millisecond)
				_, _ = executeATCmd("AT+CFUN=1")
			} else {
				logger.Get().Infof("Shannon", "Successfully reloaded radio via DM menu keypresses (Mode: %d, zero CFUN cycle)", applyMode)
			}
		} else {
			// Fallback if DM interface is absent
			_, _ = executeATCmd("AT+CFUN=0")
			time.Sleep(500 * time.Millisecond)
			_, _ = executeATCmd("AT+CFUN=1")
		}
	}

	return nil
}

// LockBands maintains backward compatibility with older controller call signatures.
func LockBands(slot int, mode string, lteBands []int, nrBands []int) error {
	req := LockRequest{
		Slot:      slot,
		Mode:      mode,
		ApplyMode: mode != "",
		LTEBands:  lteBands,
		ApplyLTE:  len(lteBands) > 0,
		SABands:   nrBands,
		ApplySA:   len(nrBands) > 0,
		NSABands:  nrBands,
		ApplyNSA:  len(nrBands) > 0,
	}
	return ApplyShannonSettings(req)
}

// ResetBands disables manual band locking and restores factory band allow-lists.
func ResetBands(slot int) error {
	req := LockRequest{
		Slot:       slot,
		Mode:       "both",
		ApplyMode:  true,
		ApplyLTE:   true,
		LTEAll:     true,
		ApplySA:    true,
		SAAll:      true,
		ApplyNSA:   true,
		NSAAll:     true,
		ApplyWCDMA: true,
		WCDMAAll:   true,
		ApplyGSM:   true,
		GSMAll:     true,
		ApplyVoNR:  true,
		VoNRState:  -1, // Default carrier policy
	}
	return ApplyShannonSettings(req)
}

// menuSetNRMode drives the Shannon service menu over /dev/umts_dm0 to trigger instant radio reload.
func menuSetNRMode(mode int) error {
	menuKey := byte('4') // SA+NSA
	switch mode {
	case 1:
		menuKey = '3' // NSA
	case 2:
		menuKey = '5' // SA
	case 3:
		menuKey = '2' // LTE only / disabled
	}

	// Execute via root script using direct sysfs writes or shannon binary
	dmScript := fmt.Sprintf(`
python3 -c '
import os, time
dm = os.open("%s", os.O_RDWR)
def w(b): os.write(dm, bytes(b))
def k(key):
    w([0x7f, 0x0f, 0, 0, 0x0c, 0, 0, 0, 0xa0, 0x08, 0x09, 0, 0, 0, 0, key, 0x7e])
    time.sleep(0.05)

w([0x7f, 0x12, 0, 0, 0x0f, 0, 0, 0, 0xa0, 0, 0x52, 0, 0, 0, 0, 1, 0, 0, 0, 0x7e])
w([0x7f, 0x16, 0, 0, 0x13, 0, 0, 0, 0xa0, 0, 0, 0, 0, 0, 0, 0x34, 0xdc, 0x12, 0xfe, 0, 0, 0, 0xc0, 0x7e])
w([0x7f, 0x0e, 0, 0, 0x0b, 0, 0, 0, 0xa0, 0, 0x72, 0, 0, 0, 0, 0x7e])
w([0x7f, 0x10, 0, 0, 0x0d, 0, 0, 0, 0xa0, 0, 0x06, 0, 0, 0, 0, 3, 3, 0x7e])
w([0x7f, 0x0f, 0, 0, 0x0c, 0, 0, 0, 0xa0, 0, 0x10, 0, 0, 0, 0, 0xff, 0x7e])
w([0x7f, 0x0f, 0, 0, 0x0c, 0, 0, 0, 0xa0, 0, 0x20, 0, 0, 0, 0, 0xff, 0x7e])
w([0x7f, 0x0f, 0, 0, 0x0c, 0, 0, 0, 0xa0, 0, 0x30, 0, 0, 0, 0, 0xff, 0x7e])
w([0x7f, 0x0f, 0, 0, 0x0c, 0, 0, 0, 0xa0, 0, 0x40, 0, 0, 0, 0, 0xff, 0x7e])
w([0x7f, 0x0f, 0, 0, 0x0c, 0, 0, 0, 0xa0, 0, 0x44, 0, 0, 0, 0, 0xff, 0x7e])
w([0x7f, 0x1b, 0, 0, 0x18, 0, 0, 0, 0xa0, 0, 0xa0, 0, 0, 0, 0, 0, 0x3e, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f, 0x7e])
w([0x7f, 0x12, 0, 0, 0x0f, 0, 0, 0, 0xa0, 0, 0x90, 0, 0, 0, 0, 0, 0, 0, 0, 0x7e])
w([0x7f, 0x12, 0, 0, 0x0f, 0, 0, 0, 0xa0, 0, 0x9a, 0, 0, 0, 0, 1, 1, 0x16, 0x10, 0x7e])

k(ord("1"))
k(ord("3"))
k(%d)
k(0x5c)
k(0x5c)
os.close(dm)
' 2>/dev/null || true
`, DMPath, menuKey)

	out, err := config.ExecSuTimeout(3*time.Second, dmScript)
	if err != nil {
		return err
	}
	_ = out
	return nil
}

// googGetNV queries a single Shannon NV item via AT+GOOGGETNV.
func googGetNV(name string, index int) ([]byte, error) {
	cmd := fmt.Sprintf("AT+GOOGGETNV=\"%s\",%d", name, index)
	resp, err := executeATCmd(cmd)
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "+GOOGGETNV:") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				hexData := parts[3]
				return parseHexCSV(hexData), nil
			}
		}
	}
	return nil, fmt.Errorf("no NV data returned for %s", name)
}

// googGetNVArray queries multi-element NV array items (e.g. format 1).
func googGetNVArray(name string) ([][]byte, error) {
	cmd := fmt.Sprintf("AT+GOOGGETNV=\"%s\",1", name)
	resp, err := executeATCmd(cmd)
	if err != nil {
		return nil, err
	}

	var results [][]byte
	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "+GOOGGETNV:") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				hexData := parts[len(parts)-2]
				bytes := parseHexCSV(hexData)
				if len(bytes) > 0 {
					results = append(results, bytes)
				}
			}
		}
	}
	return results, nil
}

// googSetNV sets a Shannon NV item via AT+GOOGSETNV.
func googSetNV(name string, index int, data []byte) error {
	hexStr := formatHexCSV(data)
	cmd := fmt.Sprintf("AT+GOOGSETNV=\"%s\",%d,\"%s\"", name, index, hexStr)
	resp, err := executeATCmd(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(resp, "ERROR") {
		return fmt.Errorf("NV set rejected: %s", resp)
	}
	return nil
}

// executeATCmd opens /dev/umts_router via root subshell and reads response.
func executeATCmd(cmd string) (string, error) {
	cleanCmd := strings.TrimSpace(cmd)
	if cleanCmd == "" {
		return "", fmt.Errorf("empty AT command")
	}

	execStr := fmt.Sprintf(
		"echo -e '%s\\r' > %s && timeout 2 cat %s 2>/dev/null | head -n 30",
		cleanCmd, RouterPath, RouterPath,
	)

	out, err := config.ExecSuTimeout(atTimeout, execStr)
	res := strings.TrimSpace(string(out))
	if err != nil && res == "" {
		return "", err
	}
	return res, nil
}

func parseHexCSV(csvStr string) []byte {
	var bytes []byte
	parts := strings.Split(csvStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) == 0 {
			continue
		}
		if val, err := strconv.ParseUint(p, 16, 8); err == nil {
			bytes = append(bytes, byte(val))
		}
	}
	return bytes
}

func formatHexCSV(data []byte) string {
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ",")
}

func bitmapToBands(bitmap []byte) []int {
	var bands []int
	for byteIdx, b := range bitmap {
		for bit := 0; bit < 8; bit++ {
			if (b & (1 << bit)) != 0 {
				bandNum := byteIdx*8 + bit + 1
				if bandNum != LteDummyBand {
					bands = append(bands, bandNum)
				}
			}
		}
	}
	return bands
}

func bandsToBitmap256(bands []int) []byte {
	bitmap := make([]byte, 32)
	for _, b := range bands {
		if b >= 1 && b <= 256 {
			x := b - 1
			bitmap[x/8] |= (1 << (x % 8))
		}
	}
	return bitmap
}

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return true
		}
	}
	return false
}
