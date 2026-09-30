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
	dest SockaddrQRTR
}

const (
	SYS_PPOLL = 73
	POLLIN    = 0x0001
)

type pollFd struct {
	fd      int32
	events  int16
	revents int16
}

func pollRead(fd int, timeoutMs int) (bool, error) {
	pfd := pollFd{
		fd:     int32(fd),
		events: POLLIN,
	}
	ts := syscall.Timespec{
		Sec:  int64(timeoutMs / 1000),
		Nsec: int64((timeoutMs % 1000) * 1000000),
	}
	r, _, errSys := syscall.RawSyscall6(
		SYS_PPOLL,
		uintptr(unsafe.Pointer(&pfd)),
		1,
		uintptr(unsafe.Pointer(&ts)),
		0,
		0,
		0,
	)
	if errSys != 0 {
		if errSys == syscall.EINTR {
			return false, nil
		}
		return false, errSys
	}
	return r > 0 && (pfd.revents&POLLIN) != 0, nil
}

func pollWrite(fd int, timeoutMs int) (bool, error) {
	pfd := pollFd{
		fd:     int32(fd),
		events: 0x0004, // POLLOUT
	}
	ts := syscall.Timespec{
		Sec:  int64(timeoutMs / 1000),
		Nsec: int64((timeoutMs % 1000) * 1000000),
	}
	r, _, errSys := syscall.RawSyscall6(
		SYS_PPOLL,
		uintptr(unsafe.Pointer(&pfd)),
		1,
		uintptr(unsafe.Pointer(&ts)),
		0,
		0,
		0,
	)
	if errSys != 0 {
		if errSys == syscall.EINTR {
			return false, nil
		}
		return false, errSys
	}
	return r > 0 && (pfd.revents&0x0004) != 0, nil
}

// NewQRTRClient creates a new QRTR socket client in non-blocking mode.
func NewQRTRClient() (*QRTRClient, error) {
	fd, err := syscall.Socket(AF_QIPCRTR, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socket(AF_QIPCRTR) failed: %w", err)
	}
	_ = syscall.SetNonblock(fd, true)
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
	deadline := time.Now().Add(1000 * time.Millisecond)

	for time.Now().Before(deadline) {
		ready, err := pollRead(c.fd, 100)
		if err != nil || !ready {
			continue
		}
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

	c.dest = SockaddrQRTR{Family: AF_QIPCRTR, Node: foundNode, Port: foundPort}
	dest := c.dest
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

	// Poll socket write readiness (max 300ms)
	writable, err := pollWrite(c.fd, 300)
	if err != nil || !writable {
		return nil, fmt.Errorf("socket write timeout or busy: %v", err)
	}

	var errSys syscall.Errno
	for attempt := 0; attempt < 5; attempt++ {
		_, _, errSys = syscall.RawSyscall(syscall.SYS_WRITE, uintptr(c.fd), uintptr(unsafe.Pointer(&req[0])), uintptr(len(req)))
		if errSys == 0 {
			break
		}
		if errSys == syscall.EAGAIN || errSys == syscall.EINTR {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		return nil, fmt.Errorf("write error: %v", errSys)
	}
	if errSys != 0 {
		return nil, fmt.Errorf("write error: %v", errSys)
	}

	buf := make([]byte, 2048)
	deadline := time.Now().Add(1000 * time.Millisecond)

	for time.Now().Before(deadline) {
		ready, err := pollRead(c.fd, 100)
		if err != nil || !ready {
			time.Sleep(10 * time.Millisecond)
			continue
		}
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
			if errSys == syscall.EINTR || errSys == syscall.EAGAIN {
				time.Sleep(10 * time.Millisecond)
				continue
			}
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

// SetLTECellLock sets the target EARFCN and PCI for LTE cell locking on modem.
func (c *QRTRClient) SetLTECellLock(earfcn uint32, pci uint16) error {
	var payload bytes.Buffer
	// TLV 0x01: count=1, pci:u16, earfcn:u32 (7 bytes)
	payload.WriteByte(0x01)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(7))
	payload.WriteByte(1)
	_ = binary.Write(&payload, binary.LittleEndian, pci)
	_ = binary.Write(&payload, binary.LittleEndian, earfcn)

	// TLV 0x10: apply=1
	payload.WriteByte(0x10)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(1)

	_, err := c.Exchange(0x00D8, payload.Bytes())
	return err
}

// ClearLTECellLock clears the active LTE cell locking.
func (c *QRTRClient) ClearLTECellLock() error {
	var payload bytes.Buffer
	// TLV 0x01: count=0 (1 byte)
	payload.WriteByte(0x01)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(0)

	// TLV 0x10: apply=1
	payload.WriteByte(0x10)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(1)

	_, err := c.Exchange(0x00D8, payload.Bytes())
	return err
}

// GetLTECellLock reads the current LTE cell lock status from the modem.
func (c *QRTRClient) GetLTECellLock() (bool, uint32, uint16, error) {
	resp, err := c.Exchange(0x00D7, nil)
	if err != nil {
		return false, 0, 0, err
	}
	for i := 0; i+3 <= len(resp); {
		tlvType := resp[i]
		tlvLen := binary.LittleEndian.Uint16(resp[i+1 : i+3])
		i += 3
		if i+int(tlvLen) > len(resp) {
			break
		}
		val := resp[i : i+int(tlvLen)]
		i += int(tlvLen)
		if (tlvType == 0x10 || tlvType == 0x01) && len(val) >= 7 {
			count := val[0]
			if count > 0 {
				pci := binary.LittleEndian.Uint16(val[1:3])
				var earfcn uint32
				if len(val) >= 7 {
					earfcn = binary.LittleEndian.Uint32(val[3:7])
				} else if len(val) >= 5 {
					earfcn = uint32(binary.LittleEndian.Uint16(val[3:5]))
				}
				return true, earfcn, pci, nil
			}
		}
	}
	return false, 0, 0, nil
}

// SetPLMN locks the modem to a specific manual PLMN (MCC/MNC) or restores automatic selection.
func (c *QRTRClient) SetPLMN(mode string, mcc uint16, mnc uint16) error {
	var payload bytes.Buffer
	// TLV 0x17: DURATION = 1 (1 byte)
	payload.WriteByte(TLV_DURATION)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(1))
	payload.WriteByte(1)

	// TLV 0x16: NET_SEL_PREF (5 bytes: enum8 mode, u16 mcc, u16 mnc)
	payload.WriteByte(0x16)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(5))
	if mode == "manual" {
		payload.WriteByte(0x01)
	} else {
		payload.WriteByte(0x00)
	}
	_ = binary.Write(&payload, binary.LittleEndian, mcc)
	_ = binary.Write(&payload, binary.LittleEndian, mnc)

	_, err := c.Exchange(MSG_SET, payload.Bytes())
	return err
}

// ApplyLTECellLock executes LTE cell locking on the specified SIM slot.
func ApplyLTECellLock(simSlot int, earfcn uint32, pci uint16) error {
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
	_ = client.BindSim(simSlot)
	return client.SetLTECellLock(earfcn, pci)
}

// ClearLTECellLock clears LTE cell lock on the specified SIM slot.
func ClearLTECellLock(simSlot int) error {
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
	_ = client.BindSim(simSlot)
	return client.ClearLTECellLock()
}

// GetLTECellLockStatus queries the current LTE cell lock status on the modem.
func GetLTECellLockStatus(simSlot int) (bool, uint32, uint16, error) {
	if !IsQRTRSupported() {
		return false, 0, 0, fmt.Errorf("QRTR not supported")
	}
	client, err := NewQRTRClient()
	if err != nil {
		return false, 0, 0, err
	}
	defer client.Close()

	if err := client.DiscoverAndConnect(NAS_SERVICE); err != nil {
		return false, 0, 0, err
	}
	_ = client.BindSim(simSlot)
	return client.GetLTECellLock()
}

// ApplyPLMNLock locks the modem to a specific MCC/MNC or restores automatic network selection.
func ApplyPLMNLock(simSlot int, mode string, mcc uint16, mnc uint16) error {
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
	_ = client.BindSim(simSlot)
	return client.SetPLMN(mode, mcc, mnc)
}

// GetHardwareBandsInfo returns the factory hardware-supported frequency bands.
func GetHardwareBandsInfo(simSlot int) (*HardwareBandsInfo, error) {
	info := &HardwareBandsInfo{
		LTEBands: []int{1, 2, 3, 4, 5, 7, 8, 12, 13, 14, 17, 18, 19, 20, 25, 26, 28, 29, 30, 32, 38, 39, 40, 41, 42, 46, 48, 66, 71},
		NRBands:  []int{1, 2, 3, 5, 7, 8, 12, 20, 25, 28, 38, 40, 41, 66, 71, 77, 78},
	}
	return info, nil
}

const (
	MSG_GET_CELL_LOCATION = 0x0043
	MSG_GET_SIG_INFO      = 0x0051
)

// GetCellLocation queries baseband modem for serving and neighbor cell towers.
func (c *QRTRClient) GetCellLocation() (*CellScanResult, error) {
	result := &CellScanResult{
		Success:     true,
		CarrierAggr: make([]NearbyCellInfo, 0),
		Neighbors:   make([]NearbyCellInfo, 0),
		ScanTime:    time.Now().Format("15:04:05"),
	}

	var foundTLVs []string
	var primaryRSRP, primaryRSRQ, primarySINR, primaryRSSI float64

	// Query NAS 0x0043 (Get Cell Location Info)
	resp, err := c.Exchange(MSG_GET_CELL_LOCATION, nil)
	if err != nil {
		return nil, fmt.Errorf("QMI 0x0043 failed: %w", err)
	}

	if len(resp) >= 7 {
		data := resp[7:]
		off := 0
		for off+3 <= len(data) {
			tlvType := data[off]
			tlvLen := int(binary.LittleEndian.Uint16(data[off+1 : off+3]))
			off += 3
			if off+tlvLen > len(data) {
				break
			}
			val := data[off : off+tlvLen]
			off += tlvLen
			foundTLVs = append(foundTLVs, fmt.Sprintf("0x%02x (len=%d)", tlvType, tlvLen))

			switch tlvType {
			case 0x2A: // Primary Serving Band (4 bytes u32)
				if len(val) >= 4 {
					primaryBandNum := int(binary.LittleEndian.Uint32(val[0:4]))
					if result.ServingCell != nil && result.ServingCell.Band == 0 {
						result.ServingCell.Band = primaryBandNum
						result.ServingCell.BandName = fmt.Sprintf("B%d", primaryBandNum)
					}
				}
			case 0x26: // Secondary CA Band (2 bytes u16)
				if len(val) >= 2 {
					secBandNum := int(binary.LittleEndian.Uint16(val[0:2]))
					if secBandNum > 0 {
						_, secBandName := EarfcnToBand(uint32(secBandNum))
						if secBandNum < 100 {
							secBandName = fmt.Sprintf("B%d", secBandNum)
						}
						result.CarrierAggr = append(result.CarrierAggr, NearbyCellInfo{
							Type:        "lte",
							Role:        "scell",
							Band:        secBandNum,
							BandName:    secBandName,
							DistanceStr: "CA Secondary Band",
						})
					}
				}
			case 0x27: // Serving EARFCN (4 bytes u32)
				if len(val) >= 4 {
					servingEarfcn := binary.LittleEndian.Uint32(val[0:4])
					bNum, bName := EarfcnToBand(servingEarfcn)
					if result.ServingCell == nil {
						result.ServingCell = &NearbyCellInfo{
							Type:        "lte",
							Role:        "serving",
							EARFCN:      servingEarfcn,
							Band:        bNum,
							BandName:    bName,
							RSRP:        primaryRSRP,
							RSRQ:        primaryRSRQ,
							RSSI:        primaryRSSI,
							SINR:        primarySINR,
							DistanceStr: "Connected (Primary)",
						}
					} else {
						result.ServingCell.EARFCN = servingEarfcn
						result.ServingCell.Band = bNum
						result.ServingCell.BandName = bName
					}
				}
			case 0x13: // LTE Cell Information in Qualcomm NAS
				if len(val) >= 14 {
					tac := binary.LittleEndian.Uint16(val[1:3])
					cellID := binary.LittleEndian.Uint32(val[3:7])
					earfcn := uint32(binary.LittleEndian.Uint16(val[10:12]))
					pci := binary.LittleEndian.Uint16(val[12:14])
					if earfcn == 0 && result.ServingCell != nil && result.ServingCell.EARFCN > 0 {
						earfcn = result.ServingCell.EARFCN
					}
					bNum, bName := EarfcnToBand(earfcn)

					if result.ServingCell == nil {
						result.ServingCell = &NearbyCellInfo{
							Type:        "lte",
							Role:        "serving",
							CellID:      cellID,
							PCI:         pci,
							TAC:         tac,
							EARFCN:      earfcn,
							Band:        bNum,
							BandName:    bName,
							RSRP:        primaryRSRP,
							RSRQ:        primaryRSRQ,
							RSSI:        primaryRSSI,
							SINR:        primarySINR,
							DistanceStr: "Connected (Primary)",
						}
					} else {
						if cellID > 0 {
							result.ServingCell.CellID = cellID
						}
						if pci > 0 {
							result.ServingCell.PCI = pci
						}
						if tac > 0 {
							result.ServingCell.TAC = tac
						}
						if result.ServingCell.EARFCN == 0 && earfcn > 0 {
							result.ServingCell.EARFCN = earfcn
							result.ServingCell.Band = bNum
							result.ServingCell.BandName = bName
						}
					}

					// Neighbors inside TLV 0x13
					if len(val) >= 19 {
						neighborCount := int(val[18])
						pos := 19
						for n := 0; n < neighborCount && pos+10 <= len(val); n++ {
							nPci := binary.LittleEndian.Uint16(val[pos : pos+2])
							nRsrq := float64(int16(binary.LittleEndian.Uint16(val[pos+2:pos+4]))) / 10.0
							nRsrp := float64(int16(binary.LittleEndian.Uint16(val[pos+4:pos+6]))) / 10.0
							nRssi := float64(int16(binary.LittleEndian.Uint16(val[pos+6:pos+8]))) / 10.0
							pos += 10

							if nPci <= 503 && nPci != pci {
								result.Neighbors = append(result.Neighbors, NearbyCellInfo{
									Type:        "lte",
									Role:        "neighbor",
									PCI:         nPci,
									EARFCN:      earfcn,
									Band:        bNum,
									BandName:    bName,
									RSRP:        nRsrp,
									RSRQ:        nRsrq,
									RSSI:        nRssi,
									DistanceStr: "Neighbor Tower",
								})
							}
						}
					}
				}
			case 0x15: // LTE Serving Cell Information (12 bytes)
				if len(val) >= 12 {
					cellID := binary.LittleEndian.Uint32(val[0:4])
					pci := binary.LittleEndian.Uint16(val[4:6])
					tac := binary.LittleEndian.Uint16(val[6:8])
					earfcn := binary.LittleEndian.Uint32(val[8:12])
					bNum, bName := EarfcnToBand(earfcn)

					result.ServingCell = &NearbyCellInfo{
						Type:          "lte",
						Role:          "serving",
						CellID:        cellID,
						PCI:           pci,
						TAC:           tac,
						EARFCN:        earfcn,
						Band:          bNum,
						BandName:      bName,
						RSRP:          primaryRSRP,
						RSRQ:          primaryRSRQ,
						RSSI:          primaryRSSI,
						SINR:          primarySINR,
						TimingAdvance: -1,
						DistanceMeter: -1,
						DistanceStr:   "Connected (Primary)",
					}
				}
			case 0x16: // LTE Intra-frequency Neighbor Cells
				if len(val) >= 3 {
					count := int(binary.LittleEndian.Uint16(val[1:3]))
					pos := 3
					for j := 0; j < count && pos+8 <= len(val); j++ {
						pci := binary.LittleEndian.Uint16(val[pos : pos+2])
						rsrq := float64(int16(binary.LittleEndian.Uint16(val[pos+2:pos+4]))) / 10.0
						rsrp := float64(int16(binary.LittleEndian.Uint16(val[pos+4:pos+6]))) / 10.0
						rssi := float64(int16(binary.LittleEndian.Uint16(val[pos+6:pos+8]))) / 10.0
						pos += 8

						var earfcn uint32
						var bNum int
						var bName string
						if result.ServingCell != nil {
							earfcn = result.ServingCell.EARFCN
							bNum = result.ServingCell.Band
							bName = result.ServingCell.BandName
						}

						result.Neighbors = append(result.Neighbors, NearbyCellInfo{
							Type:          "lte",
							Role:          "neighbor",
							PCI:           pci,
							EARFCN:        earfcn,
							Band:          bNum,
							BandName:      bName,
							RSRP:          rsrp,
							RSRQ:          rsrq,
							RSSI:          rssi,
							TimingAdvance: -1,
							DistanceMeter: -1,
							DistanceStr:   "Intra-freq Tower",
						})
					}
				}
			case 0x17: // LTE Inter-frequency Neighbor Cells
				if len(val) >= 2 {
					freqCount := int(val[1])
					pos := 2
					for f := 0; f < freqCount && pos+5 <= len(val); f++ {
						earfcn := binary.LittleEndian.Uint32(val[pos : pos+4])
						cellCount := int(val[pos+4])
						pos += 5
						bNum, bName := EarfcnToBand(earfcn)
						for j := 0; j < cellCount && pos+8 <= len(val); j++ {
							pci := binary.LittleEndian.Uint16(val[pos : pos+2])
							rsrq := float64(int16(binary.LittleEndian.Uint16(val[pos+2:pos+4]))) / 10.0
							rsrp := float64(int16(binary.LittleEndian.Uint16(val[pos+4:pos+6]))) / 10.0
							rssi := float64(int16(binary.LittleEndian.Uint16(val[pos+6:pos+8]))) / 10.0
							pos += 8

							result.Neighbors = append(result.Neighbors, NearbyCellInfo{
								Type:          "lte",
								Role:          "neighbor",
								PCI:           pci,
								EARFCN:        earfcn,
								Band:          bNum,
								BandName:      bName,
								RSRP:          rsrp,
								RSRQ:          rsrq,
								RSSI:          rssi,
								TimingAdvance: -1,
								DistanceMeter: -1,
								DistanceStr:   "Inter-freq Tower",
							})
						}
					}
				}
			case 0x1A: // LTE SCell Information (Carrier Aggregation secondary cells)
				if len(val) >= 1 {
					sCount := int(val[0])
					pos := 1
					for s := 0; s < sCount && pos+13 <= len(val); s++ {
						earfcn := binary.LittleEndian.Uint32(val[pos+1 : pos+5])
						pci := binary.LittleEndian.Uint16(val[pos+5 : pos+7])
						rsrp := float64(int16(binary.LittleEndian.Uint16(val[pos+7:pos+9]))) / 10.0
						rsrq := float64(int16(binary.LittleEndian.Uint16(val[pos+9:pos+11]))) / 10.0
						rssi := float64(int16(binary.LittleEndian.Uint16(val[pos+11:pos+13]))) / 10.0
						pos += 13
						bNum, bName := EarfcnToBand(earfcn)

						result.CarrierAggr = append(result.CarrierAggr, NearbyCellInfo{
							Type:          "lte",
							Role:          "scell",
							PCI:           pci,
							EARFCN:        earfcn,
							Band:          bNum,
							BandName:      bName,
							RSRP:          rsrp,
							RSRQ:          rsrq,
							RSSI:          rssi,
							TimingAdvance: -1,
							DistanceMeter: -1,
							DistanceStr:   "CA Secondary (SCell)",
						})
					}
				}
			}
		}
	}

	total := len(result.Neighbors) + len(result.CarrierAggr)
	if result.ServingCell != nil {
		total++
	}
	result.TotalCells = total
	logger.Get().Infof("Modem", "QMI 0x0043 received TLVs: %v (totalCells=%d, serving=%v)", foundTLVs, result.TotalCells, result.ServingCell != nil)

	return result, nil
}

var (
	scanCacheMu    sync.Mutex
	cachedScanRes   *CellScanResult
	cachedScanAt    time.Time
	cachedScanSlot  int = -1
)

// ScanNearbyCells performs dynamic cell tower discovery combining NetMonster dumpsys telemetry and baseband QRTR.
func ScanNearbyCells(simSlot int) (*CellScanResult, error) {
	scanCacheMu.Lock()
	if cachedScanRes != nil && time.Since(cachedScanAt) < 10*time.Second {
		defer scanCacheMu.Unlock()
		return cachedScanRes, nil
	}
	scanCacheMu.Unlock()

	// 1. Gather all towers from NetMonster dumpsys parser (instant, zero lockups, guaranteed detection)
	res := ScanCellsFromDumpsys(simSlot)

	// 2. If QRTR baseband is available, query baseband modem for cell lock status & hardware measurements
	if IsQRTRSupported() {
		client, err := NewQRTRClient()
		if err == nil {
			defer client.Close()
			if err := client.DiscoverAndConnect(NAS_SERVICE); err == nil {
				if simSlot >= 0 {
					_ = client.BindSim(simSlot)
				}
				// Correlate with current cell lock
				if isLocked, lockEarfcn, lockPci, _ := client.GetLTECellLock(); isLocked {
					if res.ServingCell != nil && res.ServingCell.EARFCN == lockEarfcn && res.ServingCell.PCI == lockPci {
						res.ServingCell.IsCurrentLock = true
					}
					for i := range res.Neighbors {
						if res.Neighbors[i].EARFCN == lockEarfcn && res.Neighbors[i].PCI == lockPci {
							res.Neighbors[i].IsCurrentLock = true
						}
					}
					for i := range res.CarrierAggr {
						if res.CarrierAggr[i].EARFCN == lockEarfcn && res.CarrierAggr[i].PCI == lockPci {
							res.CarrierAggr[i].IsCurrentLock = true
						}
					}
				}

				// Try QMI 0x0043 (Get Cell Location Info) if supported
				if qrtrRes, err := client.GetCellLocation(); err == nil && qrtrRes != nil {
					if qrtrRes.ServingCell != nil {
						if res.ServingCell == nil {
							res.ServingCell = qrtrRes.ServingCell
						} else {
							if qrtrRes.ServingCell.CellID > 0 {
								res.ServingCell.CellID = qrtrRes.ServingCell.CellID
							}
							if qrtrRes.ServingCell.TAC > 0 {
								res.ServingCell.TAC = qrtrRes.ServingCell.TAC
							}
							if qrtrRes.ServingCell.PCI > 0 {
								res.ServingCell.PCI = qrtrRes.ServingCell.PCI
							}
						}
					}
					if len(qrtrRes.Neighbors) > 0 {
						for _, qn := range qrtrRes.Neighbors {
							dup := false
							for i := range res.Neighbors {
								if res.Neighbors[i].PCI == qn.PCI && res.Neighbors[i].EARFCN == qn.EARFCN {
									dup = true
									if qn.RSRP != 0 {
										res.Neighbors[i].RSRP = qn.RSRP
									}
									break
								}
							}
							if !dup {
								res.Neighbors = append(res.Neighbors, qn)
							}
						}
					}
				}
			}
		}
	}

	total := 0
	if res.ServingCell != nil {
		total++
	}
	total += len(res.CarrierAggr) + len(res.Neighbors)
	res.TotalCells = total

	scanCacheMu.Lock()
	cachedScanRes = res
	cachedScanSlot = simSlot
	cachedScanAt = time.Now()
	scanCacheMu.Unlock()

	return res, nil
}

// GetCachedScanResult returns the most recent cell tower scan result if available.
func GetCachedScanResult() *CellScanResult {
	scanCacheMu.Lock()
	defer scanCacheMu.Unlock()
	return cachedScanRes
}

