package shannon

import (
	"testing"
)

func TestEncodeDecodeShannonLteUECap(t *testing.T) {
	doc := &ComboDocument{
		Version: 1,
		Bitmask: 0,
		Combos: []Combo{
			{
				BCS:           2147483648,
				ConfigMaskLow: 196,
				ConfigMaskHigh: 0,
				Components: []Component{
					{Band: 1, BwClassMimoDl: EncodeDLValue("A", 4), BwClassMimoUl: EncodeULValue("A")},
					{Band: 3, BwClassMimoDl: EncodeDLValue("C", 4), BwClassMimoUl: 0},
					{Band: 8, BwClassMimoDl: EncodeDLValue("A", 2), BwClassMimoUl: 0},
				},
			},
		},
	}

	encoded, err := EncodeShannonLteUECap(doc)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	parsed, err := ParseShannonLteUECap(encoded)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed.Version != 1 {
		t.Errorf("expected version 1, got %d", parsed.Version)
	}
	if len(parsed.Combos) != 1 {
		t.Fatalf("expected 1 combo, got %d", len(parsed.Combos))
	}

	combo := parsed.Combos[0]
	if len(combo.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(combo.Components))
	}
	if combo.Components[0].Band != 1 || combo.Components[0].DlMimo != 4 || combo.Components[0].UlClass != "A" {
		t.Errorf("unexpected comp 0: %+v", combo.Components[0])
	}
	if combo.Components[1].Band != 3 || combo.Components[1].DlClass != "C" || combo.Components[1].DlMimo != 4 {
		t.Errorf("unexpected comp 1: %+v", combo.Components[1])
	}
	if combo.Components[2].Band != 8 || combo.Components[2].DlMimo != 2 {
		t.Errorf("unexpected comp 2: %+v", combo.Components[2])
	}
}

func TestValidationRules(t *testing.T) {
	// Low band 4x4 should fail
	invalidCombo := []Combo{
		{
			ComboString: "8A4[A]",
			Components: []Component{
				{Band: 8, BwClassMimoDl: EncodeDLValue("A", 4), BwClassMimoUl: EncodeULValue("A")},
			},
		},
	}
	res := ValidateCombos(invalidCombo)
	if res.Valid {
		t.Errorf("expected invalid result for low-band 4x4 MIMO, got valid")
	}
	if res.ErrorCount == 0 {
		t.Errorf("expected error count > 0")
	}

	// Valid combo
	validCombo := []Combo{
		{
			ComboString: "1A4[A]+3C4",
			Components: []Component{
				{Band: 1, BwClassMimoDl: EncodeDLValue("A", 4), BwClassMimoUl: EncodeULValue("A")},
				{Band: 3, BwClassMimoDl: EncodeDLValue("C", 4), BwClassMimoUl: 0},
			},
		},
	}
	resValid := ValidateCombos(validCombo)
	if !resValid.Valid {
		t.Errorf("expected valid result, got errors: %v", resValid.Issues)
	}
}
