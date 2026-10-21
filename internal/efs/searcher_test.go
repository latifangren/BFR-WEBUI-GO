package efs

import (
	"testing"
)

func TestLTECellPacking(t *testing.T) {
	pci := uint32(412)
	cid := uint32(12345678)

	packed := PackLTECell(pci, cid)
	unpackedPCI, unpackedCID := UnpackLTECell(packed)

	if unpackedPCI != pci {
		t.Errorf("expected PCI %d, got %d", pci, unpackedPCI)
	}
	if unpackedCID != cid {
		t.Errorf("expected CID %d, got %d", cid, unpackedCID)
	}

	formatted := FormatSearcherItem("lte", "cell", packed)
	if formatted != "PCI: 412, CID: 12345678" {
		t.Errorf("unexpected formatted output: %s", formatted)
	}
}

func TestGSMCellPacking(t *testing.T) {
	f1 := uint16(123)
	f2 := uint8(45)
	f3 := uint32(67890)
	f4 := uint16(54321)

	packed := PackGSMCell(f1, f2, f3, f4)
	uf1, uf2, uf3, uf4 := UnpackGSMCell(packed)

	if uf1 != f1 || uf2 != f2 || uf3 != f3 || uf4 != f4 {
		t.Errorf("GSM unpack mismatch: got (%d,%d,%d,%d), want (%d,%d,%d,%d)",
			uf1, uf2, uf3, uf4, f1, f2, f3, f4)
	}

	formatted := FormatSearcherItem("gsm", "cell", packed)
	if formatted != "f1: 123, f2: 45, LAC: 67890, CID: 54321" {
		t.Errorf("unexpected GSM formatted string: %s", formatted)
	}
}

func TestNRCellPacking(t *testing.T) {
	nrArfcn := uint32(640000)
	pci := uint16(234)
	scs := uint8(3) // 60kHz SCS step

	packed := PackNRCell(nrArfcn, pci, scs)
	uArfcn, uPCI, uSCS := UnpackNRCell(packed)

	if uArfcn != nrArfcn {
		t.Errorf("expected NR-ARFCN %d, got %d", nrArfcn, uArfcn)
	}
	if uPCI != pci {
		t.Errorf("expected PCI %d, got %d", pci, uPCI)
	}
	if uSCS != scs {
		t.Errorf("expected SCS %d, got %d", scs, uSCS)
	}

	formatted := FormatSearcherItem("nr", "cell", packed)
	if formatted != "NR-ARFCN: 640000, PCI: 234, SCS: 3" {
		t.Errorf("unexpected NR formatted string: %s", formatted)
	}
}

func TestSearcherPagePacking(t *testing.T) {
	entries := []uint64{
		PackLTECell(100, 200),
		PackLTECell(101, 201),
		PackLTECell(102, 202),
	}

	packed := PackSearcherPage(entries)
	if len(packed) != SearcherPageSlots*8 {
		t.Fatalf("expected packed page size %d, got %d", SearcherPageSlots*8, len(packed))
	}

	unpacked, err := UnpackSearcherPage(packed)
	if err != nil {
		t.Fatalf("UnpackSearcherPage failed: %v", err)
	}

	if len(unpacked) != len(entries) {
		t.Fatalf("expected %d unpacked entries, got %d", len(entries), len(unpacked))
	}

	for i := range entries {
		if unpacked[i] != entries[i] {
			t.Errorf("entry %d mismatch: got %d, want %d", i, unpacked[i], entries[i])
		}
	}

	// Test empty page
	emptyPacked := PackSearcherPage([]uint64{})
	emptyUnpacked, err := UnpackSearcherPage(emptyPacked)
	if err != nil {
		t.Fatalf("UnpackSearcherPage on empty page failed: %v", err)
	}
	if len(emptyUnpacked) != 0 {
		t.Errorf("expected 0 entries from empty page, got %d", len(emptyUnpacked))
	}

	// Test short buffer
	shortUnpacked, err := UnpackSearcherPage([]byte{0x01})
	if err != nil {
		t.Fatalf("UnpackSearcherPage short buffer failed: %v", err)
	}
	if len(shortUnpacked) != 0 {
		t.Errorf("expected 0 entries from short buffer, got %d", len(shortUnpacked))
	}
}
