package partition_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestBackupGroupZip(t *testing.T) {
	tempDir := t.TempDir()
	devRootDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")

	_ = os.MkdirAll(devRootDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BLOCK_DEV_ROOT", devRootDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// Create fake core partitions
	_ = os.WriteFile(filepath.Join(devRootDir, "boot"), []byte("BOOT_DATA"), 0644)
	_ = os.WriteFile(filepath.Join(devRootDir, "dtbo"), []byte("DTBO_DATA"), 0644)

	res, err := partition.BackupGroupZip("boot_core")
	if err != nil {
		t.Fatalf("BackupGroupZip failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	if !strings.HasSuffix(res.Filename, ".zip") {
		t.Errorf("expected .zip extension, got %s", res.Filename)
	}

	// Verify zip contents
	r, err := zip.OpenReader(res.FilePath)
	if err != nil {
		t.Fatalf("failed to open created zip: %v", err)
	}

	foundEntries := make(map[string]bool)
	for _, f := range r.File {
		foundEntries[f.Name] = true
	}
	_ = r.Close()

	for _, expected := range []string{"boot.img", "dtbo.img", "checksums.sha256", "manifest.json"} {
		if !foundEntries[expected] {
			t.Errorf("expected entry '%s' inside zip", expected)
		}
	}

	// Test ListBackups includes zip
	backups, err := partition.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	if !backups[0].IsCompressed {
		t.Errorf("expected IsCompressed to be true for zip")
	}
	if !strings.Contains(backups[0].PartitionName, "(zip)") {
		t.Errorf("expected (zip) in partition name, got %s", backups[0].PartitionName)
	}

	// Test DeleteBackup for zip
	if err := partition.DeleteBackup(res.Filename); err != nil {
		t.Fatalf("DeleteBackup failed for zip: %v", err)
	}
	if _, err := os.Stat(res.FilePath); !os.IsNotExist(err) {
		t.Errorf("expected zip file to be deleted")
	}
}

func TestDeleteMultipleAndAllBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BACKUP_DIR", backupDir)

	f1 := "boot_20260101_120000.img"
	f2 := "dtbo_20260101_120000.img"
	f3 := "recovery_20260101_120000.img"

	_ = os.WriteFile(filepath.Join(backupDir, f1), []byte("test1"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, f1+".sha256"), []byte("sha1"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, f2), []byte("test2"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, f2+".sha256"), []byte("sha2"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, f3), []byte("test3"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, f3+".sha256"), []byte("sha3"), 0644)

	// Test DeleteMultipleBackups
	deleted, err := partition.DeleteMultipleBackups([]string{f1, f2})
	if err != nil {
		t.Fatalf("DeleteMultipleBackups failed: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}
	if _, err := os.Stat(filepath.Join(backupDir, f1)); !os.IsNotExist(err) {
		t.Errorf("expected f1 to be deleted")
	}
	if _, err := os.Stat(filepath.Join(backupDir, f2)); !os.IsNotExist(err) {
		t.Errorf("expected f2 to be deleted")
	}
	if _, err := os.Stat(filepath.Join(backupDir, f3)); err != nil {
		t.Errorf("expected f3 to still exist")
	}

	// Test DeleteAllBackups
	allDeleted, err := partition.DeleteAllBackups()
	if err != nil {
		t.Fatalf("DeleteAllBackups failed: %v", err)
	}
	if allDeleted != 1 {
		t.Errorf("expected 1 deleted in DeleteAllBackups, got %d", allDeleted)
	}
	if _, err := os.Stat(filepath.Join(backupDir, f3)); !os.IsNotExist(err) {
		t.Errorf("expected f3 to be deleted")
	}
}

func TestGetBackupDirectory(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_BACKUP_DIR", tempDir)

	dir := partition.GetBackupDirectory()
	if dir != tempDir {
		t.Errorf("expected %s, got %s", tempDir, dir)
	}

	t.Setenv("BFR_BACKUP_DIR", "")
	defDir := partition.GetBackupDirectory()
	if defDir == "" {
		t.Error("expected non-empty default backup dir")
	}
}

func TestBackupGroupValidation(t *testing.T) {
	// Unknown group
	_, err := partition.BackupGroup("invalid_unknown_group", false)
	if err == nil {
		t.Error("expected error for invalid group")
	}
}

func TestDeleteMultipleBackupsEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_BACKUP_DIR", tempDir)

	// Delete empty list
	deleted, err := partition.DeleteMultipleBackups([]string{})
	if err != nil || deleted != 0 {
		t.Errorf("expected 0 deleted for empty list, got %d, err: %v", deleted, err)
	}

	// Delete non-existent
	deleted, err = partition.DeleteMultipleBackups([]string{"non_existent_file.img"})
	if err == nil {
		t.Errorf("expected error for non-existent file, deleted: %d", deleted)
	}
}

func TestFlashPartitionValidation(t *testing.T) {
	// 1. IsFlashablePartition
	if !partition.IsFlashablePartition("boot") {
		t.Error("expected boot to be flashable")
	}
	if !partition.IsFlashablePartition("recovery") {
		t.Error("expected recovery to be flashable")
	}
	if partition.IsFlashablePartition("system") {
		t.Error("expected system to NOT be flashable")
	}

	// 2. FlashPartition invalid name
	_, err := partition.FlashPartition("invalid;name!", strings.NewReader("data"), 4, false)
	if err == nil {
		t.Error("expected error for invalid partition name")
	}

	// 3. FlashPartition prohibited partition
	_, err = partition.FlashPartition("system", strings.NewReader("data"), 4, false)
	if err == nil {
		t.Error("expected safety policy prohibition error for system")
	}

	// 4. FlashPartition device not found
	_, err = partition.FlashPartition("boot", strings.NewReader("data"), 4, false)
	if err == nil {
		t.Error("expected error when block device not found on host")
	}
}

func TestDiscoveryAndCategorization(t *testing.T) {
	// FindPartitionDevice invalid name
	_, err := partition.FindPartitionDevice("invalid;name!")
	if err == nil {
		t.Error("expected error for invalid partition device name")
	}

	// FindBlockDeviceRoot
	root, err := partition.FindBlockDeviceRoot()
	t.Logf("FindBlockDeviceRoot returned: %s, err: %v", root, err)

	// GetSlotInfo
	slot := partition.GetSlotInfo()
	t.Logf("GetSlotInfo: %+v", slot)

	// CategorizePartition
	cases := []string{"boot", "recovery", "modem", "system", "userdata", "misc_unknown"}
	for _, c := range cases {
		cat, isCrit := partition.CategorizePartition(c)
		t.Logf("CategorizePartition(%s): cat=%s, crit=%v", c, cat, isCrit)
	}
}

func TestFullBackupAndFlashWithMockBlockDevice(t *testing.T) {
	tempDir := t.TempDir()
	blockDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(blockDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BLOCK_DEV_ROOT", blockDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// Create mock partition block files
	bootA := filepath.Join(blockDir, "boot_a")
	modemA := filepath.Join(blockDir, "modem_a")
	dummyData := []byte("mock partition block device data header")
	_ = os.WriteFile(bootA, dummyData, 0644)
	_ = os.WriteFile(modemA, dummyData, 0644)

	// 1. ListPartitions
	parts, slot, err := partition.ListPartitions()
	if err != nil {
		t.Fatalf("ListPartitions failed: %v", err)
	}
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 partitions, got %d", len(parts))
	}
	t.Logf("Found %d partitions, current slot: %s", len(parts), slot.CurrentSlot)

	// 2. BackupPartition raw and compressed
	resRaw, err := partition.BackupPartition("boot_a", false)
	if err != nil {
		t.Fatalf("BackupPartition raw failed: %v", err)
	}
	if resRaw.Filename == "" {
		t.Error("expected non-empty backup filename")
	}

	resGz, err := partition.BackupPartition("boot_a", true)
	if err != nil {
		t.Fatalf("BackupPartition compressed failed: %v", err)
	}
	if resGz.Filename == "" {
		t.Error("expected non-empty gzip backup filename")
	}

	// 3. BackupGroup radio_efs
	groupResults, err := partition.BackupGroup("radio_efs", false)
	if err != nil {
		t.Fatalf("BackupGroup failed: %v", err)
	}
	t.Logf("BackupGroup created %d results", len(groupResults))

	// BackupGroupZip
	zipRes, err := partition.BackupGroupZip("radio_efs")
	if err != nil {
		t.Fatalf("BackupGroupZip failed: %v", err)
	}
	t.Logf("BackupGroupZip result: %+v", zipRes)

	// 4. FlashPartition
	newData := strings.NewReader("new flashed image kernel data")
	flashRes, err := partition.FlashPartition("boot_a", newData, 29, true)
	if err != nil {
		t.Fatalf("FlashPartition failed: %v", err)
	}
	if !flashRes.Success {
		t.Errorf("expected flash success, got %+v", flashRes)
	}
}

func TestDeleteBackupAndListEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(backupDir, 0755)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	// 1. DeleteBackup invalid names
	if err := partition.DeleteBackup(""); err == nil {
		t.Error("expected error deleting empty filename")
	}
	if err := partition.DeleteBackup("sub/file.img"); err == nil {
		t.Error("expected error deleting filename with slash")
	}
	if err := partition.DeleteBackup("../file.img"); err == nil {
		t.Error("expected error deleting filename with ..")
	}
	if err := partition.DeleteBackup("file.txt"); err == nil {
		t.Error("expected error deleting non-backup extension")
	}

	// 2. Delete valid file
	testBackup := filepath.Join(backupDir, "boot_20260101_120000.img")
	testSidecar := testBackup + ".sha256"
	_ = os.WriteFile(testBackup, []byte("data"), 0644)
	_ = os.WriteFile(testSidecar, []byte("hash"), 0644)

	err := partition.DeleteBackup("boot_20260101_120000.img")
	if err != nil {
		t.Fatalf("DeleteBackup failed: %v", err)
	}

	// 3. ListBackups
	// Create mock backup files (.img, .img.gz, .zip) and a subfolder and a non-backup file
	_ = os.WriteFile(filepath.Join(backupDir, "modem_20260102_130000.img.gz"), []byte("gzdata"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, "bfr_backup_radio_efs_20260102_140000.zip"), []byte("zipdata"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, "other.txt"), []byte("txt"), 0644)
	_ = os.MkdirAll(filepath.Join(backupDir, "sub_folder"), 0755)

	backups, err := partition.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 2 {
		t.Errorf("expected 2 backup entries, got %d", len(backups))
	}

	// 4. DeleteAllBackups
	delCount, err := partition.DeleteAllBackups()
	if err != nil || delCount != 2 {
		t.Errorf("DeleteAllBackups expected 2 deleted, got %d (err: %v)", delCount, err)
	}

	// 5. ListBackups on non-existent dir
	t.Setenv("BFR_BACKUP_DIR", filepath.Join(tempDir, "non_existent"))
	emptyBackups, err := partition.ListBackups()
	if err != nil || len(emptyBackups) != 0 {
		t.Errorf("expected empty list for non-existent backup dir, got %d (err: %v)", len(emptyBackups), err)
	}
}
