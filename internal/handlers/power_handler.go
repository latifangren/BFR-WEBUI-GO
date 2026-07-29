package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bfr-webui-go/internal/power"
)

type powerRequest struct {
	Action string `json:"action"`
}

func HandlePower(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req powerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}

	act := power.Action(strings.ToLower(req.Action))
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := power.Execute(act); err != nil {
			log.Printf("Power execution error: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Executing action: %s", act),
	})
}
