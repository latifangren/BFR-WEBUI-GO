package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	CookieName      = "bfr_session"
	DefaultPassword = "bfr"
	SessionDuration = 24 * time.Hour
)

type Manager struct {
	mu       sync.RWMutex
	password string
	sessions map[string]time.Time
}

func NewManager(password string) *Manager {
	if password == "" {
		password = DefaultPassword
	}
	m := &Manager{
		password: password,
		sessions: make(map[string]time.Time),
	}
	go m.cleanupLoop()
	return m
}

func (m *Manager) Authenticate(password string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if password != m.password {
		return "", false
	}

	token, err := generateToken()
	if err != nil {
		return "", false
	}
	m.sessions[token] = time.Now().Add(SessionDuration)
	return token, true
}

func (m *Manager) ValidateSession(token string) bool {
	if token == "" {
		return false
	}
	m.mu.RLock()
	exp, exists := m.sessions[token]
	m.mu.RUnlock()

	if !exists {
		return false
	}
	if time.Now().After(exp) {
		m.Logout(token)
		return false
	}
	return true
}

func (m *Manager) Logout(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

func (m *Manager) GetTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop() // M-4: stop ticker to release resources when goroutine exits.
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for token, exp := range m.sessions {
			if now.After(exp) {
				delete(m.sessions, token)
			}
		}
		m.mu.Unlock()
	}
}

// H-5: generateToken returns an error if crypto/rand fails; no timestamp fallback.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
