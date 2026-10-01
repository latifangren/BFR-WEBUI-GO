//go:build android || linux

package efs

import (
	"encoding/binary"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"bfr-webui-go/internal/logger"
)

const (
	AF_QIPCRTR           = 42
	QRTR_PORT_CTRL       = 0xfffffffe
	QRTR_TYPE_NEW_LOOKUP = 10
	QRTR_TYPE_NEW_SERVER = 4
	QRTR_DIAG_SERVICE    = 4097

	DIAG_BAD_CMD_F      = 0x13
	DIAG_BAD_PARM_F     = 0x14
	DIAG_BAD_LEN_F      = 0x15
	DIAG_BAD_DEV_F      = 0x16
	DIAG_BAD_MODE_F     = 0x17
	DIAG_BAD_SPC_F      = 0x18
	DIAG_BAD_SEC_MODE_F = 0x42

	SYS_PPOLL = 73
	POLLIN    = 0x0001
	POLLOUT   = 0x0004
)

type SockaddrQRTR struct {
	Family uint16
	Pad    uint16
	Node   uint32
	Port   uint32
}

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

// DiagClient manages communication with the Qualcomm modem DIAG subsystem over AF_QIPCRTR.
type DiagClient struct {
	mu        sync.Mutex
	fd        int
	node      uint32
	port      uint32
	rxFrames  uint64
	rxDropped uint64
}

// NewDiagClient initializes and connects to the DIAG service on the QRTR bus.
func NewDiagClient(timeout time.Duration) (*DiagClient, error) {
	fd, err := syscall.Socket(AF_QIPCRTR, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, fmt.Errorf("socket(AF_QIPCRTR) failed: %w", err)
	}
	_ = syscall.SetNonblock(fd, true)

	c := &DiagClient{fd: fd}

	// Look up DIAG service on the bus
	node, port, err := c.findDiagService(timeout)
	if err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}

	c.node = node
	c.port = port

	dest := SockaddrQRTR{Family: AF_QIPCRTR, Node: node, Port: port}
	_, _, errSys := syscall.RawSyscall(syscall.SYS_CONNECT, uintptr(fd), uintptr(unsafe.Pointer(&dest)), uintptr(unsafe.Sizeof(dest)))
	if errSys != 0 {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("connect to DIAG (%d:%d) failed: %v", node, port, errSys)
	}

	logger.Get().Infof("EFS", "Connected to Qualcomm DIAG service on QRTR node %d port %d", node, port)
	return c, nil
}

// Close terminates the DIAG socket.
func (c *DiagClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fd >= 0 {
		_ = syscall.Close(c.fd)
		c.fd = -1
	}
}

// IsOpen returns whether the socket is currently active.
func (c *DiagClient) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fd >= 0
}

// findDiagService discovers the QRTR node and port serving DIAG service 4097.
func (c *DiagClient) findDiagService(timeout time.Duration) (uint32, uint32, error) {
	var pkt [20]byte
	binary.LittleEndian.PutUint32(pkt[0:4], QRTR_TYPE_NEW_LOOKUP)
	binary.LittleEndian.PutUint32(pkt[4:8], QRTR_DIAG_SERVICE)

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
		return 0, 0, fmt.Errorf("sendto QRTR lookup failed: %v", errSys)
	}

	buf := make([]byte, 2048)
	var foundNode, foundPort uint32
	found := false
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ready, err := pollRead(c.fd, 150)
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
			inst := binary.LittleEndian.Uint32(buf[8:12])
			node := binary.LittleEndian.Uint32(buf[12:16])
			port := binary.LittleEndian.Uint32(buf[16:20])

			if srv == QRTR_DIAG_SERVICE {
				if !found || inst == 1 {
					foundNode = node
					foundPort = port
					found = true
					if inst == 1 {
						break
					}
				}
			}
		}
	}

	if !found {
		return 0, 0, fmt.Errorf("DIAG service (%d) not found on QRTR bus", QRTR_DIAG_SERVICE)
	}
	return foundNode, foundPort, nil
}

// Send transmits a raw DIAG payload over the connected socket.
func (c *DiagClient) Send(req []byte) error {
	if c.fd < 0 {
		return fmt.Errorf("socket closed")
	}
	_, _, errSys := syscall.RawSyscall(
		syscall.SYS_WRITE,
		uintptr(c.fd),
		uintptr(unsafe.Pointer(&req[0])),
		uintptr(len(req)),
	)
	if errSys != 0 {
		return fmt.Errorf("write to DIAG failed: %v", errSys)
	}
	return nil
}

// unwrap strips the QRTR/DIAG protocol wrappers from received datagrams.
func unwrap(p []byte) []byte {
	plen := len(p)
	// SM8850+ target wrapper: [u16 type=8][u16 length]
	if plen >= 8 {
		declared := int(p[2]) | (int(p[3]) << 8)
		if declared+4 == plen && p[4] == 0x7E {
			p = p[4:]
			plen -= 4
		}
	}

	// HDLC framing: 7E 01 <len16> <payload> 7E
	if plen >= 5 && p[0] == 0x7E && p[1] == 0x01 {
		declared := int(p[2]) | (int(p[3]) << 8)
		if declared > 0 && declared+4 <= plen {
			return p[4 : 4+declared]
		}
	}

	return p
}

func matchesDiag(p []byte, wantCmd int, wantSubsys int, wantSubcmd int) bool {
	if len(p) < 1 {
		return false
	}
	if wantCmd >= 0 && int(p[0]) != wantCmd {
		// Target error code block
		if p[0] >= DIAG_BAD_CMD_F && p[0] <= DIAG_BAD_SPC_F {
			return true
		}
		// Security refusal echoes rejected command
		if p[0] == DIAG_BAD_SEC_MODE_F && len(p) >= 2 && int(p[1]) == wantCmd {
			return true
		}
		return false
	}
	if wantSubsys >= 0 {
		if len(p) < 4 || int(p[1]) != wantSubsys {
			return false
		}
		if wantSubcmd >= 0 {
			subcmd := int(p[2]) | (int(p[3]) << 8)
			if subcmd != wantSubcmd {
				return false
			}
		}
	}
	return true
}

// Recv reads datagrams from the DIAG socket until a matching frame arrives or timeout expires.
func (c *DiagClient) Recv(timeout time.Duration, wantCmd int, wantSubsys int, wantSubcmd int) ([]byte, error) {
	if c.fd < 0 {
		return nil, fmt.Errorf("socket closed")
	}

	rx := make([]byte, 512*1024)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		ms := int(remaining.Milliseconds())
		if ms > 250 {
			ms = 250
		}
		ready, err := pollRead(c.fd, ms)
		if err != nil || !ready {
			continue
		}

		n, _, errSys := syscall.RawSyscall(
			syscall.SYS_READ,
			uintptr(c.fd),
			uintptr(unsafe.Pointer(&rx[0])),
			uintptr(len(rx)),
		)
		if errSys != 0 || n <= 0 {
			continue
		}
		c.rxFrames++

		payload := unwrap(rx[:n])
		if len(payload) == 0 {
			c.rxDropped++
			continue
		}

		if matchesDiag(payload, wantCmd, wantSubsys, wantSubcmd) {
			result := make([]byte, len(payload))
			copy(result, payload)
			return result, nil
		}

		c.rxDropped++
	}

	return nil, fmt.Errorf("timed out waiting for DIAG response (cmd=0x%02x, subsys=0x%02x)", wantCmd, wantSubsys)
}

// Xfer performs a synchronized send and receive over DIAG.
func (c *DiagClient) Xfer(req []byte, timeout time.Duration) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.Send(req); err != nil {
		return nil, err
	}

	wantCmd := int(req[0])
	wantSubsys := -1
	wantSubcmd := -1
	if len(req) >= 4 && req[0] == 0x4B {
		wantSubsys = int(req[1])
		wantSubcmd = int(req[2]) | (int(req[3]) << 8)
	}

	resp, err := c.Recv(timeout, wantCmd, wantSubsys, wantSubcmd)
	if err != nil {
		return nil, err
	}

	// Check if target rejected command
	if len(resp) >= 1 && resp[0] != req[0] {
		var reason string
		switch resp[0] {
		case DIAG_BAD_CMD_F:
			reason = "bad command"
		case DIAG_BAD_PARM_F:
			reason = "bad parameter"
		case DIAG_BAD_LEN_F:
			reason = "bad length"
		case DIAG_BAD_DEV_F:
			reason = "bad device"
		case DIAG_BAD_MODE_F:
			reason = "bad mode"
		case DIAG_BAD_SPC_F:
			reason = "bad SPC mode"
		case DIAG_BAD_SEC_MODE_F:
			reason = "security state does not allow command (SPC unlock required)"
		default:
			reason = fmt.Sprintf("error code 0x%02x", resp[0])
		}
		return nil, fmt.Errorf("modem rejected command: %s", reason)
	}

	return resp, nil
}

// Spc sends a 6-digit Service Programming Code (SPC) to elevate the DIAG security session.
func (c *DiagClient) Spc(spc string, timeout time.Duration) (bool, error) {
	if len(spc) != 6 {
		return false, fmt.Errorf("SPC must be exactly 6 digits")
	}

	cmd := make([]byte, 7)
	cmd[0] = 0x41
	copy(cmd[1:7], []byte(spc))

	resp, err := c.Xfer(cmd, timeout)
	if err != nil {
		return false, err
	}
	if len(resp) < 2 || resp[0] != 0x41 {
		return false, fmt.Errorf("invalid SPC response from modem")
	}

	return resp[1] != 0, nil
}

// SSR issues a Subsystem Restart (SSR) to cleanly reboot the Qualcomm modem.
func (c *DiagClient) SSR() error {
	// Subsystem 0x25, Command 0x0003
	req := []byte{0x4B, 0x25, 0x03, 0x00}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Send(req)
}

// GetStats returns frame transfer diagnostics.
func (c *DiagClient) GetStats() DiagStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return DiagStats{
		RxFrames:  c.rxFrames,
		RxDropped: c.rxDropped,
		Transport: fmt.Sprintf("qrtr %d:%d", c.node, c.port),
	}
}
