package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/handlers"
)

func TestTerminalAndScrcpyHandlers_Unauthorized(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	authMgr := auth.NewManager("")
	terminalHandler := handlers.NewTerminalHandler(authMgr)
	scrcpyHandler := handlers.NewScrcpyHandler(authMgr)

	// 1. Terminal unauthorized request
	reqTerm := httptest.NewRequest("GET", "/ws/terminal", nil)
	rrTerm := httptest.NewRecorder()
	terminalHandler.HandleWS(rrTerm, reqTerm)

	if rrTerm.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for unauthenticated terminal request, got %d", rrTerm.Code)
	}

	// 2. Scrcpy unauthorized request
	reqScrcpy := httptest.NewRequest("GET", "/ws/scrcpy", nil)
	rrScrcpy := httptest.NewRecorder()
	scrcpyHandler.HandleWS(rrScrcpy, reqScrcpy)

	if rrScrcpy.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for unauthenticated scrcpy request, got %d", rrScrcpy.Code)
	}
}
