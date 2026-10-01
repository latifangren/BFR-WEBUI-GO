//go:build android || linux

package efs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/logger"
)

const (
	DIAG_SUBSYS_CMD_F = 0x4B
	DIAG_NV_READ_F    = 0x26
	DIAG_NV_WRITE_F   = 0x27
	DIAG_SUBSYS_NV    = 0x30
	SUBSYS_NV_READ    = 0x01
	SUBSYS_NV_WRITE   = 0x02

	DIAG_SUBSYS_EFS_STD = 0x13
	DIAG_SUBSYS_EFS_ALT = 0x3E

	EFS2_HELLO           = 0
	EFS2_QUERY           = 1
	EFS2_OPEN            = 2
	EFS2_CLOSE           = 3
	EFS2_READ            = 4
	EFS2_WRITE           = 5
	EFS2_SYMLINK         = 6
	EFS2_READLINK        = 7
	EFS2_UNLINK          = 8
	EFS2_MKDIR           = 9
	EFS2_RMDIR           = 10
	EFS2_OPENDIR         = 11
	EFS2_READDIR         = 12
	EFS2_CLOSEDIR        = 13
	EFS2_RENAME          = 14
	EFS2_STAT            = 15
	EFS2_LSTAT           = 16
	EFS2_FSTAT           = 17
	EFS2_CHMOD           = 18
	EFS2_STATFS          = 19
	EFS2_PUT_V1          = 26
	EFS2_GET_V1          = 27
	EFS2_DELTREE         = 37
	EFS2_PUT             = 38
	EFS2_GET             = 39
	EFS2_SYNC_NO_WAIT    = 48
	EFS2_SYNC_GET_STATUS = 49
	EFS2_FS_IMAGE_OPEN   = 54
	EFS2_FS_IMAGE_READ   = 55
	EFS2_FS_IMAGE_CLOSE  = 56

	EFS_O_RDONLY   = 0x00000
	EFS_O_WRONLY   = 0x00001
	EFS_O_RDWR     = 0x00002
	EFS_O_CREAT    = 0x00040
	EFS_O_TRUNC    = 0x00200
	EFS_O_ITEMFILE = 0x40000
	EFS_O_AUTODIR  = 0x80000

	EFS_MAX_IO   = 1024
	NV_ITEM_SIZE = 128
)

// EfsSession manages high-level filesystem operations on Qualcomm modem EFS2.
type EfsSession struct {
	mu        sync.Mutex
	diag      *DiagClient
	method    uint8
	detected  bool
	timeout   time.Duration
	readOnly  bool
	lastErrno int32
}

// NewEfsSession creates a new session wrapping a DiagClient.
func NewEfsSession(diag *DiagClient) *EfsSession {
	return &EfsSession{
		diag:     diag,
		method:   DIAG_SUBSYS_EFS_STD,
		timeout:  4000 * time.Millisecond,
		readOnly: true, // Safe read-only by default
	}
}

func (s *EfsSession) hdr(op uint8) []byte {
	return []byte{DIAG_SUBSYS_CMD_F, s.method, op, 0x00}
}

// SetReadOnly controls whether mutating operations are permitted.
func (s *EfsSession) SetReadOnly(ro bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readOnly = ro
}

// IsReadOnly returns the active read-only lock status.
func (s *EfsSession) IsReadOnly() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readOnly
}

func (s *EfsSession) checkWrite() error {
	if s.readOnly {
		return fmt.Errorf("modem filesystem is locked in READ-ONLY mode. Unlock first in the toolbar")
	}
	return nil
}

// Detect probes for EFS2 support across alternate and standard subsystem IDs.
func (s *EfsSession) Detect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order := []uint8{DIAG_SUBSYS_EFS_ALT, DIAG_SUBSYS_EFS_STD}
	for _, m := range order {
		cmd := make([]byte, 4+0x28)
		cmd[0] = DIAG_SUBSYS_CMD_F
		cmd[1] = m
		cmd[2] = EFS2_HELLO
		cmd[3] = 0

		for i := 0; i < 6; i++ {
			binary.LittleEndian.PutUint32(cmd[4+i*4:], 0x100000)
		}
		binary.LittleEndian.PutUint32(cmd[28:], 1)          // Version
		binary.LittleEndian.PutUint32(cmd[32:], 1)          // Min version
		binary.LittleEndian.PutUint32(cmd[36:], 1)          // Max version
		binary.LittleEndian.PutUint32(cmd[40:], 0xFFFFFFFF) // Feature bits

		resp, err := s.diag.Xfer(cmd, s.timeout)
		if err == nil && len(resp) >= 4 && resp[0] == DIAG_SUBSYS_CMD_F && resp[1] == m {
			s.method = m
			s.detected = true
			logger.Get().Infof("EFS", "EFS2 detected on Qualcomm baseband via subsystem 0x%02x", m)

			// Query capabilities
			qCmd := []byte{DIAG_SUBSYS_CMD_F, m, EFS2_QUERY, 0}
			_, _ = s.diag.Xfer(qCmd, s.timeout)
			return nil
		}
	}

	return fmt.Errorf("EFS2 baseband filesystem did not respond on subsystem 0x3E or 0x13")
}

func (s *EfsSession) ensureDetected() error {
	if !s.detected {
		return s.Detect()
	}
	return nil
}

func typeOf(mode int32, entryType int32) string {
	switch mode & 0xF000 {
	case 0x4000:
		return "dir"
	case 0x8000:
		return "file"
	case 0xA000:
		return "link"
	case 0xE000:
		return "item"
	}
	switch entryType {
	case 0:
		return "file"
	case 1:
		return "dir"
	case 2:
		return "link"
	case 15:
		return "item"
	}
	return "unknown"
}

// ListDir returns all items contained directly inside an EFS2 directory path.
func (s *EfsSession) ListDir(dirPath string) ([]EfsEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDetected(); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}

	// 1. OPENDIR
	pathBytes := append([]byte(dirPath), 0)
	cmd := append(s.hdr(EFS2_OPENDIR), pathBytes...)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return nil, fmt.Errorf("opendir '%s' failed: %w", dirPath, err)
	}
	if len(resp) < 12 {
		return nil, fmt.Errorf("opendir '%s' response truncated", dirPath)
	}

	dirp := int32(binary.LittleEndian.Uint32(resp[4:8]))
	s.lastErrno = int32(binary.LittleEndian.Uint32(resp[8:12]))
	if s.lastErrno != 0 || dirp < 0 {
		return nil, fmt.Errorf("opendir '%s' failed with EFS errno %d", dirPath, s.lastErrno)
	}

	defer func() {
		closeCmd := make([]byte, 8)
		copy(closeCmd[:4], s.hdr(EFS2_CLOSEDIR))
		binary.LittleEndian.PutUint32(closeCmd[4:], uint32(dirp))
		_, _ = s.diag.Xfer(closeCmd, s.timeout)
	}()

	var entries []EfsEntry
	var seq uint32 = 1

	for {
		readCmd := make([]byte, 12)
		copy(readCmd[:4], s.hdr(EFS2_READDIR))
		binary.LittleEndian.PutUint32(readCmd[4:], uint32(dirp))
		binary.LittleEndian.PutUint32(readCmd[8:], seq)

		rResp, rErr := s.diag.Xfer(readCmd, s.timeout)
		if rErr != nil || len(rResp) < 40 {
			break
		}

		s.lastErrno = int32(binary.LittleEndian.Uint32(rResp[12:16]))
		if s.lastErrno != 0 {
			break
		}

		entryType := int32(binary.LittleEndian.Uint32(rResp[16:20]))
		mode := int32(binary.LittleEndian.Uint32(rResp[20:24]))
		size := int32(binary.LittleEndian.Uint32(rResp[24:28]))
		atime := int32(binary.LittleEndian.Uint32(rResp[28:32]))
		mtime := int32(binary.LittleEndian.Uint32(rResp[32:36]))
		ctime := int32(binary.LittleEndian.Uint32(rResp[36:40]))

		if len(rResp) <= 40 {
			break // EOF
		}

		nameBytes := rResp[40:]
		nullIdx := bytes.IndexByte(nameBytes, 0)
		if nullIdx >= 0 {
			nameBytes = nameBytes[:nullIdx]
		}
		name := strings.TrimSpace(string(nameBytes))
		if name == "" || name == "." || name == ".." {
			if name == "" {
				break
			}
			seq++
			continue
		}

		entryPath := path.Join(dirPath, name)
		ent := EfsEntry{
			Name:      name,
			Path:      entryPath,
			Type:      typeOf(mode, entryType),
			Mode:      mode,
			Size:      int64(size),
			Atime:     time.Unix(int64(atime), 0),
			Mtime:     time.Unix(int64(mtime), 0),
			Ctime:     time.Unix(int64(ctime), 0),
			EntryType: entryType,
		}

		entries = append(entries, ent)
		seq++
	}

	return entries, nil
}

// Stat returns metadata about a file or folder.
func (s *EfsSession) Stat(targetPath string, follow bool) (*EfsEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDetected(); err != nil {
		return nil, err
	}

	op := uint8(EFS2_LSTAT)
	if follow {
		op = EFS2_STAT
	}

	pathBytes := append([]byte(targetPath), 0)
	cmd := append(s.hdr(op), pathBytes...)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < 32 {
		return nil, fmt.Errorf("stat '%s' short reply (%d bytes)", targetPath, len(resp))
	}

	s.lastErrno = int32(binary.LittleEndian.Uint32(resp[4:8]))
	if s.lastErrno != 0 {
		return nil, fmt.Errorf("stat '%s' failed, EFS errno %d", targetPath, s.lastErrno)
	}

	mode := int32(binary.LittleEndian.Uint32(resp[8:12]))
	size := int32(binary.LittleEndian.Uint32(resp[12:16]))
	nlink := int32(binary.LittleEndian.Uint32(resp[16:20]))
	atime := int32(binary.LittleEndian.Uint32(resp[20:24]))
	mtime := int32(binary.LittleEndian.Uint32(resp[24:28]))
	ctime := int32(binary.LittleEndian.Uint32(resp[28:32]))

	entry := &EfsEntry{
		Name:   path.Base(targetPath),
		Path:   targetPath,
		Type:   typeOf(mode, 0),
		Mode:   mode,
		Size:   int64(size),
		Nlink:  nlink,
		Atime:  time.Unix(int64(atime), 0),
		Mtime:  time.Unix(int64(mtime), 0),
		Ctime:  time.Unix(int64(ctime), 0),
	}

	return entry, nil
}

// StatFS returns partition disk usage.
func (s *EfsSession) StatFS(targetPath string) (*StatFSInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDetected(); err != nil {
		return nil, err
	}

	pathBytes := append([]byte(targetPath), 0)
	cmd := append(s.hdr(EFS2_STATFS), pathBytes...)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < 8 {
		return nil, fmt.Errorf("statfs short response")
	}

	s.lastErrno = int32(binary.LittleEndian.Uint32(resp[4:8]))
	if s.lastErrno != 0 {
		return nil, fmt.Errorf("statfs failed, EFS errno %d", s.lastErrno)
	}

	body := resp[8:]
	words := len(body) / 4
	v := make([]uint32, words)
	for i := 0; i < words; i++ {
		v[i] = binary.LittleEndian.Uint32(body[i*4 : i*4+4])
	}

	info := &StatFSInfo{Path: targetPath}
	for i := 0; i+2 < words; i++ {
		bs := v[i]
		total := v[i+1]
		freeb := v[i+2]

		// Power of two block size between 256 and 65536
		if bs < 256 || bs > 65536 || (bs&(bs-1)) != 0 {
			continue
		}
		if total == 0 || total > (1<<24) || freeb > total {
			continue
		}

		info.BlockSize = bs
		info.TotalBlocks = total
		info.FreeBlocks = freeb
		info.TotalBytes = uint64(bs) * uint64(total)
		info.FreeBytes = uint64(bs) * uint64(freeb)
		info.UsedBytes = info.TotalBytes - info.FreeBytes
		if info.TotalBytes > 0 {
			info.UsedPercent = int((info.UsedBytes * 100) / info.TotalBytes)
		}
		break
	}

	return info, nil
}

// ReadFile reads an entire regular file or item file from modem EFS2.
func (s *EfsSession) ReadFile(filePath string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDetected(); err != nil {
		return nil, err
	}

	// 1. Try EFS2_OPEN (for regular files)
	pathBytes := append([]byte(filePath), 0)
	openCmd := make([]byte, 12+len(pathBytes))
	copy(openCmd[:4], s.hdr(EFS2_OPEN))
	binary.LittleEndian.PutUint32(openCmd[4:], EFS_O_RDONLY)
	binary.LittleEndian.PutUint32(openCmd[8:], 0)
	copy(openCmd[12:], pathBytes)

	resp, err := s.diag.Xfer(openCmd, s.timeout)
	if err == nil && len(resp) >= 12 {
		fd := int32(binary.LittleEndian.Uint32(resp[4:8]))
		errno := int32(binary.LittleEndian.Uint32(resp[8:12]))

		if fd >= 0 && errno == 0 {
			defer func() {
				clCmd := make([]byte, 8)
				copy(clCmd[:4], s.hdr(EFS2_CLOSE))
				binary.LittleEndian.PutUint32(clCmd[4:], uint32(fd))
				_, _ = s.diag.Xfer(clCmd, s.timeout)
			}()

			var out bytes.Buffer
			var offset uint32 = 0

			for {
				readCmd := make([]byte, 16)
				copy(readCmd[:4], s.hdr(EFS2_READ))
				binary.LittleEndian.PutUint32(readCmd[4:], uint32(fd))
				binary.LittleEndian.PutUint32(readCmd[8:], EFS_MAX_IO)
				binary.LittleEndian.PutUint32(readCmd[12:], offset)

				rResp, rErr := s.diag.Xfer(readCmd, s.timeout)
				if rErr != nil || len(rResp) < 20 {
					break
				}

				got := int32(binary.LittleEndian.Uint32(rResp[12:16]))
				rErrno := int32(binary.LittleEndian.Uint32(rResp[16:20]))
				if rErrno != 0 || got <= 0 {
					break
				}

				data := rResp[20:]
				if int32(len(data)) > got {
					data = data[:got]
				}
				out.Write(data)
				offset += uint32(got)
				if got < EFS_MAX_IO {
					break // Short read = EOF
				}
			}

			return out.Bytes(), nil
		}
	}

	// 2. Fallback to GET for item files (invisible to regular file I/O)
	for _, getOp := range []uint8{EFS2_GET, EFS2_GET_V1} {
		getCmd := append(s.hdr(getOp), pathBytes...)
		gResp, gErr := s.diag.Xfer(getCmd, s.timeout)
		if gErr == nil && len(gResp) >= 12 {
			dlen := int32(binary.LittleEndian.Uint32(gResp[4:8]))
			gErrno := int32(binary.LittleEndian.Uint32(gResp[8:12]))
			if gErrno == 0 && dlen >= 0 {
				data := gResp[12:]
				if int32(len(data)) > dlen {
					data = data[:dlen]
				}
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("file '%s' could not be opened or read from EFS2", filePath)
}

// IsItemPath checks if a path belongs to known Qualcomm NV item file directories.
func IsItemPath(p string) bool {
	return strings.HasPrefix(p, "/nv/item_files/") ||
		strings.HasPrefix(p, "/nv/reg_files/") ||
		strings.HasPrefix(p, "/cgps/nv/item_files/") ||
		strings.HasPrefix(p, "/sd/")
}

// WriteFile writes data to a regular or item file in modem EFS2.
func (s *EfsSession) WriteFile(filePath string, data []byte, mode int16, forceItem int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkWrite(); err != nil {
		return err
	}
	if err := s.ensureDetected(); err != nil {
		return err
	}

	isItem := IsItemPath(filePath)
	if forceItem > 0 {
		isItem = true
	} else if forceItem == 0 {
		isItem = false
	}

	pathBytes := append([]byte(filePath), 0)

	// 1. Atomic PUT for small item files
	if isItem && len(data) <= 6144 {
		cmd := make([]byte, 14+len(data)+len(pathBytes))
		copy(cmd[:4], s.hdr(EFS2_PUT))
		binary.LittleEndian.PutUint16(cmd[4:6], uint16(len(data)))
		// Flags: O_CREAT | O_WRONLY | O_TRUNC | O_ITEMFILE | O_AUTODIR
		flags := uint32(EFS_O_CREAT | EFS_O_WRONLY | EFS_O_TRUNC | EFS_O_ITEMFILE | EFS_O_AUTODIR)
		binary.LittleEndian.PutUint32(cmd[8:12], flags)
		binary.LittleEndian.PutUint16(cmd[12:14], uint16(mode))
		copy(cmd[14:14+len(data)], data)
		copy(cmd[14+len(data):], pathBytes)

		resp, err := s.diag.Xfer(cmd, s.timeout)
		if err == nil && len(resp) >= 10 {
			errno := int16(binary.LittleEndian.Uint16(resp[6:8]))
			if errno == 0 {
				_ = s.syncLocked()
				return nil
			}
		}
	}

	// 2. Regular File OPEN / WRITE / CLOSE
	oflag := uint32(EFS_O_WRONLY | EFS_O_CREAT | EFS_O_TRUNC)
	if isItem {
		oflag |= EFS_O_ITEMFILE
	} else {
		oflag |= EFS_O_AUTODIR
	}

	openCmd := make([]byte, 12+len(pathBytes))
	copy(openCmd[:4], s.hdr(EFS2_OPEN))
	binary.LittleEndian.PutUint32(openCmd[4:], oflag)
	binary.LittleEndian.PutUint32(openCmd[8:], uint32(mode))
	copy(openCmd[12:], pathBytes)

	resp, err := s.diag.Xfer(openCmd, s.timeout)
	if err != nil {
		return fmt.Errorf("open '%s' for write failed: %w", filePath, err)
	}
	if len(resp) < 12 {
		return fmt.Errorf("open '%s' response truncated", filePath)
	}

	fd := int32(binary.LittleEndian.Uint32(resp[4:8]))
	errno := int32(binary.LittleEndian.Uint32(resp[8:12]))
	if fd < 0 || errno != 0 {
		return fmt.Errorf("open '%s' for write failed with EFS errno %d", filePath, errno)
	}

	defer func() {
		clCmd := make([]byte, 8)
		copy(clCmd[:4], s.hdr(EFS2_CLOSE))
		binary.LittleEndian.PutUint32(clCmd[4:], uint32(fd))
		_, _ = s.diag.Xfer(clCmd, s.timeout)
	}()

	var offset uint32 = 0
	for offset < uint32(len(data)) {
		chunkSize := uint32(len(data)) - offset
		if chunkSize > EFS_MAX_IO {
			chunkSize = EFS_MAX_IO
		}

		wCmd := make([]byte, 12+chunkSize)
		copy(wCmd[:4], s.hdr(EFS2_WRITE))
		binary.LittleEndian.PutUint32(wCmd[4:], uint32(fd))
		binary.LittleEndian.PutUint32(wCmd[8:], offset)
		copy(wCmd[12:], data[offset:offset+chunkSize])

		wResp, wErr := s.diag.Xfer(wCmd, s.timeout)
		if wErr != nil || len(wResp) < 20 {
			return fmt.Errorf("write '%s' failed at offset %d: %v", filePath, offset, wErr)
		}
		wrote := int32(binary.LittleEndian.Uint32(wResp[12:16]))
		wErrno := int32(binary.LittleEndian.Uint32(wResp[16:20]))
		if wErrno != 0 || wrote <= 0 {
			return fmt.Errorf("write '%s' at offset %d failed with EFS errno %d", filePath, offset, wErrno)
		}
		offset += uint32(wrote)
	}

	_ = s.syncLocked()
	return nil
}

// Mkdir creates a directory in EFS2.
func (s *EfsSession) Mkdir(dirPath string, mode int16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkWrite(); err != nil {
		return err
	}
	if err := s.ensureDetected(); err != nil {
		return err
	}

	pathBytes := append([]byte(dirPath), 0)
	cmd := make([]byte, 8+len(pathBytes))
	copy(cmd[:4], s.hdr(EFS2_MKDIR))
	binary.LittleEndian.PutUint16(cmd[4:6], uint16(mode))
	copy(cmd[8:], pathBytes)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return err
	}
	if len(resp) < 8 {
		return fmt.Errorf("mkdir short reply")
	}

	errno := int32(binary.LittleEndian.Uint32(resp[4:8]))
	if errno != 0 {
		return fmt.Errorf("mkdir '%s' failed with EFS errno %d", dirPath, errno)
	}

	_ = s.syncLocked()
	return nil
}

// Unlink deletes a regular file or item file from EFS2.
func (s *EfsSession) Unlink(targetPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkWrite(); err != nil {
		return err
	}
	if err := s.ensureDetected(); err != nil {
		return err
	}

	pathBytes := append([]byte(targetPath), 0)
	cmd := append(s.hdr(EFS2_UNLINK), pathBytes...)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return err
	}
	if len(resp) < 8 {
		return fmt.Errorf("unlink short reply")
	}

	errno := int32(binary.LittleEndian.Uint32(resp[4:8]))
	if errno != 0 {
		return fmt.Errorf("unlink '%s' failed with EFS errno %d", targetPath, errno)
	}

	_ = s.syncLocked()
	return nil
}

// Rmdir deletes an empty directory from EFS2.
func (s *EfsSession) Rmdir(dirPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkWrite(); err != nil {
		return err
	}
	if err := s.ensureDetected(); err != nil {
		return err
	}

	pathBytes := append([]byte(dirPath), 0)
	cmd := append(s.hdr(EFS2_RMDIR), pathBytes...)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return err
	}
	if len(resp) < 8 {
		return fmt.Errorf("rmdir short reply")
	}

	errno := int32(binary.LittleEndian.Uint32(resp[4:8]))
	if errno != 0 {
		return fmt.Errorf("rmdir '%s' failed with EFS errno %d", dirPath, errno)
	}

	_ = s.syncLocked()
	return nil
}

// Rename moves or renames an entry in EFS2.
func (s *EfsSession) Rename(from, to string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkWrite(); err != nil {
		return err
	}
	if err := s.ensureDetected(); err != nil {
		return err
	}

	fromBytes := append([]byte(from), 0)
	toBytes := append([]byte(to), 0)
	cmd := append(s.hdr(EFS2_RENAME), append(fromBytes, toBytes...)...)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return err
	}
	if len(resp) < 8 {
		return fmt.Errorf("rename short reply")
	}

	errno := int32(binary.LittleEndian.Uint32(resp[4:8]))
	if errno != 0 {
		return fmt.Errorf("rename '%s' -> '%s' failed with EFS errno %d", from, to, errno)
	}

	_ = s.syncLocked()
	return nil
}

// Chmod updates file permissions in EFS2.
func (s *EfsSession) Chmod(targetPath string, mode int16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkWrite(); err != nil {
		return err
	}
	if err := s.ensureDetected(); err != nil {
		return err
	}

	pathBytes := append([]byte(targetPath), 0)
	cmd := make([]byte, 8+len(pathBytes))
	copy(cmd[:4], s.hdr(EFS2_CHMOD))
	binary.LittleEndian.PutUint16(cmd[4:6], uint16(mode))
	copy(cmd[8:], pathBytes)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return err
	}
	if len(resp) < 8 {
		return fmt.Errorf("chmod short reply")
	}

	errno := int32(binary.LittleEndian.Uint32(resp[4:8]))
	if errno != 0 {
		return fmt.Errorf("chmod '%s' failed with EFS errno %d", targetPath, errno)
	}

	_ = s.syncLocked()
	return nil
}

// syncLocked flushes dirty EFS journal blocks to modem NAND/UFS flash.
func (s *EfsSession) syncLocked() error {
	cmd := append(s.hdr(EFS2_SYNC_NO_WAIT), []byte("/\x00")...)
	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil || len(resp) < 14 {
		return err
	}

	token := binary.LittleEndian.Uint32(resp[6:10])
	errno := int32(binary.LittleEndian.Uint32(resp[10:14]))
	if errno != 0 {
		return fmt.Errorf("sync start failed, errno %d", errno)
	}

	// Poll status
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		pollCmd := make([]byte, 12)
		copy(pollCmd[:4], s.hdr(EFS2_SYNC_GET_STATUS))
		binary.LittleEndian.PutUint16(pollCmd[4:6], 1)
		binary.LittleEndian.PutUint32(pollCmd[6:10], token)
		pollCmd[10] = '/'
		pollCmd[11] = 0

		pResp, pErr := s.diag.Xfer(pollCmd, s.timeout)
		if pErr == nil && len(pResp) >= 11 && pResp[6] == 0 {
			return nil
		}
	}
	return nil
}

// ReadNV reads an arbitrary Qualcomm NV item.
func (s *EfsSession) ReadNV(item uint16, index int) (*NVItemData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index >= 0 {
		// Subsystem indexed NV read
		cmd := make([]byte, 4+2+2+NV_ITEM_SIZE+2)
		cmd[0] = DIAG_SUBSYS_CMD_F
		cmd[1] = DIAG_SUBSYS_NV
		cmd[2] = SUBSYS_NV_READ
		cmd[3] = 0
		binary.LittleEndian.PutUint16(cmd[4:6], item)
		binary.LittleEndian.PutUint16(cmd[6:8], uint16(index))

		resp, err := s.diag.Xfer(cmd, s.timeout)
		if err != nil {
			return nil, err
		}
		if len(resp) < len(cmd) || resp[0] != DIAG_SUBSYS_CMD_F {
			return nil, fmt.Errorf("indexed nv_read truncated or invalid")
		}

		status := binary.LittleEndian.Uint16(resp[8+NV_ITEM_SIZE : 10+NV_ITEM_SIZE])
		data := resp[8 : 8+NV_ITEM_SIZE]
		return &NVItemData{
			Item:      item,
			Index:     index,
			DataHex:   fmt.Sprintf("%x", data),
			Status:    status,
			StatusStr: NVStatusStr(status),
		}, nil
	}

	// Legacy NV read
	cmd := make([]byte, 1+2+NV_ITEM_SIZE+2)
	cmd[0] = DIAG_NV_READ_F
	binary.LittleEndian.PutUint16(cmd[1:3], item)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < len(cmd) || resp[0] != DIAG_NV_READ_F {
		return nil, fmt.Errorf("nv_read truncated or invalid")
	}

	status := binary.LittleEndian.Uint16(resp[3+NV_ITEM_SIZE : 5+NV_ITEM_SIZE])
	data := resp[3 : 3+NV_ITEM_SIZE]
	return &NVItemData{
		Item:      item,
		Index:     -1,
		DataHex:   fmt.Sprintf("%x", data),
		Status:    status,
		StatusStr: NVStatusStr(status),
	}, nil
}

// WriteNV writes raw bytes to a Qualcomm NV item.
func (s *EfsSession) WriteNV(item uint16, index int, data []byte) (*NVItemData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkWrite(); err != nil {
		return nil, err
	}

	if len(data) > NV_ITEM_SIZE {
		data = data[:NV_ITEM_SIZE]
	}

	if index >= 0 {
		cmd := make([]byte, 4+2+2+NV_ITEM_SIZE+2)
		cmd[0] = DIAG_SUBSYS_CMD_F
		cmd[1] = DIAG_SUBSYS_NV
		cmd[2] = SUBSYS_NV_WRITE
		cmd[3] = 0
		binary.LittleEndian.PutUint16(cmd[4:6], item)
		binary.LittleEndian.PutUint16(cmd[6:8], uint16(index))
		copy(cmd[8:8+len(data)], data)

		resp, err := s.diag.Xfer(cmd, s.timeout)
		if err != nil {
			return nil, err
		}
		if len(resp) < len(cmd) || resp[0] != DIAG_SUBSYS_CMD_F {
			return nil, fmt.Errorf("indexed nv_write truncated or invalid")
		}

		status := binary.LittleEndian.Uint16(resp[8+NV_ITEM_SIZE : 10+NV_ITEM_SIZE])
		return &NVItemData{
			Item:      item,
			Index:     index,
			DataHex:   fmt.Sprintf("%x", data),
			Status:    status,
			StatusStr: NVStatusStr(status),
		}, nil
	}

	cmd := make([]byte, 1+2+NV_ITEM_SIZE+2)
	cmd[0] = DIAG_NV_WRITE_F
	binary.LittleEndian.PutUint16(cmd[1:3], item)
	copy(cmd[3:3+len(data)], data)

	resp, err := s.diag.Xfer(cmd, s.timeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < len(cmd) || resp[0] != DIAG_NV_WRITE_F {
		return nil, fmt.Errorf("nv_write truncated or invalid")
	}

	status := binary.LittleEndian.Uint16(resp[3+NV_ITEM_SIZE : 5+NV_ITEM_SIZE])
	return &NVItemData{
		Item:      item,
		Index:     -1,
		DataHex:   fmt.Sprintf("%x", data),
		Status:    status,
		StatusStr: NVStatusStr(status),
	}, nil
}

// DumpImage requests the modem baseband to generate a complete .tar archive of an EFS subtree.
func (s *EfsSession) DumpImage(efsPath string, localTarPath string) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDetected(); err != nil {
		return 0, err
	}

	pathBytes := append([]byte(efsPath), 0)
	cmd := make([]byte, 8+len(pathBytes))
	copy(cmd[:4], s.hdr(EFS2_FS_IMAGE_OPEN))
	cmd[6] = 0 // TAR format
	copy(cmd[7:], pathBytes)

	resp, err := s.diag.Xfer(cmd, 15000*time.Millisecond)
	if err != nil || len(resp) < 12 {
		return 0, fmt.Errorf("fs-image open failed: %v", err)
	}

	handle := int32(binary.LittleEndian.Uint32(resp[4:8]))
	errno := int32(binary.LittleEndian.Uint32(resp[8:12]))
	if handle < 0 || errno != 0 {
		return 0, fmt.Errorf("fs-image open failed with EFS errno %d", errno)
	}

	defer func() {
		clCmd := make([]byte, 8)
		copy(clCmd[:4], s.hdr(EFS2_FS_IMAGE_CLOSE))
		binary.LittleEndian.PutUint32(clCmd[4:], uint32(handle))
		_, _ = s.diag.Xfer(clCmd, s.timeout)
	}()

	f, err := os.OpenFile(localTarPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return 0, fmt.Errorf("cannot create local archive %s: %w", localTarPath, err)
	}
	defer f.Close()

	var total uint64 = 0
	var seq uint16 = 0

	for {
		rdCmd := make([]byte, 10)
		copy(rdCmd[:4], s.hdr(EFS2_FS_IMAGE_READ))
		binary.LittleEndian.PutUint32(rdCmd[4:8], uint32(handle))
		binary.LittleEndian.PutUint16(rdCmd[8:10], seq)

		rResp, rErr := s.diag.Xfer(rdCmd, 15000*time.Millisecond)
		if rErr != nil || len(rResp) < 15 {
			return total, fmt.Errorf("fs-image read error at seq %d: %v", seq, rErr)
		}

		rErrno := int32(binary.LittleEndian.Uint32(rResp[10:14]))
		if rErrno != 0 {
			return total, fmt.Errorf("fs-image read failed, EFS errno %d", rErrno)
		}
		isEnd := rResp[14] != 0

		if len(rResp) > 15 {
			chunk := rResp[15:]
			if _, wErr := f.Write(chunk); wErr != nil {
				return total, wErr
			}
			total += uint64(len(chunk))
		}

		if isEnd {
			break
		}
		seq++
	}

	return total, nil
}
