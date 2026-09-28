package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/nas"
)

func TestHandleNASStatus(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET status
	reqGet := httptest.NewRequest("GET", "/api/nas/status", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleNASStatus(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET NAS status, got %d", rrGet.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rrGet.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	st, ok := resp["status"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected status map in response, got %v", resp["status"])
	}
	if _, ok := st["ips"]; !ok {
		t.Errorf("expected 'ips' field in NAS status")
	}
	if _, ok := st["primary_ip"]; !ok {
		t.Errorf("expected 'primary_ip' field in NAS status")
	}
	if _, ok := st["mount_urls"]; !ok {
		t.Errorf("expected 'mount_urls' field in NAS status")
	}
	if _, ok := st["port"]; !ok {
		t.Errorf("expected 'port' field in NAS status")
	}

	// POST wrong method
	reqPost := httptest.NewRequest("POST", "/api/nas/status", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleNASStatus(rrPost, reqPost)

	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST NAS status, got %d", rrPost.Code)
	}
}

func TestHandleNASStartAndStop(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// POST start payload
	cfg := nas.NASConfig{
		Enabled:   true,
		SharePath: tempDir,
		Port:      19088,
	}
	payload, _ := json.Marshal(cfg)

	reqStart := httptest.NewRequest("POST", "/api/nas/start", bytes.NewBuffer(payload))
	rrStart := httptest.NewRecorder()
	handlers.HandleNASStart(rrStart, reqStart)

	if rrStart.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST NAS start, got %d", rrStart.Code)
	}

	// POST stop
	reqStop := httptest.NewRequest("POST", "/api/nas/stop", nil)
	rrStop := httptest.NewRecorder()
	handlers.HandleNASStop(rrStop, reqStop)

	if rrStop.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST NAS stop, got %d", rrStop.Code)
	}
}
