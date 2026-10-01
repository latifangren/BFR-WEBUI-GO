//go:build !linux && !android

package efs

func (m *Manager) openLocked() (*EfsSession, error) {
	return nil, ErrNotSupported
}

func (m *Manager) closeSessionLocked() {}

func (m *Manager) unlockSPCLocked(spc string) (bool, error) {
	return false, ErrNotSupported
}

func (m *Manager) sendPasswordLocked(pwd string) (bool, error) {
	return false, ErrNotSupported
}

func (m *Manager) restartModemLocked() error {
	return ErrNotSupported
}
