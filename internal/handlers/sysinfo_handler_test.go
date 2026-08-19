package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleSysinfo(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/sysinfo", nil)
	rr := httptest.NewRecorder()

	handlers.HandleSysinfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if _, ok := resp["server_time"]; !ok {
		t.Errorf("expected server_time key in sysinfo response")
	}
}

func TestHandleVnstatStats(t *testing.T) {
	// 1. Wrong method
	reqPost := httptest.NewRequest("POST", "/api/vnstat/stats", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleVnstatStats(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST, got %d", rrPost.Code)
	}

	// 2. GET success
	reqGet := httptest.NewRequest("GET", "/api/vnstat/stats", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleVnstatStats(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET, got %d", rrGet.Code)
	}
}

func TestHandleVnstatReset(t *testing.T) {
	// 1. Wrong method
	reqGet := httptest.NewRequest("GET", "/api/vnstat/reset", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleVnstatReset(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rrGet.Code)
	}

	// 2. POST success
	reqPost := httptest.NewRequest("POST", "/api/vnstat/reset", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleVnstatReset(rrPost, reqPost)
	if rrPost.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST, got %d", rrPost.Code)
	}
}
