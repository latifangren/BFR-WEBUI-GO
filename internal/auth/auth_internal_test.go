package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractIPEdgeCases(t *testing.T) {
	if extractIP("192.168.1.1:8080") != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1")
	}
	if extractIP("192.168.1.1") != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1 when no port")
	}
	if extractIP("   ") != "" {
		t.Errorf("expected empty string")
	}
}

func TestChangePasswordValidation(t *testing.T) {
	tempDir := t.TempDir()
	authFile := filepath.Join(tempDir, "auth.json")
	mgr := NewManager(authFile)

	// Empty new password
	err := mgr.ChangePassword(DefaultPassword, "")
	if err == nil {
		t.Error("expected error for empty new password")
	}

	// Short new password (<3 chars)
	err = mgr.ChangePassword(DefaultPassword, "12")
	if err == nil {
		t.Error("expected error for short new password")
	}
}

func TestLoadConfigCorruptedAndExpiredSession(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)
	authFile := filepath.Join(tempDir, "auth.json")

	// Corrupted file -> fallback to default
	_ = os.WriteFile(authFile, []byte("invalid json {{"), 0644)
	mgr := NewManager("")
	if !mgr.IsDefaultPassword() {
		t.Error("expected default password on corrupted config file")
	}

	// Expired session validation
	mgr.mu.Lock()
	mgr.sessions["expired_token"] = time.Now().Add(-1 * time.Hour)
	mgr.mu.Unlock()

	if mgr.ValidateSession("expired_token") {
		t.Error("expected expired session to be invalid")
	}
}
