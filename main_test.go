package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPSRedirectHandler(t *testing.T) {
	handler := httpsRedirectHandler("8443")
	request := httptest.NewRequest(http.MethodGet, "http://device.local/status?full=1", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTemporaryRedirect)
	}
	if got, want := response.Header().Get("Location"), "https://device.local:8443/status?full=1"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestHTTPSRedirectHandlerRejectsPlainHTTPWrites(t *testing.T) {
	handler := httpsRedirectHandler("443")
	request := httptest.NewRequest(http.MethodPost, "http://device.local/api/auth/login", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUpgradeRequired {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUpgradeRequired)
	}
	if response.Header().Get("Location") != "" {
		t.Fatalf("unexpected redirect for request with a body: %q", response.Header().Get("Location"))
	}
}
