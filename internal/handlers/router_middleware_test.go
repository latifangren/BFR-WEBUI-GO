package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/auth"
)

func TestUnknownAPIPathReturnsNotFound(t *testing.T) {
	t.Setenv("BFR_DATA_DIR", t.TempDir())
	mux := http.NewServeMux()
	RegisterRoutes(mux, auth.NewManager("test-password"))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/not-registered", nil)
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if got := response.Header().Get("Content-Type"); got == "text/html; charset=utf-8" {
		t.Fatal("unknown API path returned the SPA document")
	}
}

func TestDefaultPasswordCannotUseWebSocketRoutes(t *testing.T) {
	t.Setenv("BFR_DATA_DIR", t.TempDir())
	authManager := auth.NewManager("")
	token, ok, _ := authManager.Authenticate(auth.DefaultPassword, "127.0.0.1:1234")
	if !ok {
		t.Fatal("failed to create default-password session")
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, authManager)
	for _, path := range []string{
		"/api/terminal/ws",
		"/api/scrcpy/ws",
		"/api/logs/logcat/stream",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
			}
		})
	}
}

func TestAcceptsGzip(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "accepted", header: "gzip", want: true},
		{name: "case insensitive", header: "br, GZip", want: true},
		{name: "quality accepted", header: "gzip;q=0.5, br;q=1", want: true},
		{name: "explicitly refused", header: "gzip;q=0, identity;q=1", want: false},
		{name: "wildcard accepted", header: "br, *;q=0.8", want: true},
		{name: "explicit refusal overrides wildcard", header: "gzip;q=0, *;q=1", want: false},
		{name: "malformed quality refused", header: "gzip;q=invalid", want: false},
		{name: "not accepted", header: "br, identity", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := acceptsGzip(test.header); got != test.want {
				t.Fatalf("acceptsGzip(%q) = %t, want %t", test.header, got, test.want)
			}
		})
	}
}
