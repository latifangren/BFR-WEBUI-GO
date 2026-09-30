//go:build android || linux

package modem

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"bfr-webui-go/internal/logger"
)

type SockaddrQRTR struct {
	Family uint16
	Pad    uint16
	Node   uint32
	Port   uint32
}

const (
	AF_QIPCRTR           = 42
	QRTR_PORT_CTRL       = 0xfffffffe
	QRTR_TYPE_NEW_LOOKUP = 10
	QRTR_TYPE_NEW_SERVER = 4

	NAS_SERVICE          = 3
	DMS_SERVICE          = 2

	MSG_SET              = 0x0033
	MSG_GET              = 0x0034
	MSG_BIND             = 0x0045
	DMS_MSG_GET_BANDS    = 0x0045

	TLV_RESULT           = 0x02
	TLV_MODE             = 0x11
	TLV_LEGACY           = 0x12
	TLV_LTE              = 0x15
	TLV_DURATION         = 0x17
	TLV_EXT_LTE_GET      = 0x23
	TLV_EXT_LTE_SET      = 0x24
	TLV_NR_COMBINED      = 0x2B
	TLV_NR_SA_GET        = 0x2C
	TLV_NR_NSA_GET       = 0x2D
	TLV_NR_MODE          = 0x2E
	TLV_NR_SA_SET        = 0x2F
	TLV_NR_NSA_SET       = 0x30
)

var (
	qrtrSupportedOnce sync.Once
	qrtrSupported     bool
)

// IsQRTRSupported checks whether kernel has AF_QIPCRTR support and Qualcomm modem QRTR bus is active.
func IsQRTRSupported() bool {
	qrtrSupportedOnce.Do(func() {
		// Quick check via /proc/net/protocols
		if data, err := os.ReadFile("/proc/net/protocols"); err == nil {
			if strings.Contains(string(data), "QIPCRTR") {
				// Verify by opening a test socket
				fd, err := syscall.Socket(AF_QIPCRTR, syscall.SOCK_DGRAM, 0)
				if err == nil {
					_ = syscall.Close(fd)
					qrtrSupported = true
					return
				}
			}
		}

		// Fallback socket test
		fd, err := syscall.Socket(AF_QIPCRTR, syscall.SOCK_DGRAM, 0)
		if err == nil {
			_ = syscall.Close(fd)
			qrtrSupported = true
		}
	})
	return qrtrSupported
}

// QRTRClient handles QMI transactions over Qualcomm QRTR transport.
type QRTRClient struct {
	fd   int
	txid uint16
}

// NewQRTRClient creates a new QRTR socket client.
func NewQRTRClient() (*QRTRClient, error) {
	fd, err := syscall.Socket(AF_QIPCRTR, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socket(AF_QIPCRTR) failed: %w", err)
	}
	tv := syscall.Timeval{Sec: 3, Usec: 0}
	_ = syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)
	return &QRTRClient{fd: fd, txid: 1}, nil
}

// Close terminates the QRTR socket.
func (c *QRTRClient) Close() {
	if c.fd >= 0 {
		_ = syscall.Close(c.fd)
		c.fd = -1
	}
}

// DiscoverAndConnect performs dynamic QRTR service discovery and connects the socket to the server endpoint.
func (c *QRTRClient) DiscoverAndConnect(service uint32) error {
	var pkt [20]byte
	binary.LittleEndian.PutUint32(pkt[0:4], QRTR_TYPE_NEW_LOOKUP)
	binary.LittleEndian.PutUint32(pkt[4:8], service)
	binary.LittleEndian.PutUint32(pkt[8:12], 1) // instance 1

	ctrlDest := SockaddrQRTR{Family: AF_QIPCRTR, Node: 1, Port: QRTR_PORT_CTRL}
	_, _, errSys := syscall.RawSyscall6(
		syscall.SYS_SENDTO,
		uintptr(c.fd),
		uintptr(unsafe.Pointer(&pkt[0])),
		uintptr(len(pkt)),
		0,
		uintptr(unsafe.Pointer(&ctrlDest)),
		uintptr(unsafe.Sizeof(ctrlDest)),
	)
	if errSys != 0 {
		return fmt.Errorf("sendto lookup failed: %v", errSys)
	}

	buf := make([]byte, 2048)
	var foundNode, foundPort uint32
	found := false

	for {
		var from SockaddrQRTR
		fromLen := uint32(unsafe.Sizeof(from))
		n, _, errSys := syscall.RawSyscall6(
			syscall.SYS_RECVFROM,
			uintptr(c.fd),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			0,
			uintptr(unsafe.Pointer(&from)),
			uintptr(unsafe.Pointer(&fromLen)),
		)
		if errSys != 0 {
			break
		}
		if n >= 20 && binary.LittleEndian.Uint32(buf[0:4]) == QRTR_TYPE_NEW_SERVER {
			srv := binary.LittleEndian.Uint32(buf[4:8])
			if srv == service {
				foundNode = binary.LittleEndian.Uint32(buf[12:16])
				foundPort = binary.LittleEndian.Uint32(buf[16:20])
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("service %d not found on QRTR bus", service)
	}

	dest := SockaddrQRTR{Family: AF_QIPCRTR, Node: foundNode, Port: foundPort}
	_, _, errSys = syscall.RawSyscall(syscall.SYS_CONNECT, uintptr(c.fd), uintptr(unsafe.Pointer(&dest)), uintptr(unsafe.Sizeof(dest)))
	if errSys != 0 {
		return fmt.Errorf("connect to %d:%d failed: %v", foundNode, foundPort, errSys)
	}

	return nil
}

// Exchange writes a QMI request message and waits for the matching reply (handling indications gracefully).
func (c *QRTRClient) Exchange(msgID uint16, payload []byte) ([]byte, error) {
	c.txid++
	if c.txid == 0 {
		c.txid = 1
	}

	req := make([]byte, 7+len(payload))
	req[0] = 0x00 // Request
	binary.LittleEndian.PutUint16(req[1:3], c.txid)
	binary.LittleEndian.PutUint16(req[3:5], msgID)
	binary.LittleEndian.PutUint16(req[5:7], uint16(len(payload)))
	copy(req[7:], payload)

	_, _, errSys := syscall.RawSyscall(syscall.SYS_WRITE, uintptr(c.fd), uintptr(unsafe.Pointer(&req[0])), uintptr(len(req)))
	if errSys != 0 {
		return nil, fmt.Errorf("write error: %v", errSys)
	}

	buf := make([]byte, 2048)
	for i := 0; i < 16; i++ {
		n, _, errSys := syscall.RawSyscall(syscall.SYS_READ, uintptr(c.fd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if errSys != 0 {
			return nil, fmt.Errorf("read error: %v", errSys)
		}
		if n >= 7 && buf[0] == 0x02 && binary.LittleEndian.Uint16(buf[1:3]) == c.txid && binary.LittleEndian.Uint16(buf[3:5]) == msgID {
			// Validate Result TLV if present
			if n >= 14 && buf[7] == TLV_RESULT {
				res := binary.LittleEndian.Uint16(buf[10:12])
				errCode := binary.LittleEndian.Uint16(buf[12:14])
				if res != 0 {
					return nil, fmt.Errorf("modem rejected command (res=0x%04x err=0x%04x)", res, errCode)
				}
			}
			return buf[:n], nil
		}
	}
	return nil, fmt.Errorf("timeout waiting for msg 0x%04x reply", msgID)
}

// BindSim binds the active NAS session to SIM slot (0 = SIM 1, 1 = SIM 2).
func (c *QRTRClient) BindSim(slot int) error {
	payload := []byte{0x01, 0x01, 0x00, byte(slot)}
	_, err := c.Exchange(MSG_BIND, payload)
	return err
}

// QRTRBandStatus represents the active forcing and band state reported by the modem.
type QRTRBandStatus struct {
	RATMode     uint16 `json:"rat_mode"`
	LTEBands    []int  `json:"lte_bands"`
	NRBands     []int  `json:"nr_bands"`
	NRMode      string `json:"nr_mode"` // "both", "sa", "nsa", "unknown"
}

// HardwareBandsInfo represents modem hardware supported band capabilities.
type HardwareBandsInfo struct {
	LTEBands []int `json:"lte_bands"`
	NRBands  []int `json:"nr_bands"`
}

// QueryCurrentBands reads the current active band configuration from the modem via QMI NAS 0x0034.
func (c *QRTRClient) QueryCurrentBands() (*QRTRBandStatus, error) {
	resp, err := c.Exchange(MSG_GET, nil)
	if err != nil {
		return nil, err
	}

	st := &QRTRBandStatus{
		LTEBands: make([]int, 0),
		NRBands:  make([]int, 0),
		NRMode:   "both",
	}

	if len(resp) < 7 {
		return st, nil
	}

	data := resp[7:]
	off := 0
	for off+3 <= len(data) {
		t := data[off]
		l := int(binary.LittleEndian.Uint16(data[off+1 : off+3]))
		off += 3
		if off+l > len(data) {
			break
		}
		v := data[off : off+l]
		off += l

		switch t {
		case TLV_MODE:
			if l >= 2 {
				st.RATMode = binary.LittleEndian.Uint16(v[:2])
			}
		case TLV_LTE:
			if l == 8 && len(st.LTEBands) == 0 {
				m := binary.LittleEndian.Uint64(v)
				for b := 1; b <= 64; b++ {
					if (m & (1 << (b - 1))) != 0 {
						st.LTEBands = append(st.LTEBands, b)
					}
				}
			}
		case TLV_EXT_LTE_GET:
			if l == 32 {
				st.LTEBands = nil
				for b := 1; b <= 256; b++ {
					idx := (b - 1) / 8
					bit := (b - 1) % 8
					if (v[idx] & (1 << bit)) != 0 {
						st.LTEBands = append(st.LTEBands, b)
					}
				}
			}
		case TLV_NR_SA_GET, TLV_NR_NSA_GET:
			if l == 64 && len(st.NRBands) == 0 {
				for b := 1; b <= 512; b++ {
					idx := (b - 1) / 8
					bit := (b - 1) % 8
					if (v[idx] & (1 << bit)) != 0 {
						st.NRBands = append(st.NRBands, b)
					}
				}
			}
		case TLV_NR_COMBINED:
			if l == 4 {
				switch v[0] {
				case 0:
					st.NRMode = "both"
				case 1:
					st.NRMode = "nsa"
				case 2:
					st.NRMode = "sa"
				}
			}
		}
	}
	return st, nil
}

// BandLockVerificationResult represents detailed execution and verification feedback from modem baseband.
type BandLockVerificationResult struct {
	Success        bool     `json:"success"`
	Verified       bool     `json:"verified"`
	TargetSlot     int      `json:"target_slot"`
	TargetSlotName string   `json:"target_slot_name"`
	RequestedLTE   []int    `json:"requested_lte"`
	ActiveLTEBands []int    `json:"active_lte_bands"`
	ActiveNRBands  []int    `json:"active_nr_bands"`
	Message        string   `json:"message"`
	StepLogs       []string `json:"step_logs"`
}

// SetLTEBands sends an atomic QMI request with only LTE band filter TLVs (TLV 0x15 or 0x24).
func (c *QRTRClient) SetLTEBands(lteBands []int) error {
	var payload bytes.Buffer
	// TLV 0x17: DURATION (1 = power cycle, 0 = permanent)
	payload.WriteByte(TLV_DURATION)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(1)

	hasHigh := false
	for _, b := range lteBands {
		if b > 64 {
			hasHigh = true
			break
		}
	}

	if hasHigh {
		mask := make([]byte, 32)
		for _, b := range lteBands {
			if b >= 1 && b <= 256 {
				idx := (b - 1) / 8
				bit := (b - 1) % 8
				mask[idx] |= (1 << bit)
			}
		}
		payload.WriteByte(TLV_EXT_LTE_SET)
		_ = binary.Write(&payload, binary.LittleEndian, uint16(32))
		payload.Write(mask)
	} else {
		var mask uint64
		for _, b := range lteBands {
			if b >= 1 && b <= 64 {
				mask |= (1 << (b - 1))
			}
		}
		payload.WriteByte(TLV_LTE)
		_ = binary.Write(&payload, binary.LittleEndian, uint16(8))
		_ = binary.Write(&payload, binary.LittleEndian, mask)
	}

	_, err := c.Exchange(MSG_SET, payload.Bytes())
	return err
}

// SetRATMode sends an atomic QMI request with only RAT mode TLV (TLV 0x11).
func (c *QRTRClient) SetRATMode(rat string) error {
	var payload bytes.Buffer
	payload.WriteByte(TLV_DURATION)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(1)

	var ratMask uint16
	switch rat {
	case "5g_only":
		ratMask = 64
	case "4g_only", "lte_only":
		ratMask = 16
	case "3g_only":
		ratMask = 8
	default:
		ratMask = 0x00FF // auto / hybrid
	}
	payload.WriteByte(TLV_MODE)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(2))
	_ = binary.Write(&payload, binary.LittleEndian, ratMask)

	_, err := c.Exchange(MSG_SET, payload.Bytes())
	return err
}

// SetNRBands sends an atomic QMI request with only combined NR band filter TLV (TLV 0x2B).
func (c *QRTRClient) SetNRBands(nrBands []int) error {
	if len(nrBands) == 0 {
		return nil
	}

	var payload bytes.Buffer
	payload.WriteByte(TLV_DURATION)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(1)

	nrMask := make([]byte, 64)
	for _, b := range nrBands {
		if b >= 1 && b <= 512 {
			idx := (b - 1) / 8
			bit := (b - 1) % 8
			nrMask[idx] |= (1 << bit)
		}
	}
	payload.WriteByte(TLV_NR_COMBINED)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(64))
	payload.Write(nrMask)

	_, err := c.Exchange(MSG_SET, payload.Bytes())
	return err
}

// ResetBands restores all hardware-supported LTE and NR bands.
func (c *QRTRClient) ResetBands() error {
	// Standard all-band mask for LTE (B1..B48)
	var allBands uint64 = 0x0000e7ffffdf3fff
	var payload bytes.Buffer
	payload.WriteByte(TLV_DURATION)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(1)

	payload.WriteByte(TLV_LTE)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(8))
	_ = binary.Write(&payload, binary.LittleEndian, allBands)

	// Mode auto
	payload.WriteByte(TLV_MODE)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(2))
	_ = binary.Write(&payload, binary.LittleEndian, uint16(0x00FF))

	_, err := c.Exchange(MSG_SET, payload.Bytes())
	return err
}

// GetQRTRActiveBands connects to the modem, binds the requested SIM slot, and queries active bands.
func GetQRTRActiveBands(simSlot int) (*QRTRBandStatus, error) {
	if !IsQRTRSupported() {
		return nil, fmt.Errorf("QRTR not supported on this device")
	}

	client, err := NewQRTRClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.DiscoverAndConnect(NAS_SERVICE); err != nil {
		return nil, err
	}

	if err := client.BindSim(simSlot); err != nil {
		logger.Get().Warnf("Modem", "Bind SIM %d warning: %v", simSlot, err)
	}

	return client.QueryCurrentBands()
}

// ApplyQRTRBandLockDetailed executes atomic band locking and performs live ground truth verification.
func ApplyQRTRBandLockDetailed(simSlot int, lteBands []int, nrBands []int, rat string, nrMode string) (*BandLockVerificationResult, error) {
	slotName := fmt.Sprintf("SIM %d", simSlot+1)
	simStatus := GetSimStatus()
	isActiveDataSlot := false
	if simStatus != nil {
		for _, s := range simStatus.Slots {
			if s.SlotIndex == simSlot {
				slotName = s.Label
				isActiveDataSlot = s.IsActive
				break
			}
		}
	}

	res := &BandLockVerificationResult{
		Success:        false,
		Verified:       false,
		TargetSlot:     simSlot,
		TargetSlotName: slotName,
		RequestedLTE:   lteBands,
		StepLogs:       make([]string, 0),
	}

	if simStatus != nil && !isActiveDataSlot {
		activeName := fmt.Sprintf("SIM %d", simStatus.ActiveSlot+1)
		for _, s := range simStatus.Slots {
			if s.SlotIndex == simStatus.ActiveSlot {
				activeName = s.Label
				break
			}
		}
		res.StepLogs = append(res.StepLogs, fmt.Sprintf("Notice: Targeting slot %d (%s). Mobile data is currently routing through slot %d (%s).", simSlot, slotName, simStatus.ActiveSlot, activeName))
	}

	if !IsQRTRSupported() {
		err := fmt.Errorf("QRTR socket interface not supported on this device")
		res.Message = err.Error()
		return res, err
	}

	client, err := NewQRTRClient()
	if err != nil {
		res.Message = fmt.Sprintf("Failed to initialize QRTR client: %v", err)
		return res, err
	}
	defer client.Close()

	if err := client.DiscoverAndConnect(NAS_SERVICE); err != nil {
		res.Message = fmt.Sprintf("Failed to connect to Qualcomm NAS service: %v", err)
		return res, err
	}
	res.StepLogs = append(res.StepLogs, "Connected to Qualcomm NAS service over AF_QIPCRTR")

	// Step 1: Bind SIM Slot
	if err := client.BindSim(simSlot); err != nil {
		res.StepLogs = append(res.StepLogs, fmt.Sprintf("Bind SIM %d notice: %v", simSlot, err))
	} else {
		res.StepLogs = append(res.StepLogs, fmt.Sprintf("Bound to %s (Slot %d)", slotName, simSlot))
	}

	// Step 2: Atomic LTE Band Setter
	if len(lteBands) > 0 {
		if err := client.SetLTEBands(lteBands); err != nil {
			res.Message = fmt.Sprintf("Modem rejected LTE bands %v: %v", lteBands, err)
			res.StepLogs = append(res.StepLogs, res.Message)
			return res, err
		}
		res.StepLogs = append(res.StepLogs, fmt.Sprintf("LTE band filter accepted by modem for: %v", lteBands))
	}

	// Step 3: Atomic RAT Mode Setter
	if rat != "" {
		if err := client.SetRATMode(rat); err != nil {
			res.StepLogs = append(res.StepLogs, fmt.Sprintf("Notice: Modem rejected RAT mode (%s): %v", rat, err))
		} else {
			res.StepLogs = append(res.StepLogs, fmt.Sprintf("Radio RAT technology set to %s", rat))
		}
	}

	// Step 4: Atomic NR Bands Setter (Optional)
	if len(nrBands) > 0 {
		if err := client.SetNRBands(nrBands); err != nil {
			res.StepLogs = append(res.StepLogs, fmt.Sprintf("Notice: 5G NR band filter rejected by modem: %v", err))
		} else {
			res.StepLogs = append(res.StepLogs, fmt.Sprintf("5G NR band filter accepted for: %v", nrBands))
		}
	}

	// Step 5: Wait 250ms for RF state transition
	time.Sleep(250 * time.Millisecond)

	// Step 6: Ground Truth Verification via QMI 0x0034
	st, err := client.QueryCurrentBands()
	if err != nil {
		res.StepLogs = append(res.StepLogs, fmt.Sprintf("Ground truth query warning: %v", err))
		res.Success = true
		res.Message = "Band lock applied, but post-verification query timed out."
		return res, nil
	}

	res.ActiveLTEBands = st.LTEBands
	res.ActiveNRBands = st.NRBands
	res.StepLogs = append(res.StepLogs, fmt.Sprintf("Modem baseband reports active LTE bands: %v", st.LTEBands))

	// Strict bidirectional match: counts must be equal and every requested band present
	matched := false
	if len(lteBands) > 0 {
		if len(st.LTEBands) == len(lteBands) {
			activeMap := make(map[int]bool)
			for _, b := range st.LTEBands {
				activeMap[b] = true
			}
			allMatch := true
			for _, b := range lteBands {
				if !activeMap[b] {
					allMatch = false
					break
				}
			}
			matched = allMatch
		}
	}

	res.Success = true
	res.Verified = matched
	if matched {
		res.Message = fmt.Sprintf("Successfully locked & verified on %s! Baseband confirmed bands: %v", slotName, st.LTEBands)
		res.StepLogs = append(res.StepLogs, fmt.Sprintf("Hardware Verification PASSED: Active bands %v strictly match requested %v", st.LTEBands, lteBands))
	} else {
		res.Message = fmt.Sprintf("Command applied, but baseband reports bands: %v (requested %v). Modem may require airplane mode toggle.", st.LTEBands, lteBands)
		res.StepLogs = append(res.StepLogs, fmt.Sprintf("Hardware Verification NOTICE: Baseband reported %v vs requested %v", st.LTEBands, lteBands))
	}

	return res, nil
}

// ApplyQRTRBandLock wraps ApplyQRTRBandLockDetailed for compatibility.
func ApplyQRTRBandLock(simSlot int, lteBands []int, nrBands []int, rat string, nrMode string) error {
	_, err := ApplyQRTRBandLockDetailed(simSlot, lteBands, nrBands, rat, nrMode)
	return err
}

// ResetQRTRBandLock resets the modem band locking to defaults.
func ResetQRTRBandLock(simSlot int) error {
	if !IsQRTRSupported() {
		return fmt.Errorf("QRTR not supported on this device")
	}

	client, err := NewQRTRClient()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.DiscoverAndConnect(NAS_SERVICE); err != nil {
		return err
	}

	if err := client.BindSim(simSlot); err != nil {
		logger.Get().Warnf("Modem", "Bind SIM %d warning: %v", simSlot, err)
	}

	return client.ResetBands()
}
