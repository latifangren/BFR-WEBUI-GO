package filemanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	IsDir       bool      `json:"is_dir"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Permissions string    `json:"permissions"`
}

const MaxReadFileSize = 5 * 1024 * 1024 // 5 MB

func ListDirectory(dirPath string) ([]FileInfo, string, error) {
	if dirPath == "" {
		if _, err := os.Stat("/sdcard"); err == nil {
			dirPath = "/sdcard"
		} else if _, err := os.Stat("/data/adb"); err == nil {
			dirPath = "/data/adb"
		} else {
			dirPath = "/"
		}
	}

	cleanPath := filepath.Clean(dirPath)
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, cleanPath, err
	}

	var result []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, FileInfo{
			Name:        info.Name(),
			Path:        filepath.Join(cleanPath, info.Name()),
			IsDir:       info.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	return result, cleanPath, nil
}

func ReadFile(filePath string) (string, error) {
	cleanPath := filepath.Clean(filePath)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory")
	}
	if info.Size() > MaxReadFileSize {
		return "", fmt.Errorf("file size exceeds limit of 5MB")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func SaveFile(filePath string, content string) error {
	cleanPath := filepath.Clean(filePath)
	return os.WriteFile(cleanPath, []byte(content), 0644)
}

func UploadFile(dirPath string, fileName string, reader io.Reader) error {
	cleanDir := filepath.Clean(dirPath)
	targetPath := filepath.Join(cleanDir, fileName)

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	return err
}

func CreateFile(filePath string) error {
	cleanPath := filepath.Clean(filePath)
	file, err := os.Create(cleanPath)
	if err != nil {
		return err
	}
	return file.Close()
}

func RenamePath(oldPath string, newPath string) error {
	cleanOld := filepath.Clean(oldPath)
	cleanNew := filepath.Clean(newPath)
	return os.Rename(cleanOld, cleanNew)
}

func DownloadFile(filePath string, w http.ResponseWriter, r *http.Request) {
	cleanPath := filepath.Clean(filePath)
	info, err := os.Stat(cleanPath)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	http.ServeFile(w, r, cleanPath)
}

func DeletePath(targetPath string) error {
	cleanPath := filepath.Clean(targetPath)
	return os.RemoveAll(cleanPath)
}

func CreateDir(dirPath string) error {
	cleanPath := filepath.Clean(dirPath)
	return os.MkdirAll(cleanPath, 0755)
}
