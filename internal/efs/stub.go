//go:build !linux && !android

package efs

import (
	"fmt"
)

var ErrNotSupported = fmt.Errorf("qualcomm EFS2 is only supported on Linux/Android ARM64 platforms")

type EfsSession struct{}

func (s *EfsSession) SetReadOnly(ro bool)                                                    {}
func (s *EfsSession) IsReadOnly() bool                                                       { return true }
func (s *EfsSession) Detect() error                                                          { return ErrNotSupported }
func (s *EfsSession) ListDir(dirPath string) ([]EfsEntry, error)                             { return nil, ErrNotSupported }
func (s *EfsSession) Stat(targetPath string, follow bool) (*EfsEntry, error)                { return nil, ErrNotSupported }
func (s *EfsSession) StatFS(targetPath string) (*StatFSInfo, error)                         { return nil, ErrNotSupported }
func (s *EfsSession) ReadFile(filePath string) ([]byte, error)                              { return nil, ErrNotSupported }
func (s *EfsSession) WriteFile(filePath string, data []byte, mode int16, forceItem int) error {
	_, err := s.WriteFileWithOptions(filePath, data, mode, WriteOptions{
		ForceItem: forceItem,
		Backup:    true,
		Verify:    true,
	})
	return err
}
func (s *EfsSession) WriteFileWithOptions(filePath string, data []byte, mode int16, opts WriteOptions) (*WriteResult, error) {
	return nil, ErrNotSupported
}
func (s *EfsSession) Mkdir(dirPath string, mode int16) error             { return ErrNotSupported }
func (s *EfsSession) Unlink(targetPath string) error                     { return ErrNotSupported }
func (s *EfsSession) Rmdir(dirPath string) error                         { return ErrNotSupported }
func (s *EfsSession) Rename(from, to string) error                       { return ErrNotSupported }
func (s *EfsSession) Chmod(targetPath string, mode int16) error          { return ErrNotSupported }
func (s *EfsSession) ReadNV(item uint16, index int) (*NVItemData, error) { return nil, ErrNotSupported }
func (s *EfsSession) WriteNV(item uint16, index int, data []byte) (*NVItemData, error) {
	return nil, ErrNotSupported
}
func (s *EfsSession) DumpImage(efsPath string, localTarPath string) (uint64, error) {
	return 0, ErrNotSupported
}
func (s *EfsSession) ReadSearcherPreferences() (*SearcherPreferences, error) {
	return nil, ErrNotSupported
}
func (s *EfsSession) WriteSearcherPreferences(prefs *SearcherPreferences, opts WriteOptions) error {
	return ErrNotSupported
}
func (s *EfsSession) ReadExtraLocks() (*ExtraLocksConfig, error) {
	return nil, ErrNotSupported
}
func (s *EfsSession) WriteExtraLocks(locks *ExtraLocksConfig, opts WriteOptions) error {
	return ErrNotSupported
}
func (s *EfsSession) ReadCapability() (*CapabilityState, error) {
	return nil, ErrNotSupported
}
func (s *EfsSession) ApplyCapability(state *CapabilityState, opts WriteOptions) error {
	return ErrNotSupported
}
func (s *EfsSession) ReadMbnAutoselect() (MbnAutoselectStatus, error) {
	return MbnAutoselectStatus{}, ErrNotSupported
}
func (s *EfsSession) WriteMbnAutoselect(simSlot int, mode int, opts WriteOptions) error {
	return ErrNotSupported
}
func (s *EfsSession) ScanMbnDirectory(baseDir string, configType string) ([]MbnConfigItem, error) {
	return nil, ErrNotSupported
}
