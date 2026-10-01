package shannon

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ParseShannonNrUECap parses a raw ShannonNrUECap .binarypb buffer into an NRComboDocument.
func ParseShannonNrUECap(data []byte) (*NRComboDocument, error) {
	doc := &NRComboDocument{
		Combos: make([]NRCombo, 0),
	}
	offset := 0

	var comboGroups []rawComboGroupNr
	var dlFeaturesPerCC []rawFeatureDlPerCCNr
	var ulFeaturesPerCC []rawFeatureUlPerCCNr

	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return nil, fmt.Errorf("error reading root tag at %d: %w", offset, err)
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // version (optional uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			doc.Version = uint32(val)

		case 2: // id (optional int32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			doc.ID = int32(val)

		case 3: // comboGroups (repeated ComboGroup)
			if wireType != wireLengthDelim {
				return nil, fmt.Errorf("expected wireLengthDelim for comboGroup, got %d", wireType)
			}
			groupBytes, n, err := readLengthDelimited(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			cg, err := parseComboGroup(groupBytes)
			if err != nil {
				return nil, err
			}
			comboGroups = append(comboGroups, *cg)

		case 6: // dlFeaturePerCCList (repeated FeatureDlPerCCNr)
			if wireType != wireLengthDelim {
				return nil, fmt.Errorf("expected wireLengthDelim for dlFeaturePerCC, got %d", wireType)
			}
			fBytes, n, err := readLengthDelimited(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f, err := parseFeatureDlPerCC(fBytes)
			if err != nil {
				return nil, err
			}
			dlFeaturesPerCC = append(dlFeaturesPerCC, *f)

		case 7: // ulFeaturePerCCList (repeated FeatureUlPerCCNr)
			if wireType != wireLengthDelim {
				return nil, fmt.Errorf("expected wireLengthDelim for ulFeaturePerCC, got %d", wireType)
			}
			fBytes, n, err := readLengthDelimited(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f, err := parseFeatureUlPerCC(fBytes)
			if err != nil {
				return nil, err
			}
			ulFeaturesPerCC = append(ulFeaturesPerCC, *f)

		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}

	// Resolve raw combos into high-level NRCombo models
	for _, group := range comboGroups {
		for _, rawCombo := range group.Combos {
			combo := resolveRawCombo(rawCombo, group, dlFeaturesPerCC, ulFeaturesPerCC)
			doc.Combos = append(doc.Combos, combo)

			switch combo.Type {
			case "ENDC":
				doc.EndcCount++
			case "NRCA":
				doc.NrcaCount++
			case "NRDC":
				doc.NrdcCount++
			}
		}
	}

	doc.TotalCombos = len(doc.Combos)
	return doc, nil
}

func readLengthDelimited(data []byte) ([]byte, int, error) {
	length, n, err := decodeVarint(data)
	if err != nil {
		return nil, 0, err
	}
	total := n + int(length)
	if total > len(data) {
		return nil, 0, errors.New("length-delimited payload truncated")
	}
	return data[n:total], total, nil
}

func parseComboGroup(data []byte) (*rawComboGroupNr, error) {
	cg := &rawComboGroupNr{}
	offset := 0

	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return nil, err
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // comboFeatures (required ComboFeatures)
			if wireType != wireLengthDelim {
				return nil, fmt.Errorf("expected wireLengthDelim for comboFeatures, got %d", wireType)
			}
			featBytes, n, err := readLengthDelimited(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			if err := parseComboFeatures(featBytes, cg); err != nil {
				return nil, err
			}

		case 2: // combos (repeated Combo)
			if wireType != wireLengthDelim {
				return nil, fmt.Errorf("expected wireLengthDelim for combo, got %d", wireType)
			}
			cBytes, n, err := readLengthDelimited(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			rc, err := parseRawCombo(cBytes)
			if err != nil {
				return nil, err
			}
			cg.Combos = append(cg.Combos, *rc)

		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}

	return cg, nil
}

func parseComboFeatures(data []byte, cg *rawComboGroupNr) error {
	offset := 0
	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return err
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // bcsNr (optional uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			cg.BcsNr = uint32(val)
		case 2: // bcsIntraEndc (optional uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			cg.BcsIntraEndc = uint32(val)
		case 3: // bcsEutra (optional uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			cg.BcsEutra = uint32(val)
		case 4: // powerClass (optional int32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			cg.PowerClass = int32(val)
		case 5: // intraBandEnDcSupport (optional int32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			cg.IntraBandEndcSupport = int32(val)
		case 6: // supportedNumTag (optional uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			cg.SupportedNumTag = uint32(val)
		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return err
			}
			offset += skipLen
		}
	}
	return nil
}

func parseRawCombo(data []byte) (*rawComboNr, error) {
	rc := &rawComboNr{}
	offset := 0

	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return nil, err
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // components (repeated Component)
			if wireType != wireLengthDelim {
				return nil, fmt.Errorf("expected wireLengthDelim for component, got %d", wireType)
			}
			compBytes, n, err := readLengthDelimited(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp, err := parseRawComponent(compBytes)
			if err != nil {
				return nil, err
			}
			rc.Components = append(rc.Components, *comp)

		case 2: // skuBitMask (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			rc.SkuBitmask = uint32(val)

		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}

	return rc, nil
}

func parseRawComponent(data []byte) (*rawComponentNr, error) {
	comp := &rawComponentNr{}
	offset := 0

	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return nil, err
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // band (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.RawBand = uint32(val)

		case 2: // bwClassDl (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.RawBwClassDl = uint32(val)

		case 3: // bwClassUl (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.RawBwClassUl = uint32(val)

		case 4: // dlFeatureIndex (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.DlFeatureIndex = uint32(val)

		case 5: // ulFeatureIndex (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.UlFeatureIndex = uint32(val)

		case 6: // dlFeaturePerCCIds (repeated uint32, packed)
			if wireType == wireLengthDelim {
				packedBytes, n, err := readLengthDelimited(data[offset:])
				if err != nil {
					return nil, err
				}
				offset += n
				pOff := 0
				for pOff < len(packedBytes) {
					id, pn, err := decodeVarint(packedBytes[pOff:])
					if err != nil {
						break
					}
					pOff += pn
					comp.DlFeaturePerCCIds = append(comp.DlFeaturePerCCIds, uint32(id))
				}
			} else if wireType == wireVarint {
				val, n, err := decodeVarint(data[offset:])
				if err != nil {
					return nil, err
				}
				offset += n
				comp.DlFeaturePerCCIds = append(comp.DlFeaturePerCCIds, uint32(val))
			}

		case 7: // ulFeaturePerCCIds (repeated uint32, packed)
			if wireType == wireLengthDelim {
				packedBytes, n, err := readLengthDelimited(data[offset:])
				if err != nil {
					return nil, err
				}
				offset += n
				pOff := 0
				for pOff < len(packedBytes) {
					id, pn, err := decodeVarint(packedBytes[pOff:])
					if err != nil {
						break
					}
					pOff += pn
					comp.UlFeaturePerCCIds = append(comp.UlFeaturePerCCIds, uint32(id))
				}
			} else if wireType == wireVarint {
				val, n, err := decodeVarint(data[offset:])
				if err != nil {
					return nil, err
				}
				offset += n
				comp.UlFeaturePerCCIds = append(comp.UlFeaturePerCCIds, uint32(val))
			}

		case 8: // srsTxSwitch (optional int32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.SrsTxSwitch = int32(val)

		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}

	return comp, nil
}

func parseFeatureDlPerCC(data []byte) (*rawFeatureDlPerCCNr, error) {
	f := &rawFeatureDlPerCCNr{}
	offset := 0
	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return nil, err
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // maxScs
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxScs = int32(val)
		case 2: // maxMimo
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxMimo = int32(val)
		case 3: // maxBw
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxBw = uint32(val)
		case 4: // maxModOrder
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxModOrder = int32(val)
		case 5: // bw90MHzSupported
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.Bw90MHzSupported = (val != 0)
		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}
	return f, nil
}

func parseFeatureUlPerCC(data []byte) (*rawFeatureUlPerCCNr, error) {
	f := &rawFeatureUlPerCCNr{}
	offset := 0
	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return nil, err
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // maxScs
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxScs = int32(val)
		case 2: // maxMimo
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxMimo = int32(val)
		case 3: // maxBw
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxBw = int32(val)
		case 4: // maxModOrder
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxModOrder = int32(val)
		case 5: // bw90MHzSupported
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.Bw90MHzSupported = (val != 0)
		case 6: // maxNumSRSResPerSet
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			f.MaxNumSRSRes = uint32(val)
		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}
	return f, nil
}

// resolveRawCombo converts raw protobuf structures to clean, typed NRCombo
func resolveRawCombo(
	rawCombo rawComboNr,
	group rawComboGroupNr,
	dlFeaturesPerCC []rawFeatureDlPerCCNr,
	ulFeaturesPerCC []rawFeatureUlPerCCNr,
) NRCombo {
	lteComponents := make([]NRComponent, 0)
	nrComponents := make([]NRComponent, 0)

	for _, rc := range rawCombo.Components {
		if rc.RawBand > 10000 {
			// 5G NR Component
			nrComp := resolveNRComponent(rc, dlFeaturesPerCC, ulFeaturesPerCC)
			nrComponents = append(nrComponents, nrComp)
		} else {
			// 4G LTE Component
			lteComp := resolveLTEComponent(rc)
			lteComponents = append(lteComponents, lteComp)
		}
	}

	// Sort components descending by band
	sort.Slice(lteComponents, func(i, j int) bool {
		return lteComponents[i].Band > lteComponents[j].Band
	})
	sort.Slice(nrComponents, func(i, j int) bool {
		return nrComponents[i].Band > nrComponents[j].Band
	})

	combo := NRCombo{
		ComponentsLte: lteComponents,
		ComponentsNr:  nrComponents,
		BCSNr:         group.BcsNr,
		BCSEutra:      group.BcsEutra,
		BCSIntraEndc:  group.BcsIntraEndc,
		SkuBitmask:    rawCombo.SkuBitmask,
	}

	switch group.PowerClass {
	case 1:
		combo.PowerClass = "PC2"
	case 2:
		combo.PowerClass = "PC1.5"
	default:
		combo.PowerClass = "Default"
	}

	// Determine Combo Type
	if len(lteComponents) > 0 && len(nrComponents) > 0 {
		combo.Type = "ENDC"
	} else if len(lteComponents) == 0 && len(nrComponents) > 0 {
		hasFR1 := false
		hasFR2 := false
		for _, comp := range nrComponents {
			if comp.IsFR2 {
				hasFR2 = true
			} else {
				hasFR1 = true
			}
		}
		if hasFR1 && hasFR2 {
			combo.Type = "NRDC"
		} else {
			combo.Type = "NRCA"
		}
	} else {
		combo.Type = "ENDC"
	}

	// Calculate total DL streams and format combo string
	combo.TotalDlStreams = 0
	for _, c := range lteComponents {
		combo.TotalDlStreams += c.MimoDl
	}
	for _, c := range nrComponents {
		combo.TotalDlStreams += c.MimoDl
	}

	combo.ComboString = formatNRComboString(combo)
	return combo
}

func decodeBwClass(raw uint32) string {
	if raw == 0 {
		return ""
	}
	// ASCII offset: 1 -> 'A', 2 -> 'B', etc.
	if raw >= 1 && raw <= 26 {
		return string(rune('A' + (raw - 1)))
	}
	return "?"
}

func resolveLTEComponent(rc rawComponentNr) NRComponent {
	// LTE Hardcoded FeatureSet:
	// Downlink: 1:4, 2:2, 3:4+4, 4:2+4, 5:4+2, 6:2+2, 7:4+4+4, 8:2+2+2, 9:4+4+4+4, 10:2+2+2+2
	// Uplink: 1:2, 2:1, 3:2+1, 4:1+1
	mimoDl := 2
	switch rc.DlFeatureIndex {
	case 1, 3, 5, 7, 9:
		mimoDl = 4
	case 2, 4, 6, 8, 10:
		mimoDl = 2
	}

	mimoUl := 1
	switch rc.UlFeatureIndex {
	case 1, 3:
		mimoUl = 2
	case 2, 4:
		mimoUl = 1
	}

	return NRComponent{
		Band:         int(rc.RawBand),
		IsNR:         false,
		IsFR2:        false,
		BwClassDl:    decodeBwClass(rc.RawBwClassDl),
		BwClassUl:    decodeBwClass(rc.RawBwClassUl),
		MimoDl:       mimoDl,
		MimoUl:       mimoUl,
		ModulationDl: "qam256",
		ModulationUl: "qam64",
		MaxScs:       15,
		MaxBwDl:      20,
		MaxBwUl:      20,
	}
}

func resolveNRComponent(
	rc rawComponentNr,
	dlFeatures []rawFeatureDlPerCCNr,
	ulFeatures []rawFeatureUlPerCCNr,
) NRComponent {
	band := int(rc.RawBand - 10000)
	isFR2 := (band >= 257 && band <= 263)

	comp := NRComponent{
		Band:         band,
		IsNR:         true,
		IsFR2:        isFR2,
		BwClassDl:    decodeBwClass(rc.RawBwClassDl),
		BwClassUl:    decodeBwClass(rc.RawBwClassUl),
		ModulationDl: "qam256",
		ModulationUl: "qam256",
		MimoDl:       2,
		MimoUl:       1,
	}

	switch rc.SrsTxSwitch {
	case 1:
		comp.SrsTxSwitch = "1T1R"
	case 2:
		comp.SrsTxSwitch = "1T2R"
	case 3:
		comp.SrsTxSwitch = "1T4R"
	case 4:
		comp.SrsTxSwitch = "2T2R"
	case 5:
		comp.SrsTxSwitch = "2T4R"
	case 6:
		comp.SrsTxSwitch = "1T4R-2T4R"
	}

	// Downlink features per CC
	for _, ccId := range rc.DlFeaturePerCCIds {
		idx := int(ccId - 1)
		if idx >= 0 && idx < len(dlFeatures) {
			feat := dlFeatures[idx]
			// Max SCS: 1=15, 2=30, 3=60, 4=120
			switch feat.MaxScs {
			case 1:
				comp.MaxScs = maxInt(comp.MaxScs, 15)
			case 2:
				comp.MaxScs = maxInt(comp.MaxScs, 30)
			case 3:
				comp.MaxScs = maxInt(comp.MaxScs, 60)
			case 4:
				comp.MaxScs = maxInt(comp.MaxScs, 120)
			}

			// MIMO DL: 1=2, 2=4, 3=8
			switch feat.MaxMimo {
			case 1:
				comp.MimoDl = maxInt(comp.MimoDl, 2)
			case 2:
				comp.MimoDl = maxInt(comp.MimoDl, 4)
			case 3:
				comp.MimoDl = maxInt(comp.MimoDl, 8)
			}

			// Max BW DL
			bw := int(feat.MaxBw)
			if feat.Bw90MHzSupported && bw == 80 {
				bw = 90
			}
			comp.MaxBwDl = maxInt(comp.MaxBwDl, bw)

			if feat.Bw90MHzSupported {
				comp.Bw90MHzSupported = true
			}

			// Modulation Order: 1=64QAM, 2=256QAM
			if feat.MaxModOrder == 1 && comp.ModulationDl != "qam256" {
				comp.ModulationDl = "qam64"
			} else if feat.MaxModOrder == 2 {
				comp.ModulationDl = "qam256"
			}
		}
	}

	// Uplink features per CC
	for _, ccId := range rc.UlFeaturePerCCIds {
		idx := int(ccId - 1)
		if idx >= 0 && idx < len(ulFeatures) {
			feat := ulFeatures[idx]
			switch feat.MaxScs {
			case 1:
				comp.MaxScs = maxInt(comp.MaxScs, 15)
			case 2:
				comp.MaxScs = maxInt(comp.MaxScs, 30)
			case 3:
				comp.MaxScs = maxInt(comp.MaxScs, 60)
			case 4:
				comp.MaxScs = maxInt(comp.MaxScs, 120)
			}

			// MIMO UL: 1=1, 2=2, 3=4
			switch feat.MaxMimo {
			case 1:
				comp.MimoUl = maxInt(comp.MimoUl, 1)
			case 2:
				comp.MimoUl = maxInt(comp.MimoUl, 2)
			case 3:
				comp.MimoUl = maxInt(comp.MimoUl, 4)
			}

			bw := int(feat.MaxBw)
			if feat.Bw90MHzSupported && bw == 80 {
				bw = 90
			}
			comp.MaxBwUl = maxInt(comp.MaxBwUl, bw)

			if feat.Bw90MHzSupported {
				comp.Bw90MHzSupported = true
			}

			if feat.MaxModOrder == 1 && comp.ModulationUl != "qam256" {
				comp.ModulationUl = "qam64"
			} else if feat.MaxModOrder == 2 {
				comp.ModulationUl = "qam256"
			}
		}
	}

	return comp
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func formatNRComboString(combo NRCombo) string {
	lteParts := make([]string, 0, len(combo.ComponentsLte))
	for _, c := range combo.ComponentsLte {
		mimoStr := fmt.Sprintf("%d", c.MimoDl)
		ulStr := ""
		if c.BwClassUl != "" {
			ulStr = fmt.Sprintf("[%s]", c.BwClassUl)
		}
		lteParts = append(lteParts, fmt.Sprintf("%d%s%s%s", c.Band, c.BwClassDl, mimoStr, ulStr))
	}

	nrParts := make([]string, 0, len(combo.ComponentsNr))
	for _, c := range combo.ComponentsNr {
		mimoStr := fmt.Sprintf("%d", c.MimoDl)
		ulStr := ""
		if c.BwClassUl != "" {
			ulStr = fmt.Sprintf("[%s]", c.BwClassUl)
		}
		nrParts = append(nrParts, fmt.Sprintf("n%d%s%s%s", c.Band, c.BwClassDl, mimoStr, ulStr))
	}

	if combo.Type == "ENDC" {
		return fmt.Sprintf("DC_%s_%s", strings.Join(lteParts, "+"), strings.Join(nrParts, "+"))
	} else if combo.Type == "NRDC" {
		return fmt.Sprintf("DC_%s", strings.Join(nrParts, "+"))
	}
	return strings.Join(nrParts, "+")
}
