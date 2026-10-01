package efs

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CarrierPolicyComponent represents a single carrier component in a Qualcomm carrier policy combo.
type CarrierPolicyComponent struct {
	Band      int    `json:"band"`
	DlClass   string `json:"dl_class"`
	DlMimo    string `json:"dl_mimo"`
	DlStreams int    `json:"dl_streams"`
	UlClass   string `json:"ul_class"`
}

// CarrierPolicyCombo represents a decoded CA combination from Qualcomm carrier policy XML.
type CarrierPolicyCombo struct {
	RawComboString string                   `json:"raw_combo_string"`
	Components     []CarrierPolicyComponent `json:"components"`
	BCS            string                   `json:"bcs"`
	TotalStreams   int                      `json:"total_streams"`
	ComboString    string                   `json:"combo_string"`
}

// CarrierPolicyParseResult contains the extracted combos and metadata from carrier policy XML.
type CarrierPolicyParseResult struct {
	TotalCombos    int                  `json:"total_combos"`
	PolicyName     string               `json:"policy_name,omitempty"`
	PolicyVer      string               `json:"policy_ver,omitempty"`
	MaxStreams     int                  `json:"max_streams"`
	BandsSupported []int                `json:"bands_supported"`
	Combos         []CarrierPolicyCombo `json:"combos"`
}

// Regex to extract component parts: <band:1-3 digits><dlClass:A-F><mimoDL:2,4 digits up to 5><ulClass:A-F or empty>
var (
	reCarrierPolicyComponent = regexp.MustCompile(`^(\d{1,3})([A-F])([24]{0,5})([A-F]?)$`)
	reCaBandCombosAttr       = regexp.MustCompile(`(?s)<ca_band_combos[^>]*string=["']([^"']+)["']`)
	reCaBandCombosBody       = regexp.MustCompile(`(?s)<ca_band_combos[^>]*>(.*?)</ca_band_combos>`)
	rePolicyHeader           = regexp.MustCompile(`name\s*=\s*["']([^"']+)["']`)
	rePolicyVer              = regexp.MustCompile(`policy_ver\s*=\s*["']([^"']+)["']`)
)

// ParseCarrierPolicyXML extracts and decodes LTE CA combinations from Qualcomm carrier policy XML.
// Based on the 3GPP and Qualcomm reverse-engineering rules from uecapabilityparser (ImportLteCarrierPolicy.kt).
func ParseCarrierPolicyXML(xmlContent string) *CarrierPolicyParseResult {
	res := &CarrierPolicyParseResult{
		Combos:         make([]CarrierPolicyCombo, 0),
		BandsSupported: make([]int, 0),
	}

	if nameM := rePolicyHeader.FindStringSubmatch(xmlContent); len(nameM) > 1 {
		res.PolicyName = strings.TrimSpace(nameM[1])
	}
	if verM := rePolicyVer.FindStringSubmatch(xmlContent); len(verM) > 1 {
		res.PolicyVer = strings.TrimSpace(verM[1])
	}

	rawBlocks := make([]string, 0)

	// 1. Check for attribute: <ca_band_combos string="..." />
	if attrMatches := reCaBandCombosAttr.FindAllStringSubmatch(xmlContent, -1); len(attrMatches) > 0 {
		for _, m := range attrMatches {
			if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
				rawBlocks = append(rawBlocks, m[1])
			}
		}
	}

	// 2. Check for tag body: <ca_band_combos>...</ca_band_combos>
	if bodyMatches := reCaBandCombosBody.FindAllStringSubmatch(xmlContent, -1); len(bodyMatches) > 0 {
		for _, m := range bodyMatches {
			if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
				// Don't re-add if it's already a string="..." attribute inside the tag
				content := strings.TrimSpace(m[1])
				if !strings.HasPrefix(content, "string=") {
					rawBlocks = append(rawBlocks, content)
				}
			}
		}
	}

	// 3. Fallback: check if the content itself is semi-colon separated combos
	if len(rawBlocks) == 0 && strings.Contains(xmlContent, ";") && strings.Contains(xmlContent, "-") {
		rawBlocks = append(rawBlocks, xmlContent)
	}

	seenCombos := make(map[string]bool)
	bandsMap := make(map[int]bool)
	maxStreams := 0

	for _, block := range rawBlocks {
		// Split by ';' or '"' or newlines
		rawEntries := strings.FieldsFunc(block, func(r rune) bool {
			return r == ';' || r == '"' || r == '\n' || r == '\r'
		})

		for _, entry := range rawEntries {
			entry = strings.TrimSpace(entry)
			if entry == "" || strings.HasPrefix(entry, "<!--") {
				continue
			}

			combo := parseQualcommComboString(entry)
			if combo != nil && len(combo.Components) > 0 {
				if !seenCombos[combo.ComboString] {
					seenCombos[combo.ComboString] = true
					res.Combos = append(res.Combos, *combo)

					if combo.TotalStreams > maxStreams {
						maxStreams = combo.TotalStreams
					}
					for _, c := range combo.Components {
						bandsMap[c.Band] = true
					}
				}
			}
		}
	}

	bands := make([]int, 0, len(bandsMap))
	for b := range bandsMap {
		bands = append(bands, b)
	}
	sort.Ints(bands)

	res.BandsSupported = bands
	res.MaxStreams = maxStreams
	res.TotalCombos = len(res.Combos)
	return res
}

func parseQualcommComboString(raw string) *CarrierPolicyCombo {
	clean := strings.TrimSpace(raw)
	// Strip optional "e-" prefix
	if strings.HasPrefix(clean, "e-") {
		clean = clean[2:]
	}

	parts := strings.Split(clean, "-")
	if len(parts) == 0 {
		return nil
	}

	var comps []CarrierPolicyComponent
	bcsStr := "All BCS"

	// If last part is BCS (e.g. "0", "mAll", "m1", "m3F"), extract it
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	hasBCS := false
	if isBCSPart(lastPart) {
		bcsStr = formatBCS(lastPart)
		hasBCS = true
	}

	limit := len(parts)
	if hasBCS {
		limit = len(parts) - 1
	}

	for i := 0; i < limit; i++ {
		compStr := strings.TrimSpace(parts[i])
		if compStr == "" {
			continue
		}
		comp := parseCarrierComponent(compStr)
		if comp != nil {
			comps = append(comps, *comp)
		}
	}

	if len(comps) == 0 {
		return nil
	}

	// Sort components descending by band for canonical display (matches uecapabilityparser)
	sort.Slice(comps, func(i, j int) bool {
		if comps[i].Band != comps[j].Band {
			return comps[i].Band > comps[j].Band
		}
		return comps[i].DlClass < comps[j].DlClass
	})

	totalStreams := 0
	comboParts := make([]string, 0, len(comps))
	for _, c := range comps {
		totalStreams += c.DlStreams
		ulSuffix := ""
		if c.UlClass != "" && c.UlClass != "None" {
			ulSuffix = fmt.Sprintf("[%s]", c.UlClass)
		}
		mimoDisplay := ""
		if c.DlMimo != "" {
			mimoDisplay = c.DlMimo
		}
		comboParts = append(comboParts, fmt.Sprintf("%d%s%s%s", c.Band, c.DlClass, mimoDisplay, ulSuffix))
	}

	return &CarrierPolicyCombo{
		RawComboString: raw,
		Components:     comps,
		BCS:            bcsStr,
		TotalStreams:   totalStreams,
		ComboString:    strings.Join(comboParts, "+"),
	}
}

func parseCarrierComponent(s string) *CarrierPolicyComponent {
	m := reCarrierPolicyComponent.FindStringSubmatch(s)
	if len(m) < 3 {
		return nil
	}

	band, err := strconv.Atoi(m[1])
	if err != nil || band <= 0 {
		return nil
	}

	dlClass := m[2]
	mimoStr := ""
	streams := 2 // default 2x2 MIMO

	if len(m) > 3 && m[3] != "" {
		mimoStr = m[3]
		calcStreams := 0
		for _, ch := range mimoStr {
			if ch >= '0' && ch <= '9' {
				calcStreams += int(ch - '0')
			}
		}
		if calcStreams > 0 {
			streams = calcStreams
		}
	}

	ulClass := ""
	if len(m) > 4 && m[4] != "" {
		ulClass = m[4]
	}

	return &CarrierPolicyComponent{
		Band:      band,
		DlClass:   dlClass,
		DlMimo:    mimoStr,
		DlStreams: streams,
		UlClass:   ulClass,
	}
}

func isBCSPart(s string) bool {
	if s == "" {
		return false
	}
	if s == "mAll" || strings.HasPrefix(s, "m") {
		return true
	}
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	return false
}

func formatBCS(s string) string {
	if s == "" || s == "mAll" {
		return "All BCS"
	}
	if strings.HasPrefix(s, "m") {
		hexStr := s[1:]
		if val, err := strconv.ParseUint(hexStr, 16, 64); err == nil {
			var bits []string
			for b := 0; b < 64; b++ {
				if (val & (1 << uint(b))) != 0 {
					bits = append(bits, strconv.Itoa(b))
				}
			}
			if len(bits) == 32 {
				return "All BCS"
			}
			if len(bits) > 0 {
				return "BCS " + strings.Join(bits, ", ")
			}
		}
		return "BCS " + s
	}
	return "BCS " + s
}
