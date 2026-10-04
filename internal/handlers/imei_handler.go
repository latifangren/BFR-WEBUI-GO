//go:build !nomodem

package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/efs"
)

// HandleEfsIMEIRead returns current IMEI 1 and IMEI 2 read from modem DIAG.
// GET /api/efs/imei
func HandleEfsIMEIRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mgr := efs.GetManager()
	info, err := mgr.ReadIMEIs()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":   true,
		"imei": info,
	})
}

// HandleEfsIMEIWrite writes a new IMEI using Normal (DIAG 0x27) and/or EFS Method.
// POST /api/efs/imei/write
// Body: { "slot": 0, "imei": "353456789012345", "method": "auto", "use_offline_mode": true, "reboot_modem": false }
func HandleEfsIMEIWrite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req efs.IMEIWriteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	mgr := efs.GetManager()
	res, err := mgr.WriteIMEI(req)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":     false,
			"error":  err.Error(),
			"result": res,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     res.Success,
		"result": res,
	})
}
