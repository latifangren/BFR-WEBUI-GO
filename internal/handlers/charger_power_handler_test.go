package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/handlers"
)

func TestHandleChargerConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET status
	reqGet := httptest.NewRequest("GET", "/api/charger/config", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleChargerConfig(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET charger config, got %d", rrGet.Code)
	}

	// POST wrong method
	reqPost := httptest.NewRequest("POST", "/api/charger/config", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleChargerConfig(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST, got %d", rrPost.Code)
	}
}

func TestHandleChargerToggle(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// POST valid payload
	payload, _ := json.Marshal(charger.Config{
		Enabled:      true,
		StartPercent: 70,
		StopPercent:  80,
	})

	reqPost := httptest.NewRequest("POST", "/api/charger/toggle", bytes.NewBuffer(payload))
	rrPost := httptest.NewRecorder()
	handlers.HandleChargerToggle(rrPost, reqPost)
	if rrPost.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST charger toggle, got %d", rrPost.Code)
	}

	// GET wrong method
	reqGet := httptest.NewRequest("GET", "/api/charger/toggle", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleChargerToggle(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rrGet.Code)
	}
}

func TestHandlePower_Validation(t *testing.T) {
	// GET wrong method
	reqGet := httptest.NewRequest("GET", "/api/power/action", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandlePower(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rrGet.Code)
	}

	// Invalid action payload
	payload, _ := json.Marshal(map[string]string{"action": "invalid_action"})
	reqPost := httptest.NewRequest("POST", "/api/power/action", bytes.NewBuffer(payload))
	rrPost := httptest.NewRecorder()
	handlers.HandlePower(rrPost, reqPost)
	if rrPost.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid power action, got %d", rrPost.Code)
	}
}
