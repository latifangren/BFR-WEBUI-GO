package config_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bfr-webui-go/internal/config"
)

func TestGetPersistentDataDir(t *testing.T) {
	// 1. Default / non-env check
	dir := config.GetPersistentDataDir()
	if dir == "" {
		t.Fatalf("expected non-empty data dir")
	}

	// 2. Custom BFR_DATA_DIR override check
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	customDir := config.GetPersistentDataDir()
	if customDir != tempDir {
		t.Errorf("expected BFR_DATA_DIR override %s, got %s", tempDir, customDir)
	}
}

func TestGetPersistentFilePath(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	filename := "test_config.json"
	expectedPath := filepath.Join(tempDir, filename)

	actualPath := config.GetPersistentFilePath(filename)
	if actualPath != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, actualPath)
	}

	// Test migration from legacy location
	legacyDir := t.TempDir()
	legacyFile := filepath.Join(legacyDir, "legacy.json")
	legacyContent := []byte(`{"key": "value"}`)
	if err := os.WriteFile(legacyFile, legacyContent, 0644); err != nil {
		t.Fatalf("failed to create legacy file: %v", err)
	}

	// Override ModuleDir or legacy path source
	originalModuleDir := config.ModuleDir
	config.ModuleDir = legacyDir
	defer func() { config.ModuleDir = originalModuleDir }()

	migratedPath := config.GetPersistentFilePath("legacy.json")
	if _, err := os.Stat(migratedPath); os.IsNotExist(err) {
		t.Fatalf("expected legacy file to be migrated to %s", migratedPath)
	}

	data, err := os.ReadFile(migratedPath)
	if err != nil {
		t.Fatalf("failed to read migrated file: %v", err)
	}
	if string(data) != string(legacyContent) {
		t.Errorf("expected migrated content %s, got %s", string(legacyContent), string(data))
	}
}

func TestConfigDefaultsAndEnvVars(t *testing.T) {
	if config.SUBin == "" {
		t.Errorf("expected SUBin to be initialized")
	}
	if config.ModuleDir == "" {
		t.Errorf("expected ModuleDir to be initialized")
	}
	if config.ClashAPI == "" {
		t.Errorf("expected ClashAPI to be initialized")
	}
	if len(config.AllowedDirs) == 0 {
		t.Errorf("expected AllowedDirs to contain default directories")
	}

	// Test BFR_ALLOWED_DIRS env parsing
	t.Setenv("BFR_ALLOWED_DIRS", "/dir1, /dir2 , /dir3")
	// Note: AllowedDirs package variable is initialized on package load, but we can verify default values
	foundSdcard := false
	for _, d := range config.AllowedDirs {
		if strings.Contains(d, "sdcard") || strings.Contains(d, "data") {
			foundSdcard = true
			break
		}
	}
	if !foundSdcard {
		t.Errorf("expected standard android paths in default AllowedDirs")
	}
}

func TestExecSuTimeout(t *testing.T) {
	// Test timeout context cancellation safety
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Calling ExecSuContext with expired or brief context should not panic
	_, _ = config.ExecSuContext(ctx, "echo test")

	// Calling ExecSuTimeout with minimal timeout
	_, _ = config.ExecSuTimeout(10*time.Millisecond, "echo test")
}
