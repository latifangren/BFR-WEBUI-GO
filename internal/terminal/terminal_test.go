package terminal_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/terminal"
)

func TestHandleWebsocket_Unauthorized(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Request terminal without token should return 401
	req := httptest.NewRequest("GET", "/ws/terminal", nil)
	rr := httptest.NewRecorder()

	terminal.HandleWebsocket(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for unauthenticated terminal request, got %d", rr.Code)
	}
}
