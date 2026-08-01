package modules

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

type ModuleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	VersionCode string `json:"version_code"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

var reModuleID = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func ListModules() ([]ModuleInfo, error) {
	modulesDir := "/data/adb/modules"
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return []ModuleInfo{}, nil
	}

	var result []ModuleInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		modID := entry.Name()
		modPath := filepath.Join(modulesDir, modID)

		disableFile := filepath.Join(modPath, "disable")
		removeFile := filepath.Join(modPath, "remove")
		isDisabled := false
		if _, err := os.Stat(disableFile); err == nil {
			isDisabled = true
		}
		if _, err := os.Stat(removeFile); err == nil {
			isDisabled = true
		}

		info := ModuleInfo{
			ID:          modID,
			Name:        modID,
			Enabled:     !isDisabled,
			Version:     "Unknown",
			Author:      "Unknown",
			Description: "",
		}

		propFile := filepath.Join(modPath, "module.prop")
		if data, err := os.ReadFile(propFile); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])

				switch key {
				case "name":
					info.Name = val
				case "version":
					info.Version = val
				case "versionCode":
					info.VersionCode = val
				case "author":
					info.Author = val
				case "description":
					info.Description = val
				}
			}
		}

		result = append(result, info)
	}

	return result, nil
}

func ToggleModule(id string, enable bool) error {
	if !reModuleID.MatchString(id) {
		return fmt.Errorf("invalid module ID: %s", id)
	}

	disableFile := fmt.Sprintf("/data/adb/modules/%s/disable", id)

	if enable {
		cmdStr := fmt.Sprintf("rm -f %s", disableFile)
		out, err := config.ExecSuTimeout(5*time.Second, cmdStr)
		if err != nil {
			return fmt.Errorf("failed to enable module: %v, out: %s", err, string(out))
		}
	} else {
		cmdStr := fmt.Sprintf("touch %s", disableFile)
		out, err := config.ExecSuTimeout(5*time.Second, cmdStr)
		if err != nil {
			return fmt.Errorf("failed to disable module: %v, out: %s", err, string(out))
		}
	}

	return nil
}

func InstallModule(reader io.Reader, fileName string) (string, error) {
	tmpDir := "/data/local/tmp"
	safeName := filepath.Base(fileName)
	if !strings.HasSuffix(strings.ToLower(safeName), ".zip") {
		return "", fmt.Errorf("module package must be a .zip file")
	}

	tmpZipPath := filepath.Join(tmpDir, "mod_inst_"+safeName)
	outFile, err := os.Create(tmpZipPath)
	if err != nil {
		return "", fmt.Errorf("failed to save module zip: %w", err)
	}
	_, err = io.Copy(outFile, reader)
	outFile.Close()
	if err != nil {
		_ = os.Remove(tmpZipPath)
		return "", fmt.Errorf("failed writing module zip: %w", err)
	}

	defer func() {
		_ = os.Remove(tmpZipPath)
	}()

	cmdStr := fmt.Sprintf(`magisk --install-module "%s" || ksud module install "%s" || apatch module install "%s"`, tmpZipPath, tmpZipPath, tmpZipPath)
	out, err := config.ExecSuTimeout(60*time.Second, cmdStr)
	if err != nil {
		return string(out), fmt.Errorf("module installation failed: %v, output: %s", err, string(out))
	}

	return string(out), nil
}
