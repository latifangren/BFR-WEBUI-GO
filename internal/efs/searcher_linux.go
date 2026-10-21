//go:build android || linux

package efs

import (
	"encoding/binary"

	"bfr-webui-go/internal/logger"
)

const (
	EfsPathCellSelectionOpt  = "/nv/item_files/modem/lte/rrc/efs/cell_selection_opt_params"
	EfsPathCellRestrictOpt   = "/nv/item_files/modem/lte/rrc/efs/cell_restrict_opt_params"
	EfsPathCellLockList      = "/nv/item_files/modem/lte/rrc/efs/cell_lock_list"
	EfsPathRegCellLockList   = "/nv/reg_files/modem/lte/rrc/csp/cell_lock_list"

	// Fitur C - Extra Locks
	EfsPathGeranLockEnabled  = "/nv/item_files/modem/geran/grr/feature_lock_arfcn_enabled"
	EfsPathGeranArfcnList    = "/nv/item_files/modem/geran/rr_efs_arfcn_list"
	EfsPathFakeBtsBarring    = "/nv/item_files/modem/geran/grr/fake_bts_cell_barring_enabled"
	EfsPathWcdmaFreqLock     = "/nv/item_files/wcdma/rrc/wcdma_rrc_freq_lock_item"
	EfsPathLteCspPciLock     = "/nv/item_files/modem/lte/rrc/csp/pci_lock"
	EfsPathLteCspEarfcnLock  = "/nv/item_files/modem/lte/rrc/csp/earfcn_lock"
	EfsPathNrCspPciLock      = "/nv/item_files/modem/nr5g/rrc/csp/pci_lock"
	EfsPathNrCspEarfcnLock   = "/nv/item_files/modem/nr5g/rrc/csp/earfcn_lock"
)

// ReadSearcherPreferences reads and unpacks RAT cell/channel preferences from EFS2.
func (s *EfsSession) ReadSearcherPreferences() (*SearcherPreferences, error) {
	prefs := &SearcherPreferences{
		GSM:  RATSearcherConfig{},
		UMTS: RATSearcherConfig{},
		LTE:  RATSearcherConfig{},
		NR:   RATSearcherConfig{},
	}

	// 1. Read LTE Cell Lock / Selection list
	if data, err := s.ReadFile(EfsPathCellLockList); err == nil && len(data) >= 8 {
		entries, _ := UnpackSearcherPage(data)
		for _, raw := range entries {
			pci, cid := UnpackLTECell(raw)
			prefs.LTE.CellBlacklist = append(prefs.LTE.CellBlacklist, SearcherItem{
				RawValue:  raw,
				Code:      pci,
				CellID:    cid,
				Formatted: FormatSearcherItem("lte", "cell", raw),
			})
		}
	}

	// 2. Read Cell Restrict Params
	if data, err := s.ReadFile(EfsPathCellRestrictOpt); err == nil && len(data) >= 8 {
		entries, _ := UnpackSearcherPage(data)
		for _, raw := range entries {
			prefs.LTE.ChannelBlacklist = append(prefs.LTE.ChannelBlacklist, SearcherItem{
				RawValue:  raw,
				Channel:   uint32(raw),
				Formatted: FormatSearcherItem("lte", "channel", raw),
			})
		}
	}

	// 3. Read Cell Selection Params
	if data, err := s.ReadFile(EfsPathCellSelectionOpt); err == nil && len(data) >= 8 {
		entries, _ := UnpackSearcherPage(data)
		for _, raw := range entries {
			prefs.LTE.PriorityChannels = append(prefs.LTE.PriorityChannels, SearcherItem{
				RawValue:  raw,
				Channel:   uint32(raw),
				Formatted: FormatSearcherItem("lte", "channel", raw),
			})
		}
	}

	return prefs, nil
}

// WriteSearcherPreferences writes packed RAT cell/channel preferences to EFS2.
func (s *EfsSession) WriteSearcherPreferences(prefs *SearcherPreferences, opts WriteOptions) error {
	if prefs == nil {
		return nil
	}

	// 1. Pack LTE Cell Blacklist
	var lteCellRaw []uint64
	for _, item := range prefs.LTE.CellBlacklist {
		if item.RawValue != 0 && item.RawValue != SearcherTerminator {
			lteCellRaw = append(lteCellRaw, item.RawValue)
		} else if item.Code != 0 || item.CellID != 0 {
			lteCellRaw = append(lteCellRaw, PackLTECell(item.Code, item.CellID))
		}
	}
	cellPage := PackSearcherPage(lteCellRaw)
	if _, err := s.WriteFileWithOptions(EfsPathCellLockList, cellPage, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsPathCellLockList, err)
	}
	// Mirror to reg_files as used by ModemCaps
	_, _ = s.WriteFileWithOptions(EfsPathRegCellLockList, cellPage, 0644, opts)

	// 2. Pack LTE Channel Blacklist
	var lteChanRaw []uint64
	for _, item := range prefs.LTE.ChannelBlacklist {
		if item.RawValue != 0 && item.RawValue != SearcherTerminator {
			lteChanRaw = append(lteChanRaw, item.RawValue)
		} else if item.Channel != 0 {
			lteChanRaw = append(lteChanRaw, uint64(item.Channel))
		}
	}
	chanPage := PackSearcherPage(lteChanRaw)
	if _, err := s.WriteFileWithOptions(EfsPathCellRestrictOpt, chanPage, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsPathCellRestrictOpt, err)
	}

	// 3. Pack Priority Channels
	var prioRaw []uint64
	for _, item := range prefs.LTE.PriorityChannels {
		if item.RawValue != 0 && item.RawValue != SearcherTerminator {
			prioRaw = append(prioRaw, item.RawValue)
		} else if item.Channel != 0 {
			prioRaw = append(prioRaw, uint64(item.Channel))
		}
	}
	prioPage := PackSearcherPage(prioRaw)
	if _, err := s.WriteFileWithOptions(EfsPathCellSelectionOpt, prioPage, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsPathCellSelectionOpt, err)
	}

	return nil
}

// ReadExtraLocks reads GERAN, WCDMA, and CSP lock files from EFS2.
func (s *EfsSession) ReadExtraLocks() (*ExtraLocksConfig, error) {
	locks := &ExtraLocksConfig{}

	// GERAN lock ARFCN enabled (1 byte)
	if data, err := s.ReadFile(EfsPathGeranLockEnabled); err == nil && len(data) >= 1 {
		locks.GeranLockArfcnEnabled = data[0] != 0
	}

	// GERAN ARFCN list (uint16 array)
	if data, err := s.ReadFile(EfsPathGeranArfcnList); err == nil && len(data) >= 2 {
		count := len(data) / 2
		for i := range count {
			locks.GeranArfcnList = append(locks.GeranArfcnList, binary.LittleEndian.Uint16(data[i*2:(i+1)*2]))
		}
	}

	// Fake BTS cell barring (1 byte)
	if data, err := s.ReadFile(EfsPathFakeBtsBarring); err == nil && len(data) >= 1 {
		locks.FakeBtsCellBarring = data[0] != 0
	}

	// WCDMA frequency lock (uint16 UARFCN)
	if data, err := s.ReadFile(EfsPathWcdmaFreqLock); err == nil && len(data) >= 2 {
		locks.WcdmaFreqLockItem = binary.LittleEndian.Uint16(data[:2])
	}

	// LTE CSP PCI lock (uint16)
	if data, err := s.ReadFile(EfsPathLteCspPciLock); err == nil && len(data) >= 2 {
		locks.LteCspPciLock = binary.LittleEndian.Uint16(data[:2])
	}

	// LTE CSP EARFCN lock (uint32)
	if data, err := s.ReadFile(EfsPathLteCspEarfcnLock); err == nil && len(data) >= 4 {
		locks.LteCspEarfcnLock = binary.LittleEndian.Uint32(data[:4])
	}

	// NR5G CSP PCI lock (uint16)
	if data, err := s.ReadFile(EfsPathNrCspPciLock); err == nil && len(data) >= 2 {
		locks.NrCspPciLock = binary.LittleEndian.Uint16(data[:2])
	}

	// NR5G CSP EARFCN lock (uint32)
	if data, err := s.ReadFile(EfsPathNrCspEarfcnLock); err == nil && len(data) >= 4 {
		locks.NrCspEarfcnLock = binary.LittleEndian.Uint32(data[:4])
	}

	return locks, nil
}

// WriteExtraLocks writes GERAN, WCDMA, and CSP lock files to EFS2.
func (s *EfsSession) WriteExtraLocks(locks *ExtraLocksConfig, opts WriteOptions) error {
	if locks == nil {
		return nil
	}

	// GERAN Lock Enabled
	geranByte := byte(0)
	if locks.GeranLockArfcnEnabled {
		geranByte = 1
	}
	_, _ = s.WriteFileWithOptions(EfsPathGeranLockEnabled, []byte{geranByte}, 0644, opts)

	// GERAN ARFCN List
	if len(locks.GeranArfcnList) > 0 {
		buf := make([]byte, len(locks.GeranArfcnList)*2)
		for i, arfcn := range locks.GeranArfcnList {
			binary.LittleEndian.PutUint16(buf[i*2:(i+1)*2], arfcn)
		}
		_, _ = s.WriteFileWithOptions(EfsPathGeranArfcnList, buf, 0644, opts)
	}

	// Fake BTS Barring
	fakeBtsByte := byte(0)
	if locks.FakeBtsCellBarring {
		fakeBtsByte = 1
	}
	_, _ = s.WriteFileWithOptions(EfsPathFakeBtsBarring, []byte{fakeBtsByte}, 0644, opts)

	// WCDMA Frequency Lock
	wcdmaBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(wcdmaBuf, locks.WcdmaFreqLockItem)
	_, _ = s.WriteFileWithOptions(EfsPathWcdmaFreqLock, wcdmaBuf, 0644, opts)

	// LTE CSP PCI Lock
	pciBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(pciBuf, locks.LteCspPciLock)
	_, _ = s.WriteFileWithOptions(EfsPathLteCspPciLock, pciBuf, 0644, opts)

	// LTE CSP EARFCN Lock
	earfcnBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(earfcnBuf, locks.LteCspEarfcnLock)
	_, _ = s.WriteFileWithOptions(EfsPathLteCspEarfcnLock, earfcnBuf, 0644, opts)

	// NR5G CSP PCI Lock
	nrPciBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(nrPciBuf, locks.NrCspPciLock)
	_, _ = s.WriteFileWithOptions(EfsPathNrCspPciLock, nrPciBuf, 0644, opts)

	// NR5G CSP EARFCN Lock
	nrEarfcnBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(nrEarfcnBuf, locks.NrCspEarfcnLock)
	_, _ = s.WriteFileWithOptions(EfsPathNrCspEarfcnLock, nrEarfcnBuf, 0644, opts)

	return nil
}
