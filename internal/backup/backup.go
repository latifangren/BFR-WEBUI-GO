package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bfr-webui-go/internal/config"
)

type BackupBundle struct {
	Timestamp time.Time         `json:"timestamp"`
	Configs   map[string]string `json:"configs"`
}

var backupFiles = []string{
	"charger_config.json",
	"ssh_config.json",
	"vnstat_data.json",
	"tweaks.json",
}

func ExportBackup() ([]byte, error) {
	bundle := BackupBundle{
		Timestamp: time.Now(),
		Configs:   make(map[string]string),
	}

	searchDirs := []string{
		config.ModuleDir,
		".",
	}

	for _, fname := range backupFiles {
		for _, dir := range searchDirs {
			p := filepath.Join(dir, fname)
			if data, err := os.ReadFile(p); err == nil {
				bundle.Configs[fname] = string(data)
				break
			}
		}
	}

	return json.MarshalIndent(bundle, "", "  ")
}

func ImportBackup(data []byte) error {
	var bundle BackupBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return fmt.Errorf("invalid backup bundle format: %w", err)
	}

	if len(bundle.Configs) == 0 {
		return fmt.Errorf("backup bundle contains no configuration files")
	}

	allowedNames := map[string]bool{
		"charger_config.json": true,
		"ssh_config.json":     true,
		"vnstat_data.json":    true,
		"tweaks.json":         true,
	}

	destDir := config.ModuleDir
	if err := os.MkdirAll(destDir, 0755); err != nil {
		destDir = "."
	}

	for fname, content := range bundle.Configs {
		if !allowedNames[fname] {
			continue
		}
		targetPath := filepath.Join(destDir, fname)
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			// Fallback to local dir
			_ = os.WriteFile(fname, []byte(content), 0644)
		}
	}

	return nil
}
