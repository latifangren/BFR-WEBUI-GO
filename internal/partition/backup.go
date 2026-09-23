package partition

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// GetBackupDirectory returns the directory for storing partition backups.
func GetBackupDirectory() string {
	if env := os.Getenv("BFR_BACKUP_DIR"); env != "" {
		_ = os.MkdirAll(env, 0755)
		return env
	}
	primary := "/sdcard/Download/BFR_Backups"
	if err := os.MkdirAll(primary, 0755); err == nil {
		return primary
	}
	fallback := filepath.Join(config.GetPersistentDataDir(), "backups")
	_ = os.MkdirAll(fallback, 0755)
	return fallback
}

// BackupPartition creates a raw .img or .img.gz backup of a partition and writes a .sha256 sidecar.
func BackupPartition(name string, compress bool) (*BackupResult, error) {
	if !partitionNameRegex.MatchString(name) {
		return nil, fmt.Errorf("invalid partition name: %s", name)
	}
	devPath, err := FindPartitionDevice(name)
	if err != nil {
		return nil, err
	}

	backupDir := GetBackupDirectory()
	timestamp := time.Now().Format("20060102_150405")
	ext := ".img"
	if compress {
		ext = ".img.gz"
	}
	filename := fmt.Sprintf("%s_%s%s", name, timestamp, ext)
	filePath := filepath.Join(backupDir, filename)

	outFile, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	hasher := sha256.New()
	var writer io.Writer = io.MultiWriter(outFile, hasher)
	var gzWriter *gzip.Writer
	if compress {
		gzWriter = gzip.NewWriter(writer)
		writer = gzWriter
	}

	srcReader, errReader := openBlockDeviceReader(devPath)
	if errReader != nil {
		_ = outFile.Close()
		_ = os.Remove(filePath)
		return nil, errReader
	}

	copiedBytes, errCopy := io.Copy(writer, srcReader)
	errCloseReader := srcReader.Close()
	if gzWriter != nil {
		if errClose := gzWriter.Close(); errClose != nil && errCopy == nil {
			errCopy = errClose
		}
	}
	if errCopy != nil {
		_ = outFile.Close()
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to copy partition data: %w", errCopy)
	}
	if errCloseReader != nil {
		_ = outFile.Close()
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to close partition reader: %w", errCloseReader)
	}
	if copiedBytes == 0 {
		_ = outFile.Close()
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("backup failed: 0 bytes read from partition device '%s'", devPath)
	}
	_ = outFile.Close()

	info, errStat := os.Stat(filePath)
	var sizeBytes int64
	if errStat == nil {
		sizeBytes = info.Size()
	}
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	sidecarContent := fmt.Sprintf("%s  %s\n", shaHex, filename)
	_ = os.WriteFile(filePath+".sha256", []byte(sidecarContent), 0644)

	logger.Get().Infof("partition", "Backup created: partition=%s file=%s size=%d sha256=%s", name, filename, sizeBytes, shaHex)

	return &BackupResult{
		Success:       true,
		PartitionName: name,
		Filename:      filename,
		FilePath:      filePath,
		SizeBytes:     sizeBytes,
		SHA256:        shaHex,
	}, nil
}

func openBlockDeviceReader(devPath string) (io.ReadCloser, error) {
	if f, err := os.Open(devPath); err == nil {
		return f, nil
	}
	cmd := exec.Command(config.SUBin, "-c", fmt.Sprintf("cat '%s'", devPath))
	stdout, errPipe := cmd.StdoutPipe()
	if errPipe != nil {
		return nil, fmt.Errorf("failed to pipe block device: %w", errPipe)
	}
	if errStart := cmd.Start(); errStart != nil {
		return nil, fmt.Errorf("failed to read block device via su: %w", errStart)
	}
	return &suReadCloser{stdout: stdout, cmd: cmd}, nil
}

type suReadCloser struct {
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func (s *suReadCloser) Read(p []byte) (n int, err error) {
	return s.stdout.Read(p)
}

func (s *suReadCloser) Close() error {
	errPipe := s.stdout.Close()
	if errWait := s.cmd.Wait(); errWait != nil {
		return errWait
	}
	return errPipe
}

// BackupGroup backs up a collection of partitions by group name ("radio_efs" or "boot_core").
func BackupGroup(group string, compress bool) ([]*BackupResult, error) {
	partitions, slotInfo, err := ListPartitions()
	if err != nil {
		return nil, err
	}

	var targets []string
	switch group {
	case "radio_efs":
		for _, p := range partitions {
			if p.Category == "radio_efs" {
				targets = append(targets, p.Name)
			}
		}
	case "boot_core":
		coreBases := map[string]bool{
			"boot": true, "init_boot": true, "vendor_boot": true,
			"recovery": true, "dtbo": true, "vbmeta": true,
		}
		for _, p := range partitions {
			base, slot := ParsePartitionSlot(p.Name)
			if coreBases[base] {
				if slotInfo.IsABDevice && slotInfo.CurrentSlot != "" {
					if slot == slotInfo.CurrentSlot || slot == "" {
						targets = append(targets, p.Name)
					}
				} else {
					targets = append(targets, p.Name)
				}
			}
		}
		if len(targets) == 0 {
			for _, p := range partitions {
				base, _ := ParsePartitionSlot(p.Name)
				if coreBases[base] {
					targets = append(targets, p.Name)
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported backup group: %s (supported: 'radio_efs', 'boot_core')", group)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no partitions found for group '%s'", group)
	}

	var results []*BackupResult
	for _, target := range targets {
		res, err := BackupPartition(target, compress)
		if err != nil {
			results = append(results, &BackupResult{
				Success:       false,
				PartitionName: target,
				Error:         err.Error(),
			})
		} else {
			results = append(results, res)
		}
	}
	return results, nil
}

// ListBackups scans the backup directory for existing .img and .img.gz backup files.
func ListBackups() ([]BackupFile, error) {
	backupDir := GetBackupDirectory()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupFile{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []BackupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".img") && !strings.HasSuffix(name, ".img.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(backupDir, name)
		sha := ""
		if sidecarData, errSidecar := os.ReadFile(fullPath + ".sha256"); errSidecar == nil {
			if fields := strings.Fields(string(sidecarData)); len(fields) > 0 {
				sha = fields[0]
			}
		}

		backups = append(backups, BackupFile{
			Filename:      name,
			Path:          fullPath,
			PartitionName: parsePartitionNameFromFilename(name),
			SizeBytes:     info.Size(),
			SizeHuman:     FormatBytesHuman(info.Size()),
			SHA256:        sha,
			IsCompressed:  strings.HasSuffix(name, ".img.gz"),
			CreatedAt:     info.ModTime(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
	return backups, nil
}

func parsePartitionNameFromFilename(filename string) string {
	base := strings.TrimSuffix(strings.TrimSuffix(filename, ".gz"), ".img")
	parts := strings.Split(base, "_")
	if len(parts) >= 3 {
		last := parts[len(parts)-1]
		secondLast := parts[len(parts)-2]
		if len(last) == 6 && len(secondLast) == 8 {
			return strings.Join(parts[:len(parts)-2], "_")
		}
	}
	return base
}

// DeleteBackup deletes a backup file and its .sha256 sidecar.
func DeleteBackup(filename string) error {
	if filename == "" || filename != filepath.Base(filename) || strings.Contains(filename, "..") {
		return fmt.Errorf("invalid backup filename: %s", filename)
	}
	if !strings.HasSuffix(filename, ".img") && !strings.HasSuffix(filename, ".img.gz") {
		return fmt.Errorf("invalid backup file extension: %s", filename)
	}

	backupDir := GetBackupDirectory()
	filePath := filepath.Join(backupDir, filename)
	_ = os.Remove(filePath + ".sha256")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	logger.Get().Infof("partition", "Deleted backup: %s", filename)
	return nil
}
