package modem

import (
	"testing"
)

func TestCalculateTowerDistance(t *testing.T) {
	tests := []struct {
		ta       int
		wantM    int
		wantSub  string
		wantZero bool
	}{
		{ta: 0, wantZero: true},
		{ta: -1, wantZero: true},
		{ta: 1283, wantZero: true},
		{ta: 1, wantM: 78, wantSub: "~78 m (TA: 1)"},
		{ta: 4, wantM: 312, wantSub: "~312 m (TA: 4)"},
		{ta: 20, wantM: 1562, wantSub: "~1.6 km (TA: 20)"},
	}

	for _, tt := range tests {
		m, s := CalculateTowerDistance(tt.ta)
		if tt.wantZero {
			if m != 0 || s != "" {
				t.Errorf("CalculateTowerDistance(%d) = (%d, %q); want 0, \"\"", tt.ta, m, s)
			}
		} else {
			if m != tt.wantM || s != tt.wantSub {
				t.Errorf("CalculateTowerDistance(%d) = (%d, %q); want (%d, %q)", tt.ta, m, s, tt.wantM, tt.wantSub)
			}
		}
	}
}

func TestMccIsoAndOperator(t *testing.T) {
	if iso := GetCountryIsoByMcc("510"); iso != "ID" {
		t.Errorf("GetCountryIsoByMcc(510) = %q; want ID", iso)
	}
	if iso := GetCountryIsoByMcc("525"); iso != "SG" {
		t.Errorf("GetCountryIsoByMcc(525) = %q; want SG", iso)
	}
	if iso := GetCountryIsoByMcc("310"); iso != "US" {
		t.Errorf("GetCountryIsoByMcc(310) = %q; want US", iso)
	}

	if op := GetOperatorByPlmn("510", "11"); op != "XL Axiata" {
		t.Errorf("GetOperatorByPlmn(510, 11) = %q; want XL Axiata", op)
	}
	if op := GetOperatorByPlmn("510", "10"); op != "Telkomsel" {
		t.Errorf("GetOperatorByPlmn(510, 10) = %q; want Telkomsel", op)
	}
	if op := GetOperatorByPlmn("510", "1"); op != "Indosat Ooredoo" {
		t.Errorf("GetOperatorByPlmn(510, 1) = %q; want Indosat Ooredoo", op)
	}
}

func TestBandResolutionAndNrRaster(t *testing.T) {
	// LTE Band 3 (1800 MHz)
	bNum, bName := GetLteBand(1850)
	if bNum != 3 || bName != "B3 (1800 MHz)" {
		t.Errorf("GetLteBand(1850) = %d, %s; want 3, B3 (1800 MHz)", bNum, bName)
	}

	// 5G NR n78 (3500 MHz) vs n77 (3700 MHz)
	// NR-ARFCN 640000 = 3600 MHz
	freq := GetNrFrequencyKhz(640000)
	if freq != 3600000 {
		t.Errorf("GetNrFrequencyKhz(640000) = %d; want 3600000", freq)
	}

	// Disambiguation with band hints
	nNum, nName := GetNrBandWithHints(640000, 78)
	if nNum != 78 || nName != "n78 (3500 MHz)" {
		t.Errorf("GetNrBandWithHints(640000, 78) = %d, %s; want 78, n78 (3500 MHz)", nNum, nName)
	}
}

func TestCarrierAggregationNoNeighborPollution(t *testing.T) {
	dumpsysText := `
mIsUsingCarrierAggregation=true
mCellBandwidths=[20000, 10000]
`
	pChannels := []PhysicalChannelInfo{
		{ConnectionStatus: "PrimaryServing", ChannelNumber: 1850, BandwidthKhz: 20000},
		{ConnectionStatus: "SecondaryServing", ChannelNumber: 2450, BandwidthKhz: 10000}, // Band 5
	}

	cells := []NetMonsterCellEntry{
		{Registered: true, Band: 3, BandName: "B3"},
		{Registered: false, Band: 1, BandName: "B1"}, // Neighbor 1 (Must NOT be in CA)
		{Registered: false, Band: 8, BandName: "B8"}, // Neighbor 2 (Must NOT be in CA)
	}

	isCA, bandLabel, bwLabel := DetectCarrierAggregation(dumpsysText, pChannels, cells)
	if !isCA {
		t.Errorf("DetectCarrierAggregation() isCA = false, want true")
	}

	// Expected: B3 + B5 [2CA Active], NOT polluted by B1 or B8
	if bandLabel != "B3 + B5 [2CA Active]" {
		t.Errorf("DetectCarrierAggregation() bandLabel = %q; want \"B3 + B5 [2CA Active]\"", bandLabel)
	}
	if bwLabel != "20 MHz + 10 MHz (30 MHz Total)" {
		t.Errorf("DetectCarrierAggregation() bwLabel = %q; want \"20 MHz + 10 MHz (30 MHz Total)\"", bwLabel)
	}
}

func TestDetectNrNsaState(t *testing.T) {
	// Case 1: Masked Android 16 dumpsys with nrAvailable and secondary NR
	dumpsysMasked := `
isNrAvailable=true
isEnDcAvailable=true
isDcNrRestricted=false
nrState=****
`
	pChannels := []PhysicalChannelInfo{
		{ConnectionStatus: "PrimaryServing", ChannelNumber: 1850, NetworkType: "LTE"},
	}
	cells := []NetMonsterCellEntry{
		{Registered: true, NetworkType: "LTE", Band: 3},
		{Registered: false, NetworkType: "NR", Band: 78},
	}

	isNSA, isSA, desc := DetectNrNsaState(dumpsysMasked, pChannels, cells)
	if !isNSA || isSA || desc != "5G NSA" {
		t.Errorf("DetectNrNsaState(masked) = (%v, %v, %q); want (true, false, \"5G NSA\")", isNSA, isSA, desc)
	}
}

func TestParseCellInfoListDetailed(t *testing.T) {
	sample := `
CellInfoLte:{mRegistered=YES,mTimeStampType=oem_ril,mTimeStamp=123456,CellIdentityLte:{mMcc=510,mMnc=11,mAlphaLong="XL",mCi=1234567,mPci=150,mTac=2000,mEarfcn=1850,mBands=[3],mBandwidth=20000},CellSignalStrengthLte:{rssi=-67,rsrp=-93,rsrq=-11,rssnr=16,cqi=14,timingAdvance=4}}
CellInfoLte:{mRegistered=NO,mTimeStampType=oem_ril,mTimeStamp=123456,CellIdentityLte:{mMcc=510,mMnc=11,mAlphaLong="XL",mCi=2147483647,mPci=240,mTac=2147483647,mEarfcn=1850,mBands=[3],mBandwidth=2147483647},CellSignalStrengthLte:{rssi=-79,rsrp=-105,rsrq=-14,rssnr=-999,cqi=2147483647,timingAdvance=2147483647}}
`
	cells := ParseCellInfoList(sample)
	if len(cells) != 2 {
		t.Fatalf("ParseCellInfoList() returned %d cells, want 2", len(cells))
	}

	serving := cells[0]
	if !serving.Registered {
		t.Errorf("cells[0].Registered = false, want true")
	}
	if serving.PhysicalCellID != 150 {
		t.Errorf("serving.PhysicalCellID = %d, want 150", serving.PhysicalCellID)
	}
	if serving.EARFCN != 1850 {
		t.Errorf("serving.EARFCN = %d, want 1850", serving.EARFCN)
	}
	if serving.CellID != 1234567 {
		t.Errorf("serving.CellID = %d, want 1234567", serving.CellID)
	}
	if serving.TAC != 2000 {
		t.Errorf("serving.TAC = %d, want 2000", serving.TAC)
	}
	if serving.RSRP != -93 || serving.RSRQ != -11 || serving.RSSI != -67 || serving.SINR != 16 {
		t.Errorf("serving signals = (%d, %d, %d, %d), want (-93, -11, -67, 16)", serving.RSRP, serving.RSRQ, serving.RSSI, serving.SINR)
	}
	if serving.TimingAdvance != 4 {
		t.Errorf("serving.TimingAdvance = %d, want 4", serving.TimingAdvance)
	}
	if serving.DistanceMeter != 312 || serving.DistanceStr != "~312 m (TA: 4)" {
		t.Errorf("serving.Distance = (%d, %q), want (312, \"~312 m (TA: 4)\")", serving.DistanceMeter, serving.DistanceStr)
	}

	neighbor := cells[1]
	if neighbor.Registered {
		t.Errorf("neighbor.Registered = true, want false")
	}
	if neighbor.PhysicalCellID != 240 {
		t.Errorf("neighbor.PhysicalCellID = %d, want 240", neighbor.PhysicalCellID)
	}
	// Verify invalid dumpsys placeholders (2147483647) were ignored
	if neighbor.CellID != -1 || neighbor.TAC != -1 {
		t.Errorf("neighbor placeholders not ignored: CellID=%d, TAC=%d", neighbor.CellID, neighbor.TAC)
	}
}
