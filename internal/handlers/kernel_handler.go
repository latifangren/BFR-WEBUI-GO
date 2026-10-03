package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/kernel"
)

type kernelProfileRequest struct {
	Profile string `json:"profile"`
}

type kernelClusterRequest struct {
	PolicyID int    `json:"policy_id"`
	Governor string `json:"governor"`
	MinFreq  int64  `json:"min_freq"`
	MaxFreq  int64  `json:"max_freq"`
}

type kernelVMRequest struct {
	Swappiness           *int   `json:"swappiness,omitempty"`
	VFSCachePressure     *int   `json:"vfs_cache_pressure,omitempty"`
	DirtyRatio           *int   `json:"dirty_ratio,omitempty"`
	DirtyBackgroundRatio *int   `json:"dirty_background_ratio,omitempty"`
	TCPCongestion        string `json:"tcp_congestion,omitempty"`
}

func HandleKernelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := kernel.GetManager().GetStatus()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(status)
}

func HandleKernelProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req kernelProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Profile == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid profile payload"})
		return
	}

	err := kernel.GetManager().ApplyProfile(req.Profile)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"profile": req.Profile,
	})
}

func HandleKernelCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req kernelClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid cluster payload"})
		return
	}

	err := kernel.GetManager().SetClusterConfig(req.PolicyID, req.Governor, req.MinFreq, req.MaxFreq)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func HandleKernelVM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req kernelVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid VM payload"})
		return
	}

	err := kernel.GetManager().SetVMConfig(req.Swappiness, req.VFSCachePressure, req.DirtyRatio, req.DirtyBackgroundRatio, req.TCPCongestion)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
