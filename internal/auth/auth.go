package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	CookieName    = "bfr_session"
	DefaultPassword = "bfr"
	SessionDuration = 24 * time.Hour
)

type Manager struct {
	mu          sync.RWMutex
	password    string
	sessions    map[string]time.Time
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

	token := generateToken()
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

func generateToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return strconvTimeHex()
	}
	return hex.EncodeToString(bytes)
}

func strconvTimeHex() string {
	return hex.EncodeToString([]byte(time.Now().String()))
}
