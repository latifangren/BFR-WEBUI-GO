package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandlePing_Validation(t *testing.T) {
	// Invalid method GET
	reqGet := httptest.NewRequest("GET", "/api/network/ping", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandlePing(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rrGet.Code)
	}

	// Empty host
	reqEmpty := httptest.NewRequest("POST", "/api/network/ping", bytes.NewBufferString(`{"host":""}`))
	rrEmpty := httptest.NewRecorder()
	handlers.HandlePing(rrEmpty, reqEmpty)
	if rrEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty host, got %d", rrEmpty.Code)
	}
}

func TestHandleDNS(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/network/dns", nil)
	rr := httptest.NewRecorder()
	handlers.HandleDNS(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET DNS, got %d", rr.Code)
	}
}

func TestHandleTTL(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/network/ttl", nil)
	rr := httptest.NewRecorder()
	handlers.HandleTTL(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET TTL, got %d", rr.Code)
	}
}

func TestHandleDNSReset(t *testing.T) {
	// Invalid method GET
	reqGet := httptest.NewRequest("GET", "/api/network/dns/reset", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleDNSReset(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET on reset, got %d", rrGet.Code)
	}

	// Valid POST reset
	reqPost := httptest.NewRequest("POST", "/api/network/dns/reset", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleDNSReset(rrPost, reqPost)
	if rrPost.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST on reset, got %d", rrPost.Code)
	}
}
