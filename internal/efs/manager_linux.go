//go:build android || linux

package efs

import (
	"fmt"
	"time"
)

func (m *Manager) openLocked() (*EfsSession, error) {
	diag, err := NewDiagClient(4000 * time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("DIAG connection failed: %w", err)
	}

	sess := NewEfsSession(diag)
	if err := sess.Detect(); err != nil {
		diag.Close()
		return nil, fmt.Errorf("EFS2 detection failed: %w", err)
	}

	return sess, nil
}

func (m *Manager) closeSessionLocked() {
	if m.session != nil && m.session.diag != nil {
		m.session.diag.Close()
	}
}

func (m *Manager) unlockSPCLocked(spc string) (bool, error) {
	if m.session == nil || m.session.diag == nil {
		return false, fmt.Errorf("no active DIAG connection")
	}
	return m.session.diag.Spc(spc, 4000*time.Millisecond)
}

func (m *Manager) restartModemLocked() error {
	if m.session == nil || m.session.diag == nil {
		return fmt.Errorf("no active DIAG connection")
	}

	// Flush dirty journal blocks before restart
	_ = m.session.syncLocked()

	// Send SSR packet
	if err := m.session.diag.SSR(); err != nil {
		return fmt.Errorf("SSR command failed: %w", err)
	}

	// Close session as modem is rebooting
	m.closeSessionLocked()
	m.session = nil

	return nil
}
