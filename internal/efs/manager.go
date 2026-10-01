package efs

import (
	"fmt"
	"sync"

	"bfr-webui-go/internal/logger"
)

// Manager coordinates Qualcomm EFS2 and DIAG lifecycle and sessions.
type Manager struct {
	mu          sync.Mutex
	session     *EfsSession
	unlockedSPC bool
	lastError   string
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager returns the singleton EFS manager.
func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{}
	})
	return instance
}

// GetStatus returns the current connection and filesystem state.
func (m *Manager) GetStatus() Status {
	m.mu.Lock()
	defer m.mu.Unlock()

	st := Status{
		Connected:   m.session != nil,
		ReadOnly:    true,
		UnlockedSPC: m.unlockedSPC,
		Error:       m.lastError,
	}

	if m.session != nil {
		st.ReadOnly = m.session.IsReadOnly()
		if statfs, err := m.session.StatFS("/"); err == nil {
			st.StatFS = statfs
		}
	}

	return st
}

// SetReadOnly configures write access protection.
func (m *Manager) SetReadOnly(ro bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		m.session.SetReadOnly(ro)
	}
}

// GetSession returns the active EFS session, opening a new one if necessary.
func (m *Manager) GetSession() (*EfsSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session != nil {
		return m.session, nil
	}

	sess, err := m.openLocked()
	if err != nil {
		m.lastError = err.Error()
		return nil, err
	}
	m.session = sess
	m.lastError = ""
	return m.session, nil
}

// Close terminates any active EFS session and DIAG socket.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeLocked()
}

func (m *Manager) closeLocked() {
	if m.session != nil {
		m.closeSessionLocked()
		m.session = nil
	}
}

// UnlockSPC elevates the DIAG session using the provided 6-digit SPC code.
func (m *Manager) UnlockSPC(spc string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session == nil {
		sess, err := m.openLocked()
		if err != nil {
			return false, err
		}
		m.session = sess
	}

	ok, err := m.unlockSPCLocked(spc)
	if err != nil {
		return false, err
	}
	m.unlockedSPC = ok
	if ok {
		logger.Get().Infof("EFS", "DIAG security level elevated with SPC")
	}
	return ok, nil
}

// SendPassword sends a 16-hex/digit security password to unlock protected NV items.
func (m *Manager) SendPassword(pwd string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session == nil {
		sess, err := m.openLocked()
		if err != nil {
			return false, err
		}
		m.session = sess
	}

	ok, err := m.sendPasswordLocked(pwd)
	if err != nil {
		return false, err
	}
	if ok {
		logger.Get().Infof("EFS", "DIAG security password accepted")
	}
	return ok, nil
}

// RestartModem triggers an SSR (Subsystem Restart) over DIAG.
func (m *Manager) RestartModem() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session == nil {
		sess, err := m.openLocked()
		if err != nil {
			return err
		}
		m.session = sess
	}

	if m.session.IsReadOnly() {
		return fmt.Errorf("modem restart is blocked while EFS is in READ-ONLY mode")
	}

	return m.restartModemLocked()
}
