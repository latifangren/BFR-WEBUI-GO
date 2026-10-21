package efs

import (
	"testing"
)

func TestBuildAndParseTokenString(t *testing.T) {
	orig := &CapabilityState{
		LTE: LTESectionState{
			LTECA:  true,
			ULCA:   true,
			DL256Q: true,
			DLMIMO: true,
			HORXD:  true,
			Rx0:    true,
			VoLTE:  true,
		},
		NSA: NSASectionState{
			NSALTE:    true,
			NSALTEDSS: true,
		},
		NR: NRSectionState{
			NRCA:   true,
			DL256Q: true,
			VoNR:   true,
			RedCap: true,
		},
	}

	tokenStr := BuildTokenString(orig)
	expectedTokens := []string{
		"[LTE_CA]", "[LTE_UL_CA]", "[LTE_DL_256Q]", "[LTE_DL_MIMO]", "[LTE_HORXD]", "[LTE_RX0]", "[LTE_VOICE]",
		"[NSA_LTE]", "[NSA_LTE_DSS]",
		"[NR_CA]", "[NR_DL_256Q]", "[NR_VOICE]", "[NR_REDCAP]",
	}

	for _, token := range expectedTokens {
		if !testingContains(tokenStr, token) {
			t.Errorf("expected token string to contain %s, got: %s", token, tokenStr)
		}
	}

	parsed := ParseTokenString(tokenStr)
	if !parsed.LTE.LTECA || !parsed.LTE.ULCA || !parsed.LTE.DL256Q || !parsed.LTE.DLMIMO || !parsed.LTE.VoLTE {
		t.Errorf("parsed LTE state mismatch")
	}
	if !parsed.NSA.NSALTE || !parsed.NSA.NSALTEDSS {
		t.Errorf("parsed NSA state mismatch")
	}
	if !parsed.NR.NRCA || !parsed.NR.DL256Q || !parsed.NR.VoNR || !parsed.NR.RedCap {
		t.Errorf("parsed NR state mismatch")
	}

	// Unchecked tokens should be false
	if parsed.LTE.Rx1 || parsed.NR.UL256Q || parsed.NSA.NSANDDS {
		t.Errorf("expected inactive tokens to be false")
	}
}

func TestValidateCAFilter(t *testing.T) {
	validCases := []string{
		"",
		"1A-3A-7A",
		"3C-7A",
		"1A-3A-7A-8A",
		"3C4A-7A2A[0]",
		"1A-3C[4]",
		"1A-3A[m3]; 3C-7A[mAll]",
	}

	for _, tc := range validCases {
		ok, errMsg := ValidateCAFilter(tc)
		if !ok {
			t.Errorf("expected valid for %q, got error: %s", tc, errMsg)
		}
	}

	invalidCases := []struct {
		filter string
		err    string
	}{
		{"1A-3A[invalid]", "invalid bandwidth class set"},
		{"1A-3A[0", "unclosed bandwidth class set"},
		{"invalid_comp", "invalid CA component"},
		{"1-3A", "invalid CA component"},
	}

	for _, tc := range invalidCases {
		ok, errMsg := ValidateCAFilter(tc.filter)
		if ok {
			t.Errorf("expected invalid for %q, but got valid", tc.filter)
		} else if !testingContains(errMsg, tc.err) {
			t.Errorf("expected error containing %q, got %q", tc.err, errMsg)
		}
	}
}

func testingContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && len(s) > 0 && searchSubstr(s, substr)))
}

func searchSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
