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
	NetworkType    string // "LTE", "NR"
	Band           int
	BandName       string
	Bandwidth      string
	PhysicalCellID int
	EARFCN         int
	RSRP           int
	RSRQ           int
	RSSI           int
	SINR           int
	Operator       string
	MCC            string
	MNC            string
}

var (
	rePhysicalChannelBlock = regexp.MustCompile(`\{mConnectionStatus=[^\}]+\}`)
	reConnectionStatus     = regexp.MustCompile(`mConnectionStatus=([^,}\s]+)`)
	rePhysicalCellID       = regexp.MustCompile(`mPhysicalCellId=([0-9]+)`)
	reBandwidthKhz         = regexp.MustCompile(`mCellBandwidthDownlinkKhz=([0-9]+)`)
	reDownlinkChannel      = regexp.MustCompile(`mDownlinkChannelNumber=([0-9]+)`)
	reChannelNumber        = regexp.MustCompile(`mChannelNumber=([0-9]+)`)

	reCellInfoLte = regexp.MustCompile(`CellInfoLte:\{([^\}]*CellIdentityLte:[^\}]*\}[^\}]*)\}`)
	reCellInfoNr  = regexp.MustCompile(`CellInfoNr:\{([^\}]*CellIdentityNr:[^\}]*\}[^\}]*)\}`)
	reRegistered  = regexp.MustCompile(`mRegistered=(YES|NO|true|false)`)
	reBands       = regexp.MustCompile(`mBands=\[([0-9]+)\]`)
	reBandwidth   = regexp.MustCompile(`mBandwidth=([0-9]+)`)
	reRSRP        = regexp.MustCompile(`rsrp=(-?[0-9]+)`)
	reRSRQ        = regexp.MustCompile(`rsrq=(-?[0-9]+)`)
	reRSSI        = regexp.MustCompile(`rssi=(-?[0-9]+)`)
	reSINR        = regexp.MustCompile(`(?:rssnr|sinr|snr)=(-?[0-9]+)`)
	reAlphaLong   = regexp.MustCompile(`mAlphaLong=([^,;}\n\r]+)`)
	reAlphaShort  = regexp.MustCompile(`mAlphaShort=([^,;}\n\r]+)`)
	reMCC         = regexp.MustCompile(`mMcc=([0-9]+)`)
	reMNC         = regexp.MustCompile(`mMnc=([0-9]+)`)
	reCellWidths  = regexp.MustCompile(`mCellBandwidths=\[([^\]]+)\]`)
)

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
// Ported from NetMonster (cz.mroczis.netmonster.core.telephony.mapper.cell.CellMapperLte).
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
		}

		if regM := reRegistered.FindStringSubmatch(blk); len(regM) > 1 {
			entry.Registered = regM[1] == "YES" || regM[1] == "true"
		}
		if bM := reBands.FindStringSubmatch(blk); len(bM) > 1 {
			if b, err := strconv.Atoi(bM[1]); err == nil {
				entry.Band = b
				if netType == "NR" {
					entry.BandName = fmt.Sprintf("n%d", b)
				} else {
					entry.BandName = fmt.Sprintf("B%d", b)
				}
			}
		}
		if bwM := reBandwidth.FindStringSubmatch(blk); len(bwM) > 1 {
			if bw, err := strconv.Atoi(bwM[1]); err == nil && bw > 0 && bw != 2147483647 {
				entry.Bandwidth = FormatBandWidthKhz(bw)
			}
		}
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

		cells = append(cells, entry)
	}

	return cells
}

// DetectCarrierAggregation resolves Carrier Aggregation status and band aggregation list.
// Ported from NetMonster (cz.mroczis.netmonster.core.feature.detect.DetectorLteAdvancedServiceState).
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
		}
		if pc.BandwidthKhz > 0 {
			bandwidthList = append(bandwidthList, pc.BandwidthKhz)
		}
	}

	// NetMonster Rule 3: Distinct bands from CellInfo serving + neighbors
	var servingBand int
	var neighborBands []int
	for _, c := range cells {
		if c.Registered && c.Band > 0 {
			servingBand = c.Band
		} else if !c.Registered && c.Band > 0 {
			neighborBands = append(neighborBands, c.Band)
		}
	}

	if servingBand > 0 {
		caBands = append(caBands, fmt.Sprintf("B%d", servingBand))
		if isCA {
			for _, nb := range neighborBands {
				if nb != servingBand {
					bStr := fmt.Sprintf("B%d", nb)
					already := false
					for _, existing := range caBands {
						if existing == bStr {
							already = true
							break
						}
					}
					if !already {
						caBands = append(caBands, bStr)
					}
				}
			}
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

	// Fast dumpsys query (<15ms)
	out, err := exec.Command(config.SUBin, "-c", "dumpsys telephony.registry 2>/dev/null | grep -E 'mPhysicalChannelConfigs|mCellInfo|mSignalStrength|mServiceState|mChannelNumber|notifyServiceStateForSubscriber' | head -n 120").Output()
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

	// 2. Parse CellInfo List for serving/neighbor metrics & bands
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
				sig.Band = fmt.Sprintf("B%d", c.Band)
			}
			if c.Bandwidth != "" && sig.Bandwidth == "Unknown" {
				sig.Bandwidth = c.Bandwidth
			}
			if c.Operator != "" {
				sig.Operator = c.Operator
			}
			break
		}
	}

	// 3. Channel Number / EARFCN parsing from mChannelNumber
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
	_, caBandStr, caBwStr := DetectCarrierAggregation(text, pChannels, cells)
	if caBandStr != "" {
		sig.Band = caBandStr
	}
	if caBwStr != "" && caBwStr != "Unknown" {
		sig.Bandwidth = caBwStr
	}

	// 5. Operator Resolution
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

	// 6. Signal Metrics from SignalStrength fallback if not extracted from CellInfo
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

	// 7. Check cached scan result if present
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

	out, err := exec.Command(config.SUBin, "-c", "dumpsys telephony.registry 2>/dev/null | grep -E 'mPhysicalChannelConfigs|mCellInfo|mSignalStrength|mServiceState|mChannelNumber' | head -n 140").Output()
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
			res.ServingCell = &NearbyCellInfo{
				Type:        "lte",
				Role:        "serving",
				PCI:         primaryPCI,
				EARFCN:      primaryChannel,
				Band:        bNum,
				BandName:    bName,
				RSRP:        float64(c.RSRP),
				RSRQ:        float64(c.RSRQ),
				RSSI:        float64(c.RSSI),
				SINR:        float64(c.SINR),
				DistanceStr: "Connected (Primary)",
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

	// Secondary CA Carriers from physical channels
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

	// Neighbor cells
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
			res.Neighbors = append(res.Neighbors, NearbyCellInfo{
				Type:        "lte",
				Role:        "neighbor",
				PCI:         pci,
				EARFCN:      earfcn,
				Band:        bNum,
				BandName:    bName,
				RSRP:        float64(c.RSRP),
				RSRQ:        float64(c.RSRQ),
				RSSI:        float64(c.RSSI),
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

