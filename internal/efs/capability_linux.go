//go:build android || linux

package efs

import (
	"encoding/binary"
	"fmt"

	"bfr-webui-go/internal/logger"
)

// ReadCapability inspects known EFS RRC capability files and reconstructs the CapabilityState.
func (s *EfsSession) ReadCapability() (*CapabilityState, error) {
	state := &CapabilityState{
		LTE:   LTESectionState{MaxMIMOLayer: 4},
		NSA:   NSASectionState{},
		NR:    NRSectionState{},
		Files: make([]CapabilityFileStatus, 0),
	}

	// 1. LTE CA Disable file
	if data, err := s.ReadFile(EfsCapLteCADisable); err == nil && len(data) >= 1 {
		caDisabled := data[0] != 0
		state.LTE.LTECA = !caDisabled
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapLteCADisable,
			Present:     true,
			Readable:    true,
			WriteOnly:   true, // ModemCaps footer: LTE CA is write-only on some basebands
			Size:        int64(len(data)),
			Description: "LTE Carrier Aggregation Disable toggle",
		})
	} else {
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapLteCADisable,
			Present:     false,
			Readable:    false,
			WriteOnly:   true,
			Description: "LTE Carrier Aggregation Disable toggle",
		})
	}

	// 2. LTE Max MIMO Layers
	if data, err := s.ReadFile(EfsCapLteMaxMimoLayer); err == nil && len(data) >= 1 {
		layers := data[0]
		if layers >= 4 {
			state.LTE.DLMIMO = true
			state.LTE.MaxMIMOLayer = 4
		} else {
			state.LTE.DLMIMO = false
			state.LTE.MaxMIMOLayer = layers
		}
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapLteMaxMimoLayer,
			Present:     true,
			Readable:    true,
			Size:        int64(len(data)),
			Description: "LTE Max MIMO Layers (2 or 4)",
		})
	} else {
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapLteMaxMimoLayer,
			Present:     false,
			Readable:    false,
			Description: "LTE Max MIMO Layers (2 or 4)",
		})
	}

	// 3. LTE ARD (Adaptive Rx Diversity)
	if data, err := s.ReadFile(EfsCapLteML1ARD); err == nil && len(data) >= 1 {
		state.LTE.HORXD = data[0] != 0
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapLteML1ARD,
			Present:     true,
			Readable:    true,
			WriteOnly:   true,
			Size:        int64(len(data)),
			Description: "LTE ML1 Adaptive Rx Diversity (ARD)",
		})
	}

	// 4. LTE ML1 Rx Select
	if data, err := s.ReadFile(EfsCapLteML1RxSelect); err == nil && len(data) >= 1 {
		mask := data[0]
		state.LTE.Rx0 = (mask & 0x01) != 0
		state.LTE.Rx1 = (mask & 0x02) != 0
		state.LTE.Rx2 = (mask & 0x04) != 0
		state.LTE.Rx3 = (mask & 0x08) != 0
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapLteML1RxSelect,
			Present:     true,
			Readable:    true,
			WriteOnly:   true,
			Size:        int64(len(data)),
			Description: "LTE ML1 Rx Chain Selection",
		})
	}

	// 5. NR CA Downgrade 1CC
	if data, err := s.ReadFile(EfsCapNrCapNrcaDowngrade1CC); err == nil && len(data) >= 1 {
		downgrade := data[0] != 0
		state.NR.NRCA = !downgrade
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapNrCapNrcaDowngrade1CC,
			Present:     true,
			Readable:    true,
			Size:        int64(len(data)),
			Description: "NR Carrier Aggregation Downgrade 1CC",
		})
	}

	// 6. NR DL 256-QAM Control
	if data, err := s.ReadFile(EfsCapNrCapControlDL256Q); err == nil && len(data) >= 1 {
		state.NR.DL256Q = data[0] != 0
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapNrCapControlDL256Q,
			Present:     true,
			Readable:    true,
			Size:        int64(len(data)),
			Description: "NR RRC DL 256-QAM Capability Control",
		})
	}

	// 7. NR UL 256-QAM Disable
	if data, err := s.ReadFile(EfsCapNrCapDisableUL256Q); err == nil && len(data) >= 1 {
		ulDisabled := data[0] != 0
		state.NR.UL256Q = !ulDisabled
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapNrCapDisableUL256Q,
			Present:     true,
			Readable:    true,
			Size:        int64(len(data)),
			Description: "NR RRC UL 256-QAM Disable toggle",
		})
	}

	// 8. NR Limit RF MIMO
	if data, err := s.ReadFile(EfsCapNrCapLimitRFMIMO); err == nil && len(data) >= 1 {
		state.NR.LimitRFMIMO = data[0] != 0
		state.Files = append(state.Files, CapabilityFileStatus{
			Path:        EfsCapNrCapLimitRFMIMO,
			Present:     true,
			Readable:    true,
			Size:        int64(len(data)),
			Description: "NR RRC Limit RF MIMO",
		})
	}

	state.TokenString = BuildTokenString(state)
	return state, nil
}

// ApplyCapability validates and writes RRC capability configurations to modem EFS2.
func (s *EfsSession) ApplyCapability(state *CapabilityState, opts WriteOptions) error {
	if state == nil {
		return nil
	}

	// Validate Band-Combo filter grammar if provided
	if state.LTE.CAFilter != "" {
		if ok, errMsg := ValidateCAFilter(state.LTE.CAFilter); !ok {
			return fmt.Errorf("invalid CA filter '%s': %s", state.LTE.CAFilter, errMsg)
		}
	}

	// 1. LTE CA Disable toggle
	caDisableByte := byte(0)
	if !state.LTE.LTECA {
		caDisableByte = 1
	}
	if _, err := s.WriteFileWithOptions(EfsCapLteCADisable, []byte{caDisableByte}, 0644, opts); err != nil {
		return fmt.Errorf("failed to write %s: %w", EfsCapLteCADisable, err)
	}

	// 2. LTE Max MIMO Layers
	layers := state.LTE.MaxMIMOLayer
	if layers == 0 {
		if state.LTE.DLMIMO {
			layers = 4
		} else {
			layers = 2
		}
	}
	mimoBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(mimoBuf, uint16(layers))
	if _, err := s.WriteFileWithOptions(EfsCapLteMaxMimoLayer, mimoBuf, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsCapLteMaxMimoLayer, err)
	}

	// 3. LTE ML1 ARD (Adaptive Rx Diversity)
	ardByte := byte(0)
	if state.LTE.HORXD {
		ardByte = 1
	}
	if _, err := s.WriteFileWithOptions(EfsCapLteML1ARD, []byte{ardByte}, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsCapLteML1ARD, err)
	}

	// 4. LTE ML1 Rx Select
	var rxMask byte = 0
	if state.LTE.Rx0 {
		rxMask |= 0x01
	}
	if state.LTE.Rx1 {
		rxMask |= 0x02
	}
	if state.LTE.Rx2 {
		rxMask |= 0x04
	}
	if state.LTE.Rx3 {
		rxMask |= 0x08
	}
	if rxMask != 0 {
		if _, err := s.WriteFileWithOptions(EfsCapLteML1RxSelect, []byte{rxMask}, 0644, opts); err != nil {
			logger.Get().Warnf("EFS", "Write %s warning: %v", EfsCapLteML1RxSelect, err)
		}
	}

	// 5. NR CA Downgrade 1CC
	nrcaDowngradeByte := byte(0)
	if !state.NR.NRCA {
		nrcaDowngradeByte = 1
	}
	if _, err := s.WriteFileWithOptions(EfsCapNrCapNrcaDowngrade1CC, []byte{nrcaDowngradeByte}, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsCapNrCapNrcaDowngrade1CC, err)
	}

	// 6. NR DL 256-QAM Control
	nrDL256QByte := byte(0)
	if state.NR.DL256Q {
		nrDL256QByte = 1
	}
	if _, err := s.WriteFileWithOptions(EfsCapNrCapControlDL256Q, []byte{nrDL256QByte}, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsCapNrCapControlDL256Q, err)
	}

	// 7. NR UL 256-QAM Disable
	nrUL256QDisableByte := byte(0)
	if !state.NR.UL256Q {
		nrUL256QDisableByte = 1
	}
	if _, err := s.WriteFileWithOptions(EfsCapNrCapDisableUL256Q, []byte{nrUL256QDisableByte}, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsCapNrCapDisableUL256Q, err)
	}

	// 8. NR Limit RF MIMO
	nrLimitRFMIMOByte := byte(0)
	if state.NR.LimitRFMIMO {
		nrLimitRFMIMOByte = 1
	}
	if _, err := s.WriteFileWithOptions(EfsCapNrCapLimitRFMIMO, []byte{nrLimitRFMIMOByte}, 0644, opts); err != nil {
		logger.Get().Warnf("EFS", "Write %s warning: %v", EfsCapNrCapLimitRFMIMO, err)
	}

	logger.Get().Infof("EFS", "Applied RRC capabilities (tokens: %s)", BuildTokenString(state))
	return nil
}
