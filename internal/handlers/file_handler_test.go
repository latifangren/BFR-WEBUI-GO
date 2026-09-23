package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

func TestHandleFilesUpload_NoFiles(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("path", "/data")
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/api/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	handlers.HandleFilesUpload(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for no files uploaded, got %d", rr.Code)
	}
}

func TestHandleFilesUpload_PluralAndSingularFields(t *testing.T) {
	// Test plural 'files' field with invalid/empty path
	bodyPlural := &bytes.Buffer{}
	writerPlural := multipart.NewWriter(bodyPlural)
	part1, _ := writerPlural.CreateFormFile("files", "test1.txt")
	_, _ = part1.Write([]byte("content 1"))
	part2, _ := writerPlural.CreateFormFile("files", "test2.txt")
	_, _ = part2.Write([]byte("content 2"))
	_ = writerPlural.Close()

	reqPlural := httptest.NewRequest("POST", "/api/files/upload", bodyPlural)
	reqPlural.Header.Set("Content-Type", writerPlural.FormDataContentType())
	rrPlural := httptest.NewRecorder()
	handlers.HandleFilesUpload(rrPlural, reqPlural)
	if rrPlural.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid/empty path with 'files', got %d", rrPlural.Code)
	}

	// Test singular 'file' field with query param path
	bodySingular := &bytes.Buffer{}
	writerSingular := multipart.NewWriter(bodySingular)
	partSingular, _ := writerSingular.CreateFormFile("file", "test.txt")
	_, _ = partSingular.Write([]byte("content"))
	_ = writerSingular.Close()

	reqSingular := httptest.NewRequest("POST", "/api/files/upload?path=/nonexistent_dir_12345", bodySingular)
	reqSingular.Header.Set("Content-Type", writerSingular.FormDataContentType())
	rrSingular := httptest.NewRecorder()
	handlers.HandleFilesUpload(rrSingular, reqSingular)
	if rrSingular.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid dir via query param with 'file', got %d", rrSingular.Code)
	}
}
