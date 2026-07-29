package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/filemanager"
)

type fileSaveRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type filePathRequest struct {
	Path string `json:"path"`
}

func HandleFilesList(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("path")
	files, currentPath, err := filemanager.ListDirectory(dirPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"path":  currentPath,
		"files": files,
	})
}

func HandleFilesRead(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	content, err := filemanager.ReadFile(filePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"path":    filePath,
		"content": content,
	})
}

func HandleFilesSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}
	err := filemanager.SaveFile(req.Path, req.Content)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to parse form"})
		return
	}
	dirPath := r.FormValue("path")
	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	err = filemanager.UploadFile(dirPath, header.Filename, file)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesDownload(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	filemanager.DownloadFile(filePath, w, r)
}

func HandleFilesDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req filePathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid path"})
		return
	}
	err := filemanager.DeletePath(req.Path)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req filePathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid directory path"})
		return
	}
	err := filemanager.CreateDir(req.Path)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}
