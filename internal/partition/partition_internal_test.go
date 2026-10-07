package partition

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSuReadCloserAndOpenDevice(t *testing.T) {
	tempDir := t.TempDir()
	devFile := filepath.Join(tempDir, "mock_block")
	_ = os.WriteFile(devFile, []byte("mock block data"), 0644)

	// 1. openBlockDeviceReader on existing file
	reader, err := openBlockDeviceReader(devFile)
	if err != nil {
		t.Fatalf("openBlockDeviceReader failed: %v", err)
	}
	defer reader.Close()

	buf := make([]byte, 4)
	n, err := reader.Read(buf)
	if err != nil || n != 4 {
		t.Errorf("read failed: %d, err: %v", n, err)
	}

	// 2. suReadCloser methods
	cmd := exec.Command("cmd.exe", "/c", "echo test")
	s := &suReadCloser{
		stdout: io.NopCloser(strings.NewReader("pipe data")),
		cmd:    cmd,
	}
	pBuf := make([]byte, 4)
	_, _ = s.Read(pBuf)
	_ = s.Close()
}

func TestBackupGroupAllTargets(t *testing.T) {
	tempDir := t.TempDir()
	blockDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(blockDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BLOCK_DEV_ROOT", blockDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// Create mock files for various partition groups
	_ = os.WriteFile(filepath.Join(blockDir, "boot_a"), []byte("boot"), 0644)
	_ = os.WriteFile(filepath.Join(blockDir, "modem_a"), []byte("modem"), 0644)
	_ = os.WriteFile(filepath.Join(blockDir, "fsg"), []byte("fsg"), 0644)

	// critical_boot group
	_, _ = BackupGroup("critical_boot", false)

	// full_modem group
	_, _ = BackupGroup("full_modem", false)
}

func TestBackupGroupZipFullSuccess(t *testing.T) {
	tempDir := t.TempDir()
	blockDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(blockDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BLOCK_DEV_ROOT", blockDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// Create boot_a mock device file with valid size
	bootA := filepath.Join(blockDir, "boot_a")
	_ = os.WriteFile(bootA, []byte("boot partition mock binary image data 1234567890"), 0644)

	res, err := BackupGroupZip("boot_core")
	if err != nil {
		t.Fatalf("BackupGroupZip failed: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success true, got false")
	}
	t.Logf("Created group zip: %s, sha256: %s", res.Filename, res.SHA256)
}

func TestFlashAndBackupEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	blockDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(blockDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BLOCK_DEV_ROOT", blockDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// Create an empty partition file (0 bytes)
	emptyPart := filepath.Join(blockDir, "empty_part_a")
	_ = os.WriteFile(emptyPart, []byte{}, 0644)

	// BackupPartition with 0 bytes partition should fail
	_, err := BackupPartition("empty_part_a", false)
	if err == nil {
		t.Error("expected error backing up 0-byte partition")
	}

	// FlashPartition with size mismatch
	bootA := filepath.Join(blockDir, "boot_a")
	_ = os.WriteFile(bootA, []byte("existing boot"), 0644)

	// Expected size 100, provided 4
	_, err = FlashPartition("boot_a", strings.NewReader("data"), 100, false)
	if err == nil {
		t.Error("expected error for upload size mismatch")
	}
}

func TestSlotInfoMockAndBlockDevSize(t *testing.T) {
	tempDir := t.TempDir()
	blockDir := filepath.Join(tempDir, "by-name")
	_ = os.MkdirAll(blockDir, 0755)
	_ = os.WriteFile(filepath.Join(blockDir, "boot_a"), []byte("data"), 0644)

	t.Setenv("BFR_BLOCK_DEV_ROOT", blockDir)
	t.Setenv("BFR_MOCK_SLOT_SUFFIX", "_a")

	slot := GetSlotInfo()
	if !slot.IsABDevice || slot.CurrentSlot != "a" {
		t.Errorf("slot info mismatch: %+v", slot)
	}

	// getBlockDevSizeBytes empty paths
	sz0 := getBlockDevSizeBytes()
	if sz0 != 0 {
		t.Errorf("expected 0 size for empty paths, got %d", sz0)
	}

	// getBlockDevSizeBytes regular file
	szFile := getBlockDevSizeBytes(filepath.Join(blockDir, "boot_a"))
	t.Logf("getBlockDevSizeBytes on file: %d", szFile)
}
