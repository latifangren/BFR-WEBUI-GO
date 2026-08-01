package filemanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
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

// H-2 & H-4: SanitizePath cleans path, resolves symlinks, and enforces that target stays within allowed base directories.
func SanitizePath(userPath string) (string, error) {
	if userPath == "" {
		return "", fmt.Errorf("empty path")
	}
	clean := filepath.Clean(userPath)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
		clean = filepath.Clean(clean)
	}

	var evalPath string
	if _, err := os.Lstat(clean); err == nil {
		eval, err := filepath.EvalSymlinks(clean)
		if err != nil {
			return "", fmt.Errorf("symlink resolution error: %w", err)
		}
		evalPath = eval
	} else {
		dir := filepath.Dir(clean)
		base := filepath.Base(clean)
		evalDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			evalDir = dir
		}
		evalPath = filepath.Join(evalDir, base)
	}

	evalPath = filepath.Clean(evalPath)

	for _, b := range config.AllowedDirs {
		cleanBase := filepath.Clean(b)
		evalBase := cleanBase
		if target, err := filepath.EvalSymlinks(cleanBase); err == nil {
			evalBase = target
		}

		if evalPath == cleanBase || strings.HasPrefix(evalPath, cleanBase+"/") ||
			evalPath == evalBase || strings.HasPrefix(evalPath, evalBase+"/") {
			return evalPath, nil
		}
	}
	return "", fmt.Errorf("access denied: path %s is outside allowed directories", clean)
}

func ListDirectory(dirPath string) ([]FileInfo, string, error) {
	if dirPath == "" {
		if _, err := os.Stat("/sdcard"); err == nil {
			dirPath = "/sdcard"
		} else if _, err := os.Stat("/data/adb"); err == nil {
			dirPath = "/data/adb"
		} else {
			dirPath = "/data/local/tmp"
		}
	}

	cleanPath, err := SanitizePath(dirPath)
	if err != nil {
		return nil, dirPath, err
	}

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
	cleanPath, err := SanitizePath(filePath)
	if err != nil {
		return "", err
	}
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
	cleanPath, err := SanitizePath(filePath)
	if err != nil {
		return err
	}
	return os.WriteFile(cleanPath, []byte(content), 0644)
}

func UploadFile(dirPath string, fileName string, reader io.Reader) error {
	cleanDir, err := SanitizePath(dirPath)
	if err != nil {
		return err
	}
	safeFileName := filepath.Base(fileName)
	targetPath, err := SanitizePath(filepath.Join(cleanDir, safeFileName))
	if err != nil {
		return err
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	return err
}

func CreateFile(filePath string) error {
	cleanPath, err := SanitizePath(filePath)
	if err != nil {
		return err
	}
	file, err := os.Create(cleanPath)
	if err != nil {
		return err
	}
	return file.Close()
}

func RenamePath(oldPath string, newPath string) error {
	cleanOld, err := SanitizePath(oldPath)
	if err != nil {
		return err
	}
	cleanNew, err := SanitizePath(newPath)
	if err != nil {
		return err
	}
	return os.Rename(cleanOld, cleanNew)
}

func DownloadFile(filePath string, w http.ResponseWriter, r *http.Request) {
	cleanPath, err := SanitizePath(filePath)
	if err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
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
	cleanPath, err := SanitizePath(targetPath)
	if err != nil {
		return err
	}
	for _, base := range config.AllowedDirs {
		if cleanPath == filepath.Clean(base) {
			return fmt.Errorf("cannot delete root allowed directory: %s", cleanPath)
		}
	}
	return os.RemoveAll(cleanPath)
}

func CreateDir(dirPath string) error {
	cleanPath, err := SanitizePath(dirPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(cleanPath, 0755)
}
