package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/modem"
)

func TestHandleModemSignal(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET signal info
	reqGet := httptest.NewRequest("GET", "/api/modem/signal", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModemSignal(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET modem signal, got %d", rrGet.Code)
	}

	// POST wrong method
	reqPost := httptest.NewRequest("POST", "/api/modem/signal", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleModemSignal(rrPost, reqPost)

	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST modem signal, got %d", rrPost.Code)
	}
}

func TestHandleModemBands(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// 1. GET bands config
	reqGet := httptest.NewRequest("GET", "/api/modem/bands", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModemBands(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET modem bands, got %d", rrGet.Code)
	}

	// 2. POST apply bands config
	cfg := modem.BandConfig{
		Engine:       "universal",
		PreferredRAT: "4g_only",
		LTEBands:     []int{1, 3, 8},
	}
	payload, _ := json.Marshal(cfg)
	reqPost := httptest.NewRequest("POST", "/api/modem/bands", bytes.NewBuffer(payload))
	rrPost := httptest.NewRecorder()
	handlers.HandleModemBands(rrPost, reqPost)

	if rrPost.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST modem bands, got %d", rrPost.Code)
	}
}

func TestHandleModemAT_Validation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET wrong method
	reqGet := httptest.NewRequest("GET", "/api/modem/at", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModemAT(rrGet, reqGet)

	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET modem at, got %d", rrGet.Code)
	}

	// POST empty AT command payload
	payload, _ := json.Marshal(map[string]string{"command": ""})
	reqBad := httptest.NewRequest("POST", "/api/modem/at", bytes.NewBuffer(payload))
	rrBad := httptest.NewRecorder()
	handlers.HandleModemAT(rrBad, reqBad)

	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty AT command, got %d", rrBad.Code)
	}

	// POST valid AT command payload
	payloadOk, _ := json.Marshal(map[string]string{"command": "AT"})
	reqOk := httptest.NewRequest("POST", "/api/modem/at", bytes.NewBuffer(payloadOk))
	rrOk := httptest.NewRecorder()
	handlers.HandleModemAT(rrOk, reqOk)

	if rrOk.Code != http.StatusOK {
		t.Errorf("expected status 200 for valid AT command execution, got %d", rrOk.Code)
	}
}

func TestHandleModemReset(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	reqPost := httptest.NewRequest("POST", "/api/modem/reset", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleModemReset(rrPost, reqPost)

	if rrPost.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST modem reset, got %d", rrPost.Code)
	}
}

func TestHandleModemCellLock(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET cell lock status
	reqGet := httptest.NewRequest("GET", "/api/modem/cell_lock?slot=0", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModemCellLock(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET cell lock, got %d", rrGet.Code)
	}

	// POST bad payload
	reqBad := httptest.NewRequest("POST", "/api/modem/cell_lock", bytes.NewBufferString("{bad-json}"))
	rrBad := httptest.NewRecorder()
	handlers.HandleModemCellLock(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for bad cell lock payload, got %d", rrBad.Code)
	}

	// POST invalid params
	payloadInvalid, _ := json.Marshal(map[string]interface{}{"slot": 0, "earfcn": 0, "pci": 9999})
	reqInvalid := httptest.NewRequest("POST", "/api/modem/cell_lock", bytes.NewBuffer(payloadInvalid))
	rrInvalid := httptest.NewRecorder()
	handlers.HandleModemCellLock(rrInvalid, reqInvalid)
	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for out-of-range cell lock params, got %d", rrInvalid.Code)
	}
}

func TestHandleModemCellLockClear(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// POST clear
	reqPost := httptest.NewRequest("POST", "/api/modem/cell_lock/clear", bytes.NewBufferString(`{"slot": 0}`))
	rrPost := httptest.NewRecorder()
	handlers.HandleModemCellLockClear(rrPost, reqPost)

	// On host stub, it returns Bad Request because QRTR not supported
	if rrPost.Code != http.StatusBadRequest && rrPost.Code != http.StatusOK {
		t.Errorf("unexpected status for cell lock clear: %d", rrPost.Code)
	}
}

func TestHandleModemPlmnLock(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// POST bad json
	reqBad := httptest.NewRequest("POST", "/api/modem/plmn_lock", bytes.NewBufferString("invalid"))
	rrBad := httptest.NewRecorder()
	handlers.HandleModemPlmnLock(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for bad plmn json, got %d", rrBad.Code)
	}
}

func TestHandleModemHardwareBands(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET hardware bands
	reqGet := httptest.NewRequest("GET", "/api/modem/hardware_bands?slot=0", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModemHardwareBands(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET hardware bands, got %d", rrGet.Code)
	}
}

func TestHandleModemCellScan(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET cell scan
	reqGet := httptest.NewRequest("GET", "/api/modem/cell_scan?slot=0", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModemCellScan(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET cell scan, got %d", rrGet.Code)
	}
}
