package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleNetworkTweaks_Get(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/network/tweaks", nil)
	rr := httptest.NewRecorder()

	handlers.HandleNetworkTweaks(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse network tweaks GET response: %v", err)
	}

	if _, ok := resp["preset_dns"]; !ok {
		t.Errorf("expected preset_dns key in response")
	}
}

func TestHandleNetworkTweaks_PostInvalidAction(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/network/tweaks?action=invalid", bytes.NewBufferString("{}"))
	rr := httptest.NewRecorder()

	handlers.HandleNetworkTweaks(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid action, got %d", rr.Code)
	}
}

func TestHandlePing_Validation(t *testing.T) {
	// Wrong HTTP method
	reqGet := httptest.NewRequest("GET", "/api/ping", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandlePing(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rrGet.Code)
	}

	// Empty host
	payload, _ := json.Marshal(map[string]interface{}{"host": "", "count": 2})
	reqEmpty := httptest.NewRequest("POST", "/api/ping", bytes.NewBuffer(payload))
	rrEmpty := httptest.NewRecorder()
	handlers.HandlePing(rrEmpty, reqEmpty)
	if rrEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty host, got %d", rrEmpty.Code)
	}
}

func TestHandleDNS(t *testing.T) {
	// GET preset DNS
	reqGet := httptest.NewRequest("GET", "/api/network/dns", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleDNS(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET DNS, got %d", rrGet.Code)
	}
}

func TestHandleTTL(t *testing.T) {
	// GET TTL status
	reqGet := httptest.NewRequest("GET", "/api/network/ttl", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleTTL(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET TTL, got %d", rrGet.Code)
	}
}

func TestHandleRPS(t *testing.T) {
	// GET RPS configs
	reqGet := httptest.NewRequest("GET", "/api/network/rps", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleRPS(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET RPS, got %d", rrGet.Code)
	}

	// POST empty interface
	payload, _ := json.Marshal(map[string]string{"interface": "", "bitmask": "f"})
	reqPostBad := httptest.NewRequest("POST", "/api/network/rps", bytes.NewBuffer(payload))
	rrPostBad := httptest.NewRecorder()
	handlers.HandleRPS(rrPostBad, reqPostBad)
	if rrPostBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty interface, got %d", rrPostBad.Code)
	}
}
