package modem

import (
	"fmt"
	"strings"
)

// BandEntity defines a channel range and its corresponding 3GPP frequency band name and number.
// Ported from NetMonster Core (cz.mroczis.netmonster.core.db.BandTableLte & BandTableNr).
type BandEntity struct {
	MinChannel int
	MaxChannel int
	Name       string
	Number     int
}

// 3GPP 36.101 LTE Band Table (FDD & TDD)
var lteBandTable = []BandEntity{
	{0, 599, "2100", 1},
	{600, 1199, "1900", 2},
	{1200, 1949, "1800", 3},
	{1950, 2399, "AWS", 4},
	{2400, 2649, "850", 5},
	{2650, 2749, "900", 6},
	{2750, 3449, "2600", 7},
	{3450, 3799, "900", 8},
	{3800, 4149, "1800", 9},
	{4150, 4749, "AWS", 10},
	{4750, 5009, "1500", 11},
	{5010, 5179, "700", 12},
	{5180, 5279, "700", 13},
	{5280, 5729, "700", 14},
	{5730, 5849, "700", 17},
	{5850, 5999, "800", 18},
	{6000, 6149, "800", 19},
	{6150, 6449, "800", 20},
	{6450, 6599, "1500", 21},
	{6600, 7499, "3500", 22},
	{7500, 7699, "2000", 23},
	{7700, 8039, "1600", 24},
	{8040, 8689, "1900", 25},
	{8690, 9039, "850", 26},
	{9040, 9209, "800", 27},
	{9210, 9659, "700", 28},
	{9660, 9769, "700", 29},
	{9770, 9869, "2300", 30},
	{9870, 9919, "450", 31},
	{9920, 10359, "1500", 32},

	// TDD Ranges
	{36000, 36199, "1900", 33},
	{36200, 36349, "2000", 34},
	{36350, 36949, "PCS", 35},
	{36950, 37549, "PCS", 36},
	{37550, 37749, "PCS", 37},
	{37750, 38249, "2600", 38},
	{38250, 38649, "1900", 39},
	{38650, 39649, "2300", 40},
	{39650, 41589, "2500", 41},
	{41590, 43589, "3500", 42},
	{43590, 45589, "3700", 43},
	{45590, 46589, "700", 44},
	{46590, 46789, "1500", 45},
	{46790, 54539, "5200", 46},
	{54540, 55239, "5900", 47},
	{55240, 56739, "3600", 48},
	{56740, 58239, "3600", 49},
	{58240, 59089, "1500", 50},
	{59090, 59139, "1500", 51},
	{59140, 60139, "3300", 52},
	{60140, 60254, "2500", 53},

	// Extended LTE bands
	{65536, 66435, "2100", 65},
	{66436, 67335, "AWS", 66},
	{67336, 67535, "700", 67},
	{67536, 67835, "700", 68},
	{67836, 68335, "2500", 69},
	{68336, 68585, "AWS", 70},
	{68586, 68935, "600", 71},
	{68936, 68985, "450", 72},
	{68986, 69035, "450", 73},
	{69036, 69465, "L", 74},
	{69466, 70315, "1500", 75},
	{70316, 70365, "1500", 76},
	{70366, 70545, "700", 85},
	{70546, 70595, "410", 87},
	{70596, 70645, "410", 88},
	{70646, 70655, "700", 103},
}

// 3GPP 38.101-1 NR Band Table
var nrBandTable = []BandEntity{
	{123400, 130400, "600", 71},
	{143400, 145600, "700", 29},
	{145800, 149200, "700", 12},
	{151600, 160600, "700", 28},
	{151600, 153600, "700", 14},
	{158200, 164200, "800", 20},
	{171800, 178800, "850", 26},
	{172000, 175000, "800", 18},
	{173800, 178800, "850", 5},
	{185000, 192000, "900", 8},
	{285400, 286400, "1500", 51},
	{286400, 303400, "1500", 75},
	{361000, 376000, "1800", 3},
	{376000, 384000, "1900", 39},
	{386000, 398000, "1900", 2},
	{399000, 404000, "AWS", 70},
	{422000, 440000, "2100", 1},
	{422000, 440000, "AWS", 66},
	{460000, 480000, "2300", 40},
	{470000, 472000, "2300", 30},
	{499200, 538000, "2500", 41},
	{514000, 524000, "2600", 38},
	{524000, 538000, "2600", 7},
	{620000, 680000, "3700", 77},
	{620000, 653333, "3500", 78},
	{636667, 646666, "3600", 48},
	{693334, 733333, "4500", 79},
	{743334, 795000, "5200", 46},
	{790334, 795000, "5900", 47},
}

// GetNrFrequencyKhz calculates downlink frequency in kHz from NR-ARFCN.
// Ported from 3GPP 38.101-1 / 5.4.2.1 NR-ARFCN and channel raster.
func GetNrFrequencyKhz(arfcn int) int {
	if arfcn <= 600000 {
		return 5 * arfcn
	}
	return 3000000 + 15*(arfcn-600000)
}

// NetMonster 16-bit integer overflow mapping: MCC -> list of invalid ranges where EARFCN = value + 65536.
var earfcnOverflowRanges = map[string][][2]int{
	"228": {{1950, 1999}, {3930, 4779}}, // Switzerland, b67 + b75
	"302": {{1200, 1799}},                // Canada, b66
	"310": {{1200, 1799}, {3050, 3399}}, // USA, b66 + b71
	"311": {{1200, 1799}, {3050, 3399}}, // USA, b66 + b71
	"466": {{900, 1199}},                // Taiwan, b66
	"730": {{900, 1799}},                // Chile, b66
}

// GetLteBandWithMcc resolves LTE band number and frequency string from EARFCN, with optional MCC overflow correction.
func GetLteBandWithMcc(earfcn int, mcc string) (int, string) {
	if earfcn <= 0 || earfcn == 2147483647 {
		return 0, ""
	}

	// Check if country-specific 16-bit overflow fix applies
	fixedEarfcn := earfcn
	if ranges, ok := earfcnOverflowRanges[mcc]; ok {
		for _, r := range ranges {
			if earfcn >= r[0] && earfcn <= r[1] {
				fixedEarfcn = earfcn + 65536
				break
			}
		}
	}

	for _, b := range lteBandTable {
		if fixedEarfcn >= b.MinChannel && fixedEarfcn <= b.MaxChannel {
			return b.Number, fmt.Sprintf("B%d (%s MHz)", b.Number, b.Name)
		}
	}

	// Generic fallback if still unresolved and in 65536 range
	if fixedEarfcn == earfcn && earfcn < 65536 {
		cand := earfcn + 65536
		for _, b := range lteBandTable {
			if cand >= b.MinChannel && cand <= b.MaxChannel {
				return b.Number, fmt.Sprintf("B%d (%s MHz)", b.Number, b.Name)
			}
		}
	}

	return 0, ""
}

// GetLteBand resolves LTE band number and frequency string from EARFCN.
func GetLteBand(earfcn int) (int, string) {
	return GetLteBandWithMcc(earfcn, "")
}

// GetNrBandWithHints resolves 5G NR band number and frequency string with band hint disambiguation & raster check.
func GetNrBandWithHints(nrarfcn int, bandHints ...int) (int, string) {
	if nrarfcn <= 0 || nrarfcn == 2147483647 {
		return 0, ""
	}

	// 1. Gather all candidate bands matching ARFCN range
	var candidates []BandEntity
	for _, b := range nrBandTable {
		if nrarfcn >= b.MinChannel && nrarfcn <= b.MaxChannel {
			candidates = append(candidates, b)
		}
	}

	if len(candidates) == 0 {
		return 0, ""
	}

	// 2. If band hints are provided (e.g. from mBands), match against them first
	if len(bandHints) > 0 {
		for _, hint := range bandHints {
			for _, c := range candidates {
				if c.Number == hint {
					return c.Number, fmt.Sprintf("n%d (%s MHz)", c.Number, c.Name)
				}
			}
		}
	}

	if len(candidates) == 1 {
		return candidates[0].Number, fmt.Sprintf("n%d (%s MHz)", candidates[0].Number, candidates[0].Name)
	}

	// 3. Raster frequency check (NetMonster 5000 kHz raster rule)
	freq := GetNrFrequencyKhz(nrarfcn)
	var rasterMatches []BandEntity
	for _, c := range candidates {
		startFreq := GetNrFrequencyKhz(c.MinChannel)
		if (startFreq-freq)%5000 == 0 {
			rasterMatches = append(rasterMatches, c)
		}
	}

	if len(rasterMatches) == 1 {
		return rasterMatches[0].Number, fmt.Sprintf("n%d (%s MHz)", rasterMatches[0].Number, rasterMatches[0].Name)
	} else if len(rasterMatches) > 1 {
		return rasterMatches[0].Number, fmt.Sprintf("n%d (%s MHz)", rasterMatches[0].Number, rasterMatches[0].Name)
	}

	// Default fallback to first candidate
	return candidates[0].Number, fmt.Sprintf("n%d (%s MHz)", candidates[0].Number, candidates[0].Name)
}

// GetNrBand resolves 5G NR band number and frequency string from NR-ARFCN.
func GetNrBand(nrarfcn int) (int, string) {
	return GetNrBandWithHints(nrarfcn)
}

// FormatBandWidthKhz converts kHz (e.g. 20000 or 10000) or Hz into a readable MHz string.
func FormatBandWidthKhz(khz int) string {
	if khz <= 0 || khz == 2147483647 {
		return ""
	}
	if khz >= 1000000 {
		return fmt.Sprintf("%d MHz", khz/1000000)
	}
	if khz >= 1000 {
		return fmt.Sprintf("%d MHz", khz/1000)
	}
	return fmt.Sprintf("%d MHz", khz)
}

// FormatMultiBandwidths formats an array of bandwidths (in kHz or MHz) into e.g. "20 MHz + 10 MHz (30 MHz)".
func FormatMultiBandwidths(widths []int) string {
	var parts []string
	totalKhz := 0
	for _, w := range widths {
		if w > 0 && w != 2147483647 {
			s := FormatBandWidthKhz(w)
			if s != "" {
				parts = append(parts, s)
				if w >= 1000 {
					totalKhz += w / 1000
				} else {
					totalKhz += w
				}
			}
		}
	}
	if len(parts) == 0 {
		return "Unknown"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("%s (%d MHz Total)", strings.Join(parts, " + "), totalKhz)
}
