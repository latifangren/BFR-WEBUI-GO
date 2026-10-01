package shannon

import (
	"os"
	"testing"
)

func TestParseShannonNrUECapSub62(t *testing.T) {
	testFilePath := "F:/GITHUB/uecapabilityparser/src/test/resources/shannon/input/sub6_2.binarypb"
	data, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Skipf("Test file not found at %s, skipping: %v", testFilePath, err)
		return
	}

	doc, err := ParseShannonNrUECap(data)
	if err != nil {
		t.Fatalf("Failed to parse ShannonNrUECap: %v", err)
	}

	if doc.TotalCombos == 0 {
		t.Fatalf("Expected combos, got 0")
	}

	t.Logf("Parsed %d combos (Version: %d, ID: %d, NRCA: %d, ENDC: %d, NRDC: %d)",
		doc.TotalCombos, doc.Version, doc.ID, doc.NrcaCount, doc.EndcCount, doc.NrdcCount)

	for i, c := range doc.Combos {
		t.Logf("Combo #%d [%s]: %s (Total DL Streams: %d, BCS NR: %d)",
			i+1, c.Type, c.ComboString, c.TotalDlStreams, c.BCSNr)
	}
}

func TestParseShannonNrUECapSub6(t *testing.T) {
	testFilePath := "F:/GITHUB/uecapabilityparser/src/test/resources/shannon/input/sub6.binarypb"
	data, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Skipf("Test file not found at %s, skipping: %v", testFilePath, err)
		return
	}

	doc, err := ParseShannonNrUECap(data)
	if err != nil {
		t.Fatalf("Failed to parse ShannonNrUECap: %v", err)
	}

	if doc.TotalCombos == 0 {
		t.Fatalf("Expected combos, got 0")
	}

	t.Logf("Sub6: Parsed %d combos (Version: %d, ID: %d, ENDC: %d, NRCA: %d, NRDC: %d)",
		doc.TotalCombos, doc.Version, doc.ID, doc.EndcCount, doc.NrcaCount, doc.NrdcCount)

	if len(doc.Combos) > 0 {
		t.Logf("First combo: %s (Type: %s, Streams: %d)", doc.Combos[0].ComboString, doc.Combos[0].Type, doc.Combos[0].TotalDlStreams)
	}
}

func TestParseShannonNrUECapPixel9(t *testing.T) {
	testFilePath := "F:/GITHUB/uecapabilityparser/src/test/resources/shannon/input/pixel9.binarypb"
	data, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Skipf("Test file not found at %s, skipping: %v", testFilePath, err)
		return
	}

	doc, err := ParseShannonNrUECap(data)
	if err != nil {
		t.Fatalf("Failed to parse Pixel 9 ShannonNrUECap: %v", err)
	}

	t.Logf("Pixel 9: Parsed %d combos (Version: %d, ID: %d, ENDC: %d, NRCA: %d, NRDC: %d)",
		doc.TotalCombos, doc.Version, doc.ID, doc.EndcCount, doc.NrcaCount, doc.NrdcCount)
}

