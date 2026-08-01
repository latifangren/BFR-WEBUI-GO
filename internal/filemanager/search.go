package filemanager

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"bfr-webui-go/internal/config"
)

func SearchFiles(rootDir string, query string) ([]FileInfo, error) {
	cleanRoot, err := SanitizePath(rootDir)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []FileInfo
	const maxResults = 500

	err = filepath.WalkDir(cleanRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if queryLower == "" || strings.Contains(strings.ToLower(d.Name()), queryLower) {
			info, err := d.Info()
			if err == nil {
				results = append(results, FileInfo{
					Name:        info.Name(),
					Path:        path,
					IsDir:       info.IsDir(),
					Size:        info.Size(),
					ModTime:     info.ModTime(),
					Permissions: info.Mode().String(),
				})

				if len(results) >= maxResults {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	return results, err
}

func GetStorageUsage() (total, free, used int64, percent float64, err error) {
	targets := []string{"/sdcard", "/data", "/storage/emulated/0"}
	targetDir := ""

	for _, t := range targets {
		if _, err := SanitizePath(t); err == nil {
			targetDir = t
			break
		}
	}

	if targetDir == "" && len(config.AllowedDirs) > 0 {
		targetDir = config.AllowedDirs[0]
	}

	out, err := exec.Command(config.SUBin, "-c", fmt.Sprintf("df -k %s", targetDir)).Output()
	if err != nil {
		out, err = exec.Command("df", "-k", targetDir).Output()
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("storage usage stat error: %w", err)
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) >= 2 {
		fields := strings.Fields(lines[len(lines)-1])
		if len(fields) >= 4 {
			totalKb, _ := strconv.ParseInt(fields[1], 10, 64)
			usedKb, _ := strconv.ParseInt(fields[2], 10, 64)
			freeKb, _ := strconv.ParseInt(fields[3], 10, 64)

			total = totalKb * 1024
			used = usedKb * 1024
			free = freeKb * 1024
			if total > 0 {
				percent = (float64(used) / float64(total)) * 100
			}
			return total, free, used, percent, nil
		}
	}

	return 0, 0, 0, 0, fmt.Errorf("failed to parse storage usage output")
}
