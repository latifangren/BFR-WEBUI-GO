package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bfr-webui-go/internal/backup"
	"bfr-webui-go/internal/logger"
)

func HandleBackupExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := backup.ExportBackup()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	logger.Get().Infof("backup", "Exported configuration backup")
	fileName := fmt.Sprintf("bfr_backup_%s.json", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

func HandleBackupImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Empty or invalid backup payload"})
		return
	}

	err = backup.ImportBackup(body)
	if err == nil {
		logger.Get().Infof("backup", "Imported configuration backup successfully")
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleCloudBackupConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg := backup.GetCloudManager().GetConfig()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read body"})
		return
	}
	defer r.Body.Close()

	var cfg backup.CloudConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	if err := backup.GetCloudManager().SaveConfig(cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	updated := backup.GetCloudManager().GetConfig()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func HandleCloudBackupSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := backup.GetCloudManager().SyncBackup()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	cfg := backup.GetCloudManager().GetConfig()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"last_sync": cfg.LastSync,
	})
}
