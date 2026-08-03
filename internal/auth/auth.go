package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

const (
	CookieName      = "bfr_session"
	DefaultPassword = "bfr"
	SessionDuration = 24 * time.Hour
)

type AuthConfig struct {
	PasswordHash string `json:"password_hash"`
	IsDefault    bool   `json:"is_default"`
}

type Manager struct {
	mu           sync.RWMutex
	dataPath     string
	passwordHash string
	isDefault    bool
	sessions     map[string]time.Time
}

func HashPassword(password string, salt string) string {
	if salt == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		salt = hex.EncodeToString(b)
	}
	hash := sha256.Sum256([]byte(salt + password))
	return fmt.Sprintf("%s:%s", salt, hex.EncodeToString(hash[:]))
}

func VerifyPassword(password string, stored string) bool {
	if stored == "" {
		return false
	}
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return password == stored
	}
	salt := parts[0]
	expectedHash := parts[1]
	hash := sha256.Sum256([]byte(salt + password))
	return hex.EncodeToString(hash[:]) == expectedHash
}

func NewManager(password string) *Manager {
	m := &Manager{
		dataPath: config.GetPersistentFilePath("auth.json"),
		sessions: make(map[string]time.Time),
	}
	m.loadConfig(password)
	go m.cleanupLoop()
	return m
}

func (m *Manager) loadConfig(envPass string) {
	if data, err := os.ReadFile(m.dataPath); err == nil {
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		var cfg AuthConfig
		if err := json.Unmarshal(data, &cfg); err == nil && cfg.PasswordHash != "" {
			m.passwordHash = cfg.PasswordHash
			m.isDefault = cfg.IsDefault
			return
		}
	}

	passToUse := DefaultPassword
	m.isDefault = true
	if envPass != "" {
		passToUse = envPass
		m.isDefault = false
	}

	m.passwordHash = HashPassword(passToUse, "")
	_ = m.saveConfigLocked()
}

func (m *Manager) saveConfigLocked() error {
	cfg := AuthConfig{
		PasswordHash: m.passwordHash,
		IsDefault:    m.isDefault,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(m.dataPath)
	_ = os.MkdirAll(dir, 0755)
	return os.WriteFile(m.dataPath, data, 0644)
}

func (m *Manager) IsDefaultPassword() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isDefault
}

func (m *Manager) Authenticate(password string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	valid := false
	if m.isDefault && password == DefaultPassword {
		valid = true
	} else if VerifyPassword(password, m.passwordHash) {
		valid = true
	}

	if !valid {
		return "", false
	}

	token, err := generateToken()
	if err != nil {
		return "", false
	}
	m.sessions[token] = time.Now().Add(SessionDuration)
	return token, true
}

func (m *Manager) ChangePassword(currentPass, newPass string) error {
	newPass = strings.TrimSpace(newPass)
	if newPass == "" {
		return fmt.Errorf("new password cannot be empty")
	}
	if len(newPass) < 3 {
		return fmt.Errorf("new password must be at least 3 characters long")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	valid := false
	if m.isDefault && currentPass == DefaultPassword {
		valid = true
	} else if VerifyPassword(currentPass, m.passwordHash) {
		valid = true
	}

	if !valid {
		return fmt.Errorf("incorrect current password")
	}

	m.passwordHash = HashPassword(newPass, "")
	m.isDefault = false

	if err := m.saveConfigLocked(); err != nil {
		logger.Get().Errorf("auth", "Failed to save auth.json: %v", err)
		return fmt.Errorf("failed to save new password: %w", err)
	}

	logger.Get().Infof("auth", "Password updated successfully")
	return nil
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
	defer ticker.Stop()
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

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(b), nil
}