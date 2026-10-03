package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestThemeWallpapersFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "theme_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("BFR_DATA_DIR", tempDir)

	// 1. Initial list (should be empty)
	req := httptest.NewRequest(http.MethodGet, "/api/theme/wallpapers", nil)
	w := httptest.NewRecorder()
	HandleThemeWallpapers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var listResp struct {
		Success bool            `json:"success"`
		Data    []WallpaperItem `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	if len(listResp.Data) != 0 {
		t.Fatalf("expected 0 wallpapers, got %d", len(listResp.Data))
	}

	// 2. Upload test image
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, err := mw.CreateFormFile("file", "test_cyber.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	testContent := []byte("fake-image-bytes-header-data")
	if _, err := part.Write(testContent); err != nil {
		t.Fatalf("failed to write form part: %v", err)
	}
	mw.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/theme/wallpapers/upload", &b)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	HandleThemeWallpaperUpload(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusOK {
		t.Fatalf("expected 200 upload, got %d: %s", uploadRec.Code, uploadRec.Body.String())
	}
	var uploadResp struct {
		Success bool          `json:"success"`
		Data    WallpaperItem `json:"data"`
	}
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("failed to parse upload response: %v", err)
	}
	if uploadResp.Data.ID == "" {
		t.Fatalf("expected non-empty wallpaper id")
	}

	// 3. Serve the uploaded file
	serveReq := httptest.NewRequest(http.MethodGet, "/api/theme/wallpapers/file?id="+uploadResp.Data.ID, nil)
	serveRec := httptest.NewRecorder()
	HandleThemeWallpaperFile(serveRec, serveReq)

	if serveRec.Code != http.StatusOK {
		t.Fatalf("expected 200 serve, got %d", serveRec.Code)
	}
	servedBytes, _ := io.ReadAll(serveRec.Body)
	if string(servedBytes) != string(testContent) {
		t.Fatalf("served bytes mismatch")
	}

	// 4. Verify listed
	listRec2 := httptest.NewRecorder()
	HandleThemeWallpapers(listRec2, req)
	_ = json.Unmarshal(listRec2.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 wallpaper, got %d", len(listResp.Data))
	}

	// 5. Delete wallpaper
	delReq := httptest.NewRequest(http.MethodDelete, "/api/theme/wallpapers/delete?id="+uploadResp.Data.ID, nil)
	delRec := httptest.NewRecorder()
	HandleThemeWallpaperDelete(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 delete, got %d", delRec.Code)
	}

	// 6. Verify deleted
	fileCheck := filepath.Join(tempDir, "wallpapers", uploadResp.Data.ID)
	if _, err := os.Stat(fileCheck); !os.IsNotExist(err) {
		t.Fatalf("file still exists on disk after delete")
	}
}
