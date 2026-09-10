package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleGovernorStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/governor", nil)
	rr := httptest.NewRecorder()
	handlers.HandleGovernorStatus(rr, req)

	// Might return 200 or 500 depending on platform, but method validation:
	reqPost := httptest.NewRequest("POST", "/api/governor", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleGovernorStatus(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST to HandleGovernorStatus, got %d", rrPost.Code)
	}
}

func TestHandleGovernorSet_InvalidPayload(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/governor", bytes.NewBufferString("{}"))
	rr := httptest.NewRecorder()
	handlers.HandleGovernorSet(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty governor payload, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == nil || resp["error"] == "<nil>" {
		t.Errorf("expected error message in response, got %v", resp["error"])
	}
}
