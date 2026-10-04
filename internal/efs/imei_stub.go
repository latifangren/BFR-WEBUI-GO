//go:build !linux && !android

package efs

// ReadIMEIs returns ErrNotSupported on non-Linux/Android platforms.
func (m *Manager) ReadIMEIs() (*IMEIInfo, error) {
	return nil, ErrNotSupported
}

// WriteIMEI returns ErrNotSupported on non-Linux/Android platforms.
func (m *Manager) WriteIMEI(req IMEIWriteReq) (*IMEIWriteResult, error) {
	return nil, ErrNotSupported
}
