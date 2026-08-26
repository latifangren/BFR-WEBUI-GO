package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
)

type Shortcut struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	IconURL   string `json:"icon_url"`
	CreatedAt int64  `json:"created_at"`
}

var shortcutsMu sync.Mutex

func getShortcutsFilePath() string {
	return config.GetPersistentFilePath("shortcuts.json")
}

func loadShortcuts() ([]Shortcut, error) {
	filePath := getShortcutsFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Shortcut{}, nil
		}
		return nil, err
	}
	var list []Shortcut
	if err := json.Unmarshal(data, &list); err != nil {
		return []Shortcut{}, nil
	}
	if list == nil {
		list = []Shortcut{}
	}
	return list, nil
}

func saveShortcuts(list []Shortcut) error {
	filePath := getShortcutsFilePath()
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteFileAtomic(filePath, data, 0644)
}

func HandleShortcutsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()

	list, err := loadShortcuts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func HandleShortcutsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		URL     string `json:"url"`
		IconURL string `json:"icon_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	if req.Title == "" || req.URL == "" {
		http.Error(w, "Title and URL are required", http.StatusBadRequest)
		return
	}

	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()

	list, err := loadShortcuts()
	if err != nil {
		list = []Shortcut{}
	}

	if req.ID == "" {
		newID := fmt.Sprintf("sc_%d", time.Now().UnixNano())
		list = append(list, Shortcut{
			ID:        newID,
			Title:     req.Title,
			URL:       req.URL,
			IconURL:   req.IconURL,
			CreatedAt: time.Now().Unix(),
		})
	} else {
		found := false
		for i := range list {
			if list[i].ID == req.ID {
				list[i].Title = req.Title
				list[i].URL = req.URL
				list[i].IconURL = req.IconURL
				found = true
				break
			}
		}
		if !found {
			list = append(list, Shortcut{
				ID:        req.ID,
				Title:     req.Title,
				URL:       req.URL,
				IconURL:   req.IconURL,
				CreatedAt: time.Now().Unix(),
			})
		}
	}

	if err := saveShortcuts(list); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"shortcuts": list,
	})
}

func HandleShortcutsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "Shortcut ID required", http.StatusBadRequest)
		return
	}

	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()

	list, err := loadShortcuts()
	if err != nil {
		list = []Shortcut{}
	}

	newList := make([]Shortcut, 0, len(list))
	for _, item := range list {
		if item.ID != req.ID {
			newList = append(newList, item)
		}
	}

	if err := saveShortcuts(newList); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"shortcuts": newList,
	})
}
