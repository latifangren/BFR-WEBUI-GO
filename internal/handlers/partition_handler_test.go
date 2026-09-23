package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleListPartitions(t *testing.T) {
	// Wrong method
	reqPost := httptest.NewRequest("POST", "/api/partitions", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleListPartitions(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST on /api/partitions, got %d", rrPost.Code)
	}

	// Valid GET
	reqGet := httptest.NewRequest("GET", "/api/partitions", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleListPartitions(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected 200 for GET on /api/partitions, got %d", rrGet.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rrGet.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}
	if _, ok := resp["partitions"]; !ok {
		t.Errorf("expected 'partitions' field in response")
	}
	if _, ok := resp["slot_info"]; !ok {
		t.Errorf("expected 'slot_info' field in response")
	}
}

func TestHandleBackupPartition_Validation(t *testing.T) {
	// Wrong method
	reqGet := httptest.NewRequest("GET", "/api/partitions/backup", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleBackupPartition(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on /api/partitions/backup, got %d", rrGet.Code)
	}

	// Empty payload
	reqEmpty := httptest.NewRequest("POST", "/api/partitions/backup", bytes.NewBufferString("{}"))
	rrEmpty := httptest.NewRecorder()
	handlers.HandleBackupPartition(rrEmpty, reqEmpty)
	if rrEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty partition name, got %d", rrEmpty.Code)
	}

	// Non-existent partition device
	payload, _ := json.Marshal(map[string]interface{}{
		"partition": "nonexistent_part_12345",
		"compress":  false,
	})
	reqBad := httptest.NewRequest("POST", "/api/partitions/backup", bytes.NewBuffer(payload))
	rrBad := httptest.NewRecorder()
	handlers.HandleBackupPartition(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for nonexistent partition, got %d", rrBad.Code)
	}
}

func TestHandleBackupGroup_Validation(t *testing.T) {
	// Wrong method
	reqGet := httptest.NewRequest("GET", "/api/partitions/backup-group", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleBackupGroup(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on /api/partitions/backup-group, got %d", rrGet.Code)
	}

	// Invalid group
	payload, _ := json.Marshal(map[string]interface{}{
		"group": "invalid_group_xyz",
	})
	reqBad := httptest.NewRequest("POST", "/api/partitions/backup-group", bytes.NewBuffer(payload))
	rrBad := httptest.NewRecorder()
	handlers.HandleBackupGroup(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid group, got %d", rrBad.Code)
	}
}

func TestHandleBackupGroup_Success(t *testing.T) {
	tempDir := t.TempDir()
	devRootDir := filepath.Join(tempDir, "by-name")
	backupDir := filepath.Join(tempDir, "backups")

	_ = os.MkdirAll(devRootDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BLOCK_DEV_ROOT", devRootDir)
	t.Setenv("BFR_BACKUP_DIR", backupDir)

	_ = os.WriteFile(filepath.Join(devRootDir, "boot"), []byte("BOOT_BYTES"), 0644)

	payload, _ := json.Marshal(map[string]interface{}{
		"group": "boot_core",
	})
	req := httptest.NewRequest("POST", "/api/partitions/backup-group", bytes.NewBuffer(payload))
	rr := httptest.NewRecorder()
	handlers.HandleBackupGroup(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid backup-group, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success: true, got %v", resp["success"])
	}
	if resp["group"] != "boot_core" {
		t.Errorf("expected group: 'boot_core', got %v", resp["group"])
	}
	if resp["filename"] == nil || resp["filename"] == "" {
		t.Errorf("expected filename in response")
	}
	if resp["sha256"] == nil || resp["sha256"] == "" {
		t.Errorf("expected sha256 in response")
	}
}

func TestHandleListBackups(t *testing.T) {
	// Wrong method
	reqPost := httptest.NewRequest("POST", "/api/partitions/backups", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleListBackups(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST on /api/partitions/backups, got %d", rrPost.Code)
	}

	// Valid GET
	reqGet := httptest.NewRequest("GET", "/api/partitions/backups", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleListBackups(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected 200 for GET on /api/partitions/backups, got %d", rrGet.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rrGet.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}
	if _, ok := resp["backups"]; !ok {
		t.Errorf("expected 'backups' field in response")
	}
}

func TestHandleDownloadBackup_Validation(t *testing.T) {
	// Wrong method
	reqPost := httptest.NewRequest("POST", "/api/partitions/download", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleDownloadBackup(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST on /api/partitions/download, got %d", rrPost.Code)
	}

	// Path traversal attempt
	reqBad := httptest.NewRequest("GET", "/api/partitions/download?filename=../../etc/passwd", nil)
	rrBad := httptest.NewRecorder()
	handlers.HandleDownloadBackup(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal filename, got %d", rrBad.Code)
	}

	// Non-existent file
	reqNotFound := httptest.NewRequest("GET", "/api/partitions/download?filename=nonexistent_file_12345.img", nil)
	rrNotFound := httptest.NewRecorder()
	handlers.HandleDownloadBackup(rrNotFound, reqNotFound)
	if rrNotFound.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent backup file, got %d", rrNotFound.Code)
	}
}

func TestHandleDeleteBackup_Validation(t *testing.T) {
	// Wrong method
	reqGet := httptest.NewRequest("GET", "/api/partitions/backup/delete", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleDeleteBackup(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on /api/partitions/backup/delete, got %d", rrGet.Code)
	}

	// Empty filename
	reqEmpty := httptest.NewRequest("POST", "/api/partitions/backup/delete", bytes.NewBufferString("{}"))
	rrEmpty := httptest.NewRecorder()
	handlers.HandleDeleteBackup(rrEmpty, reqEmpty)
	if rrEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty filename, got %d", rrEmpty.Code)
	}

	// Non-img extension
	payload, _ := json.Marshal(map[string]string{"filename": "test.txt"})
	reqInvalid := httptest.NewRequest("POST", "/api/partitions/backup/delete", bytes.NewBuffer(payload))
	rrInvalid := httptest.NewRecorder()
	handlers.HandleDeleteBackup(rrInvalid, reqInvalid)
	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-image extension, got %d", rrInvalid.Code)
	}
}

func TestHandleDeleteBackup_BatchAndAll(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	_ = os.MkdirAll(backupDir, 0755)

	t.Setenv("BFR_BACKUP_DIR", backupDir)

	f1 := "boot_20260101_120000.img"
	f2 := "dtbo_20260101_120000.img"
	f3 := "recovery_20260101_120000.img"

	_ = os.WriteFile(filepath.Join(backupDir, f1), []byte("test1"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, f2), []byte("test2"), 0644)
	_ = os.WriteFile(filepath.Join(backupDir, f3), []byte("test3"), 0644)

	// Batch delete: f1, f2
	payloadBatch, _ := json.Marshal(map[string]interface{}{
		"filenames": []string{f1, f2},
	})
	reqBatch := httptest.NewRequest("POST", "/api/partitions/backup/delete", bytes.NewBuffer(payloadBatch))
	rrBatch := httptest.NewRecorder()
	handlers.HandleDeleteBackup(rrBatch, reqBatch)

	if rrBatch.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch delete, got %d: %s", rrBatch.Code, rrBatch.Body.String())
	}
	var respBatch map[string]interface{}
	_ = json.Unmarshal(rrBatch.Body.Bytes(), &respBatch)
	if respBatch["deleted"] != float64(2) {
		t.Errorf("expected deleted: 2, got %v", respBatch["deleted"])
	}

	// Delete all: remaining f3
	payloadAll, _ := json.Marshal(map[string]interface{}{
		"all": true,
	})
	reqAll := httptest.NewRequest("POST", "/api/partitions/backup/delete", bytes.NewBuffer(payloadAll))
	rrAll := httptest.NewRecorder()
	handlers.HandleDeleteBackup(rrAll, reqAll)

	if rrAll.Code != http.StatusOK {
		t.Fatalf("expected 200 for delete all, got %d: %s", rrAll.Code, rrAll.Body.String())
	}
	var respAll map[string]interface{}
	_ = json.Unmarshal(rrAll.Body.Bytes(), &respAll)
	if respAll["deleted"] != float64(1) {
		t.Errorf("expected deleted: 1, got %v", respAll["deleted"])
	}
}

func TestHandleFlashPartition_SafetyRejection(t *testing.T) {
	// Method check
	reqGet := httptest.NewRequest("GET", "/api/partitions/flash", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleFlashPartition(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on /api/partitions/flash, got %d", rrGet.Code)
	}

	// Missing safety_ack
	bodyNoAck := &bytes.Buffer{}
	writerNoAck := multipart.NewWriter(bodyNoAck)
	_ = writerNoAck.WriteField("partition", "boot")
	_ = writerNoAck.Close()

	reqNoAck := httptest.NewRequest("POST", "/api/partitions/flash", bodyNoAck)
	reqNoAck.Header.Set("Content-Type", writerNoAck.FormDataContentType())
	rrNoAck := httptest.NewRecorder()
	handlers.HandleFlashPartition(rrNoAck, reqNoAck)
	if rrNoAck.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing safety_ack, got %d", rrNoAck.Code)
	}

	// Safety Whitelist Rejection: attempt to flash 'super' or 'modem'
	bodyRejected := &bytes.Buffer{}
	writerRejected := multipart.NewWriter(bodyRejected)
	_ = writerRejected.WriteField("partition", "super")
	_ = writerRejected.WriteField("safety_ack", "I_UNDERSTAND_THE_RISKS")
	part, _ := writerRejected.CreateFormFile("file", "super.img")
	_, _ = part.Write([]byte("fake image data"))
	_ = writerRejected.Close()

	reqRejected := httptest.NewRequest("POST", "/api/partitions/flash", bodyRejected)
	reqRejected.Header.Set("Content-Type", writerRejected.FormDataContentType())
	rrRejected := httptest.NewRecorder()
	handlers.HandleFlashPartition(rrRejected, reqRejected)
	if rrRejected.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when attempting to flash non-whitelisted partition 'super', got %d", rrRejected.Code)
	}

	// Safety Whitelist Rejection: attempt to flash 'modem'
	bodyModem := &bytes.Buffer{}
	writerModem := multipart.NewWriter(bodyModem)
	_ = writerModem.WriteField("partition", "modem_a")
	_ = writerModem.WriteField("safety_ack", "I_UNDERSTAND_THE_RISKS")
	partM, _ := writerModem.CreateFormFile("file", "modem.img")
	_, _ = partM.Write([]byte("fake modem data"))
	_ = writerModem.Close()

	reqModem := httptest.NewRequest("POST", "/api/partitions/flash", bodyModem)
	reqModem.Header.Set("Content-Type", writerModem.FormDataContentType())
	rrModem := httptest.NewRecorder()
	handlers.HandleFlashPartition(rrModem, reqModem)
	if rrModem.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when attempting to flash non-whitelisted partition 'modem_a', got %d", rrModem.Code)
	}
}
