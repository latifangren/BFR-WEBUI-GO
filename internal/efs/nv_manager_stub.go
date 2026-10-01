//go:build !linux && !android

package efs

// NVManager is a stub implementation for non-Linux/Android platforms (e.g. Windows IDE).
type NVManager struct{}

// NewNVManager returns an NVManager stub.
func NewNVManager(diag any) *NVManager {
	return &NVManager{}
}

// GetManagerNV returns ErrNotSupported on non-Linux/Android systems.
func GetManagerNV() (*NVManager, error) {
	return nil, ErrNotSupported
}

// ReadNV stub.
func (n *NVManager) ReadNV(id uint16, index uint16) (*NVReadResult, error) {
	return nil, ErrNotSupported
}

// WriteNV stub.
func (n *NVManager) WriteNV(id uint16, index uint16, dataHex string) (*NVWriteResult, error) {
	return nil, ErrNotSupported
}

// BackupItems stub.
func (n *NVManager) BackupItems(ids []uint16, filename string) (<-chan NVBackupProgress, error) {
	return nil, ErrNotSupported
}

// BackupRange stub.
func (n *NVManager) BackupRange(startID, endID uint16, filename string) (<-chan NVBackupProgress, error) {
	return nil, ErrNotSupported
}

// RestoreFromFile stub.
func (n *NVManager) RestoreFromFile(filename string) (<-chan NVBackupProgress, error) {
	return nil, ErrNotSupported
}
