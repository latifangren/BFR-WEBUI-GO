package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modules"
)

type moduleToggleRequest struct {
	ID     string `json:"id"`
	Enable bool   `json:"enable"`
}

func HandleModulesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mods, err := modules.ListModules()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"modules": mods,
	})
}

func HandleModulesToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req moduleToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}

	err := modules.ToggleModule(req.ID, req.Enable)
	if err == nil {
		logger.Get().Infof("modules", "Module %s toggled: enable=%v", req.ID, req.Enable)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleModulesInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(64 << 20) // 64MB max module size
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to parse form"})
		return
	}

	file, header, err := r.FormFile("module")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "No module file uploaded"})
		return
	}
	defer file.Close()

	output, err := modules.InstallModule(file, header.Filename)
	if err == nil {
		logger.Get().Infof("modules", "Module installed successfully: %s", header.Filename)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"output":  output,
		"error":   fmt.Sprintf("%v", err),
	})
}
