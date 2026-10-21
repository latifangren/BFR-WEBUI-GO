package efs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEfsBackupWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_EFS_BACKUP_DIR", tempDir)

	testPath := "/nv/item_files/modem/lte/rrc/efs/max_num_mimo_lyr"
	testData := []byte{0x04, 0x00, 0x00, 0x00}

	// 1. Save backup
	entry, err := SaveEfsBackup(testPath, testData)
	if err != nil {
		t.Fatalf("SaveEfsBackup failed: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("expected non-empty backup ID")
	}
	if entry.OriginalPath != testPath {
		t.Fatalf("expected path %s, got %s", testPath, entry.OriginalPath)
	}
	if entry.Size != int64(len(testData)) {
		t.Fatalf("expected size %d, got %d", len(testData), entry.Size)
	}

	// Verify file on disk
	expectedFile := filepath.Join(tempDir, entry.Filename)
	diskData, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read written backup file: %v", err)
	}
	if !bytes.Equal(diskData, testData) {
		t.Fatalf("disk data %v does not match test data %v", diskData, testData)
	}

	// 2. List backups
	list, err := ListEfsBackups("")
	if err != nil {
		t.Fatalf("ListEfsBackups failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 backup entry, got %d", len(list))
	}
	if list[0].ID != entry.ID {
		t.Fatalf("expected entry ID %s, got %s", entry.ID, list[0].ID)
	}

	// Filter by path
	filtered, err := ListEfsBackups(testPath)
	if err != nil {
		t.Fatalf("ListEfsBackups with filter failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered entry, got %d", len(filtered))
	}

	nonExistent, err := ListEfsBackups("/non/existent/path")
	if err != nil {
		t.Fatalf("ListEfsBackups non-existent filter failed: %v", err)
	}
	if len(nonExistent) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(nonExistent))
	}

	// 3. Get backup
	retrieved, data, err := GetEfsBackup(entry.ID)
	if err != nil {
		t.Fatalf("GetEfsBackup failed: %v", err)
	}
	if retrieved.ID != entry.ID {
		t.Fatalf("expected retrieved ID %s, got %s", entry.ID, retrieved.ID)
	}
	if !bytes.Equal(data, testData) {
		t.Fatalf("retrieved data does not match test data")
	}

	// 4. Delete backup
	if err := DeleteEfsBackup(entry.ID); err != nil {
		t.Fatalf("DeleteEfsBackup failed: %v", err)
	}

	// Ensure deleted
	afterDel, err := ListEfsBackups("")
	if err != nil {
		t.Fatalf("ListEfsBackups after delete failed: %v", err)
	}
	if len(afterDel) != 0 {
		t.Fatalf("expected 0 backups after deletion, got %d", len(afterDel))
	}

	if _, err := os.Stat(expectedFile); !os.IsNotExist(err) {
		t.Fatalf("expected backup file on disk to be removed")
	}
}
