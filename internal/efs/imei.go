package efs

import (
	"fmt"
	"regexp"
	"strings"
)

// IMEIInfo holds current IMEI numbers read from the modem.
type IMEIInfo struct {
	IMEI1   string `json:"imei1"`
	IMEI2   string `json:"imei2"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// IMEIWriteReq defines parameters for an IMEI write operation.
type IMEIWriteReq struct {
	Slot           int    `json:"slot"`             // 0 for SIM 1 (NV 550), 1 for SIM 2 (NV 2497 / 5014)
	IMEI           string `json:"imei"`             // 14 or 15 digit IMEI string
	Method         string `json:"method"`           // "auto", "normal", "efs"
	UseOfflineMode bool   `json:"use_offline_mode"` // If true, switches modem to Offline-D (0x29) during write
	RebootModem    bool   `json:"reboot_modem"`     // If true, issues SSR / reset after successful write
}

// IMEIWriteResult contains the detailed execution log and outcome of an IMEI write operation.
type IMEIWriteResult struct {
	Success    bool     `json:"success"`
	Slot       int      `json:"slot"`
	IMEI       string   `json:"imei"`
	MethodUsed string   `json:"method_used"`
	StepLogs   []string `json:"step_logs"`
	Error      string   `json:"error,omitempty"`
}

var imeiDigitsRegex = regexp.MustCompile(`^[0-9]{14,15}$`)

// CalculateLuhn calculates the Luhn check digit for a 14-digit IMEI string.
func CalculateLuhn(core14 string) byte {
	if len(core14) != 14 {
		return '0'
	}
	sum := 0
	for i := 0; i < 14; i++ {
		d := int(core14[i] - '0')
		if i%2 == 1 { // Doubling odd index positions (2nd, 4th, 6th, ..., 14th digit in 1-based index)
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	check := (10 - (sum % 10)) % 10
	return byte('0' + check)
}

// ValidateIMEI checks whether a given string is a valid 14 or 15 digit IMEI with correct Luhn check digit.
func ValidateIMEI(imei string) bool {
	imei = strings.TrimSpace(imei)
	if !imeiDigitsRegex.MatchString(imei) {
		return false
	}
	if len(imei) == 14 {
		return true
	}
	expected := CalculateLuhn(imei[:14])
	return imei[14] == expected
}

// NormalizeIMEI validates and normalizes an input IMEI string into a full 15-digit IMEI.
// If 14 digits are provided, the 15th Luhn digit is automatically computed and appended.
func NormalizeIMEI(input string) (string, error) {
	imei := strings.TrimSpace(input)
	if !imeiDigitsRegex.MatchString(imei) {
		return "", fmt.Errorf("IMEI must be 14 or 15 digits (got %d chars)", len(imei))
	}
	if len(imei) == 14 {
		check := CalculateLuhn(imei)
		return imei + string(check), nil
	}
	expected := CalculateLuhn(imei[:14])
	if imei[14] != expected {
		return "", fmt.Errorf("invalid Luhn checksum for IMEI: expected '%c', got '%c'", expected, imei[14])
	}
	return imei, nil
}

// EncodeIMEIToBCD converts a 15-digit IMEI into Qualcomm 3GPP TS 24.008 BCD format (9 bytes).
// Format:
//   byte[0] = 0x08 (Length prefix)
//   byte[1] = ((digit[0] << 4) | 0x0A) // Type 4 Identity + first digit
//   byte[2] = ((digit[2] << 4) | digit[1])
//   byte[3] = ((digit[4] << 4) | digit[3])
//   byte[4] = ((digit[6] << 4) | digit[5])
//   byte[5] = ((digit[8] << 4) | digit[7])
//   byte[6] = ((digit[10] << 4) | digit[9])
//   byte[7] = ((digit[12] << 4) | digit[11])
//   byte[8] = ((digit[14] << 4) | digit[13])
func EncodeIMEIToBCD(imei string) ([]byte, error) {
	norm, err := NormalizeIMEI(imei)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 9)
	out[0] = 0x08
	out[1] = ((norm[0] - '0') << 4) | 0x0A

	for i := 0; i < 7; i++ {
		dLow := norm[1+i*2] - '0'
		dHigh := norm[2+i*2] - '0'
		out[2+i] = (dHigh << 4) | dLow
	}

	return out, nil
}
