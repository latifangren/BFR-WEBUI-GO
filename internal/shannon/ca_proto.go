package shannon

import (
	"errors"
	"fmt"
	"strings"
)

// Wire type constants for Protobuf encoding
const (
	wireVarint     = 0
	wireFixed64    = 1
	wireLengthDelim = 2
	wireFixed32    = 5
)

// decodeVarint decodes an unsigned varint from data, returning the value and bytes read.
func decodeVarint(data []byte) (uint64, int, error) {
	var val uint64
	var shift uint
	for i, b := range data {
		val |= uint64(b&0x7F) << shift
		if (b & 0x80) == 0 {
			return val, i + 1, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, errors.New("varint overflow")
		}
	}
	return 0, 0, errors.New("unexpected EOF in varint")
}

// encodeVarint appends an unsigned varint to buf.
func encodeVarint(buf []byte, val uint64) []byte {
	for val >= 0x80 {
		buf = append(buf, byte(val|0x80))
		val >>= 7
	}
	buf = append(buf, byte(val))
	return buf
}

// encodeTag encodes a protobuf field number and wire type.
func encodeTag(buf []byte, fieldNum int, wireType int) []byte {
	return encodeVarint(buf, uint64(fieldNum<<3|wireType))
}

// ParseShannonLteUECap decodes a binary .binarypb buffer into a ComboDocument.
func ParseShannonLteUECap(data []byte) (*ComboDocument, error) {
	doc := &ComboDocument{}
	offset := 0

	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return nil, fmt.Errorf("error reading root tag at %d: %w", offset, err)
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 7)

		switch fieldNum {
		case 1: // version (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			doc.Version = int32(val)
		case 2: // combos (repeated Combo)
			if wireType != wireLengthDelim {
				return nil, fmt.Errorf("expected length-delimited wire type for combo, got %d", wireType)
			}
			length, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			if offset+int(length) > len(data) {
				return nil, errors.New("combo payload truncated")
			}
			comboBytes := data[offset : offset+int(length)]
			offset += int(length)

			combo, err := parseCombo(comboBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse combo: %w", err)
			}
			doc.Combos = append(doc.Combos, *combo)
		case 3: // bitmask (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			doc.Bitmask = uint32(val)
		default:
			// Skip unknown fields
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}

	return doc, nil
}

func parseCombo(data []byte) (*Combo, error) {
	combo := &Combo{}
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
				return nil, fmt.Errorf("expected length-delimited wire type for component, got %d", wireType)
			}
			length, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			if offset+int(length) > len(data) {
				return nil, errors.New("component payload truncated")
			}
			compBytes := data[offset : offset+int(length)]
			offset += int(length)

			comp, err := parseComponent(compBytes)
			if err != nil {
				return nil, err
			}
			combo.Components = append(combo.Components, *comp)
		case 2: // bcs (optional uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			combo.BCS = uint32(val)
		case 3: // conf_id_1 (required uint64)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			combo.ConfigMaskLow = val
		case 4: // conf_id_2 (required uint32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			combo.ConfigMaskHigh = uint32(val)
		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}

	// Calculate helper visual fields
	combo.ConfIDs = MasksToConfIDs(combo.ConfigMaskLow, combo.ConfigMaskHigh)
	combo.ComboString = FormatComboString(combo.Components)
	combo.TotalStreams = CalculateTotalStreams(combo.Components)

	return combo, nil
}

func parseComponent(data []byte) (*Component, error) {
	comp := &Component{
		Band:          1,
		BwClassMimoDl: 32768,
		BwClassMimoUl: 0,
	}
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
		case 1: // band (required int32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.Band = int32(val)
		case 2: // bwClassMimoDl (required int32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.BwClassMimoDl = int32(val)
		case 3: // bwClassMimoUl (required int32)
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			comp.BwClassMimoUl = int32(val)
		default:
			skipLen, err := skipField(data[offset:], wireType)
			if err != nil {
				return nil, err
			}
			offset += skipLen
		}
	}

	comp.DlClass, comp.DlMimo = DecodeDLValue(comp.BwClassMimoDl)
	comp.UlClass = DecodeULValue(comp.BwClassMimoUl)

	return comp, nil
}

func skipField(data []byte, wireType int) (int, error) {
	switch wireType {
	case wireVarint:
		_, n, err := decodeVarint(data)
		return n, err
	case wireFixed64:
		if len(data) < 8 {
			return 0, errors.New("truncated fixed64")
		}
		return 8, nil
	case wireLengthDelim:
		length, n, err := decodeVarint(data)
		if err != nil {
			return 0, err
		}
		if len(data) < n+int(length) {
			return 0, errors.New("truncated length-delimited field")
		}
		return n + int(length), nil
	case wireFixed32:
		if len(data) < 4 {
			return 0, errors.New("truncated fixed32")
		}
		return 4, nil
	default:
		return 0, fmt.Errorf("unsupported wire type: %d", wireType)
	}
}

// EncodeShannonLteUECap encodes a ComboDocument into valid ShannonLteUECap .binarypb data.
func EncodeShannonLteUECap(doc *ComboDocument) ([]byte, error) {
	buf := make([]byte, 0, 1024*64)

	// Field 1: version (uint32)
	buf = encodeTag(buf, 1, wireVarint)
	buf = encodeVarint(buf, uint64(doc.Version))

	// Field 2: combos (repeated Combo)
	for _, combo := range doc.Combos {
		comboBytes := encodeCombo(&combo)
		buf = encodeTag(buf, 2, wireLengthDelim)
		buf = encodeVarint(buf, uint64(len(comboBytes)))
		buf = append(buf, comboBytes...)
	}

	// Field 3: bitmask (uint32)
	buf = encodeTag(buf, 3, wireVarint)
	buf = encodeVarint(buf, uint64(doc.Bitmask))

	return buf, nil
}

func encodeCombo(combo *Combo) []byte {
	buf := make([]byte, 0, 128)

	// Field 1: components (repeated Component)
	for _, comp := range combo.Components {
		compBytes := encodeComponent(&comp)
		buf = encodeTag(buf, 1, wireLengthDelim)
		buf = encodeVarint(buf, uint64(len(compBytes)))
		buf = append(buf, compBytes...)
	}

	// Field 2: bcs (optional uint32)
	if combo.BCS != 0 {
		buf = encodeTag(buf, 2, wireVarint)
		buf = encodeVarint(buf, uint64(combo.BCS))
	}

	// Field 3: configMaskLow (uint64)
	buf = encodeTag(buf, 3, wireVarint)
	buf = encodeVarint(buf, combo.ConfigMaskLow)

	// Field 4: configMaskHigh (uint32)
	buf = encodeTag(buf, 4, wireVarint)
	buf = encodeVarint(buf, uint64(combo.ConfigMaskHigh))

	return buf
}

func encodeComponent(comp *Component) []byte {
	buf := make([]byte, 0, 16)

	// Field 1: band (int32)
	buf = encodeTag(buf, 1, wireVarint)
	buf = encodeVarint(buf, uint64(comp.Band))

	// Field 2: bwClassMimoDl (int32)
	buf = encodeTag(buf, 2, wireVarint)
	buf = encodeVarint(buf, uint64(comp.BwClassMimoDl))

	// Field 3: bwClassMimoUl (int32)
	buf = encodeTag(buf, 3, wireVarint)
	buf = encodeVarint(buf, uint64(comp.BwClassMimoUl))

	return buf
}

// DecodeDLValue decodes bwClassMimoDl into class letter ("A"-"F") and MIMO streams (2 or 4).
func DecodeDLValue(val int32) (string, int) {
	mimo := 2
	if (val & 1) == 1 {
		mimo = 4
	}
	base := val &^ 1
	var class string
	switch base {
	case 32768:
		class = "A"
	case 16384:
		class = "B"
	case 8192:
		class = "C"
	case 4096:
		class = "D"
	case 2048:
		class = "E"
	case 1024:
		class = "F"
	default:
		class = "?"
	}
	return class, mimo
}

// EncodeDLValue encodes bandwidth class and MIMO to int32.
func EncodeDLValue(class string, mimo int) int32 {
	var base int32
	switch strings.ToUpper(class) {
	case "A":
		base = 32768
	case "B":
		base = 16384
	case "C":
		base = 8192
	case "D":
		base = 4096
	case "E":
		base = 2048
	case "F":
		base = 1024
	default:
		base = 32768
	}
	if mimo == 4 {
		return base | 1
	}
	return base
}

// DecodeULValue decodes bwClassMimoUl into string representation ("None", "A", "C", "D").
func DecodeULValue(val int32) string {
	switch val {
	case 32768:
		return "A"
	case 8192:
		return "C"
	case 4096:
		return "D"
	case 0:
		return "None"
	default:
		return fmt.Sprintf("%d", val)
	}
}

// EncodeULValue encodes UL string class to int32 value.
func EncodeULValue(class string) int32 {
	switch strings.ToUpper(strings.TrimSpace(class)) {
	case "A":
		return 32768
	case "C":
		return 8192
	case "D":
		return 4096
	default:
		return 0
	}
}

// ConfIDsToMasks converts slice of conf_id (0-95) to uint64 low and uint32 high masks.
func ConfIDsToMasks(confIDs []int) (uint64, uint32) {
	var low uint64
	var high uint32

	for _, id := range confIDs {
		if id < 0 || id > 95 {
			continue
		}
		if id == 0 {
			// conf_id 0 is default/implicit, do not set bit 0 unless explicitly configured
			continue
		}
		if id < 64 {
			low |= 1 << uint(id)
		} else {
			high |= 1 << uint(id-64)
		}
	}
	return low, high
}

// MasksToConfIDs decodes bitmasks into slice of active conf_id numbers.
func MasksToConfIDs(low uint64, high uint32) []int {
	ids := make([]int, 0, 16)
	for i := 1; i < 64; i++ {
		if (low & (1 << uint(i))) != 0 {
			ids = append(ids, i)
		}
	}
	for i := 0; i < 32; i++ {
		if (high & (1 << uint(i))) != 0 {
			ids = append(ids, 64+i)
		}
	}
	return ids
}

// FormatComboString formats components into a human-readable combo string (e.g. "1A4-3C4-8A2").
func FormatComboString(components []Component) string {
	parts := make([]string, 0, len(components))
	for _, comp := range components {
		dlClass, dlMimo := DecodeDLValue(comp.BwClassMimoDl)
		ulClass := DecodeULValue(comp.BwClassMimoUl)
		ulSuffix := ""
		if ulClass != "None" && ulClass != "" {
			ulSuffix = fmt.Sprintf("[%s]", ulClass)
		}
		parts = append(parts, fmt.Sprintf("%d%s%d%s", comp.Band, dlClass, dlMimo, ulSuffix))
	}
	return strings.Join(parts, "+")
}

// CalculateTotalStreams returns total spatial streams across all components.
func CalculateTotalStreams(components []Component) int {
	total := 0
	for _, comp := range components {
		_, mimo := DecodeDLValue(comp.BwClassMimoDl)
		total += mimo
	}
	return total
}
