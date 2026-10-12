package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type WallpaperItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	URL       string `json:"url"`
	UpdatedAt int64  `json:"updated_at"`
}

var allowedWallpaperExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".jfif": true,
	".png":  true,
	".webp": true,
	".gif":  true,
	".avif": true,
	".bmp":  true,
}
// GetWallpapersDir returns the persistent directory for storing user wallpapers.
func GetWallpapersDir() string {
	dir := filepath.Join(config.GetPersistentDataDir(), "wallpapers")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func sanitizeWallpaperName(name string) string {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, name)
	if clean == "" {
		return "wallpaper.jpg"
	}
	return clean
}

// maxWallpaperDirBytes caps total stored wallpapers at 200MB to prevent
// disk-fill abuse on the phone's /data partition.
const maxWallpaperDirBytes = 200 << 20

func wallpapersDirSize() int64 {
	var total int64
	entries, err := os.ReadDir(GetWallpapersDir())
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil && !info.IsDir() {
			total += info.Size()
		}
	}
	return total
}

// HandleThemeWallpapers handles GET /api/theme/wallpapers (list).
func HandleThemeWallpapers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := GetWallpapersDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	var items []WallpaperItem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !allowedWallpaperExts[ext] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		displayName := entry.Name()
		// Remove timestamp prefix if present
		if parts := strings.SplitN(displayName, "_", 2); len(parts) == 2 && len(parts[0]) >= 10 {
			displayName = parts[1]
		}

		items = append(items, WallpaperItem{
			ID:        entry.Name(),
			Name:      displayName,
			Size:      info.Size(),
			URL:       "/api/theme/wallpapers/file?id=" + entry.Name(),
			UpdatedAt: info.ModTime().Unix(),
		})
	}

	// Sort newest first
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt > items[j].UpdatedAt
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    items,
	})
}

// HandleThemeWallpaperUpload handles POST /api/theme/wallpapers/upload.
func HandleThemeWallpaperUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Max 30MB limit
	r.Body = http.MaxBytesReader(w, r.Body, 30<<20)
	if err := r.ParseMultipartForm(30 << 20); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "File too large (max 30MB) or invalid multipart body",
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "No file provided in form field 'file'",
		})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedWallpaperExts[ext] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Unsupported image format. Allowed: JPG, JPEG, JFIF, PNG, WEBP, GIF, AVIF, BMP",
		})
		return
	}

	if wallpapersDirSize() > maxWallpaperDirBytes {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInsufficientStorage)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Wallpaper storage quota exceeded (200MB). Delete old wallpapers first.",
		})
		return
	}

	baseName := sanitizeWallpaperName(filepath.Base(header.Filename))
	savedName := fmt.Sprintf("%d_%s", time.Now().Unix(), baseName)
	destPath := filepath.Join(GetWallpapersDir(), savedName)

	dst, err := os.Create(destPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to save file: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(destPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to write image data: " + err.Error(),
		})
		return
	}

	logger.Get().Infof("theme", "Wallpaper uploaded: %s (%d bytes)", savedName, written)

	item := WallpaperItem{
		ID:        savedName,
		Name:      baseName,
		Size:      written,
		URL:       "/api/theme/wallpapers/file?id=" + savedName,
		UpdatedAt: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    item,
	})
}

// HandleThemeWallpaperFile handles GET /api/theme/wallpapers/file?id=...
func HandleThemeWallpaperFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := filepath.Base(r.URL.Query().Get("id"))
	if id == "" || id == "." || id == "/" {
		http.NotFound(w, r)
		return
	}

	filePath := filepath.Join(GetWallpapersDir(), id)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=86400, must-revalidate")
	http.ServeFile(w, r, filePath)
}

// HandleThemeWallpaperDelete handles DELETE or POST /api/theme/wallpapers/delete?id=...
func HandleThemeWallpaperDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := filepath.Base(r.URL.Query().Get("id"))
	if id == "" || id == "." || id == "/" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing or invalid wallpaper id parameter",
		})
		return
	}

	filePath := filepath.Join(GetWallpapersDir(), id)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	logger.Get().Infof("theme", "Wallpaper deleted: %s", id)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Wallpaper deleted successfully",
	})
}
