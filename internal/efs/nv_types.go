package efs

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// NVReadResult represents the result of reading a single NV item.
type NVReadResult struct {
	ID        uint16     `json:"id"`
	Name      string     `json:"name"`
	Category  string     `json:"category"`
	Index     uint16     `json:"index,omitempty"`
	DataHex   string     `json:"data_hex"`
	Formatted string     `json:"formatted"`
	DataType  NVDataType `json:"data_type"`
	Size      int        `json:"size"`
	Success   bool       `json:"success"`
	Error     string     `json:"error,omitempty"`
}

// NVWriteResult represents the result of writing a single NV item.
type NVWriteResult struct {
	ID      uint16 `json:"id"`
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// NVBackupEntry is one item stored in a backup file.
type NVBackupEntry struct {
	ID         uint16 `json:"id"`
	Name       string `json:"name"`
	Index      uint16 `json:"index,omitempty"`
	IsExtended bool   `json:"is_extended,omitempty"`
	DataHex    string `json:"data_hex"`
}

// NVBackupFile is the JSON schema for NV backup files.
type NVBackupFile struct {
	Version   string          `json:"version"`
	Timestamp string          `json:"timestamp"`
	Total     int             `json:"total_items"`
	Items     []NVBackupEntry `json:"items"`
}

// NVBackupProgress reports progress during backup/restore operations.
type NVBackupProgress struct {
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	ItemName  string `json:"item_name"`
	Success   int    `json:"success"`
	Failed    int    `json:"failed"`
	Skipped   int    `json:"skipped"`
	Done      bool   `json:"done"`
	Error     string `json:"error,omitempty"`
}

// ─── Formatters ──────────────────────────────────────────────────────────────

// FormatNVValue converts raw NV bytes into a human-readable string.
// Handles IMEI (BCD), IMSI/MDN (BCD), ESN/MEID (hex), SPC (ASCII), and generic types.
func FormatNVValue(id uint16, data []byte) string {
	if len(data) == 0 {
		return "[No Data]"
	}

	info := GetNVItem(id)
	name := ""
	dt := NVDataHex
	if info != nil {
		name = strings.ToLower(info.Name)
		dt = info.DataType
	}

	// IMEI — BCD packed, length prefix at byte[0]
	if strings.Contains(name, "imei") || id == 550 || id == 2497 || id == 5014 {
		return decodeBCDNibbles(data, true)
	}

	// IMSI / MDN / MIN — BCD packed
	if strings.Contains(name, "imsi") || strings.Contains(name, "mdn") ||
		strings.Contains(name, "min") || dt == NVDataBCD {
		return decodeBCDNibbles(data, false)
	}

	// ESN / MEID — big-endian hex display
	if strings.Contains(name, "esn") || strings.Contains(name, "meid") {
		return "0x" + strings.ToUpper(hex.EncodeToString(reverseBytes(trimTrailingZero(data))))
	}

	// SPC — 6 ASCII digit code
	if strings.Contains(name, "spc") {
		return formatSPC(data)
	}

	// MCC / MNC — little-endian uint16
	if id == 176 || id == 177 || name == "mcc" || name == "mnc" {
		if len(data) >= 2 {
			val := binary.LittleEndian.Uint16(data[:2])
			if val == 0 || val == 0xFFFF {
				return "Not Set"
			}
			return fmt.Sprintf("%d", val)
		}
	}

	// ASCII / string types
	if dt == NVDataASCII || strings.Contains(name, "ver") || strings.Contains(name, "banner") {
		s := extractASCII(data)
		if s != "" {
			return fmt.Sprintf("%q", s)
		}
	}

	// Numeric fallbacks by byte width
	switch len(data) {
	case 1:
		v := data[0]
		return fmt.Sprintf("%d (0x%02X)", v, v)
	case 2:
		v := binary.LittleEndian.Uint16(data)
		return fmt.Sprintf("%d (0x%04X)", v, v)
	case 4:
		v := binary.LittleEndian.Uint32(data)
		return fmt.Sprintf("%d (0x%08X)", v, v)
	case 8:
		v := binary.LittleEndian.Uint64(data)
		return fmt.Sprintf("%d (0x%016X)", v, v)
	}

	// Hex dump fallback
	h := strings.ToUpper(hex.EncodeToString(data))
	if len(h) > 64 {
		h = h[:61] + "..."
	}
	return fmt.Sprintf("[%dB] %s", len(data), h)
}

// decodeBCDNibbles decodes a BCD-packed byte slice into a digit string.
// If hasLengthPrefix is true, byte[0] is treated as a length hint and skipped.
func decodeBCDNibbles(data []byte, hasLengthPrefix bool) string {
	start := 0
	if hasLengthPrefix && len(data) > 1 && data[0] > 0 && data[0] < 16 {
		start = 1
	}
	var sb strings.Builder
	for i := start; i < len(data); i++ {
		b := data[i]
		if b == 0x00 || b == 0xFF {
			continue
		}
		lo := b & 0x0F
		hi := (b >> 4) & 0x0F
		if lo <= 9 {
			sb.WriteByte('0' + lo)
		}
		if hi <= 9 {
			sb.WriteByte('0' + hi)
		}
	}
	if sb.Len() == 0 {
		return "Not Set"
	}
	return sb.String()
}

func formatSPC(data []byte) string {
	var sb strings.Builder
	for i := 0; i < len(data) && i < 6; i++ {
		c := data[i]
		if c >= '0' && c <= '9' {
			sb.WriteByte(c)
		} else {
			sb.WriteByte('0' + (c % 10))
		}
	}
	for sb.Len() < 6 {
		sb.WriteByte('0')
	}
	return fmt.Sprintf("%q", sb.String())
}

func extractASCII(data []byte) string {
	var sb strings.Builder
	for _, c := range data {
		if c == 0 {
			break
		}
		if c >= 32 && c <= 126 {
			sb.WriteByte(c)
		}
	}
	return strings.TrimSpace(sb.String())
}

func reverseBytes(b []byte) []byte {
	out := make([]byte, len(b))
	for i, v := range b {
		out[len(b)-1-i] = v
	}
	return out
}

func trimTrailingZero(b []byte) []byte {
	end := len(b)
	for end > 0 && b[end-1] == 0 {
		end--
	}
	if end == 0 {
		return b[:1]
	}
	return b[:end]
}

func isAllZeroHex(h string) bool {
	for _, c := range h {
		if c != '0' {
			return false
		}
	}
	return true
}

// ─── Backup file helpers ──────────────────────────────────────────────────────

func writeBackupFile(filename string, entries []NVBackupEntry) error {
	backup := NVBackupFile{
		Version:   "1.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Total:     len(entries),
		Items:     entries,
	}
	b, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup: %w", err)
	}
	if err := os.WriteFile(filename, b, 0o644); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}
	return nil
}
