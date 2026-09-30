//go:build linux || android

package shannon

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

const (
	RouterPath = "/dev/umts_router"
	DMPath     = "/dev/umts_dm0"
	atTimeout  = 3 * time.Second
)

var mu sync.Mutex

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

// GetStatus checks hardware capability and current NV mode.
func GetStatus() DeviceStatus {
	mu.Lock()
	defer mu.Unlock()

	st := DeviceStatus{
		IsShannon:   IsSupported(),
		ActiveSlot:  0,
		Mode:        "both",
		ActiveLTEBands: []int{1, 3, 8},
		ActiveNRBands:  []int{1, 3, 28, 40, 77, 78},
	}

	if _, err := os.Stat(RouterPath); err == nil {
		st.RouterAvailable = true
	}
	if _, err := os.Stat(DMPath); err == nil {
		st.DMAvailable = true
	}

	// Query device model & SoC
	model, _ := config.ExecSuTimeout(time.Second, "getprop ro.product.model 2>/dev/null || true")
	st.DeviceModel = strings.TrimSpace(string(model))

	soc, _ := config.ExecSuTimeout(time.Second, "getprop ro.soc.model 2>/dev/null || true")
	st.SoC = strings.TrimSpace(string(soc))
	if st.SoC == "" {
		socMan, _ := config.ExecSuTimeout(time.Second, "getprop ro.soc.manufacturer 2>/dev/null || true")
		st.SoC = strings.TrimSpace(string(socMan))
	}

	if !st.RouterAvailable {
		return st
	}

	// Query Shannon current LTE manual band pref if available
	resp, err := executeATCmd("AT+GOOGGETNV=\"LTE.BAND_PREF.MANUAL\",0")
	if err == nil && strings.Contains(resp, "+GOOGGETNV:") {
		bands := parseHexBitmaskResponse(resp)
		if len(bands) > 0 {
			st.ActiveLTEBands = bands
		}
	}

	// Query mode
	respMode, err := executeATCmd("AT+GOOGGETNV=\"NR.CONFIG.MODE\",0")
	if err == nil {
		if strings.Contains(respMode, "\"01\"") {
			st.Mode = "nsa"
		} else if strings.Contains(respMode, "\"02\"") {
			st.Mode = "sa"
		} else if strings.Contains(respMode, "\"03\"") {
			st.Mode = "lte_only"
		} else {
			st.Mode = "both"
		}
	}

	return st
}

// LockBands applies manual frequency band locks on the Shannon modem.
func LockBands(slot int, mode string, lteBands []int, nrBands []int) error {
	mu.Lock()
	defer mu.Unlock()

	if _, err := os.Stat(RouterPath); err != nil {
		return fmt.Errorf("shannon modem interface %s not found on this device", RouterPath)
	}

	// 1. Convert LTE bands slice to hex bitmask bytes
	hexCsv := bandsToHexCsv(lteBands)

	// Set manual LTE band pref
	cmdSetLTE := fmt.Sprintf("AT+GOOGSETNV=\"LTE.BAND_PREF.MANUAL\",%d,\"%s\"", slot, hexCsv)
	if _, err := executeATCmd(cmdSetLTE); err != nil {
		logger.Get().Warnf("Shannon", "Failed to set manual LTE bands: %v", err)
	}

	// Enable manual LTE band pref (01 = enabled)
	cmdEnable := fmt.Sprintf("AT+GOOGSETNV=\"LTE.BAND_PREF.MANUAL_ENABLE\",%d,\"01\"", slot)
	if _, err := executeATCmd(cmdEnable); err != nil {
		logger.Get().Warnf("Shannon", "Failed to enable manual LTE bands: %v", err)
	}

	// 2. Set NR operation mode
	modeVal := "00" // SA + NSA
	switch mode {
	case "sa":
		modeVal = "02"
	case "nsa":
		modeVal = "01"
	case "lte_only":
		modeVal = "03"
	}
	cmdMode := fmt.Sprintf("AT+GOOGSETNV=\"NR.CONFIG.MODE\",%d,\"%s\"", slot, modeVal)
	_, _ = executeATCmd(cmdMode)

	logger.Get().Infof("Shannon", "Locked bands on slot %d (LTE: %v, Mode: %s)", slot, lteBands, mode)
	return nil
}

// ResetBands disables manual band locking and restores factory band allow-lists.
func ResetBands(slot int) error {
	mu.Lock()
	defer mu.Unlock()

	if _, err := os.Stat(RouterPath); err != nil {
		return fmt.Errorf("shannon modem interface %s not found", RouterPath)
	}

	// 00 disables manual band locking
	cmdDisable := fmt.Sprintf("AT+GOOGSETNV=\"LTE.BAND_PREF.MANUAL_ENABLE\",%d,\"00\"", slot)
	_, err := executeATCmd(cmdDisable)
	if err != nil {
		return err
	}

	// Reset NR mode to SA + NSA (00)
	cmdMode := fmt.Sprintf("AT+GOOGSETNV=\"NR.CONFIG.MODE\",%d,\"00\"", slot)
	_, _ = executeATCmd(cmdMode)

	logger.Get().Infof("Shannon", "Reset bands on slot %d to factory auto", slot)
	return nil
}

// executeATCmd opens /dev/umts_router via root subshell with stty or direct cat/echo.
func executeATCmd(cmd string) (string, error) {
	cleanCmd := strings.TrimSpace(cmd)
	if cleanCmd == "" {
		return "", fmt.Errorf("empty AT command")
	}

	// Shannon accepts carriage return terminated AT string written to /dev/umts_router
	execStr := fmt.Sprintf(
		"echo -e '%s\\r' > %s && timeout 2 cat %s 2>/dev/null | head -n 15",
		cleanCmd, RouterPath, RouterPath,
	)

	out, err := config.ExecSuTimeout(atTimeout, execStr)
	res := strings.TrimSpace(string(out))
	if err != nil && res == "" {
		return "", err
	}
	return res, nil
}

// bandsToHexCsv converts band numbers (e.g. [1, 3, 8]) to Shannon byte-oriented hex CSV (e.g. "85,00,...").
func bandsToHexCsv(bands []int) string {
	if len(bands) == 0 {
		return "00"
	}

	mask := new(big.Int)
	for _, b := range bands {
		if b > 0 && b <= 256 {
			bit := new(big.Int).Lsh(big.NewInt(1), uint(b-1))
			mask.Or(mask, bit)
		}
	}

	bytesLen := (mask.BitLen() + 7) / 8
	if bytesLen == 0 {
		bytesLen = 1
	}

	rawBytes := mask.Bytes()
	// Big.Int Bytes() is big-endian, reverse for Shannon little-endian byte array
	le := make([]byte, bytesLen)
	for i, b := range rawBytes {
		le[bytesLen-1-i] = b
	}

	parts := make([]string, len(le))
	for i, b := range le {
		parts[i] = fmt.Sprintf("%02X", b)
	}

	return strings.Join(parts, ",")
}

// parseHexBitmaskResponse extracts band numbers from +GOOGGETNV: ..., "01,00,04"
func parseHexBitmaskResponse(resp string) []int {
	bands := make([]int, 0)
	idx := strings.Index(resp, "\"")
	if idx == -1 {
		return bands
	}
	endIdx := strings.LastIndex(resp, "\"")
	if endIdx <= idx {
		return bands
	}

	hexData := resp[idx+1 : endIdx]
	parts := strings.Split(hexData, ",")
	for byteIdx, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) == 0 {
			continue
		}
		var val byte
		_, _ = fmt.Sscanf(p, "%02x", &val)
		for bit := 0; bit < 8; bit++ {
			if (val & (1 << bit)) != 0 {
				bandNum := byteIdx*8 + bit + 1
				bands = append(bands, bandNum)
			}
		}
	}

	return bands
}

// Dummy reference to avoid unused bytes import
var _ = bytes.TrimSpace
