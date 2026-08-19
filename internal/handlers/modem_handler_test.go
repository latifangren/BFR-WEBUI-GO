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
