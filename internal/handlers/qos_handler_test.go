package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/qos"
)

func TestHandleQoSStatus(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET status success
	reqGet := httptest.NewRequest("GET", "/api/qos/status", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleQoSStatus(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET QoS status, got %d", rrGet.Code)
	}

	// POST wrong method
	reqPost := httptest.NewRequest("POST", "/api/qos/status", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleQoSStatus(rrPost, reqPost)

	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST, got %d", rrPost.Code)
	}
}

func TestHandleQoSApplyAndClear(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// 1. Apply QoS payload
	cfg := qos.QoSConfig{
		Enabled:        true,
		Engine:         "iptables",
		GlobalDownload: 100,
		GlobalUpload:   50,
	}
	payload, _ := json.Marshal(cfg)

	reqApply := httptest.NewRequest("POST", "/api/qos/apply", bytes.NewBuffer(payload))
	rrApply := httptest.NewRecorder()
	handlers.HandleQoSApply(rrApply, reqApply)

	if rrApply.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST QoS apply, got %d", rrApply.Code)
	}

	// 2. Clear QoS
	reqClear := httptest.NewRequest("POST", "/api/qos/clear", nil)
	rrClear := httptest.NewRecorder()
	handlers.HandleQoSClear(rrClear, reqClear)

	if rrClear.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST QoS clear, got %d", rrClear.Code)
	}
}
