package efs

import (
	"testing"
)

func TestParseCarrierPolicyXML(t *testing.T) {
	sampleXML := `<?xml version="1.0" encoding="UTF-8"?>
<policy name="pixel_test" policy_ver="16.30.4">
  <initial>
    <actions>
      <ca_band_combos string="4A2A-2A2-mAll;5B22A-m1;66C44A-2A4-m3;12A2A-4A4A-m3F;41D222A-0;" />
    </actions>
  </initial>
</policy>`

	res := ParseCarrierPolicyXML(sampleXML)
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	if res.PolicyName != "pixel_test" {
		t.Errorf("expected policy_name 'pixel_test', got '%s'", res.PolicyName)
	}

	if res.PolicyVer != "16.30.4" {
		t.Errorf("expected policy_ver '16.30.4', got '%s'", res.PolicyVer)
	}

	if res.TotalCombos != 5 {
		t.Fatalf("expected 5 combos, got %d", res.TotalCombos)
	}

	// Verify 5B22A-m1
	// Components: B5 Class B, MIMO 22 -> 2+2=4 streams, UL Class A
	// BCS: m1 -> BCS 0
	combo5B := res.Combos[1]
	if len(combo5B.Components) != 1 {
		t.Fatalf("expected 1 component for 5B22A, got %d", len(combo5B.Components))
	}
	c5 := combo5B.Components[0]
	if c5.Band != 5 || c5.DlClass != "B" || c5.DlStreams != 4 || c5.UlClass != "A" {
		t.Errorf("unexpected component: %+v", c5)
	}
	if combo5B.BCS != "BCS 0" {
		t.Errorf("expected 'BCS 0', got '%s'", combo5B.BCS)
	}

	// Verify 66C44A-2A4-m3
	// Components: B66 (C, 44 -> 8 streams, UL A) and B2 (A, 4 -> 4 streams)
	// Total streams: 12
	// BCS: m3 (bits 0, 1) -> "BCS 0, 1"
	combo66 := res.Combos[2]
	if combo66.TotalStreams != 12 {
		t.Errorf("expected 12 total streams, got %d", combo66.TotalStreams)
	}
	if combo66.BCS != "BCS 0, 1" {
		t.Errorf("expected 'BCS 0, 1', got '%s'", combo66.BCS)
	}

	// Verify 12A2A-4A4A-m3F
	// m3F -> bits 0, 1, 2, 3, 4, 5
	combo12 := res.Combos[3]
	if combo12.BCS != "BCS 0, 1, 2, 3, 4, 5" {
		t.Errorf("expected 'BCS 0, 1, 2, 3, 4, 5', got '%s'", combo12.BCS)
	}

	// Verify 41D222A-0
	// 41D222A -> 3 CCs (2+2+2 = 6 streams), UL Class A, BCS 0
	combo41 := res.Combos[4]
	if combo41.TotalStreams != 6 {
		t.Errorf("expected 6 total streams for 41D222A, got %d", combo41.TotalStreams)
	}
	if combo41.BCS != "BCS 0" {
		t.Errorf("expected 'BCS 0', got '%s'", combo41.BCS)
	}

	// Check max streams
	if res.MaxStreams != 12 {
		t.Errorf("expected max streams 12, got %d", res.MaxStreams)
	}

	// Check bands supported
	expectedBands := []int{2, 4, 5, 12, 41, 66}
	if len(res.BandsSupported) != len(expectedBands) {
		t.Fatalf("expected %d bands, got %d (%v)", len(expectedBands), len(res.BandsSupported), res.BandsSupported)
	}
	for i, b := range expectedBands {
		if res.BandsSupported[i] != b {
			t.Errorf("band at %d: expected %d, got %d", i, b, res.BandsSupported[i])
		}
	}
}

func TestParseMultipleBlocksAndPrunes(t *testing.T) {
	rawPrune := `
	3A4A-1A4-m3;3A4-1A4A-m3;7A4A-1A4-m3;7A4-1A4A-m3;20A2A-1A4-0;28A2A-1A4-m3;
	2A2A-2A2-0;2A2A-4A2-m7;2A2-4A2A-m7;5A2A-3A4-m1F;5A2-3A4A-m1F;7A4A-3A4-m3;
	3C44A-0;3C44C-0;7C44A-m7;7C44C-m7;40C44A-m3;40C44C-m3;41C44A-mF;41C44C-mF;
	`
	res := ParseCarrierPolicyXML(rawPrune)
	if res.TotalCombos != 20 {
		t.Errorf("expected 20 combos, got %d", res.TotalCombos)
	}
	if len(res.BandsSupported) == 0 {
		t.Errorf("expected bands detected")
	}
}

