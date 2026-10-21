package efs

import (
	"fmt"
	"regexp"
	"strings"
)

// Regex for individual CA combo components: e.g. "3C", "3A4A", "7C4B", "1A"
// Band: 1-3 digits, Class: A-F, optional Streams: 2|4, optional Uplink Class: A-F
var reCAComponent = regexp.MustCompile(`^(\d{1,3})([A-F])([24]?)([A-F]?)$`)

// BuildTokenString reconstructs the 38-token string representation in ModemCaps order.
func BuildTokenString(state *CapabilityState) string {
	if state == nil {
		return ""
	}

	var sb strings.Builder

	// 1. LTE Tokens
	if state.LTE.LTECA {
		sb.WriteString("[LTE_CA]")
	}
	if state.LTE.CAFilter != "" {
		sb.WriteString("[LTE_CA_FILTER]")
	}
	if state.LTE.ULCA {
		sb.WriteString("[LTE_UL_CA]")
	}
	if state.LTE.DL256Q {
		sb.WriteString("[LTE_DL_256Q]")
	}
	if state.LTE.UL256Q {
		sb.WriteString("[LTE_UL_256Q]")
	}
	if state.LTE.DLMIMO {
		sb.WriteString("[LTE_DL_MIMO]")
	}
	if state.LTE.HORXD {
		sb.WriteString("[LTE_HORXD]")
	}
	if state.LTE.Rx0 {
		sb.WriteString("[LTE_RX0]")
	}
	if state.LTE.Rx1 {
		sb.WriteString("[LTE_RX1]")
	}
	if state.LTE.Rx2 {
		sb.WriteString("[LTE_RX2]")
	}
	if state.LTE.Rx3 {
		sb.WriteString("[LTE_RX3]")
	}
	if state.LTE.VoLTE {
		sb.WriteString("[LTE_VOICE]")
	}

	// 2. NSA Tokens
	if state.NSA.NSALTE {
		sb.WriteString("[NSA_LTE]")
	}
	if state.NSA.NSALTEDSS {
		sb.WriteString("[NSA_LTE_DSS]")
	}
	if state.NSA.NSANDDS {
		sb.WriteString("[NSA_NDDS]")
	}
	if state.NSA.NSALTEDSDA {
		sb.WriteString("[NSA_LTE_DSDA]")
	}

	// 3. NR Tokens
	if state.NR.NRCA {
		sb.WriteString("[NR_CA]")
	}
	if state.NR.DL256Q {
		sb.WriteString("[NR_DL_256Q]")
	}
	if state.NR.UL256Q {
		sb.WriteString("[NR_UL_256Q]")
	}
	if state.NR.DLMIMO {
		sb.WriteString("[NR_DL_MIMO]")
	}
	if state.NR.ULMIMO {
		sb.WriteString("[NR_UL_MIMO]")
	}
	if state.NR.RedCap {
		sb.WriteString("[NR_REDCAP]")
	}
	if state.NR.VoNR {
		sb.WriteString("[NR_VOICE]")
	}
	if state.NR.NRNRDSDA {
		sb.WriteString("[NR_NR_DSDA]")
	}

	return sb.String()
}

// ParseTokenString constructs a CapabilityState by checking for tokens in the string.
func ParseTokenString(tokenStr string) *CapabilityState {
	state := &CapabilityState{
		TokenString: tokenStr,
		LTE:         LTESectionState{},
		NSA:         NSASectionState{},
		NR:          NRSectionState{},
	}

	has := func(token string) bool {
		return strings.Contains(tokenStr, token)
	}

	// LTE
	state.LTE.LTECA = has("[LTE_CA]")
	state.LTE.ULCA = has("[LTE_UL_CA]")
	state.LTE.DL256Q = has("[LTE_DL_256Q]")
	state.LTE.UL256Q = has("[LTE_UL_256Q]")
	state.LTE.DLMIMO = has("[LTE_DL_MIMO]")
	state.LTE.HORXD = has("[LTE_HORXD]")
	state.LTE.Rx0 = has("[LTE_RX0]")
	state.LTE.Rx1 = has("[LTE_RX1]")
	state.LTE.Rx2 = has("[LTE_RX2]")
	state.LTE.Rx3 = has("[LTE_RX3]")
	state.LTE.VoLTE = has("[LTE_VOICE]")

	// NSA
	state.NSA.NSALTE = has("[NSA_LTE]")
	state.NSA.NSALTEDSS = has("[NSA_LTE_DSS]")
	state.NSA.NSANDDS = has("[NSA_NDDS]")
	state.NSA.NSALTEDSDA = has("[NSA_LTE_DSDA]")

	// NR
	state.NR.NRCA = has("[NR_CA]")
	state.NR.DL256Q = has("[NR_DL_256Q]")
	state.NR.UL256Q = has("[NR_UL_256Q]")
	state.NR.DLMIMO = has("[NR_DL_MIMO]")
	state.NR.ULMIMO = has("[NR_UL_MIMO]")
	state.NR.RedCap = has("[NR_REDCAP]")
	state.NR.VoNR = has("[NR_VOICE]")
	state.NR.NRNRDSDA = has("[NR_NR_DSDA]")

	return state
}

// ValidateCAFilter validates the LTE CA filter string against ModemCaps grammar:
// Band+Class+[Streams][Uplink Class]-...[Bandwidth Class Set]
// Example valid filters: "1A-3A-7A", "3C-7A", "3C4A-7A2A[0]"
func ValidateCAFilter(filter string) (bool, string) {
	trimmed := strings.TrimSpace(filter)
	if trimmed == "" {
		return true, "" // Empty filter is allowed (no filter applied)
	}

	// Split multiple combos by semicolon if present
	combos := strings.Split(trimmed, ";")
	for _, rawCombo := range combos {
		combo := strings.TrimSpace(strings.ToUpper(rawCombo))
		if combo == "" {
			continue
		}

		// Check optional BW Class Set suffix: e.g. [0], [1], [2], [4], [m3], [m7], [mAll]
		comboClean := combo
		if idx := strings.Index(combo, "["); idx != -1 {
			if !strings.HasSuffix(combo, "]") {
				return false, fmt.Sprintf("unclosed bandwidth class set bracket in '%s'", combo)
			}
			bwSet := combo[idx+1 : len(combo)-1]
			validBwSets := map[string]bool{
				"0": true, "1": true, "2": true, "4": true, "M3": true, "M7": true, "MALL": true,
			}
			if !validBwSets[bwSet] {
				return false, fmt.Sprintf("invalid bandwidth class set '[%s]'", bwSet)
			}
			comboClean = combo[:idx]
		}

		// Split components by "-"
		components := strings.Split(comboClean, "-")
		if len(components) == 0 {
			return false, "empty combo definition"
		}

		for _, comp := range components {
			comp = strings.TrimSpace(comp)
			if !reCAComponent.MatchString(comp) {
				return false, fmt.Sprintf("invalid CA component '%s' (expected format e.g. 1A, 3C, 7A4A)", comp)
			}
		}
	}

	return true, ""
}
