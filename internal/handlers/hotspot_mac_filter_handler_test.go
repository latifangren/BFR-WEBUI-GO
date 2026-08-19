package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/hotspot"
)

func TestHandleHotspotMACFilter(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// GET status
	reqGet := httptest.NewRequest("GET", "/api/hotspot/mac-filter", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleHotspotMACFilter(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET MAC filter, got %d", rrGet.Code)
	}

	// POST apply MAC filter
	cfg := hotspot.MACFilterConfig{
		Mode:        "blacklist",
		BlockedMACs: []string{"00:11:22:33:44:55"},
	}
	payload, _ := json.Marshal(cfg)

	reqPost := httptest.NewRequest("POST", "/api/hotspot/mac-filter", bytes.NewBuffer(payload))
	rrPost := httptest.NewRecorder()
	handlers.HandleHotspotMACFilter(rrPost, reqPost)

	if rrPost.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST MAC filter, got %d", rrPost.Code)
	}
}
