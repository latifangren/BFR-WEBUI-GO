package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

const (
	CookieName          = "bfr_session"
	DefaultPassword     = "bfr"
	SessionDuration     = 24 * time.Hour
	rateLimitMaxFails   = 5
	rateLimitWindowSecs = 60
)

type AuthConfig struct {
	PasswordHash string `json:"password_hash"`
	IsDefault    bool   `json:"is_default"`
}

// ipRateLimit tracks failed login attempts per IP.
type ipRateLimit struct {
	failedAttempts int
	resetAt        time.Time
}

type Manager struct {
	mu           sync.RWMutex
	dataPath     string
	passwordHash string
	isDefault    bool
	sessions     map[string]time.Time
	rateLimits   map[string]*ipRateLimit
}

var globalManager *Manager

func GetManager() *Manager {
	return globalManager
}

// extractIP strips the port from a remote address string.
func extractIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return strings.TrimSpace(remoteAddr)
	}
	return strings.TrimSpace(host)
}

// isBcryptHash reports whether stored looks like a bcrypt hash.
func isBcryptHash(stored string) bool {
	return strings.HasPrefix(stored, "$2a$") ||
		strings.HasPrefix(stored, "$2b$") ||
		strings.HasPrefix(stored, "$2y$")
}

// legacySHA256Hash reproduces the pre-bcrypt "salt:hex" scheme. It is only
// used as an emergency fallback if bcrypt itself fails (practically
// unreachable) and to document the old format for VerifyPassword.
func legacySHA256Hash(password, salt string) string {
	if salt == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		salt = hex.EncodeToString(b)
	}
	hash := sha256.Sum256([]byte(salt + password))
	return fmt.Sprintf("%s:%s", salt, hex.EncodeToString(hash[:]))
}

// HashPassword creates a bcrypt hash of the password. The salt parameter is
// ignored — bcrypt generates a unique salt per hash — and is kept only for
// backward compatibility with existing callers.
func HashPassword(password string, salt string) string {
	_ = salt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return legacySHA256Hash(password, "")
	}
	return string(hash)
}

// VerifyPassword checks a password against a stored hash. It accepts bcrypt
// hashes (current), legacy "salt:hex" SHA256 hashes, and — for very old
// configs — plaintext. Legacy formats are upgraded to bcrypt on the next
// successful login (see Authenticate).
func VerifyPassword(password string, stored string) bool {
	if stored == "" {
		return false
	}
	if isBcryptHash(stored) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil
	}
	if parts := strings.SplitN(stored, ":", 2); len(parts) == 2 {
		sum := sha256.Sum256([]byte(parts[0] + password))
		got := hex.EncodeToString(sum[:])
		return subtle.ConstantTimeCompare([]byte(got), []byte(parts[1])) == 1
	}
	// Very old plaintext fallback (kept for backward compatibility).
	return subtle.ConstantTimeCompare([]byte(password), []byte(stored)) == 1
}

func NewManager(password string) *Manager {
	m := &Manager{
		dataPath:   config.GetPersistentFilePath("auth.json"),
		sessions:   make(map[string]time.Time),
		rateLimits: make(map[string]*ipRateLimit),
	}
	m.loadConfig(password)
	go m.cleanupLoop()
	globalManager = m
	return m
}

func (m *Manager) loadConfig(envPass string) {
	if data, err := os.ReadFile(m.dataPath); err == nil {
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		var cfg AuthConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			m.passwordHash = HashPassword(DefaultPassword, "")
			m.isDefault = true
			_ = m.saveConfigLocked()
			return
		}
		if cfg.PasswordHash != "" {
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
	return config.WriteFileAtomic(m.dataPath, data, 0644)
}

func (m *Manager) IsDefaultPassword() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isDefault
}

func (m *Manager) Authenticate(password, remoteAddr string) (token string, ok bool, rateLimited bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ip := extractIP(remoteAddr)
	now := time.Now()

	lim, exists := m.rateLimits[ip]
	if exists {
		if now.After(lim.resetAt) {
			lim.failedAttempts = 0
			lim.resetAt = now.Add(time.Duration(rateLimitWindowSecs) * time.Second)
		} else if lim.failedAttempts >= rateLimitMaxFails {
			return "", false, true
		}
	} else {
		lim = &ipRateLimit{
			resetAt: now.Add(time.Duration(rateLimitWindowSecs) * time.Second),
		}
		m.rateLimits[ip] = lim
	}

	valid := false
	upgradeHash := false
	if m.isDefault && password == DefaultPassword {
		valid = true
	} else if VerifyPassword(password, m.passwordHash) {
		valid = true
		// Upgrade legacy hashes to bcrypt on successful login. The default
		// password is deliberately left alone so the UI keeps nagging the
		// user to change it.
		if !m.isDefault && !isBcryptHash(m.passwordHash) {
			m.passwordHash = HashPassword(password, "")
			upgradeHash = true
		}
	}

	if !valid {
		lim.failedAttempts++
		return "", false, false
	}

	if upgradeHash {
		_ = m.saveConfigLocked()
	}

	// Reset failed attempts counter on successful login
	delete(m.rateLimits, ip)

	t, err := generateToken()
	if err != nil {
		return "", false, false
	}
	m.sessions[t] = now.Add(SessionDuration)
	return t, true, false
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

	previousHash := m.passwordHash
	previousDefault := m.isDefault
	m.passwordHash = HashPassword(newPass, "")
	m.isDefault = false

	if err := m.saveConfigLocked(); err != nil {
		m.passwordHash = previousHash
		m.isDefault = previousDefault
		logger.Get().Errorf("auth", "Failed to save auth.json: %v", err)
		return fmt.Errorf("failed to save new password: %w", err)
	}

	clear(m.sessions)
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
		for ip, lim := range m.rateLimits {
			if now.After(lim.resetAt) {
				delete(m.rateLimits, ip)
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
