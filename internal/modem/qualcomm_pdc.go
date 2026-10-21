//go:build (android || linux) && !nomodem

package modem

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

const (
	PDC_SERVICE = 36 // 0x24: Qualcomm Persistent Device Configuration service

	// PDC QMI Message IDs
	QMI_PDC_GET_SELECTED_CONFIG = 0x0022
	QMI_PDC_SET_SELECTED_CONFIG = 0x0023
	QMI_PDC_LIST_CONFIGS        = 0x0024
	QMI_PDC_ACTIVATE_CONFIG     = 0x0025
	QMI_PDC_GET_CONFIG_INFO     = 0x0026
	QMI_PDC_DEACTIVATE_CONFIG   = 0x0027

	// Config Types
	PDC_CONFIG_TYPE_MODEM_SW = 0
	PDC_CONFIG_TYPE_MODEM_HW = 1
)

var (
	pdcMu sync.Mutex
)

// PdcClient handles Persistent Device Configuration (MBN) communication over QRTR.
type PdcClient struct {
	client *QRTRClient
}

// NewPdcClient initializes and connects to the Qualcomm PDC service on QRTR.
func NewPdcClient() (*PdcClient, error) {
	if !IsQRTRSupported() {
		return nil, fmt.Errorf("QRTR not supported on this device")
	}

	client, err := NewQRTRClient()
	if err != nil {
		return nil, fmt.Errorf("create QRTR client: %w", err)
	}

	if err := client.DiscoverAndConnect(PDC_SERVICE); err != nil {
		client.Close()
		return nil, fmt.Errorf("connect to PDC service (36): %w", err)
	}

	return &PdcClient{client: client}, nil
}

// Close closes the underlying QRTR socket.
func (p *PdcClient) Close() {
	if p.client != nil {
		p.client.Close()
	}
}

// GetSelectedConfig queries QMI PDC for the currently active profile identifier.
func (p *PdcClient) GetSelectedConfig(configType uint32, simSlot int) (string, error) {
	var payload bytes.Buffer
	// TLV 0x01: config_type (uint32)
	payload.WriteByte(0x01)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(4))
	_ = binary.Write(&payload, binary.LittleEndian, configType)

	// TLV 0x10: subscription_id (uint32)
	payload.WriteByte(0x10)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(4))
	_ = binary.Write(&payload, binary.LittleEndian, uint32(simSlot))
	resp, err := p.client.Exchange(QMI_PDC_GET_SELECTED_CONFIG, payload.Bytes())
	if err != nil {
		return "", err
	}

	// Parse TLV 0x02: result code
	if len(resp) < 7 {
		return "", fmt.Errorf("PDC response truncated")
	}

	// Scan TLVs for config_id (TLV 0x11 or 0x12)
	offset := 0
	for offset+3 <= len(resp) {
		t := resp[offset]
		length := int(binary.LittleEndian.Uint16(resp[offset+1 : offset+3]))
		offset += 3
		if offset+length > len(resp) {
			break
		}
		val := resp[offset : offset+length]
		offset += length

		if (t == 0x11 || t == 0x12 || t == 0x10) && len(val) > 0 {
			// Extract string from null-terminated or raw bytes
			name := strings.TrimRight(string(val), "\x00")
			if len(name) > 2 {
				return name, nil
			}
		}
	}

	return "", nil
}

// SetSelectedConfig selects an MBN configuration profile in QMI PDC.
func (p *PdcClient) SetSelectedConfig(configID string, configType uint32, simSlot int) error {
	var payload bytes.Buffer
	// TLV 0x01: config_type (uint32)
	payload.WriteByte(0x01)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(4))
	_ = binary.Write(&payload, binary.LittleEndian, configType)

	// TLV 0x02: config_id (raw string bytes + null terminator)
	idBytes := append([]byte(configID), 0)
	payload.WriteByte(0x02)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(len(idBytes)))
	payload.Write(idBytes)

	// TLV 0x10: subscription_id (uint32)
	payload.WriteByte(0x10)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(4))
	_ = binary.Write(&payload, binary.LittleEndian, uint32(simSlot))
	resp, err := p.client.Exchange(QMI_PDC_SET_SELECTED_CONFIG, payload.Bytes())
	if err != nil {
		return err
	}

	if len(resp) >= 7 && resp[0] == 0x02 {
		result := binary.LittleEndian.Uint16(resp[3:5])
		if result != 0 {
			errCode := binary.LittleEndian.Uint16(resp[5:7])
			return fmt.Errorf("PDC set selected config error: %d", errCode)
		}
	}

	return nil
}

// ActivateConfig sends the activation command to load the selected config.
func (p *PdcClient) ActivateConfig(configType uint32, simSlot int) error {
	var payload bytes.Buffer
	// TLV 0x01: config_type (uint32)
	payload.WriteByte(0x01)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(4))
	_ = binary.Write(&payload, binary.LittleEndian, configType)

	// TLV 0x10: subscription_id (uint32)
	payload.WriteByte(0x10)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(4))
	_ = binary.Write(&payload, binary.LittleEndian, uint32(simSlot))
	resp, err := p.client.Exchange(QMI_PDC_ACTIVATE_CONFIG, payload.Bytes())
	if err != nil {
		return err
	}

	if len(resp) >= 7 && resp[0] == 0x02 {
		result := binary.LittleEndian.Uint16(resp[3:5])
		if result != 0 {
			errCode := binary.LittleEndian.Uint16(resp[5:7])
			return fmt.Errorf("PDC activate config error: %d", errCode)
		}
	}

	return nil
}

// ─── Previous Profile Rollback Storage ────────────────────────────────────────

type previousProfileStore struct {
	Profile   string    `json:"profile"`
	Timestamp time.Time `json:"timestamp"`
}

func getPreviousProfilePath() string {
	return config.GetPersistentFilePath("previous_mbn_profile.json")
}

// SavePreviousMbnProfile records the profile being replaced for one-click recovery.
func SavePreviousMbnProfile(profile string) {
	if profile == "" {
		return
	}
	st := previousProfileStore{
		Profile:   profile,
		Timestamp: time.Now().UTC(),
	}
	if b, err := json.Marshal(st); err == nil {
		_ = os.WriteFile(getPreviousProfilePath(), b, 0644)
	}
}

// GetPreviousMbnProfile retrieves the remembered previous profile name if available.
func GetPreviousMbnProfile() string {
	b, err := os.ReadFile(getPreviousProfilePath())
	if err != nil {
		return ""
	}
	var st previousProfileStore
	if err := json.Unmarshal(b, &st); err != nil {
		return ""
	}
	return st.Profile
}

// QueryActiveMbnProfile queries active PDC profile or falls back to known name.
func QueryActiveMbnProfile(simSlot int) (string, error) {
	pdcMu.Lock()
	defer pdcMu.Unlock()

	client, err := NewPdcClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	return client.GetSelectedConfig(PDC_CONFIG_TYPE_MODEM_SW, simSlot)
}

// ApplyMbnProfile selects and activates an MBN profile on the specified SIM slot.
func ApplyMbnProfile(profileID string, simSlot int) error {
	pdcMu.Lock()
	defer pdcMu.Unlock()

	client, err := NewPdcClient()
	if err != nil {
		return fmt.Errorf("connect PDC: %w", err)
	}
	defer client.Close()

	// Query current profile first and save for rollback
	if current, err := client.GetSelectedConfig(PDC_CONFIG_TYPE_MODEM_SW, simSlot); err == nil && current != "" {
		SavePreviousMbnProfile(current)
	}

	// 1. Set Selected Config
	if err := client.SetSelectedConfig(profileID, PDC_CONFIG_TYPE_MODEM_SW, simSlot); err != nil {
		return fmt.Errorf("set selected config '%s': %w", profileID, err)
	}

	// 2. Activate Config
	if err := client.ActivateConfig(PDC_CONFIG_TYPE_MODEM_SW, simSlot); err != nil {
		return fmt.Errorf("activate config '%s': %w", profileID, err)
	}

	logger.Get().Infof("PDC", "Successfully activated MBN profile '%s' on SIM%d", profileID, simSlot+1)
	return nil
}
