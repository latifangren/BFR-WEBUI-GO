package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleModulesList(t *testing.T) {
	// GET modules list
	reqGet := httptest.NewRequest("GET", "/api/modules/list", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModulesList(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET modules list, got %d", rrGet.Code)
	}

	// POST wrong method
	reqPost := httptest.NewRequest("POST", "/api/modules/list", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleModulesList(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST, got %d", rrPost.Code)
	}
}

func TestHandleModulesToggle_Validation(t *testing.T) {
	// GET wrong method
	reqGet := httptest.NewRequest("GET", "/api/modules/toggle", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModulesToggle(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rrGet.Code)
	}

	// Empty ID payload
	payload, _ := json.Marshal(map[string]interface{}{"id": "", "enable": true})
	reqBad := httptest.NewRequest("POST", "/api/modules/toggle", bytes.NewBuffer(payload))
	rrBad := httptest.NewRecorder()
	handlers.HandleModulesToggle(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty module ID, got %d", rrBad.Code)
	}
}

func TestHandleModulesInstall_Validation(t *testing.T) {
	// GET wrong method
	reqGet := httptest.NewRequest("GET", "/api/modules/install", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleModulesInstall(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rrGet.Code)
	}

	// POST missing file
	reqPost := httptest.NewRequest("POST", "/api/modules/install", bytes.NewBufferString("invalid body"))
	rrPost := httptest.NewRecorder()
	handlers.HandleModulesInstall(rrPost, reqPost)
	if rrPost.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing file, got %d", rrPost.Code)
	}
}
