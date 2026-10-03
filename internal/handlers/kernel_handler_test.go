package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleKernelStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/kernel/status", nil)
	rr := httptest.NewRecorder()
	handlers.HandleKernelStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	// Test wrong method
	reqPost := httptest.NewRequest(http.MethodPost, "/api/kernel/status", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleKernelStatus(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", rrPost.Code)
	}
}

func TestHandleKernelProfile_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/kernel/profile", bytes.NewBufferString(`{"profile": ""}`))
	rr := httptest.NewRecorder()
	handlers.HandleKernelProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestHandleKernelCluster_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/kernel/cluster", bytes.NewBufferString(`invalid json`))
	rr := httptest.NewRecorder()
	handlers.HandleKernelCluster(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestHandleKernelVM_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/kernel/vm", bytes.NewBufferString(`{bad}`))
	rr := httptest.NewRecorder()
	handlers.HandleKernelVM(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
	}
}
