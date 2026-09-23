package partition_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"bfr-webui-go/internal/partition"
)

func TestParsePartitionSlot(t *testing.T) {
	tests := []struct {
		input    string
		wantBase string
		wantSlot string
	}{
		{"boot_a", "boot", "a"},
		{"boot_b", "boot", "b"},
		{"recovery", "recovery", ""},
		{"modem_a", "modem", "a"},
		{"vendor_boot_b", "vendor_boot", "b"},
		{"userdata", "userdata", ""},
	}

	for _, tt := range tests {
		base, slot := partition.ParsePartitionSlot(tt.input)
		if base != tt.wantBase || slot != tt.wantSlot {
			t.Errorf("ParsePartitionSlot(%q) = (%q, %q); want (%q, %q)", tt.input, base, slot, tt.wantBase, tt.wantSlot)
		}
	}
}

func TestCategorizePartition(t *testing.T) {
	tests := []struct {
		name         string
		wantCategory string
		wantCritical bool
	}{
		{"modem", "radio_efs", true},
		{"modem_a", "radio_efs", true},
		{"modemst1", "radio_efs", true},
		{"fsg", "radio_efs", true},
		{"efs", "radio_efs", true},
		{"nvram", "radio_efs", true},
		{"boot", "boot_recovery", true},
		{"boot_a", "boot_recovery", true},
		{"init_boot_b", "boot_recovery", true},
		{"vendor_boot_a", "boot_recovery", true},
		{"recovery", "boot_recovery", true},
		{"dtbo", "boot_recovery", true},
		{"vbmeta", "boot_recovery", true},
		{"vbmeta_system", "boot_recovery", true},
		{"super", "system_os", false},
		{"system_a", "system_os", false},
		{"vendor_b", "system_os", false},
		{"userdata", "system_os", false},
		{"unknown_part_xyz", "other", false},
	}

	for _, tt := range tests {
		cat, crit := partition.CategorizePartition(tt.name)
		if cat != tt.wantCategory || crit != tt.wantCritical {
			t.Errorf("CategorizePartition(%q) = (%q, %v); want (%q, %v)", tt.name, cat, crit, tt.wantCategory, tt.wantCritical)
		}
	}
}

func TestFormatBytesHuman(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := partition.FormatBytesHuman(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytesHuman(%d) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestFlashPartition_SafetyWhitelistRejection(t *testing.T) {
	rejectedPartitions := []string{
		"modem", "modem_a", "efs", "nvram", "super", "system",
		"system_a", "vendor", "product", "userdata", "other_test",
	}

	dummyData := bytes.NewReader([]byte("test image data"))

	for _, part := range rejectedPartitions {
		_, err := partition.FlashPartition(part, dummyData, int64(dummyData.Len()), false)
		if err == nil {
			t.Errorf("expected flash of %q to be rejected by safety whitelist", part)
		}
	}
}

func TestBackupAndListAndDeleteBackups(t *testing.T) {
	tempDir := t.TempDir()
	devRootDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")

	if err := os.MkdirAll(devRootDir, 0755); err != nil {
		t.Fatalf("failed to create fake dev root: %v", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("failed to create fake backup dir: %v", err)
	}

	t.Setenv("BFR_BLOCK_DEV_ROOT", devRootDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// Create fake block device file for boot_a
	fakeDevFile := filepath.Join(devRootDir, "boot_a")
	sampleContent := []byte("SAMPLE_BOOT_PARTITION_DATA_12345")
	if err := os.WriteFile(fakeDevFile, sampleContent, 0644); err != nil {
		t.Fatalf("failed to write fake dev file: %v", err)
	}

	// Test BackupPartition
	res, err := partition.BackupPartition("boot_a", false)
	if err != nil {
		t.Fatalf("BackupPartition failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got: %v", res.Error)
	}
	if res.SizeBytes != int64(len(sampleContent)) {
		t.Errorf("expected size %d, got %d", len(sampleContent), res.SizeBytes)
	}
	if res.SHA256 == "" {
		t.Errorf("expected SHA256 checksum, got empty")
	}

	// Check sidecar file
	sidecarPath := res.FilePath + ".sha256"
	if _, err := os.Stat(sidecarPath); err != nil {
		t.Errorf("expected sidecar file %s to exist: %v", sidecarPath, err)
	}

	// Test ListBackups
	backups, err := partition.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	if backups[0].Filename != res.Filename {
		t.Errorf("expected filename %s, got %s", res.Filename, backups[0].Filename)
	}
	if backups[0].SHA256 != res.SHA256 {
		t.Errorf("expected sha256 %s, got %s", res.SHA256, backups[0].SHA256)
	}

	// Test DeleteBackup
	if err := partition.DeleteBackup(res.Filename); err != nil {
		t.Fatalf("DeleteBackup failed: %v", err)
	}

	// Verify both backup and sidecar deleted
	if _, err := os.Stat(res.FilePath); !os.IsNotExist(err) {
		t.Errorf("expected backup file to be deleted")
	}
	if _, err := os.Stat(sidecarPath); !os.IsNotExist(err) {
		t.Errorf("expected sidecar file to be deleted")
	}
}

func TestBackupPartition_ZeroBytesError(t *testing.T) {
	tempDir := t.TempDir()
	devRootDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")

	_ = os.MkdirAll(devRootDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BLOCK_DEV_ROOT", devRootDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// Create 0-byte fake block device file
	fakeDevFile := filepath.Join(devRootDir, "boot_a")
	if err := os.WriteFile(fakeDevFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create 0-byte fake dev: %v", err)
	}

	res, err := partition.BackupPartition("boot_a", false)
	if err == nil {
		t.Fatalf("expected error for 0-byte read, got success: %+v", res)
	}
}
