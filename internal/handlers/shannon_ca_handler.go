package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
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

// HandleShannonNRStatus returns device capability and loaded 5G NR/EN-DC combo document stats.
func HandleShannonNRStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := shannon.GetNREditorStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

// HandleShannonNRGetDocument returns the full active in-memory NRComboDocument.
func HandleShannonNRGetDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	doc := shannon.GetCurrentNRDocument()
	w.Header().Set("Content-Type", "application/json")
	if doc == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"loaded": false,
			"combos": []shannon.NRCombo{},
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"loaded":       true,
		"version":      doc.Version,
		"id":           doc.ID,
		"total_combos": doc.TotalCombos,
		"endc_count":   doc.EndcCount,
		"nrca_count":   doc.NrcaCount,
		"nrdc_count":   doc.NrdcCount,
		"source_file":  doc.SourceFile,
		"combos":       doc.Combos,
	})
}

// HandleShannonNRLoadFile loads a 5G NR / EN-DC .binarypb from disk.
func HandleShannonNRLoadFile(w http.ResponseWriter, r *http.Request) {
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

	doc, err := shannon.LoadNRCAFile(req.Path)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Warnf("ShannonNR", "Failed to load NR CA file: %v", err)
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
		"status":  shannon.GetNREditorStatus(),
	})
}

// HandleShannonNRUpload parses an uploaded 5G NR / EN-DC .binarypb file from multipart form.
func HandleShannonNRUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
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

	doc, err := shannon.LoadNRCARawBytes(header.Filename, data)
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
		"status":  shannon.GetNREditorStatus(),
	})
}

// HandleShannonNRDemo loads the built-in Pixel Sub-6 / EN-DC demo dataset.
func HandleShannonNRDemo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	doc, err := shannon.LoadDemoNRDocument()
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
		"combos":  doc.Combos,
		"status":  shannon.GetNREditorStatus(),
	})
}

// HandleShannonNRDownload streams the active 5G NR / EN-DC binary buffer.
func HandleShannonNRDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, sourcePath := shannon.GetCurrentNRBytes()
	if len(data) == 0 {
		http.Error(w, "No active NR document loaded", http.StatusNotFound)
		return
	}

	filename := "shannon_nr_uecap.binarypb"
	if sourcePath != "" {
		base := strings.TrimSuffix(filepath.Base(sourcePath), ".binarypb")
		filename = base + "_exported.binarypb"
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

// HandleShannonCAClear unloads and clears the active LTE CA document from memory.
func HandleShannonCAClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shannon.ClearCADocument()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  shannon.GetCAEditorStatus(),
	})
}

// HandleShannonNRClear unloads and clears the active 5G NR / EN-DC document from memory.
func HandleShannonNRClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shannon.ClearNRDocument()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  shannon.GetNREditorStatus(),
	})
}


