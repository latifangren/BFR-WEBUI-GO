package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleBackupExportAndImport(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// 1. Export GET
	reqExport := httptest.NewRequest("GET", "/api/backup/export", nil)
	rrExport := httptest.NewRecorder()
	handlers.HandleBackupExport(rrExport, reqExport)

	if rrExport.Code != http.StatusOK {
		t.Errorf("expected status 200 for backup export, got %d", rrExport.Code)
	}

	// 2. Import invalid payload POST
	reqImportBad := httptest.NewRequest("POST", "/api/backup/import", bytes.NewBufferString(""))
	rrImportBad := httptest.NewRecorder()
	handlers.HandleBackupImport(rrImportBad, reqImportBad)

	if rrImportBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty backup import, got %d", rrImportBad.Code)
	}
}

func TestHandleCloudBackupConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	reqGet := httptest.NewRequest("GET", "/api/backup/cloud/config", nil)
	rrGet := httptest.NewRecorder()

	handlers.HandleCloudBackupConfig(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET cloud backup config, got %d", rrGet.Code)
	}
}

func TestHandleProxyStatusAndControl(t *testing.T) {
	// Status GET
	reqStatus := httptest.NewRequest("GET", "/api/proxy/status", nil)
	rrStatus := httptest.NewRecorder()
	handlers.HandleProxyStatus(rrStatus, reqStatus)

	if rrStatus.Code != http.StatusOK {
		t.Errorf("expected status 200 for proxy status, got %d", rrStatus.Code)
	}

	// Control invalid mode POST
	payloadBadMode, _ := json.Marshal(map[string]string{"mode": "invalid_mode"})
	reqBadMode := httptest.NewRequest("POST", "/api/proxy/control", bytes.NewBuffer(payloadBadMode))
	rrBadMode := httptest.NewRecorder()

	handlers.HandleProxyControl(rrBadMode, reqBadMode)
	if rrBadMode.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid proxy mode, got %d", rrBadMode.Code)
	}
}
