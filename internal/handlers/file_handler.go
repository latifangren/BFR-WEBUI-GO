package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/filemanager"
	"bfr-webui-go/internal/logger"
)

type fileSaveRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type filePathRequest struct {
	Path string `json:"path"`
}

type fileRenameRequest struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

type fileCreateRequest struct {
	Path string `json:"path"`
}

type fileCopyMoveRequest struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type fileBatchRequest struct {
	Action  string   `json:"action"`
	Paths   []string `json:"paths"`
	DestDir string   `json:"dest_dir"`
}

type filePermissionsRequest struct {
	Path  string `json:"path"`
	Mode  string `json:"mode"`
	Owner string `json:"owner"`
}

type fileCompressRequest struct {
	Paths   []string `json:"paths"`
	DestZip string   `json:"dest_zip"`
}

type fileExtractRequest struct {
	ZipPath string `json:"zip_path"`
	DestDir string `json:"dest_dir"`
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
	if err == nil {
		logger.Get().Infof("filemanager", "File saved: path=%s", req.Path)
	}
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
	if err == nil {
		logger.Get().Infof("filemanager", "Path deleted: path=%s", req.Path)
	}
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

func HandleFilesRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileRenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OldPath == "" || req.NewPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid rename payload"})
		return
	}
	err := filemanager.RenamePath(req.OldPath, req.NewPath)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid file path"})
		return
	}
	err := filemanager.CreateFile(req.Path)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesCopy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileCopyMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Src == "" || req.Dst == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid copy payload"})
		return
	}
	err := filemanager.CopyPath(req.Src, req.Dst)
	if err == nil {
		logger.Get().Infof("filemanager", "Path copied: src=%s dst=%s", req.Src, req.Dst)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileCopyMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Src == "" || req.Dst == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid move payload"})
		return
	}
	err := filemanager.MovePath(req.Src, req.Dst)
	if err == nil {
		logger.Get().Infof("filemanager", "Path moved: src=%s dst=%s", req.Src, req.Dst)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Paths) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid batch request payload"})
		return
	}

	var err error
	switch req.Action {
	case "delete":
		err = filemanager.BatchDelete(req.Paths)
	case "copy":
		err = filemanager.BatchCopy(req.Paths, req.DestDir)
	case "move":
		err = filemanager.BatchMove(req.Paths, req.DestDir)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid batch action (allowed: delete, copy, move)"})
		return
	}

	if err == nil {
		logger.Get().Infof("filemanager", "Batch action '%s' executed on %d paths", req.Action, len(req.Paths))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req filePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid permissions request payload"})
		return
	}
	err := filemanager.ChangePermissions(req.Path, req.Mode, req.Owner)
	if err == nil {
		logger.Get().Infof("filemanager", "Permissions updated: path=%s mode=%s owner=%s", req.Path, req.Mode, req.Owner)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesCompress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileCompressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Paths) == 0 || req.DestZip == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid compress request payload"})
		return
	}
	err := filemanager.CompressZip(req.Paths, req.DestZip)
	if err == nil {
		logger.Get().Infof("filemanager", "Zip created: dest_zip=%s", req.DestZip)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fileExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ZipPath == "" || req.DestDir == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid extract request payload"})
		return
	}
	err := filemanager.ExtractZip(req.ZipPath, req.DestDir)
	if err == nil {
		logger.Get().Infof("filemanager", "Zip extracted: zip_path=%s dest_dir=%s", req.ZipPath, req.DestDir)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleFilesSearch(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("path")
	query := r.URL.Query().Get("query")
	files, err := filemanager.SearchFiles(dirPath, query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"path":  dirPath,
		"query": query,
		"files": files,
	})
}

func HandleFilesStorage(w http.ResponseWriter, r *http.Request) {
	total, free, used, pct, err := filemanager.GetStorageUsage()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"total":    total,
		"free":     free,
		"used":     used,
		"used_pct": pct,
	})
}
