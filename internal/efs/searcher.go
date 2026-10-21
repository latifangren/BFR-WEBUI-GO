package efs

import (
	"encoding/binary"
	"fmt"
)

// SearcherItem represents an individual channel, cell, or code rule.
type SearcherItem struct {
	RawValue  uint64 `json:"raw_value"`
	Channel   uint32 `json:"channel,omitempty"`   // ARFCN, UARFCN, EARFCN, or NR-ARFCN
	Code      uint32 `json:"code,omitempty"`      // BSIC, PSC, or PCI
	CellID    uint32 `json:"cell_id,omitempty"`   // CID
	Area      uint32 `json:"area,omitempty"`      // LAC or TAC
	SCS       uint8  `json:"scs,omitempty"`       // Sub-Carrier Spacing step (NR)
	Formatted string `json:"formatted,omitempty"` // Formatted string representation
}

// RATSearcherConfig holds preferences for one radio access technology (GSM, UMTS, LTE, NR).
type RATSearcherConfig struct {
	CellBlacklist        []SearcherItem `json:"cell_blacklist"`
	ChannelBlacklist     []SearcherItem `json:"channel_blacklist"`
	ChannelCodeWhitelist []SearcherItem `json:"channel_code_whitelist"`
	PriorityChannels     []SearcherItem `json:"priority_channels"` // Max 6 entries (UMTS: 1 entry)
}

// SearcherPreferences holds all RAT searcher configurations.
type SearcherPreferences struct {
	GSM  RATSearcherConfig `json:"gsm"`
	UMTS RATSearcherConfig `json:"umts"`
	LTE  RATSearcherConfig `json:"lte"`
	NR   RATSearcherConfig `json:"nr"`
}

// ExtraLocksConfig holds lock settings outside standard LTE cell lock.
type ExtraLocksConfig struct {
	// GERAN (GSM)
	GeranLockArfcnEnabled bool     `json:"geran_lock_arfcn_enabled"`
	GeranArfcnList        []uint16 `json:"geran_arfcn_list"`
	FakeBtsCellBarring    bool     `json:"fake_bts_cell_barring_enabled"`

	// WCDMA (3G)
	WcdmaFreqLockItem uint16 `json:"wcdma_freq_lock_item"` // UARFCN (0 if disabled)

	// LTE CSP
	LteCspPciLock    uint16 `json:"lte_csp_pci_lock"`
	LteCspEarfcnLock uint32 `json:"lte_csp_earfcn_lock"`

	// NR5G CSP
	NrCspPciLock    uint16 `json:"nr_csp_pci_lock"`
	NrCspEarfcnLock uint32 `json:"nr_csp_earfcn_lock"`
}

const (
	// SearcherTerminator denotes an empty or terminator entry in Qualcomm searcher arrays (-1).
	SearcherTerminator uint64 = 0xFFFFFFFFFFFFFFFF

	// SearcherPageSlots is the maximum number of long[30] entries per page.
	SearcherPageSlots = 30
)

// ─── Bit Packing & Unpacking Helpers ──────────────────────────────────────────

// PackLTECell packs PCI and Cell ID (CID) into a 64-bit word: (PCI << 32) | CID.
func PackLTECell(pci uint32, cid uint32) uint64 {
	return (uint64(pci) << 32) | uint64(cid)
}

// UnpackLTECell unpacks PCI and CID from a 64-bit word.
func UnpackLTECell(val uint64) (pci uint32, cid uint32) {
	return uint32(val >> 32), uint32(val & 0xFFFFFFFF)
}

// PackGSMCell packs 4 GSM fields into a 64-bit word: (f1 << 48) | (f2 << 40) | (f3 << 16) | f4.
func PackGSMCell(f1 uint16, f2 uint8, f3 uint32, f4 uint16) uint64 {
	return (uint64(f1) << 48) | (uint64(f2) << 40) | ((uint64(f3) & 0xFFFFFF) << 16) | uint64(f4)
}

// UnpackGSMCell unpacks 4 GSM fields from a 64-bit word.
func UnpackGSMCell(val uint64) (f1 uint16, f2 uint8, f3 uint32, f4 uint16) {
	f1 = uint16((val >> 48) & 0xFFFF)
	f2 = uint8((val >> 40) & 0xFF)
	f3 = uint32((val >> 16) & 0xFFFFFF)
	f4 = uint16(val & 0xFFFF)
	return
}

// PackNRCell packs NR-ARFCN, PCI, and SCS step into a 64-bit word.
func PackNRCell(nrArfcn uint32, pci uint16, scs uint8) uint64 {
	return (uint64(nrArfcn) << 32) | (uint64(pci) << 16) | uint64(scs&0x0F)
}

// UnpackNRCell unpacks NR-ARFCN, PCI, and SCS step from a 64-bit word.
func UnpackNRCell(val uint64) (nrArfcn uint32, pci uint16, scs uint8) {
	nrArfcn = uint32(val >> 32)
	pci = uint16((val >> 16) & 0xFFFF)
	scs = uint8(val & 0x0F)
	return
}

// PackSearcherPage encodes up to 30 uint64 entries into 240 bytes little-endian, terminated with -1.
func PackSearcherPage(entries []uint64) []byte {
	buf := make([]byte, SearcherPageSlots*8)
	// Initialize all slots to -1 (SearcherTerminator)
	for i := range SearcherPageSlots {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], SearcherTerminator)
	}

	count := len(entries)
	if count > SearcherPageSlots {
		count = SearcherPageSlots
	}

	for i := range count {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], entries[i])
	}
	return buf
}

// UnpackSearcherPage decodes 240-byte little-endian page into a slice of valid uint64 entries.
func UnpackSearcherPage(data []byte) ([]uint64, error) {
	if len(data) < 8 {
		return []uint64{}, nil
	}

	totalSlots := len(data) / 8
	if totalSlots > SearcherPageSlots {
		totalSlots = SearcherPageSlots
	}

	var results []uint64
	for i := range totalSlots {
		val := binary.LittleEndian.Uint64(data[i*8 : (i+1)*8])
		if val == SearcherTerminator {
			break // Stop at terminator
		}
		results = append(results, val)
	}
	return results, nil
}

// FormatSearcherItem returns a readable representation for a searcher entry.
func FormatSearcherItem(rat string, itemType string, val uint64) string {
	if val == SearcherTerminator {
		return "Empty"
	}
	switch rat {
	case "lte":
		if itemType == "cell" {
			pci, cid := UnpackLTECell(val)
			return fmt.Sprintf("PCI: %d, CID: %d", pci, cid)
		}
		return fmt.Sprintf("EARFCN: %d", uint32(val))
	case "gsm":
		if itemType == "cell" {
			f1, f2, f3, f4 := UnpackGSMCell(val)
			return fmt.Sprintf("f1: %d, f2: %d, LAC: %d, CID: %d", f1, f2, f3, f4)
		}
		return fmt.Sprintf("ARFCN: %d", uint32(val))
	case "umts":
		if itemType == "cell" {
			return fmt.Sprintf("PSC: %d, CID: %d", uint32(val>>32), uint32(val&0xFFFFFFFF))
		}
		return fmt.Sprintf("UARFCN: %d", uint32(val))
	case "nr":
		if itemType == "cell" {
			arfcn, pci, scs := UnpackNRCell(val)
			return fmt.Sprintf("NR-ARFCN: %d, PCI: %d, SCS: %d", arfcn, pci, scs)
		}
		return fmt.Sprintf("NR-ARFCN: %d", uint32(val))
	default:
		return fmt.Sprintf("0x%016X", val)
	}
}
