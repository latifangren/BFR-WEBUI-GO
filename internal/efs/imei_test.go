package efs

import (
	"encoding/hex"
	"testing"
)

func TestLuhnAndNormalize(t *testing.T) {
	// Example valid IMEI: 353456789012345
	// Check Luhn for first 14 digits: "35345678901234"
	core := "35345678901234"
	check := CalculateLuhn(core)
	expectedFull := core + string(check)

	if !ValidateIMEI(expectedFull) {
		t.Fatalf("Expected %s to be valid Luhn IMEI", expectedFull)
	}

	norm, err := NormalizeIMEI(core)
	if err != nil {
		t.Fatalf("NormalizeIMEI failed for 14 digits: %v", err)
	}
	if norm != expectedFull {
		t.Fatalf("Expected normalized %s, got %s", expectedFull, norm)
	}

	// Invalid Luhn
	invalidFull := core + string((check-'0'+1)%10+'0')
	if ValidateIMEI(invalidFull) {
		t.Fatalf("Expected %s to be invalid Luhn IMEI", invalidFull)
	}
}

func TestEncodeIMEIToBCD(t *testing.T) {
	imei := "864201041234560" // Valid IMEI or let's compute check digit
	norm, err := NormalizeIMEI(imei[:14])
	if err != nil {
		t.Fatalf("NormalizeIMEI error: %v", err)
	}

	bcd, err := EncodeIMEIToBCD(norm)
	if err != nil {
		t.Fatalf("EncodeIMEIToBCD error: %v", err)
	}

	if len(bcd) != 9 {
		t.Fatalf("Expected 9 bytes, got %d", len(bcd))
	}
	if bcd[0] != 0x08 {
		t.Fatalf("Expected prefix 0x08, got 0x%02X", bcd[0])
	}

	// Verify decoding with decodeBCDNibbles
	decoded := decodeBCDNibbles(bcd, true)
	if decoded != norm {
		t.Fatalf("Decoded mismatch: expected %s, got %s (hex=%s)", norm, decoded, hex.EncodeToString(bcd))
	}
}

func TestDecodeBCDNibblesZerosAndEdgeCases(t *testing.T) {
	// IMEI containing consecutive zero digits (e.g. 35005678901234)
	norm, err := NormalizeIMEI("35005678901234")
	if err != nil {
		t.Fatalf("NormalizeIMEI failed: %v", err)
	}
	bcd, err := EncodeIMEIToBCD(norm)
	if err != nil {
		t.Fatalf("EncodeIMEIToBCD failed: %v", err)
	}

	decoded := decodeBCDNibbles(bcd, true)
	if decoded != norm {
		t.Fatalf("Decoded mismatch for zeros: expected %s, got %s (hex=%s)", norm, decoded, hex.EncodeToString(bcd))
	}

	// Buffer containing all 0x00 should return "Not Set"
	allZeros := make([]byte, 16)
	if got := decodeBCDNibbles(allZeros, true); got != "Not Set" {
		t.Fatalf("Expected 'Not Set' for all 0x00, got %s", got)
	}

	// Buffer containing all 0xFF should return "Not Set"
	allFF := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	if got := decodeBCDNibbles(allFF, true); got != "Not Set" {
		t.Fatalf("Expected 'Not Set' for all 0xFF, got %s", got)
	}
}
