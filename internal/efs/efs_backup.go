package efs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// EfsBackupEntry represents metadata for a backed-up EFS file.
type EfsBackupEntry struct {
	ID           string    `json:"id"`
	OriginalPath string    `json:"original_path"`
	Timestamp    time.Time `json:"timestamp"`
	Size         int64     `json:"size"`
	SHA256       string    `json:"sha256"`
	Filename     string    `json:"filename"` // File name within backup directory
}

// efsBackupManifest manages the persisted list of EFS backups.
type efsBackupManifest struct {
	Version string           `json:"version"`
	Backups []EfsBackupEntry `json:"backups"`
}

var (
	backupMu sync.Mutex
)

// GetEfsBackupDir returns the path to the EFS file backups directory.
func GetEfsBackupDir() string {
	if dir := os.Getenv("BFR_EFS_BACKUP_DIR"); dir != "" {
		_ = os.MkdirAll(dir, 0755)
		return dir
	}
	dir := filepath.Join(config.GetPersistentDataDir(), "efs_backups")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func getManifestPath() string {
	return filepath.Join(GetEfsBackupDir(), "manifest.json")
}

func loadManifestLocked() (*efsBackupManifest, error) {
	mPath := getManifestPath()
	data, err := os.ReadFile(mPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &efsBackupManifest{
				Version: "1.0",
				Backups: []EfsBackupEntry{},
			}, nil
		}
		return nil, fmt.Errorf("read backup manifest: %w", err)
	}

	var m efsBackupManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return &efsBackupManifest{
			Version: "1.0",
			Backups: []EfsBackupEntry{},
		}, nil
	}
	return &m, nil
}

func saveManifestLocked(m *efsBackupManifest) error {
	mPath := getManifestPath()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup manifest: %w", err)
	}
	return os.WriteFile(mPath, data, 0644)
}

// SaveEfsBackup saves the previous content of an EFS file to disk before it is overwritten.
func SaveEfsBackup(originalPath string, data []byte) (*EfsBackupEntry, error) {
	backupMu.Lock()
	defer backupMu.Unlock()

	backupDir := GetEfsBackupDir()
	now := time.Now().UTC()

	sum := sha256.Sum256(data)
	hashHex := hex.EncodeToString(sum[:])

	// Create unique ID
	idRaw := fmt.Sprintf("%s_%d_%s", originalPath, now.UnixNano(), hashHex[:8])
	idHash := sha256.Sum256([]byte(idRaw))
	id := hex.EncodeToString(idHash[:8])

	// Generate safe filename: <timestamp>_<sanitized_path>.bak
	sanitized := strings.ReplaceAll(strings.Trim(originalPath, "/"), "/", "_")
	if len(sanitized) > 64 {
		sanitized = sanitized[len(sanitized)-64:]
	}
	filename := fmt.Sprintf("%s_%s_%s.bak", now.Format("20060102_150405"), id, sanitized)
	filePath := filepath.Join(backupDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("write backup file '%s': %w", filePath, err)
	}

	entry := EfsBackupEntry{
		ID:           id,
		OriginalPath: originalPath,
		Timestamp:    now,
		Size:         int64(len(data)),
		SHA256:       hashHex,
		Filename:     filename,
	}

	manifest, err := loadManifestLocked()
	if err != nil {
		manifest = &efsBackupManifest{Version: "1.0", Backups: []EfsBackupEntry{}}
	}
	manifest.Backups = append(manifest.Backups, entry)
	if err := saveManifestLocked(manifest); err != nil {
		logger.Get().Warnf("EFS", "Failed to update backup manifest: %v", err)
	}

	logger.Get().Infof("EFS", "Auto-backup created for %s (id: %s, size: %d bytes)", originalPath, id, len(data))
	return &entry, nil
}

// ListEfsBackups returns all available backups, optionally filtered by originalPath.
func ListEfsBackups(filterPath string) ([]EfsBackupEntry, error) {
	backupMu.Lock()
	defer backupMu.Unlock()

	manifest, err := loadManifestLocked()
	if err != nil {
		return nil, err
	}

	var results []EfsBackupEntry
	for _, b := range manifest.Backups {
		if filterPath == "" || b.OriginalPath == filterPath {
			results = append(results, b)
		}
	}

	// Sort newest first
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	return results, nil
}

// GetEfsBackup returns the metadata and raw content of a backup by ID.
func GetEfsBackup(id string) (*EfsBackupEntry, []byte, error) {
	backupMu.Lock()
	defer backupMu.Unlock()

	manifest, err := loadManifestLocked()
	if err != nil {
		return nil, nil, err
	}

	var found *EfsBackupEntry
	for _, b := range manifest.Backups {
		if b.ID == id {
			copyEntry := b
			found = &copyEntry
			break
		}
	}

	if found == nil {
		return nil, nil, fmt.Errorf("backup entry with ID '%s' not found", id)
	}

	filePath := filepath.Join(GetEfsBackupDir(), found.Filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read backup file '%s': %w", filePath, err)
	}

	return found, data, nil
}

// DeleteEfsBackup deletes a backup by ID from both disk and manifest.
func DeleteEfsBackup(id string) error {
	backupMu.Lock()
	defer backupMu.Unlock()

	manifest, err := loadManifestLocked()
	if err != nil {
		return err
	}

	var newBackups []EfsBackupEntry
	var targetFilename string
	for _, b := range manifest.Backups {
		if b.ID == id {
			targetFilename = b.Filename
		} else {
			newBackups = append(newBackups, b)
		}
	}

	if targetFilename == "" {
		return fmt.Errorf("backup entry with ID '%s' not found", id)
	}

	manifest.Backups = newBackups
	_ = saveManifestLocked(manifest)

	filePath := filepath.Join(GetEfsBackupDir(), targetFilename)
	_ = os.Remove(filePath)
	logger.Get().Infof("EFS", "Backup deleted (id: %s, file: %s)", id, targetFilename)
	return nil
}
