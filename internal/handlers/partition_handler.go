package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"bfr-webui-go/internal/partition"
)

type backupPartitionRequest struct {
	Partition string `json:"partition"`
	Compress  bool   `json:"compress"`
}

type backupGroupRequest struct {
	Group    string `json:"group"`
	Compress bool   `json:"compress"`
}

type deleteBackupRequest struct {
	Filename  string   `json:"filename,omitempty"`
	Filenames []string `json:"filenames,omitempty"`
	All       bool     `json:"all,omitempty"`
}

// HandleListPartitions returns list of partitions and current slot info
func HandleListPartitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	partitions, slotInfo, err := partition.ListPartitions()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"partitions": partitions,
		"slot_info":  slotInfo,
	})
}

// HandleBackupPartition creates a backup of a specific partition
func HandleBackupPartition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req backupPartitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Partition == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request: partition name required"})
		return
	}

	result, err := partition.BackupPartition(req.Partition, req.Compress)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HandleBackupGroup backs up a predefined group of partitions into a single .zip archive
func HandleBackupGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req backupGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Group != "radio_efs" && req.Group != "boot_core") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid group: must be 'radio_efs' or 'boot_core'"})
		return
	}

	res, err := partition.BackupGroupZip(req.Group)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"group":      req.Group,
		"filename":   res.Filename,
		"file_path":  res.FilePath,
		"size_bytes": res.SizeBytes,
		"size_human": partition.FormatBytesHuman(res.SizeBytes),
		"sha256":     res.SHA256,
	})
}

// HandleListBackups returns all existing backup files
func HandleListBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backups, err := partition.ListBackups()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"backups": backups,
	})
}

// HandleDownloadBackup serves a backup image file for download
func HandleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" || filename != filepath.Base(filename) || strings.Contains(filename, "..") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	backupDir := partition.GetBackupDirectory()
	filePath := filepath.Join(backupDir, filename)
	if _, err := os.Stat(filePath); err != nil {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}

// HandleDeleteBackup deletes one, multiple, or all backup files
func HandleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req deleteBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if req.All {
		count, err := partition.DeleteAllBackups()
		if err != nil && count == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"deleted": count,
		})
		return
	}

	if len(req.Filenames) > 0 {
		count, err := partition.DeleteMultipleBackups(req.Filenames)
		if err != nil && count == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"deleted": count,
		})
		return
	}

	if req.Filename != "" {
		if err := partition.DeleteBackup(req.Filename); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"deleted": 1,
		})
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "Filename, filenames, or all flag required",
	})
}

// HandleFlashPartition flashes an uploaded image to a whitelisted partition
func HandleFlashPartition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32MB max in-memory
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to parse multipart form"})
		return
	}

	partName := r.FormValue("partition")
	if partName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Partition name required"})
		return
	}

	safetyAck := r.FormValue("safety_ack")
	if safetyAck != "I_UNDERSTAND_THE_RISKS" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Safety acknowledgement required: safety_ack must be 'I_UNDERSTAND_THE_RISKS'",
		})
		return
	}

	autoBackupStr := r.FormValue("auto_backup")
	autoBackup := autoBackupStr == "true" || autoBackupStr == "1"

	file, header, err := r.FormFile("file")
	if err != nil {
		if r.MultipartForm != nil && r.MultipartForm.File != nil && len(r.MultipartForm.File["files"]) > 0 {
			file, err = r.MultipartForm.File["files"][0].Open()
			header = r.MultipartForm.File["files"][0]
		}
	}
	if err != nil || file == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Image file required"})
		return
	}
	defer file.Close()

	result, err := partition.FlashPartition(partName, file, header.Size, autoBackup)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
