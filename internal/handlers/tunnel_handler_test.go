package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/tunnel"
)

func TestHandleTunnelStatus(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET status
	reqGet := httptest.NewRequest("GET", "/api/tunnel/status", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleTunnelStatus(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET tunnel status, got %d", rrGet.Code)
	}

	// POST wrong method
	reqPost := httptest.NewRequest("POST", "/api/tunnel/status", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleTunnelStatus(rrPost, reqPost)

	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST tunnel status, got %d", rrPost.Code)
	}
}

func TestHandleTunnelStartAndStop(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// POST start payload
	cfg := tunnel.TunnelConfig{
		Engine:  "cloudflare",
		Enabled: false,
	}
	payload, _ := json.Marshal(cfg)

	reqStart := httptest.NewRequest("POST", "/api/tunnel/start", bytes.NewBuffer(payload))
	rrStart := httptest.NewRecorder()
	handlers.HandleTunnelStart(rrStart, reqStart)

	if rrStart.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST tunnel start, got %d", rrStart.Code)
	}

	// POST stop
	reqStop := httptest.NewRequest("POST", "/api/tunnel/stop", nil)
	rrStop := httptest.NewRecorder()
	handlers.HandleTunnelStop(rrStop, reqStop)

	if rrStop.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST tunnel stop, got %d", rrStop.Code)
	}
}

func TestHandleTunnelUploadBinary_Validation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET wrong method
	reqGet := httptest.NewRequest("GET", "/api/tunnel/upload", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleTunnelUploadBinary(rrGet, reqGet)

	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET tunnel upload, got %d", rrGet.Code)
	}

	// POST missing file
	reqPost := httptest.NewRequest("POST", "/api/tunnel/upload", bytes.NewBufferString("invalid form"))
	rrPost := httptest.NewRecorder()
	handlers.HandleTunnelUploadBinary(rrPost, reqPost)

	if rrPost.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing binary file, got %d", rrPost.Code)
	}
}
