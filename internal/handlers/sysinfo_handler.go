package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/sysinfo"
)

func HandleSysinfo(w http.ResponseWriter, r *http.Request) {
	stats, err := sysinfo.GetStats()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
