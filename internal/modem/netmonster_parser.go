package modem

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

// PhysicalChannelInfo represents a physical channel config extracted via NetMonster's logic.
type PhysicalChannelInfo struct {
	ConnectionStatus string // "PrimaryServing", "SecondaryServing", "None"
	BandwidthKhz     int
	ChannelNumber    int
	PhysicalCellID   int
	NetworkType      string
}

// NetMonsterCellEntry represents a parsed cellular tower entry from CellInfo.
type NetMonsterCellEntry struct {
	Registered     bool
	NetworkType    string // "LTE", "NR", "GSM", "WCDMA"
	Band           int
	BandName       string
	Bandwidth      string
	PhysicalCellID int
	EARFCN         int
	CellID         int64
	TAC            int
	RSRP           int
	RSRQ           int
	RSSI           int
	SINR           int
	TimingAdvance  int
	DistanceMeter  int
	DistanceStr    string
	Operator       string
	MCC            string
	MNC            string
}

var (
	rePhysicalChannelBlock = regexp.MustCompile(`\{mConnectionStatus=[^\}]+\}`)
	reConnectionStatus     = regexp.MustCompile(`mConnectionStatus=([^,}\s]+)`)
	rePhysicalCellID       = regexp.MustCompile(`(?:mPhysicalCellId|mPci|pci)=([0-9]+)`)
	reBandwidthKhz         = regexp.MustCompile(`mCellBandwidthDownlinkKhz=([0-9]+)`)
	reDownlinkChannel      = regexp.MustCompile(`(?:mDownlinkChannelNumber|mChannelNumber)=([0-9]+)`)
	reChannelNumber        = regexp.MustCompile(`(?:mChannelNumber|mEarfcn|mNrArfcn)=([0-9]+)`)

	reRegistered    = regexp.MustCompile(`mRegistered=(YES|NO|true|false)`)
	reBands         = regexp.MustCompile(`mBands=\[([0-9,\s]+)\]`)
	reBandwidth     = regexp.MustCompile(`mBandwidth=([0-9]+)`)
	reRSRP          = regexp.MustCompile(`rsrp=(-?[0-9]+)`)
	reRSRQ          = regexp.MustCompile(`rsrq=(-?[0-9]+)`)
	reRSSI          = regexp.MustCompile(`rssi=(-?[0-9]+)`)
	reSINR          = regexp.MustCompile(`(?:rssnr|sinr|snr)=(-?[0-9]+)`)
	reSsRsrp        = regexp.MustCompile(`ssRsrp=(-?[0-9]+)`)
	reCsiRsrp       = regexp.MustCompile(`csiRsrp=(-?[0-9]+)`)
	reSsRsrq        = regexp.MustCompile(`ssRsrq=(-?[0-9]+)`)
	reCsiRsrq       = regexp.MustCompile(`csiRsrq=(-?[0-9]+)`)
	reSsSinr        = regexp.MustCompile(`ssSinr=(-?[0-9]+)`)
	reCsiSinr       = regexp.MustCompile(`csiSinr=(-?[0-9]+)`)
	reTimingAdvance = regexp.MustCompile(`(?:timingAdvance|mTimingAdvance|ta)=([0-9]+)`)
	rePci           = regexp.MustCompile(`(?:mPci|pci)=([0-9]+)`)
	reEarfcn        = regexp.MustCompile(`(?:mEarfcn|mNrArfcn|mUarfcn|mArfcn)=([0-9]+)`)
	reCi            = regexp.MustCompile(`(?:mCi|mNci|ci|nci)=([0-9]+)`)
	reTac           = regexp.MustCompile(`(?:mTac|tac)=([0-9]+)`)
	reAlphaLong     = regexp.MustCompile(`mAlphaLong=([^,;}\n\r]+)`)
	reAlphaShort    = regexp.MustCompile(`mAlphaShort=([^,;}\n\r]+)`)
	reMCC           = regexp.MustCompile(`mMcc=([0-9]+)`)
	reMNC           = regexp.MustCompile(`mMnc=([0-9]+)`)
	reCellWidths    = regexp.MustCompile(`mCellBandwidths=\[([^\]]+)\]`)
)

// CalculateTowerDistance calculates approximate distance in meters to the cell tower using 3GPP Timing Advance.
// Ported from NetMonster (cz.mroczis.netmonster.core.model.signal.SignalLte).
// 1 TA step = 78.12 meters (one-way radio wave propagation speed).
func CalculateTowerDistance(ta int) (int, string) {
	if ta <= 0 || ta > 1282 {
		return 0, ""
	}
	meters := int(float64(ta) * 78.12)
	if meters < 1000 {
		return meters, fmt.Sprintf("~%d m (TA: %d)", meters, ta)
	}
	return meters, fmt.Sprintf("~%.1f km (TA: %d)", float64(meters)/1000.0, ta)
}

// ParsePhysicalChannelConfigs extracts unredacted physical channel parameters.
// Ported from NetMonster (cz.mroczis.netmonster.core.model.config.PhysicalChannelConfig).
func ParsePhysicalChannelConfigs(text string) []PhysicalChannelInfo {
	var list []PhysicalChannelInfo
	matches := rePhysicalChannelBlock.FindAllString(text, -1)
	for _, blk := range matches {
		info := PhysicalChannelInfo{
			PhysicalCellID: -1,
			BandwidthKhz:   -1,
			ChannelNumber:  -1,
		}

		if m := reConnectionStatus.FindStringSubmatch(blk); len(m) > 1 {
			info.ConnectionStatus = m[1]
		}
		if m := rePhysicalCellID.FindStringSubmatch(blk); len(m) > 1 {
			if id, err := strconv.Atoi(m[1]); err == nil && id >= 0 && id <= 1007 {
				info.PhysicalCellID = id
			}
		}
		if m := reBandwidthKhz.FindStringSubmatch(blk); len(m) > 1 {
			if bw, err := strconv.Atoi(m[1]); err == nil && bw > 0 && bw != 2147483647 {
				info.BandwidthKhz = bw
			}
		}
		if m := reDownlinkChannel.FindStringSubmatch(blk); len(m) > 1 {
			if ch, err := strconv.Atoi(m[1]); err == nil && ch > 0 && ch != 2147483647 {
				info.ChannelNumber = ch
			}
		}

		list = append(list, info)
	}
	return list
}

var reCellInfoSplit = regexp.MustCompile(`CellInfo(Lte|Nr|Wcdma|Gsm):\{`)

// ParseCellInfoList extracts all serving and neighbor cell towers.
// Ported from NetMonster (cz.mroczis.netmonster.core.telephony.mapper.cell.CellMapperLte & CellMapperNr).
func ParseCellInfoList(text string) []NetMonsterCellEntry {
	var cells []NetMonsterCellEntry
	locs := reCellInfoSplit.FindAllStringSubmatchIndex(text, -1)
	if len(locs) == 0 {
		return cells
	}

	for i, loc := range locs {
		netType := strings.ToUpper(text[loc[2]:loc[3]])
		start := loc[0]
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		blk := text[start:end]

		entry := NetMonsterCellEntry{
			NetworkType:    netType,
			RSRP:           -999,
			RSRQ:           -999,
			RSSI:           -999,
			SINR:           -999,
			PhysicalCellID: -1,
			EARFCN:         -1,
			CellID:         -1,
			TAC:            -1,
		}

		if regM := reRegistered.FindStringSubmatch(blk); len(regM) > 1 {
			entry.Registered = regM[1] == "YES" || regM[1] == "true"
		}

		// PCI extraction
		if pciM := rePci.FindStringSubmatch(blk); len(pciM) > 1 {
			if id, err := strconv.Atoi(pciM[1]); err == nil && id >= 0 && id <= 1007 {
				entry.PhysicalCellID = id
			}
		}

		// EARFCN / NR-ARFCN extraction
		if earfcnM := reEarfcn.FindStringSubmatch(blk); len(earfcnM) > 1 {
			if ch, err := strconv.Atoi(earfcnM[1]); err == nil && ch > 0 && ch != 2147483647 {
				entry.EARFCN = ch
			}
		}

		// Global Cell ID extraction
		if ciM := reCi.FindStringSubmatch(blk); len(ciM) > 1 {
			if ci, err := strconv.ParseInt(ciM[1], 10, 64); err == nil && ci > 0 && ci != 2147483647 && ci != 268435455 {
				entry.CellID = ci
			}
		}

		// TAC extraction
		if tacM := reTac.FindStringSubmatch(blk); len(tacM) > 1 {
			if tac, err := strconv.Atoi(tacM[1]); err == nil && tac > 0 && tac != 2147483647 && tac != 65535 {
				entry.TAC = tac
			}
		}

		// Band extraction & resolution
		if bM := reBands.FindStringSubmatch(blk); len(bM) > 1 {
			rawBands := strings.Split(bM[1], ",")
			if len(rawBands) > 0 {
				if b, err := strconv.Atoi(strings.TrimSpace(rawBands[0])); err == nil && b > 0 {
					entry.Band = b
					if netType == "NR" {
						entry.BandName = fmt.Sprintf("n%d", b)
					} else {
						entry.BandName = fmt.Sprintf("B%d", b)
					}
				}
			}
		}

		// If band was not directly reported in mBands, infer from channel number
		if entry.Band == 0 && entry.EARFCN > 0 {
			if netType == "NR" {
				if bNum, bName := GetNrBand(entry.EARFCN); bNum > 0 {
					entry.Band = bNum
					entry.BandName = bName
				}
			} else {
				if bNum, bName := GetLteBand(entry.EARFCN); bNum > 0 {
					entry.Band = bNum
					entry.BandName = bName
				}
			}
		}

		if bwM := reBandwidth.FindStringSubmatch(blk); len(bwM) > 1 {
			if bw, err := strconv.Atoi(bwM[1]); err == nil && bw > 0 && bw != 2147483647 {
				entry.Bandwidth = FormatBandWidthKhz(bw)
			}
		}

		// Signal Metrics extraction (LTE & NR)
		if rsrpM := reRSRP.FindStringSubmatch(blk); len(rsrpM) > 1 {
			if v, err := strconv.Atoi(rsrpM[1]); err == nil && v != 2147483647 {
				entry.RSRP = v
			}
		}
		if rsrqM := reRSRQ.FindStringSubmatch(blk); len(rsrqM) > 1 {
			if v, err := strconv.Atoi(rsrqM[1]); err == nil && v != 2147483647 {
				entry.RSRQ = v
			}
		}
		if rssiM := reRSSI.FindStringSubmatch(blk); len(rssiM) > 1 {
			if v, err := strconv.Atoi(rssiM[1]); err == nil && v != 2147483647 {
				entry.RSSI = v
			}
		}
		if sinrM := reSINR.FindStringSubmatch(blk); len(sinrM) > 1 {
			if v, err := strconv.Atoi(sinrM[1]); err == nil && v != 2147483647 {
				entry.SINR = v
			}
		}

		// 5G NR SS & CSI Metrics fallback
		if netType == "NR" {
			if entry.RSRP == -999 {
				if m := reSsRsrp.FindStringSubmatch(blk); len(m) > 1 {
					if v, err := strconv.Atoi(m[1]); err == nil && v != 2147483647 {
						entry.RSRP = v
					}
				} else if m := reCsiRsrp.FindStringSubmatch(blk); len(m) > 1 {
					if v, err := strconv.Atoi(m[1]); err == nil && v != 2147483647 {
						entry.RSRP = v
					}
				}
			}
			if entry.RSRQ == -999 {
				if m := reSsRsrq.FindStringSubmatch(blk); len(m) > 1 {
					if v, err := strconv.Atoi(m[1]); err == nil && v != 2147483647 {
						entry.RSRQ = v
					}
				} else if m := reCsiRsrq.FindStringSubmatch(blk); len(m) > 1 {
					if v, err := strconv.Atoi(m[1]); err == nil && v != 2147483647 {
						entry.RSRQ = v
					}
				}
			}
			if entry.SINR == -999 {
				if m := reSsSinr.FindStringSubmatch(blk); len(m) > 1 {
					if v, err := strconv.Atoi(m[1]); err == nil && v != 2147483647 {
						entry.SINR = v
					}
				} else if m := reCsiSinr.FindStringSubmatch(blk); len(m) > 1 {
					if v, err := strconv.Atoi(m[1]); err == nil && v != 2147483647 {
						entry.SINR = v
					}
				}
			}
		}

		// NetMonster Negative Inversion Quirk (some modems report positive signal values e.g. 85 instead of -85 dBm)
		if entry.RSRP > 0 && entry.RSRP <= 140 {
			entry.RSRP = -entry.RSRP
		}
		if entry.RSRQ > 0 && entry.RSRQ <= 30 {
			entry.RSRQ = -entry.RSRQ
		}
		if entry.RSSI > 0 && entry.RSSI <= 140 {
			entry.RSSI = -entry.RSSI
		}

		// Bluejay (Pixel 6a) SNR division bug normalization
		if entry.SINR > 30 {
			entry.SINR = entry.SINR / 10
		}

		// Timing Advance extraction (only valid for registered serving cell)
		if taM := reTimingAdvance.FindStringSubmatch(blk); len(taM) > 1 {
			if ta, err := strconv.Atoi(taM[1]); err == nil && ta > 0 && ta <= 1282 && ta != 2147483647 {
				if entry.Registered {
					entry.TimingAdvance = ta
					entry.DistanceMeter, entry.DistanceStr = CalculateTowerDistance(ta)
				}
			}
		}

		// Operator & PLMN
		if alphaM := reAlphaLong.FindStringSubmatch(blk); len(alphaM) > 1 {
			val := strings.Trim(alphaM[1], "\"', ")
			if val != "" && val != "null" {
				entry.Operator = val
			}
		}
		if mccM := reMCC.FindStringSubmatch(blk); len(mccM) > 1 {
			entry.MCC = mccM[1]
		}
		if mncM := reMNC.FindStringSubmatch(blk); len(mncM) > 1 {
			entry.MNC = mncM[1]
		}

		// Pixel Tensor Postprocessor: In Tensor devices, GSM neighbors are mistakenly reported as LTE
		// with no band, no RSRQ, and RSRP actually being RSSI.
		if entry.NetworkType == "LTE" && !entry.Registered && entry.Band == 0 && entry.EARFCN <= 0 && entry.RSRQ == -999 && entry.RSRP != -999 {
			entry.NetworkType = "GSM"
			entry.RSSI = entry.RSRP
			entry.RSRP = -999
		}

		cells = append(cells, entry)
	}

	return cells
}

// DetectNrNsaState identifies 5G Non-Standalone (NSA / EN-DC) and Standalone (SA) status.
// Ported from NetMonster (cz.mroczis.netmonster.core.feature.NrNsaStateParser).
func DetectNrNsaState(dumpsysText string, pChannels []PhysicalChannelInfo, cells []NetMonsterCellEntry) (isNSA bool, isSA bool, desc string) {
	// 1. Direct NR Standalone (SA): Registered primary cell is NR
	for _, c := range cells {
		if c.Registered && c.NetworkType == "NR" {
			return false, true, "5G SA"
		}
	}

	// 2. 5G NSA (Non-Standalone EN-DC) detection
	hasNrStateConnected := strings.Contains(dumpsysText, "nrState=CONNECTED") ||
		strings.Contains(dumpsysText, "mNrState=CONNECTED") ||
		strings.Contains(dumpsysText, "nsaState=5") // Huawei Android P+

	isNrAvailable := strings.Contains(dumpsysText, "isNrAvailable=true") ||
		strings.Contains(dumpsysText, "isNrAvailable = true")
	isEnDcAvailable := strings.Contains(dumpsysText, "isEnDcAvailable=true") ||
		strings.Contains(dumpsysText, "isEnDcAvailable = true") ||
		strings.Contains(dumpsysText, "EnDc=true")
	isDcNrRestricted := strings.Contains(dumpsysText, "isDcNrRestricted=true") ||
		strings.Contains(dumpsysText, "isDcNrRestricted = true")
	is5GAllocated := strings.Contains(dumpsysText, "5G Allocated=true") // LG Android P+

	hasSecondaryNrCell := false
	for _, c := range cells {
		if c.NetworkType == "NR" {
			hasSecondaryNrCell = true
			break
		}
	}
	for _, pc := range pChannels {
		if pc.NetworkType == "NR" {
			hasSecondaryNrCell = true
			break
		}
	}

	// Connected if explicit flag or (Android 12-16 Baklava sensor masking: nrAvailable + secondary NR cell)
	if hasNrStateConnected || (isEnDcAvailable && is5GAllocated) {
		return true, false, "5G NSA"
	}
	if isNrAvailable && !isDcNrRestricted && hasSecondaryNrCell {
		return true, false, "5G NSA"
	}
	if isNrAvailable && hasSecondaryNrCell {
		return true, false, "5G NSA"
	}

	return false, false, ""
}

// DetectCarrierAggregation resolves Carrier Aggregation status and band aggregation list.
// Ported from NetMonster (cz.mroczis.netmonster.core.feature.detect.DetectorLteAdvancedServiceState & DetectorLteAdvancedPhysicalChannel).
func DetectCarrierAggregation(dumpsysText string, pChannels []PhysicalChannelInfo, cells []NetMonsterCellEntry) (bool, string, string) {
	isCA := false

	// NetMonster Rule 1: ServiceState flags
	if strings.Contains(dumpsysText, "isUsingCarrierAggregation=true") ||
		strings.Contains(dumpsysText, "mIsUsingCarrierAggregation=true") ||
		strings.Contains(dumpsysText, "mIsUsingCarrierAggregation = true") ||
		strings.Contains(dumpsysText, "accessNetworkTechnology=LTE-CA") ||
		strings.Contains(dumpsysText, "AdvanceMode1") {
		isCA = true
	}

	// NetMonster Rule 2: PhysicalChannelConfigs with SecondaryServing
	var caBands []string
	var bandwidthList []int

	for _, pc := range pChannels {
		if pc.ConnectionStatus == "SecondaryServing" {
			isCA = true
			if pc.ChannelNumber > 0 {
				if bNum, _ := GetLteBand(pc.ChannelNumber); bNum > 0 {
					bStr := fmt.Sprintf("B%d", bNum)
					found := false
					for _, existing := range caBands {
						if existing == bStr {
							found = true
							break
						}
					}
					if !found {
						caBands = append(caBands, bStr)
					}
				}
			}
		}
		if pc.BandwidthKhz > 0 {
			bandwidthList = append(bandwidthList, pc.BandwidthKhz)
		}
	}

	// NetMonster Rule 3: Primary Serving Band
	var servingBand int
	for _, c := range cells {
		if c.Registered && c.Band > 0 {
			servingBand = c.Band
			break
		}
	}

	if servingBand > 0 {
		sBandStr := fmt.Sprintf("B%d", servingBand)
		hasServing := false
		for _, b := range caBands {
			if b == sBandStr {
				hasServing = true
				break
			}
		}
		if !hasServing {
			caBands = append([]string{sBandStr}, caBands...)
		}
	}

	// If dumpsys contains cell bandwidths array e.g. [20000, 10000]
	if m := reCellWidths.FindStringSubmatch(dumpsysText); len(m) > 1 {
		items := strings.Split(m[1], ",")
		if len(items) > 1 {
			isCA = true
		}
		if len(bandwidthList) == 0 {
			for _, item := range items {
				if w, err := strconv.Atoi(strings.TrimSpace(item)); err == nil && w > 0 && w != 2147483647 {
					bandwidthList = append(bandwidthList, w)
				}
			}
		}
	}

	bandLabel := ""
	if len(caBands) > 1 && isCA {
		bandLabel = fmt.Sprintf("%s [%dCA Active]", strings.Join(caBands, " + "), len(caBands))
	} else if len(caBands) == 1 && isCA {
		bandLabel = fmt.Sprintf("%s [CA Active]", caBands[0])
	} else if len(caBands) >= 1 {
		bandLabel = caBands[0]
	}

	bwLabel := FormatMultiBandwidths(bandwidthList)

	return isCA, bandLabel, bwLabel
}

// QueryNetMonsterSignal executes the high-speed NetMonster-based telemetry pipeline.
func QueryNetMonsterSignal(activeSlot int) SignalInfo {
	sig := SignalInfo{
		RSRP:        -999,
		RSRQ:        -999,
		SINR:        -999,
		RSSI:        -999,
		Operator:    "Unknown",
		NetworkType: "LTE",
		Band:        "Unknown",
		CellID:      "Unknown",
		TAC:         "Unknown",
		PCI:         "Unknown",
		EARFCN:      0,
		Bandwidth:   "Unknown",
	}

	// Fast dumpsys query (<20ms, increased to 350 lines to prevent Dual SIM / high neighbor truncation)
	out, err := exec.Command(config.SUBin, "-c", "dumpsys telephony.registry 2>/dev/null | grep -E 'mPhysicalChannelConfigs|mCellInfo|mSignalStrength|mServiceState|mChannelNumber|notifyServiceStateForSubscriber' | head -n 350").Output()
	if err != nil || len(out) == 0 {
		return sig
	}
	text := string(out)

	// 1. Parse Physical Channel Configs for unredacted PhysicalCellId & Bandwidth
	pChannels := ParsePhysicalChannelConfigs(text)
	for _, pc := range pChannels {
		if pc.ConnectionStatus == "PrimaryServing" || pc.PhysicalCellID >= 0 {
			if pc.PhysicalCellID >= 0 {
				sig.PCI = fmt.Sprintf("%d", pc.PhysicalCellID)
			}
			if pc.ChannelNumber > 0 {
				sig.EARFCN = pc.ChannelNumber
			}
			if pc.BandwidthKhz > 0 && sig.Bandwidth == "Unknown" {
				sig.Bandwidth = FormatBandWidthKhz(pc.BandwidthKhz)
			}
			break
		}
	}

	// 2. Parse CellInfo List for serving/neighbor metrics, bands & timing advance
	cells := ParseCellInfoList(text)
	for _, c := range cells {
		if c.Registered {
			if c.RSRP != -999 {
				sig.RSRP = c.RSRP
			}
			if c.RSRQ != -999 {
				sig.RSRQ = c.RSRQ
			}
			if c.RSSI != -999 {
				sig.RSSI = c.RSSI
			}
			if c.SINR != -999 {
				sig.SINR = c.SINR
			}
			if c.Band > 0 && (sig.Band == "Unknown" || sig.Band == "") {
				sig.Band = c.BandName
				if sig.Band == "" {
					sig.Band = fmt.Sprintf("B%d", c.Band)
				}
			}
			if c.Bandwidth != "" && sig.Bandwidth == "Unknown" {
				sig.Bandwidth = c.Bandwidth
			}
			if c.Operator != "" {
				sig.Operator = c.Operator
			}
			if c.CellID > 0 {
				sig.CellID = fmt.Sprintf("%d", c.CellID)
			}
			if c.TAC > 0 {
				sig.TAC = fmt.Sprintf("%d", c.TAC)
			}
			if c.TimingAdvance > 0 {
				sig.TimingAdvance = c.TimingAdvance
				sig.DistanceMeter = c.DistanceMeter
				sig.DistanceStr = c.DistanceStr
			}
			break
		}
	}

	// 3. Channel Number / EARFCN parsing from mChannelNumber if missing
	if sig.EARFCN == 0 {
		matches := reChannelNumber.FindAllStringSubmatch(text, -1)
		for i := len(matches) - 1; i >= 0; i-- {
			if len(matches[i]) > 1 {
				if ch, err := strconv.Atoi(matches[i][1]); err == nil && ch > 0 && ch != 2147483647 {
					sig.EARFCN = ch
					if sig.Band == "Unknown" || sig.Band == "" {
						if bNum, _ := GetLteBand(ch); bNum > 0 {
							sig.Band = fmt.Sprintf("B%d", bNum)
						}
					}
					break
				}
			}
		}
	}

	// 4. NetMonster Carrier Aggregation Detection
	isCA, caBandStr, caBwStr := DetectCarrierAggregation(text, pChannels, cells)
	if caBandStr != "" {
		sig.Band = caBandStr
	}
	if caBwStr != "" && caBwStr != "Unknown" {
		sig.Bandwidth = caBwStr
	}

	// 5. 5G NR NSA & SA State Detection (NetMonster NrNsaStateParser)
	isNSA, isSA, _ := DetectNrNsaState(text, pChannels, cells)
	if isSA {
		sig.NetworkType = "5G SA"
	} else if isNSA {
		if isCA {
			sig.NetworkType = "5G NSA (EN-DC + CA)"
		} else {
			sig.NetworkType = "5G NSA"
		}
	} else if isCA {
		sig.NetworkType = "4G LTE-A"
	} else {
		sig.NetworkType = "4G LTE"
	}

	// 6. Timing Advance Fallback from dumpsys text if not yet resolved
	if sig.TimingAdvance == 0 {
		if taM := reTimingAdvance.FindStringSubmatch(text); len(taM) > 1 {
			if ta, err := strconv.Atoi(taM[1]); err == nil && ta > 0 && ta <= 1282 && ta != 2147483647 {
				sig.TimingAdvance = ta
				sig.DistanceMeter, sig.DistanceStr = CalculateTowerDistance(ta)
			}
		}
	}

	// 7. Operator Resolution (getprop -> offline PLMN table fallback)
	if propOut, err := exec.Command(config.SUBin, "-c", "getprop gsm.operator.alpha").Output(); err == nil {
		propStr := strings.TrimSpace(string(propOut))
		if propStr != "" {
			parts := strings.Split(propStr, ",")
			if activeSlot < len(parts) && strings.TrimSpace(parts[activeSlot]) != "" {
				sig.Operator = strings.TrimSpace(parts[activeSlot])
			} else {
				for _, p := range parts {
					if p = strings.TrimSpace(p); p != "" && p != "Unknown" && p != "3" {
						sig.Operator = p
						break
					}
				}
			}
		}
	}

	// Offline PLMN table fallback from MCC/MNC if operator still unknown
	if sig.Operator == "Unknown" || sig.Operator == "" {
		for _, c := range cells {
			if c.Registered && c.MCC != "" && c.MNC != "" {
				if op := GetOperatorByPlmn(c.MCC, c.MNC); op != "" {
					sig.Operator = op
					break
				}
			}
		}
	}

	// 8. Signal Metrics from SignalStrength fallback if not extracted from CellInfo
	if sig.RSRP == -999 || sig.RSRQ == -999 || sig.RSSI == -999 || sig.SINR == -999 {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if sig.RSRP == -999 && (strings.Contains(line, "rsrp=") || strings.Contains(line, "rsrp:")) {
				if v := extractIntFromLine(line, "rsrp"); v != -999 && v != 2147483647 {
					sig.RSRP = v
				}
			}
			if sig.RSRQ == -999 && (strings.Contains(line, "rsrq=") || strings.Contains(line, "rsrq:")) {
				if v := extractIntFromLine(line, "rsrq"); v != -999 && v != 2147483647 {
					sig.RSRQ = v
				}
			}
			if sig.RSSI == -999 && (strings.Contains(line, "rssi=") || strings.Contains(line, "rssi:")) {
				if v := extractIntFromLine(line, "rssi"); v != -999 && v != 2147483647 {
					sig.RSSI = v
				}
			}
			if sig.SINR == -999 && (strings.Contains(line, "rssnr=") || strings.Contains(line, "sinr=") || strings.Contains(line, "snr=")) {
				if v := extractIntFromLine(line, "rssnr"); v != -999 && v != 2147483647 {
					sig.SINR = v
				} else if v := extractIntFromLine(line, "sinr"); v != -999 && v != 2147483647 {
					sig.SINR = v
				}
			}
		}
	}

	// 9. Check cached scan result if present for masked CellID/TAC
	if cached := GetCachedScanResult(); cached != nil && cached.ServingCell != nil {
		if sig.CellID == "Unknown" && cached.ServingCell.CellID > 0 {
			sig.CellID = fmt.Sprintf("%d", cached.ServingCell.CellID)
		}
		if sig.PCI == "Unknown" && cached.ServingCell.PCI > 0 {
			sig.PCI = fmt.Sprintf("%d", cached.ServingCell.PCI)
		}
		if sig.TAC == "Unknown" && cached.ServingCell.TAC > 0 {
			sig.TAC = fmt.Sprintf("%d", cached.ServingCell.TAC)
		}
	}

	CalculateSignalMetrics(&sig)
	return sig
}

// ScanCellsFromDumpsys discovers all serving, secondary CA, and neighbor cell towers using NetMonster logic.
func ScanCellsFromDumpsys(activeSlot int) *CellScanResult {
	res := &CellScanResult{
		Success:     true,
		CarrierAggr: make([]NearbyCellInfo, 0),
		Neighbors:   make([]NearbyCellInfo, 0),
		ScanTime:    time.Now().Format("15:04:05"),
	}

	out, err := exec.Command(config.SUBin, "-c", "dumpsys telephony.registry 2>/dev/null | grep -E 'mPhysicalChannelConfigs|mCellInfo|mSignalStrength|mServiceState|mChannelNumber' | head -n 350").Output()
	if err != nil || len(out) == 0 {
		return res
	}
	text := string(out)

	pChannels := ParsePhysicalChannelConfigs(text)
	cells := ParseCellInfoList(text)

	var primaryPCI uint16
	var primaryBW string
	var primaryChannel uint32

	for _, pc := range pChannels {
		if pc.PhysicalCellID >= 0 && primaryPCI == 0 {
			primaryPCI = uint16(pc.PhysicalCellID)
		}
		if pc.ChannelNumber > 0 && primaryChannel == 0 {
			primaryChannel = uint32(pc.ChannelNumber)
		}
		if pc.BandwidthKhz > 0 && primaryBW == "" {
			primaryBW = FormatBandWidthKhz(pc.BandwidthKhz)
		}
	}

	// Channel number fallback
	if primaryChannel == 0 {
		matches := reChannelNumber.FindAllStringSubmatch(text, -1)
		for i := len(matches) - 1; i >= 0; i-- {
			if len(matches[i]) > 1 {
				if ch, err := strconv.Atoi(matches[i][1]); err == nil && ch > 0 && ch != 2147483647 {
					primaryChannel = uint32(ch)
					break
				}
			}
		}
	}

	// Build serving cell
	for _, c := range cells {
		if c.Registered {
			bNum := c.Band
			bName := c.BandName
			if primaryChannel > 0 {
				if num, name := EarfcnToBand(primaryChannel); num > 0 {
					bNum = num
					bName = name
				}
			}
			distStr := c.DistanceStr
			if distStr == "" {
				distStr = "Connected (Primary)"
			}

			netType := strings.ToLower(c.NetworkType)
			if netType == "" {
				netType = "lte"
			}

			res.ServingCell = &NearbyCellInfo{
				Type:          netType,
				Role:          "serving",
				PCI:           primaryPCI,
				EARFCN:        primaryChannel,
				Band:          bNum,
				BandName:      bName,
				CellID:        uint32(c.CellID),
				TAC:           uint16(c.TAC),
				RSRP:          float64(c.RSRP),
				RSRQ:          float64(c.RSRQ),
				RSSI:          float64(c.RSSI),
				SINR:          float64(c.SINR),
				TimingAdvance: c.TimingAdvance,
				DistanceMeter: c.DistanceMeter,
				DistanceStr:   distStr,
			}
			break
		}
	}

	// Fallback serving cell if no cell marked registered
	if res.ServingCell == nil && (primaryPCI > 0 || primaryChannel > 0) {
		bNum, bName := EarfcnToBand(primaryChannel)
		res.ServingCell = &NearbyCellInfo{
			Type:        "lte",
			Role:        "serving",
			PCI:         primaryPCI,
			EARFCN:      primaryChannel,
			Band:        bNum,
			BandName:    bName,
			DistanceStr: "Connected (Primary)",
		}
	}

	// Secondary CA Carriers strictly from physical channels (NetMonster PhysicalChannelPostprocessor)
	for _, pc := range pChannels {
		if pc.ConnectionStatus == "SecondaryServing" && pc.ChannelNumber > 0 {
			bNum, bName := EarfcnToBand(uint32(pc.ChannelNumber))
			res.CarrierAggr = append(res.CarrierAggr, NearbyCellInfo{
				Type:        "lte",
				Role:        "scell",
				PCI:         uint16(pc.PhysicalCellID),
				EARFCN:      uint32(pc.ChannelNumber),
				Band:        bNum,
				BandName:    bName,
				DistanceStr: "CA Secondary Carrier",
			})
		}
	}

	// Neighbor cells (NetMonster CellMapperLte / CellMapperNr)
	for _, c := range cells {
		if !c.Registered {
			earfcn := primaryChannel
			if c.EARFCN > 0 {
				earfcn = uint32(c.EARFCN)
			}
			bNum := c.Band
			bName := c.BandName
			if bNum == 0 && earfcn > 0 {
				bNum, bName = EarfcnToBand(earfcn)
			}
			pci := uint16(0)
			if c.PhysicalCellID > 0 {
				pci = uint16(c.PhysicalCellID)
			}

			netType := strings.ToLower(c.NetworkType)
			if netType == "" {
				netType = "lte"
			}

			res.Neighbors = append(res.Neighbors, NearbyCellInfo{
				Type:        netType,
				Role:        "neighbor",
				PCI:         pci,
				EARFCN:      earfcn,
				Band:        bNum,
				BandName:    bName,
				CellID:        uint32(c.CellID),
				TAC:           uint16(c.TAC),
				RSRP:        float64(c.RSRP),
				RSRQ:        float64(c.RSRQ),
				RSSI:        float64(c.RSSI),
				SINR:        float64(c.SINR),
				DistanceStr: "Neighbor Base Station",
			})
		}
	}

	total := 0
	if res.ServingCell != nil {
		total++
	}
	total += len(res.CarrierAggr) + len(res.Neighbors)
	res.TotalCells = total

	return res
}
