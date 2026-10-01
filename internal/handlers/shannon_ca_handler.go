package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/shannon"
)

// HandleShannonCAStatus returns device compatibility and loaded CA combo document stats.
func HandleShannonCAStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := shannon.GetCAEditorStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

// HandleShannonCAGetDocument returns the full in-memory ComboDocument.
func HandleShannonCAGetDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	doc := shannon.GetCurrentDocument()
	w.Header().Set("Content-Type", "application/json")
	if doc == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"loaded": false,
			"combos": []shannon.Combo{},
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"loaded":  true,
		"version": doc.Version,
		"bitmask": doc.Bitmask,
		"combos":  doc.Combos,
	})
}

// HandleShannonCALoadFile loads a .binarypb from server storage or custom path.
func HandleShannonCALoadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Valid file path required"})
		return
	}

	doc, err := shannon.LoadCAFile(req.Path)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Warnf("ShannonCA", "Failed to load CA file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"combos":  doc.Combos,
		"status":  shannon.GetCAEditorStatus(),
	})
}

// HandleShannonCAUpload parses an uploaded .binarypb file directly from multipart form.
func HandleShannonCAUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 10MB max upload
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to read file"})
		return
	}

	doc, err := shannon.LoadCARawBytes(header.Filename, data)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"combos":  doc.Combos,
		"status":  shannon.GetCAEditorStatus(),
	})
}

// HandleShannonCASaveFile saves current document to target path.
func HandleShannonCASaveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path   string          `json:"path"`
		Combos []shannon.Combo `json:"combos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Target save path required"})
		return
	}

	if len(req.Combos) > 0 {
		shannon.UpdateCurrentDocument(req.Combos)
	}

	err := shannon.SaveCAFile(req.Path)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully saved %d combos to %s", len(req.Combos), req.Path),
	})
}

// HandleShannonCADownload streams the active document as a downloadable .binarypb file.
func HandleShannonCADownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	doc := shannon.GetCurrentDocument()
	if doc == nil {
		http.Error(w, "No active document", http.StatusNotFound)
		return
	}

	data, err := shannon.EncodeShannonLteUECap(doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := "shannon_lte_uecap_mod.binarypb"
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

// HandleShannonCAValidate runs validation rules against submitted or active combos.
func HandleShannonCAValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Combos []shannon.Combo `json:"combos"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	combosToTest := req.Combos
	if len(combosToTest) == 0 {
		if doc := shannon.GetCurrentDocument(); doc != nil {
			combosToTest = doc.Combos
		}
	}

	res := shannon.ValidateCombos(combosToTest)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleShannonCAAutoGen generates fallback combination subsets.
func HandleShannonCAAutoGen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req shannon.AutoGenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid payload"})
		return
	}

	generated := shannon.AutoGenerateCombos(req)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(generated),
		"combos":  generated,
	})
}

// HandleShannonCAPrune filters combinations by allowed bands and removes intra-band duplicates.
func HandleShannonCAPrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Combos           []shannon.Combo `json:"combos"`
		AllowedBands     []int32         `json:"allowed_bands"`
		MaxCombos        int             `json:"max_combos"`
		ExcludeIntraBand bool            `json:"exclude_intra_band"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid payload"})
		return
	}

	combosToPrune := req.Combos
	if len(combosToPrune) == 0 {
		if doc := shannon.GetCurrentDocument(); doc != nil {
			combosToPrune = doc.Combos
		}
	}

	pruned := shannon.PruneCombos(combosToPrune, req.AllowedBands, req.MaxCombos, req.ExcludeIntraBand)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(pruned),
		"combos":  pruned,
	})
}
