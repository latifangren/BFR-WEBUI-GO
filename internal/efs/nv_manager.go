//go:build android || linux

package efs

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"bfr-webui-go/internal/logger"
)

// NV DIAG command codes — standard (0x26/0x27), extended (0x75/0x76), subsystem (0x4B).
const (
	diagNVReadStd  = 0x26 // DIAG_NV_READ_F
	diagNVWriteStd = 0x27 // DIAG_NV_WRITE_F
	diagNVReadExt  = 0x75 // DIAG_NV_READ_EXT_F
	diagNVWriteExt = 0x76 // DIAG_NV_WRITE_EXT_F
	diagSubsysCmd  = 0x4B // DIAG_SUBSYS_CMD_F

	// Subsystem IDs
	subsysNVID = 0x0B // NV Subsystem

	// Subsystem NV command codes
	subsysNVRead  = 0x0026
	subsysNVWrite = 0x0027

	// Error response bytes
	diagBadCmd  = 0x13
	diagBadParm = 0x42

	// NV packet payload size (128 bytes of NV data after header)
	nvPayloadSize = 128

	// Timeout per NV operation
	nvOpTimeout = 2 * time.Second
)

// NVManager handles NV item read/write/backup/restore over the DIAG transport.
// It uses the same DiagClient as EfsSession — no extra connections needed.
type NVManager struct {
	diag *DiagClient
}

// NewNVManager creates an NVManager wrapping the provided DiagClient.
func NewNVManager(diag *DiagClient) *NVManager {
	return &NVManager{diag: diag}
}

// GetManagerNV returns an NVManager using the singleton EFS DiagClient.
// Opens a new DIAG connection if one is not already active.
func GetManagerNV() (*NVManager, error) {
	m := GetManager()
	sess, err := m.GetSession()
	if err != nil {
		return nil, fmt.Errorf("DIAG session unavailable: %w", err)
	}
	return NewNVManager(sess.diag), nil
}

// ─── Core NV Read ────────────────────────────────────────────────────────────

// ReadNV reads a single NV item using 3-tier auto-fallback:
//   1. Standard DIAG 0x26
//   2. Subsystem DIAG 0x4B (subsys 0x0B)
//   3. Extended DIAG 0x75
func (n *NVManager) ReadNV(id uint16, index uint16) (*NVReadResult, error) {
	info := GetNVItem(id)
	name := fmt.Sprintf("NV_%d", id)
	cat := "Unknown"
	dt := NVDataHex
	if info != nil {
		name = info.Name
		cat = info.Category
		dt = info.DataType
	}

	res := &NVReadResult{
		ID:       id,
		Name:     name,
		Category: cat,
		Index:    index,
		DataType: dt,
	}

	// Tier 1: Standard read (0x26)
	data, err := n.sendNVReadStd(id)
	if err != nil {
		// Tier 2: Subsystem read (0x4B)
		logger.Get().Infof("NV", "NV %d: standard read failed (%v), trying subsystem 0x4B", id, err)
		data, err = n.sendNVReadSubsys(id, index)
		if err != nil {
			// Tier 3: Extended read (0x75)
			logger.Get().Infof("NV", "NV %d: subsystem read failed (%v), trying extended 0x75", id, err)
			data, err = n.sendNVReadExt(id, index)
			if err != nil {
				res.Error = err.Error()
				return res, err
			}
		}
	}

	// Trim to declared size if known
	if info != nil && info.Size > 0 && len(data) > int(info.Size) {
		data = data[:info.Size]
	}

	res.DataHex = hex.EncodeToString(data)
	res.Size = len(data)
	res.Formatted = FormatNVValue(id, data)
	res.Success = true
	return res, nil
}

// ─── Core NV Write ───────────────────────────────────────────────────────────

// WriteNV writes a single NV item using standard DIAG 0x27.
// dataHex must be a hex-encoded string of the raw bytes to write.
func (n *NVManager) WriteNV(id uint16, index uint16, dataHex string) (*NVWriteResult, error) {
	info := GetNVItem(id)
	name := fmt.Sprintf("NV_%d", id)
	if info != nil {
		name = info.Name
		if info.ReadOnly {
			return &NVWriteResult{ID: id, Name: name, Error: "this NV item is read-only"},
				fmt.Errorf("NV %d (%s) is read-only", id, name)
		}
	}

	data, err := hex.DecodeString(strings.ReplaceAll(dataHex, " ", ""))
	if err != nil {
		return nil, fmt.Errorf("invalid hex data: %w", err)
	}

	res := &NVWriteResult{ID: id, Name: name}

	// Full security bypass: tries SPC 000000 + all known Qualcomm engineering passwords.
	// This is needed for protected NV items (IMEI 550, 2497, 5014) and matches what
	// field tools like QLM IMEI Repair do under the hood.
	logger.Get().Infof("NV", "Running full security bypass for NV %d before write...", id)
	if ok := n.diag.BypassSecurity(4 * time.Second); ok {
		logger.Get().Infof("NV", "Security bypass accepted by modem for NV %d", id)
	} else {
		logger.Get().Warnf("NV", "Security bypass did not get explicit acceptance for NV %d (will still attempt write)", id)
	}

	// Tier 1: Standard write (0x27)
	if err := n.sendNVWriteStd(id, data); err != nil {
		logger.Get().Infof("NV", "NV %d: standard write failed (%v), trying subsystem 0x4B", id, err)
		// Tier 2: Subsystem write (0x4B, cmd 0x0027)
		if err2 := n.sendNVWriteSubsys(id, index, data); err2 != nil {
			logger.Get().Infof("NV", "NV %d: subsystem write failed (%v), trying extended 0x76", id, err2)
			// Tier 3: Extended write (0x76)
			if err3 := n.sendNVWriteExt(id, index, data); err3 != nil {
				res.Error = fmt.Sprintf("std write: %v; subsys: %v; ext write: %v", err, err2, err3)
				return res, fmt.Errorf("%s", res.Error)
			}
		}
	}

	res.Success = true
	logger.Get().Infof("NV", "NV %d (%s) written successfully (%d bytes)", id, name, len(data))
	return res, nil
}

// ─── Backup ──────────────────────────────────────────────────────────────────

// BackupItems reads a list of NV item IDs and writes them to a JSON file.
// Progress is reported via the returned channel; caller must drain it.
func (n *NVManager) BackupItems(ids []uint16, filename string) (<-chan NVBackupProgress, error) {
	ch := make(chan NVBackupProgress, 32)

	go func() {
		defer close(ch)

		var entries []NVBackupEntry
		success, failed, skipped := 0, 0, 0
		total := len(ids)

		for i, id := range ids {
			info := GetNVItem(id)
			name := fmt.Sprintf("NV_%d", id)
			if info != nil {
				name = info.Name
			}

			ch <- NVBackupProgress{
				Current: i + 1, Total: total, ItemName: name,
				Success: success, Failed: failed, Skipped: skipped,
			}

			res, err := n.ReadNV(id, 0)
			if err != nil || !res.Success || res.DataHex == "" {
				failed++
				continue
			}

			// Skip all-zero data (item not populated)
			if isAllZeroHex(res.DataHex) {
				skipped++
				continue
			}

			entries = append(entries, NVBackupEntry{
				ID:      id,
				Name:    name,
				DataHex: res.DataHex,
			})
			success++
		}

		if err := writeBackupFile(filename, entries); err != nil {
			ch <- NVBackupProgress{
				Done: true, Total: total,
				Success: success, Failed: failed, Skipped: skipped,
				Error: err.Error(),
			}
			return
		}

		ch <- NVBackupProgress{
			Done: true, Total: total,
			Success: success, Failed: failed, Skipped: skipped,
		}
		logger.Get().Infof("NV", "Backup complete: %d ok, %d failed, %d skipped → %s",
			success, failed, skipped, filename)
	}()

	return ch, nil
}

// BackupRange reads all NV items in the inclusive ID range [startID, endID].
// Uses a fast 100ms watchdog per item to skip unpopulated IDs.
// Progress is reported via the returned channel.
func (n *NVManager) BackupRange(startID, endID uint16, filename string) (<-chan NVBackupProgress, error) {
	ch := make(chan NVBackupProgress, 64)

	go func() {
		defer close(ch)

		total := int(endID) - int(startID) + 1
		var entries []NVBackupEntry
		success, failed, skipped := 0, 0, 0

		for id := startID; id <= endID; id++ {
			info := GetNVItem(id)
			name := fmt.Sprintf("NV_%d", id)
			if info != nil {
				name = info.Name
			}

			ch <- NVBackupProgress{
				Current: int(id-startID) + 1, Total: total, ItemName: name,
				Success: success, Failed: failed, Skipped: skipped,
			}

			// Fast per-item read with short deadline
			done := make(chan *NVReadResult, 1)
			go func(itemID uint16) {
				r, _ := n.ReadNV(itemID, 0)
				done <- r
			}(id)

			var res *NVReadResult
			select {
			case res = <-done:
			case <-time.After(250 * time.Millisecond):
				skipped++
				continue
			}

			if res == nil || !res.Success || res.DataHex == "" || isAllZeroHex(res.DataHex) {
				failed++
				continue
			}

			entries = append(entries, NVBackupEntry{
				ID:      id,
				Name:    name,
				DataHex: res.DataHex,
			})
			success++
		}

		if err := writeBackupFile(filename, entries); err != nil {
			ch <- NVBackupProgress{
				Done: true, Total: total,
				Success: success, Failed: failed, Skipped: skipped,
				Error: err.Error(),
			}
			return
		}

		ch <- NVBackupProgress{
			Done: true, Total: total,
			Success: success, Failed: failed, Skipped: skipped,
		}
		logger.Get().Infof("NV", "Range backup %d-%d done: %d ok, %d failed, %d skipped → %s",
			startID, endID, success, failed, skipped, filename)
	}()

	return ch, nil
}

// ─── Restore ─────────────────────────────────────────────────────────────────

// RestoreFromFile reads a JSON backup file and writes all items back to the modem.
// Progress is reported via the returned channel.
func (n *NVManager) RestoreFromFile(filename string) (<-chan NVBackupProgress, error) {
	f, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read backup file: %w", err)
	}

	var backup NVBackupFile
	if err := json.Unmarshal(f, &backup); err != nil {
		return nil, fmt.Errorf("invalid backup file format: %w", err)
	}

	ch := make(chan NVBackupProgress, 32)

	go func() {
		defer close(ch)

		total := len(backup.Items)
		success, failed := 0, 0

		for i, entry := range backup.Items {
			ch <- NVBackupProgress{
				Current: i + 1, Total: total, ItemName: entry.Name,
				Success: success, Failed: failed,
			}

			res, err := n.WriteNV(entry.ID, entry.Index, entry.DataHex)
			if err != nil || !res.Success {
				failed++
				logger.Get().Warnf("NV", "Restore NV %d (%s) failed: %v", entry.ID, entry.Name, err)
				continue
			}
			success++
		}

		ch <- NVBackupProgress{
			Done: true, Total: total,
			Success: success, Failed: failed,
		}
		logger.Get().Infof("NV", "Restore complete: %d ok, %d failed from %s", success, failed, filename)
	}()

	return ch, nil
}

// ─── Low-level DIAG packet builders ─────────────────────────────────────────

// sendNVReadStd sends DIAG_NV_READ_F (0x26) and extracts the NV data payload.
func (n *NVManager) sendNVReadStd(id uint16) ([]byte, error) {
	cmd := make([]byte, 3+nvPayloadSize)
	cmd[0] = diagNVReadStd
	binary.LittleEndian.PutUint16(cmd[1:3], id)
	// bytes [3..130] = 128 zero-padded NV data area

	resp, err := n.diag.Xfer(cmd, nvOpTimeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < 3 {
		return nil, fmt.Errorf("NV std read: response too short (%d bytes)", len(resp))
	}
	if resp[0] == diagBadCmd || resp[0] == diagBadParm {
		return nil, fmt.Errorf("NV std read: modem rejected item %d (0x%02x)", id, resp[0])
	}
	if resp[0] != diagNVReadStd {
		return nil, fmt.Errorf("NV std read: unexpected response byte 0x%02x", resp[0])
	}
	// [0]=cmd [1..2]=id [3..130]=data [131..132]=status
	if len(resp) < 4 {
		return nil, fmt.Errorf("NV std read: response missing data")
	}
	data := resp[3:]
	// Trim trailing status bytes (last 2 = uint16 NV status)
	if len(data) >= 2 {
		status := binary.LittleEndian.Uint16(data[len(data)-2:])
		if status != 0 {
			return nil, fmt.Errorf("NV std read: item %d status=%d (%s)", id, status, NVStatusStr(status))
		}
		data = data[:len(data)-2]
	}
	return data, nil
}

// sendNVReadSubsys sends DIAG_SUBSYS_CMD_F (0x4B) with subsystem 0x0B for NV read.
func (n *NVManager) sendNVReadSubsys(id uint16, index uint16) ([]byte, error) {
	// Header: [0x4B, subsysID, subcmd_lo, subcmd_hi, id_lo, id_hi, idx_lo, idx_hi] + 128 bytes padding
	cmd := make([]byte, 8+nvPayloadSize)
	cmd[0] = diagSubsysCmd
	cmd[1] = subsysNVID
	binary.LittleEndian.PutUint16(cmd[2:4], subsysNVRead)
	binary.LittleEndian.PutUint16(cmd[4:6], id)
	binary.LittleEndian.PutUint16(cmd[6:8], index)

	resp, err := n.diag.Xfer(cmd, nvOpTimeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < 8 {
		return nil, fmt.Errorf("NV subsys read: response too short")
	}
	if resp[0] == diagBadCmd || resp[0] == diagBadParm {
		return nil, fmt.Errorf("NV subsys read: modem rejected item %d", id)
	}
	data := resp[8:]
	if len(data) >= 2 {
		data = data[:len(data)-2]
	}
	return data, nil
}

// sendNVReadExt sends DIAG_NV_READ_EXT_F (0x75) for extended NV read.
func (n *NVManager) sendNVReadExt(id uint16, index uint16) ([]byte, error) {
	cmd := make([]byte, 5+nvPayloadSize)
	cmd[0] = diagNVReadExt
	binary.LittleEndian.PutUint16(cmd[1:3], id)
	binary.LittleEndian.PutUint16(cmd[3:5], index)

	resp, err := n.diag.Xfer(cmd, nvOpTimeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < 5 {
		return nil, fmt.Errorf("NV ext read: response too short")
	}
	if resp[0] == diagBadCmd || resp[0] == diagBadParm {
		return nil, fmt.Errorf("NV ext read: modem rejected item %d", id)
	}
	if resp[0] != diagNVReadExt {
		return nil, fmt.Errorf("NV ext read: unexpected response 0x%02x", resp[0])
	}
	data := resp[5:]
	if len(data) >= 2 {
		status := binary.LittleEndian.Uint16(data[len(data)-2:])
		if status != 0 {
			return nil, fmt.Errorf("NV ext read: item %d status=%d (%s)", id, status, NVStatusStr(status))
		}
		data = data[:len(data)-2]
	}
	return data, nil
}

// sendNVWriteStd sends DIAG_NV_WRITE_F (0x27).
// Packet format: [CMD:1B][ItemID:2B][Data:128B][Status:2B] = 133 bytes
// Response:      [CMD:1B][ItemID:2B][Data:128B][Status:2B] = 133 bytes
// Status uint16 is at offset 3 + 128 = 131! (Index 3..4 is part of data payload)
func (n *NVManager) sendNVWriteStd(id uint16, data []byte) error {
	cmd := make([]byte, 1+2+nvPayloadSize+2)
	cmd[0] = diagNVWriteStd
	binary.LittleEndian.PutUint16(cmd[1:3], id)
	copy(cmd[3:], data)

	resp, err := n.diag.Xfer(cmd, nvOpTimeout)
	if err != nil {
		return err
	}
	if len(resp) < 3 {
		return fmt.Errorf("NV std write: response too short (%d bytes)", len(resp))
	}
	if resp[0] == diagBadCmd || resp[0] == diagBadParm {
		return fmt.Errorf("NV std write: modem rejected item %d (0x%02x)", id, resp[0])
	}
	if resp[0] != diagNVWriteStd {
		return fmt.Errorf("NV std write: unexpected response 0x%02x", resp[0])
	}

	// Status is at the end of the packet: [131:133] for 133-byte packet, or last 2 bytes
	if len(resp) >= 5 {
		statusOffset := len(resp) - 2
		if len(resp) >= 3+nvPayloadSize+2 {
			statusOffset = 3 + nvPayloadSize
		}
		status := binary.LittleEndian.Uint16(resp[statusOffset : statusOffset+2])
		if status != 0 {
			return fmt.Errorf("NV std write: item %d rejected, status=%d (%s)", id, status, NVStatusStr(status))
		}
	}
	return nil
}

// sendNVWriteSubsys sends DIAG_SUBSYS_CMD_F (0x4B) with subsys 0x0B and subcmd 0x0027.
// Packet format: [0x4B][Subsys:0x0B][Subcmd:0x0027][ItemID:2B][Index:2B][Data:128B][Status:2B]
func (n *NVManager) sendNVWriteSubsys(id uint16, index uint16, data []byte) error {
	cmd := make([]byte, 4+2+2+nvPayloadSize+2)
	cmd[0] = diagSubsysCmd
	cmd[1] = subsysNVID
	binary.LittleEndian.PutUint16(cmd[2:4], subsysNVWrite)
	binary.LittleEndian.PutUint16(cmd[4:6], id)
	binary.LittleEndian.PutUint16(cmd[6:8], index)
	copy(cmd[8:], data)

	resp, err := n.diag.Xfer(cmd, nvOpTimeout)
	if err != nil {
		return err
	}
	if len(resp) < 8 {
		return fmt.Errorf("NV subsys write: response too short (%d bytes)", len(resp))
	}
	if resp[0] == diagBadCmd || resp[0] == diagBadParm {
		return fmt.Errorf("NV subsys write: modem rejected item %d (0x%02x)", id, resp[0])
	}
	if resp[0] != diagSubsysCmd {
		return fmt.Errorf("NV subsys write: unexpected response 0x%02x", resp[0])
	}

	if len(resp) >= 10 {
		statusOffset := len(resp) - 2
		if len(resp) >= 8+nvPayloadSize+2 {
			statusOffset = 8 + nvPayloadSize
		}
		status := binary.LittleEndian.Uint16(resp[statusOffset : statusOffset+2])
		if status != 0 {
			return fmt.Errorf("NV subsys write: item %d rejected, status=%d (%s)", id, status, NVStatusStr(status))
		}
	}
	return nil
}

// sendNVWriteExt sends DIAG_NV_WRITE_EXT_F (0x76).
func (n *NVManager) sendNVWriteExt(id uint16, index uint16, data []byte) error {
	cmd := make([]byte, 1+4+nvPayloadSize+2)
	cmd[0] = diagNVWriteExt
	binary.LittleEndian.PutUint16(cmd[1:3], id)
	binary.LittleEndian.PutUint16(cmd[3:5], index)
	copy(cmd[5:], data)

	resp, err := n.diag.Xfer(cmd, nvOpTimeout)
	if err != nil {
		return err
	}
	if len(resp) < 5 {
		return fmt.Errorf("NV ext write: response too short (%d bytes)", len(resp))
	}
	if resp[0] == diagBadCmd || resp[0] == diagBadParm {
		return fmt.Errorf("NV ext write: modem rejected item %d (0x%02x)", id, resp[0])
	}
	if resp[0] != diagNVWriteExt {
		return fmt.Errorf("NV ext write: unexpected response 0x%02x", resp[0])
	}

	if len(resp) >= 7 {
		statusOffset := len(resp) - 2
		if len(resp) >= 5+nvPayloadSize+2 {
			statusOffset = 5 + nvPayloadSize
		}
		status := binary.LittleEndian.Uint16(resp[statusOffset : statusOffset+2])
		if status != 0 {
			return fmt.Errorf("NV ext write: item %d rejected, status=%d (%s)", id, status, NVStatusStr(status))
		}
	}
	return nil
}


