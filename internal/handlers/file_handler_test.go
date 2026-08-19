package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleFilesList_InvalidPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/files/list?path=/nonexistent_forbidden_dir_12345", nil)
	rr := httptest.NewRecorder()

	handlers.HandleFilesList(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for forbidden/invalid dir, got %d", rr.Code)
	}
}

func TestHandleFilesRead_InvalidPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/files/read?path=/nonexistent_forbidden_file.txt", nil)
	rr := httptest.NewRecorder()

	handlers.HandleFilesRead(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for forbidden/invalid file, got %d", rr.Code)
	}
}

func TestHandleFilesSave_InvalidPayload(t *testing.T) {
	// Wrong method
	reqGet := httptest.NewRequest("GET", "/api/files/save", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleFilesSave(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET on save, got %d", rrGet.Code)
	}

	// Empty path payload
	payload, _ := json.Marshal(map[string]string{"path": "", "content": "test"})
	reqBad := httptest.NewRequest("POST", "/api/files/save", bytes.NewBuffer(payload))
	rrBad := httptest.NewRecorder()
	handlers.HandleFilesSave(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty path, got %d", rrBad.Code)
	}
}

func TestHandleFilesDelete_InvalidPayload(t *testing.T) {
	reqGet := httptest.NewRequest("GET", "/api/files/delete", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleFilesDelete(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET on delete, got %d", rrGet.Code)
	}

	payload, _ := json.Marshal(map[string]string{"path": ""})
	reqBad := httptest.NewRequest("POST", "/api/files/delete", bytes.NewBuffer(payload))
	rrBad := httptest.NewRecorder()
	handlers.HandleFilesDelete(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty path, got %d", rrBad.Code)
	}
}

func TestHandleFilesMkdir_InvalidPayload(t *testing.T) {
	reqGet := httptest.NewRequest("GET", "/api/files/mkdir", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleFilesMkdir(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET on mkdir, got %d", rrGet.Code)
	}

	payload, _ := json.Marshal(map[string]string{"path": ""})
	reqBad := httptest.NewRequest("POST", "/api/files/mkdir", bytes.NewBuffer(payload))
	rrBad := httptest.NewRecorder()
	handlers.HandleFilesMkdir(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty path, got %d", rrBad.Code)
	}
}

func TestHandleFilesUpload_InvalidPayload(t *testing.T) {
	reqGet := httptest.NewRequest("GET", "/api/files/upload", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleFilesUpload(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET on upload, got %d", rrGet.Code)
	}

	reqPostNoFile := httptest.NewRequest("POST", "/api/files/upload", bytes.NewBufferString("invalid form"))
	rrPost := httptest.NewRecorder()
	handlers.HandleFilesUpload(rrPost, reqPostNoFile)
	if rrPost.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing multipart file, got %d", rrPost.Code)
	}
}
