package auth_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bfr-webui-go/internal/auth"
)

func setupTestAuthManager(t *testing.T, initialPassword string) (*auth.Manager, string) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	m := auth.NewManager(initialPassword)
	return m, tempDir
}

func TestHashAndVerifyPassword(t *testing.T) {
	password := "mySecretPass123"

	// Test hashing with auto-generated salt
	hashed := auth.HashPassword(password, "")
	if hashed == "" {
		t.Fatalf("expected non-empty hash string")
	}
	if !strings.Contains(hashed, ":") {
		t.Errorf("expected hash format 'salt:hash', got: %s", hashed)
	}

	// Verify valid password
	if !auth.VerifyPassword(password, hashed) {
		t.Errorf("expected password verification to succeed for valid password")
	}

	// Verify invalid password
	if auth.VerifyPassword("wrongPassword", hashed) {
		t.Errorf("expected password verification to fail for invalid password")
	}

	// Verify empty stored hash
	if auth.VerifyPassword(password, "") {
		t.Errorf("expected password verification to fail for empty stored hash")
	}

	// Test legacy un-salted stored password fallback
	legacyStored := "plainTextPass"
	if !auth.VerifyPassword("plainTextPass", legacyStored) {
		t.Errorf("expected legacy plain text verification to succeed")
	}
}

func TestManager_DefaultPassword(t *testing.T) {
	m, _ := setupTestAuthManager(t, "")

	if !m.IsDefaultPassword() {
		t.Errorf("expected IsDefaultPassword() to be true when initialized with default password")
	}

	token, ok, rateLimited := m.Authenticate(auth.DefaultPassword, "127.0.0.1:1234")
	if !ok || rateLimited || token == "" {
		t.Errorf("expected successful authentication with default password, got ok=%v, rateLimited=%v, token=%s", ok, rateLimited, token)
	}
}

func TestManager_CustomPassword(t *testing.T) {
	customPass := "customPass123"
	m, _ := setupTestAuthManager(t, customPass)

	if m.IsDefaultPassword() {
		t.Errorf("expected IsDefaultPassword() to be false when initialized with custom password")
	}

	token, ok, rateLimited := m.Authenticate(customPass, "192.168.1.100:54321")
	if !ok || rateLimited || token == "" {
		t.Errorf("expected successful authentication with custom password")
	}

	// Attempt with wrong password
	_, okWrong, _ := m.Authenticate("wrong", "192.168.1.100:54321")
	if okWrong {
		t.Errorf("expected failed authentication for wrong password")
	}
}

func TestManager_RateLimiting(t *testing.T) {
	m, _ := setupTestAuthManager(t, "secret")
	remoteAddr := "10.0.0.1:8080"

	// 5 failed attempts
	for i := 0; i < 5; i++ {
		_, ok, rateLimited := m.Authenticate("wrongPass", remoteAddr)
		if ok {
			t.Errorf("attempt %d: expected authentication to fail", i+1)
		}
		if rateLimited {
			t.Errorf("attempt %d: should not be rate limited yet", i+1)
		}
	}

	// 6th attempt should be rate limited
	_, ok, rateLimited := m.Authenticate("secret", remoteAddr)
	if ok {
		t.Errorf("6th attempt: expected authentication to fail due to rate limit")
	}
	if !rateLimited {
		t.Errorf("6th attempt: expected rateLimited to be true")
	}

	// Different IP should not be rate limited
	_, okOther, rateLimitedOther := m.Authenticate("secret", "10.0.0.2:8080")
	if !okOther || rateLimitedOther {
		t.Errorf("different IP should authenticate successfully and not be rate limited")
	}
}

func TestManager_SessionValidationAndLogout(t *testing.T) {
	m, _ := setupTestAuthManager(t, "pass")

	token, ok, _ := m.Authenticate("pass", "127.0.0.1:1111")
	if !ok {
		t.Fatalf("failed to authenticate")
	}

	if !m.ValidateSession(token) {
		t.Errorf("expected session token to be valid")
	}

	if m.ValidateSession("invalid_token") {
		t.Errorf("expected invalid token to return false")
	}

	if m.ValidateSession("") {
		t.Errorf("expected empty token to return false")
	}

	// Test Logout
	m.Logout(token)
	if m.ValidateSession(token) {
		t.Errorf("expected session token to be invalid after logout")
	}
}

func TestManager_ChangePassword(t *testing.T) {
	m, tempDir := setupTestAuthManager(t, "")

	// Fail change password with invalid current password
	err := m.ChangePassword("wrongDefault", "newSecret123")
	if err == nil {
		t.Errorf("expected error when changing password with incorrect current password")
	}

	// Succeed change password from default
	err = m.ChangePassword(auth.DefaultPassword, "newSecret123")
	if err != nil {
		t.Fatalf("unexpected error changing password: %v", err)
	}

	if m.IsDefaultPassword() {
		t.Errorf("expected IsDefaultPassword() to be false after password change")
	}

	// Verify persistence: auth.json created
	authFile := filepath.Join(tempDir, "auth.json")
	if _, err := os.Stat(authFile); os.IsNotExist(err) {
		t.Errorf("expected auth.json file to exist at %s", authFile)
	}

	// Authenticate with new password
	_, okNew, _ := m.Authenticate("newSecret123", "127.0.0.1:9999")
	if !okNew {
		t.Errorf("expected authentication to succeed with new password")
	}

	// Authenticate with old default password should fail
	_, okOld, _ := m.Authenticate(auth.DefaultPassword, "127.0.0.1:9999")
	if okOld {
		t.Errorf("expected authentication with old default password to fail")
	}
}

func TestManager_GetTokenFromRequest(t *testing.T) {
	m, _ := setupTestAuthManager(t, "pass")

	reqWithoutCookie := httptest.NewRequest("GET", "/", nil)
	if token := m.GetTokenFromRequest(reqWithoutCookie); token != "" {
		t.Errorf("expected empty token from request without cookie, got %s", token)
	}

	reqWithCookie := httptest.NewRequest("GET", "/", nil)
	reqWithCookie.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: "test_session_token_123",
	})

	if token := m.GetTokenFromRequest(reqWithCookie); token != "test_session_token_123" {
		t.Errorf("expected token 'test_session_token_123', got %s", token)
	}
}

func TestGetManager(t *testing.T) {
	m, _ := setupTestAuthManager(t, "pass")
	if global := auth.GetManager(); global != m {
		t.Errorf("expected GetManager() to return created manager instance")
	}
}
