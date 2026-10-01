package shannon

import (
	"fmt"
	"strings"
)

// Known hardware limits & tables for Google Pixel 9/10 Shannon modems
var (
	lowBands   = map[int32]bool{5: true, 8: true, 12: true, 13: true, 14: true, 17: true, 18: true, 19: true, 20: true, 26: true, 28: true, 29: true, 71: true}
	sdlBands   = map[int32]bool{29: true, 32: true, 75: true}
	tddBands   = map[int32]bool{38: true, 39: true, 40: true, 41: true, 42: true, 48: true}
	no4x4Bands = map[int32]bool{5: true, 8: true, 12: true, 13: true, 14: true, 17: true, 18: true, 19: true, 20: true, 21: true, 26: true, 28: true, 29: true, 71: true}
)

const maxDLSpatialStreams = 24

// ValidateCombos performs strict hardware capability and format integrity checks.
func ValidateCombos(combos []Combo) ValidationResult {
	res := ValidationResult{
		Valid:  true,
		Issues: make([]string, 0),
	}

	seenComboSignatures := make(map[string]int)

	for idx, combo := range combos {
		comboNum := idx + 1
		hasUL := false
		streams := 0
		lowBandCount := 0
		hasLow20 := false
		hasLow28 := false
		hasBand7 := false
		hasBand38 := false
		hasULBand7 := false
		hasULBand38 := false

		for _, comp := range combo.Components {
			_, mimo := DecodeDLValue(comp.BwClassMimoDl)
			streams += mimo

			// Check 4x4 on unsupported low bands
			if mimo > 2 && no4x4Bands[comp.Band] {
				res.Issues = append(res.Issues, fmt.Sprintf("Combo #%d: Band %d has 4x4 MIMO assigned, but hardware only supports 2x2 on low bands.", comboNum, comp.Band))
				res.ErrorCount++
			}

			// Check Uplink on SDL bands (Downlink Only)
			if comp.BwClassMimoUl != 0 && sdlBands[comp.Band] {
				res.Issues = append(res.Issues, fmt.Sprintf("Combo #%d: Band %d is Supplemental Downlink (SDL) and cannot have Uplink assigned.", comboNum, comp.Band))
				res.ErrorCount++
			}

			if comp.BwClassMimoUl != 0 {
				hasUL = true
			}

			if lowBands[comp.Band] {
				lowBandCount++
				if comp.Band == 20 {
					hasLow20 = true
				}
				if comp.Band == 28 {
					hasLow28 = true
				}
			}

			if comp.Band == 7 {
				hasBand7 = true
				if comp.BwClassMimoUl != 0 {
					hasULBand7 = true
				}
			}
			if comp.Band == 38 {
				hasBand38 = true
				if comp.BwClassMimoUl != 0 {
					hasULBand38 = true
				}
			}
		}

		// Check Missing Uplink
		if !hasUL && len(combo.Components) > 0 {
			res.Issues = append(res.Issues, fmt.Sprintf("Combo #%d (%s): Missing uplink assignment. At least one CC must have Uplink enabled.", comboNum, combo.ComboString))
			res.ErrorCount++
		}

		// Check Spatial Streams limit
		if streams > maxDLSpatialStreams {
			res.Issues = append(res.Issues, fmt.Sprintf("Combo #%d (%s): Total spatial streams (%d) exceeds modem hardware maximum (%d).", comboNum, combo.ComboString, streams, maxDLSpatialStreams))
			res.ErrorCount++
		}

		// Check Low Band CA limitation (only 20+28 supported on Shannon)
		if lowBandCount > 1 {
			if !(lowBandCount == 2 && hasLow20 && hasLow28) {
				res.Issues = append(res.Issues, fmt.Sprintf("Combo #%d (%s): Unsupported multi-low-band aggregation. Shannon hardware only supports 20+28 for low-band CA.", comboNum, combo.ComboString))
				res.ErrorCount++
			}
		}

		// Check B7 + B38 UL conflict
		if hasBand7 && hasBand38 && (hasULBand7 || hasULBand38) {
			res.Issues = append(res.Issues, fmt.Sprintf("Combo #%d (%s): Simultaneous 7+38 combination cannot have uplink assigned on B7 or B38 due to RF duplexer conflict.", comboNum, combo.ComboString))
			res.ErrorCount++
		}

		// Check duplicates
		sig := combo.ComboString
		if prevIdx, exists := seenComboSignatures[sig]; exists {
			res.Issues = append(res.Issues, fmt.Sprintf("Combo #%d is an exact duplicate of Combo #%d (%s).", comboNum, prevIdx, sig))
			res.WarningCount++
		} else {
			seenComboSignatures[sig] = comboNum
		}
	}

	if res.ErrorCount > 0 {
		res.Valid = false
	}

	return res
}

// PruneCombos filters combinations to match allowed bands and eliminate unnecessary intra-band duplicates.
func PruneCombos(combos []Combo, allowedBands []int32, maxCombos int, excludeIntraBand bool) []Combo {
	allowedMap := make(map[int32]bool)
	for _, b := range allowedBands {
		allowedMap[b] = true
	}

	filtered := make([]Combo, 0, len(combos))

	for _, combo := range combos {
		keep := true
		bandCounts := make(map[int32]int)

		for _, comp := range combo.Components {
			if len(allowedMap) > 0 && !allowedMap[comp.Band] {
				keep = false
				break
			}
			bandCounts[comp.Band]++
		}

		if !keep {
			continue
		}

		if excludeIntraBand {
			hasIntra := false
			for _, count := range bandCounts {
				if count > 1 {
					hasIntra = true
					break
				}
			}
			if hasIntra {
				continue
			}
		}

		filtered = append(filtered, combo)
		if maxCombos > 0 && len(filtered) >= maxCombos {
			break
		}
	}

	return filtered
}

// AutoGenerateCombos generates fallback CA subsets from large input combos.
func AutoGenerateCombos(req AutoGenRequest) []Combo {
	generated := make([]Combo, 0, 32)
	seen := make(map[string]bool)

	lowMask, highMask := ConfIDsToMasks(req.ConfIDs)
	bcs := req.BCSOption
	if bcs == 0 {
		bcs = 2147483648 // BCS 0 default
	}

	for _, rawCombo := range req.InputCombos {
		comps := parseRawComboComponents(rawCombo)
		if len(comps) == 0 {
			continue
		}

		// Generate combinations up to maxCC
		maxCC := req.MaxCC
		if maxCC <= 0 || maxCC > 5 {
			maxCC = 5
		}

		for k := 1; k <= len(comps) && k <= maxCC; k++ {
			subsets := generateSubsets(comps, k)
			for _, subset := range subsets {
				// Assign UL
				subsetWithUL := assignUplink(subset, req.ULOption)

				// Adjust MIMO if streams exceed 24
				adjusted := nerfSpatialStreams(subsetWithUL)

				comboStr := FormatComboString(adjusted)
				if seen[comboStr] {
					continue
				}
				seen[comboStr] = true

				combo := Combo{
					Components:    adjusted,
					BCS:           bcs,
					ConfigMaskLow: lowMask,
					ConfigMaskHigh: highMask,
					ConfIDs:       req.ConfIDs,
					ComboString:   comboStr,
					TotalStreams:  CalculateTotalStreams(adjusted),
				}
				generated = append(generated, combo)
			}
		}
	}

	return generated
}

func parseRawComboComponents(s string) []Component {
	parts := strings.Split(strings.TrimSpace(s), "+")
	comps := make([]Component, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var band int32
		class := "A"
		_, err := fmt.Sscanf(p, "%d%s", &band, &class)
		if err != nil {
			var b int32
			if _, err2 := fmt.Sscanf(p, "%d", &b); err2 == nil {
				band = b
				class = "A"
			} else {
				continue
			}
		}
		if band <= 0 {
			continue
		}

		mimo := 2
		if !no4x4Bands[band] {
			mimo = 4
		}

		comps = append(comps, Component{
			Band:          band,
			BwClassMimoDl: EncodeDLValue(class, mimo),
			BwClassMimoUl: 0,
			DlClass:       class,
			DlMimo:        mimo,
			UlClass:       "None",
		})
	}

	return comps
}

func generateSubsets(comps []Component, k int) [][]Component {
	var result [][]Component
	var helper func(start int, current []Component)
	helper = func(start int, current []Component) {
		if len(current) == k {
			copied := make([]Component, k)
			copy(copied, current)
			result = append(result, copied)
			return
		}
		for i := start; i < len(comps); i++ {
			helper(i+1, append(current, comps[i]))
		}
	}
	helper(0, []Component{})
	return result
}

func assignUplink(comps []Component, ulOption string) []Component {
	res := make([]Component, len(comps))
	copy(res, comps)

	ulAssigned := false
	for i := range res {
		if !sdlBands[res[i].Band] {
			res[i].BwClassMimoUl = EncodeULValue("A")
			res[i].UlClass = "A"
			ulAssigned = true
			if ulOption != "fdd_aa" && ulOption != "tdd_aa" && ulOption != "tdd_fdd" {
				break // Single UL default
			}
		}
	}

	// Fallback to first if all were SDL
	if !ulAssigned && len(res) > 0 {
		res[0].BwClassMimoUl = EncodeULValue("A")
		res[0].UlClass = "A"
	}

	return res
}

func nerfSpatialStreams(comps []Component) []Component {
	res := make([]Component, len(comps))
	copy(res, comps)

	for CalculateTotalStreams(res) > maxDLSpatialStreams {
		// Find highest MIMO component and nerf to 2x2
		nerfed := false
		for i := range res {
			class, mimo := DecodeDLValue(res[i].BwClassMimoDl)
			if mimo == 4 {
				res[i].BwClassMimoDl = EncodeDLValue(class, 2)
				res[i].DlMimo = 2
				nerfed = true
				break
			}
		}
		if !nerfed {
			break
		}
	}

	return res
}
