package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	tempDir := t.TempDir()
	targetFile := filepath.Join(tempDir, "atomic_test.txt")

	data := []byte("hello atomic write")
	err := WriteFileAtomic(targetFile, data, 0644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	read, err := os.ReadFile(targetFile)
	if err != nil || string(read) != string(data) {
		t.Errorf("content mismatch: %s (err: %v)", string(read), err)
	}

	// Overwrite atomically
	data2 := []byte("overwritten atomic content")
	err = WriteFileAtomic(targetFile, data2, 0644)
	if err != nil {
		t.Fatalf("WriteFileAtomic overwrite failed: %v", err)
	}

	read2, _ := os.ReadFile(targetFile)
	if string(read2) != string(data2) {
		t.Errorf("content mismatch after overwrite: %s", string(read2))
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_KEY_EXISTS", "custom_val")
	if envOrDefault("TEST_KEY_EXISTS", "def") != "custom_val" {
		t.Error("expected custom_val when env exists")
	}

	if envOrDefault("TEST_KEY_NOT_EXISTS_XYZ", "def") != "def" {
		t.Error("expected default value when env does not exist")
	}
}

func TestWriteFileAtomicInvalidDir(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "existing_file")
	_ = os.WriteFile(filePath, []byte("data"), 0644)

	// Trying to create a file inside an existing regular file causes MkdirAll to fail
	impossiblePath := filepath.Join(filePath, "sub", "target.txt")
	err := WriteFileAtomic(impossiblePath, []byte("data"), 0644)
	if err == nil {
		t.Error("expected error creating file in impossible directory")
	}
}

func TestGetPersistentDataDirUnset(t *testing.T) {
	// Unset BFR_DATA_DIR
	t.Setenv("BFR_DATA_DIR", "")
	dir := GetPersistentDataDir()
	if dir == "" {
		t.Error("expected non-empty default data dir")
	}
}

func TestGetPersistentDataDirModuleDirBranches(t *testing.T) {
	t.Setenv("BFR_DATA_DIR", "")

	// 1. ModuleDir set
	origModule := ModuleDir
	defer func() { ModuleDir = origModule }()

	ModuleDir = t.TempDir()
	d1 := GetPersistentDataDir()
	if d1 == "" {
		t.Error("expected non-empty data dir when ModuleDir set")
	}

	// 2. ModuleDir empty
	ModuleDir = ""
	d2 := GetPersistentDataDir()
	if d2 == "" {
		t.Errorf("expected non-empty data dir when ModuleDir empty, got %s", d2)
	}
}

func TestGetPersistentFilePathLegacyMigration(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", filepath.Join(tempDir, "data"))

	origModule := ModuleDir
	defer func() { ModuleDir = origModule }()
	ModuleDir = filepath.Join(tempDir, "module")
	_ = os.MkdirAll(ModuleDir, 0755)

	// Create legacy file in ModuleDir
	legacyFile := filepath.Join(ModuleDir, "legacy_cfg.json")
	_ = os.WriteFile(legacyFile, []byte(`{"legacy":true}`), 0644)

	// Calling GetPersistentFilePath should migrate it to targetPath
	targetPath := GetPersistentFilePath("legacy_cfg.json")
	data, err := os.ReadFile(targetPath)
	if err != nil || string(data) != `{"legacy":true}` {
		t.Errorf("legacy migration failed: %s, err: %v", string(data), err)
	}
}
