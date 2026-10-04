//go:build !nomodem

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"bfr-webui-go/internal/efs"
)

// HandleEfsNVItems returns the full built-in NV item catalog, optionally filtered by category.
// GET /api/efs/nv/items?category=Identity
func HandleEfsNVItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cat := r.URL.Query().Get("category")

	var items []efs.NVItem
	if cat != "" && cat != "all" {
		items = efs.GetNVItemsByCategory(cat)
	} else {
		items = efs.GetAllNVItems()
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":         true,
		"items":      items,
		"categories": efs.NVCategories,
		"total":      len(items),
	})
}

// HandleEfsNVRead reads a single NV item from the modem via DIAG.
// POST /api/efs/nv/read
// Body: { "id": 550, "index": 0 }
func HandleEfsNVRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID    uint16 `json:"id"`
		Index uint16 `json:"index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	nvm, err := efs.GetManagerNV()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	res, err := nvm.ReadNV(req.ID, req.Index)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error(), "result": res})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": res})
}

// HandleEfsNVWrite writes a single NV item to the modem via DIAG.
// POST /api/efs/nv/write
// Body: { "id": 550, "index": 0, "data_hex": "080123456789ABFF" }
func HandleEfsNVWrite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID      uint16 `json:"id"`
		Index   uint16 `json:"index"`
		DataHex string `json:"data_hex"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	if strings.TrimSpace(req.DataHex) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "data_hex is required"})
		return
	}

	nvm, err := efs.GetManagerNV()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	res, err := nvm.WriteNV(req.ID, req.Index, req.DataHex)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error(), "result": res})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": res})
}

// HandleEfsNVBackup backs up a list of NV item IDs (or a range) to a JSON file on the device.
// POST /api/efs/nv/backup
// Body: { "ids": [550, 2497, 5014], "filename": "/sdcard/nv_backup.json" }
//    OR: { "start_id": 0, "end_id": 1000, "filename": "/sdcard/nv_range.json" }
func HandleEfsNVBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDs      []uint16 `json:"ids"`
		StartID  *uint16  `json:"start_id"`
		EndID    *uint16  `json:"end_id"`
		Filename string   `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	if req.Filename == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "filename is required"})
		return
	}

	nvm, err := efs.GetManagerNV()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// Choose mode: range or item list
	var ch <-chan efs.NVBackupProgress
	if req.StartID != nil && req.EndID != nil {
		ch, err = nvm.BackupRange(*req.StartID, *req.EndID, req.Filename)
	} else if len(req.IDs) > 0 {
		ch, err = nvm.BackupItems(req.IDs, req.Filename)
	} else {
		// Default: backup all catalog items
		all := efs.GetAllNVItems()
		ids := make([]uint16, len(all))
		for i, item := range all {
			ids[i] = item.ID
		}
		ch, err = nvm.BackupItems(ids, req.Filename)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// Drain progress channel and wait for completion
	var last efs.NVBackupProgress
	for p := range ch {
		last = p
	}

	if last.Error != "" {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": last.Error,
			"stats": last,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"filename": req.Filename,
		"stats":    last,
	})
}

// HandleEfsNVRestore restores NV items from a previously saved JSON backup file.
// POST /api/efs/nv/restore
// Body: { "filename": "/sdcard/nv_backup.json" }
func HandleEfsNVRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	if req.Filename == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "filename is required"})
		return
	}

	nvm, err := efs.GetManagerNV()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	ch, err := nvm.RestoreFromFile(req.Filename)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	var last efs.NVBackupProgress
	for p := range ch {
		last = p
	}

	if last.Error != "" {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": last.Error,
			"stats": last,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":    true,
		"stats": last,
	})
}

// HandleEfsNVFormat formats raw NV hex data for display without hitting the modem.
// GET /api/efs/nv/format?id=550&hex=080123456789ABFF
func HandleEfsNVFormat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.URL.Query().Get("id")
	hexStr := r.URL.Query().Get("hex")

	if idStr == "" || hexStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "id and hex are required"})
		return
	}

	idVal, err := strconv.ParseUint(idStr, 10, 16)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid id"})
		return
	}

	id := uint16(idVal)
	data := make([]byte, 0, len(hexStr)/2)
	for i := 0; i+1 < len(hexStr); i += 2 {
		var b byte
		if _, err := strconv.ParseUint(hexStr[i:i+2], 16, 8); err == nil {
			b = byte(mustParseHexByte(hexStr[i : i+2]))
		}
		data = append(data, b)
	}

	info := efs.GetNVItem(id)
	formatted := efs.FormatNVValue(id, data)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"id":        id,
		"info":      info,
		"formatted": formatted,
	})
}

func mustParseHexByte(s string) uint64 {
	v, _ := strconv.ParseUint(s, 16, 8)
	return v
}
