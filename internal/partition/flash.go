package partition

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// FlashWhitelist defines base partition names that are permitted to be flashed.
var FlashWhitelist = map[string]bool{
	"boot":          true,
	"init_boot":     true,
	"vendor_boot":   true,
	"recovery":      true,
	"dtbo":          true,
	"vbmeta":        true,
	"vbmeta_system": true,
	"vbmeta_vendor": true,
}

// IsFlashablePartition checks if a partition's base name is on the safety whitelist.
func IsFlashablePartition(name string) bool {
	base, _ := ParsePartitionSlot(name)
	return FlashWhitelist[strings.ToLower(base)]
}

// FlashPartition writes an image to a partition with safety whitelist checks.
func FlashPartition(name string, reader io.Reader, size int64, autoBackup bool) (*FlashResult, error) {
	if !partitionNameRegex.MatchString(name) {
		return nil, fmt.Errorf("invalid partition name: %s", name)
	}

	base, _ := ParsePartitionSlot(name)
	if !IsFlashablePartition(name) {
		return nil, fmt.Errorf("flashing partition '%s' (base '%s') is prohibited by safety policy", name, base)
	}

	devPath, err := FindPartitionDevice(name)
	if err != nil {
		return nil, err
	}

	partSize := getBlockDevSizeBytes(devPath)

	var backupFilename string
	if autoBackup {
		bakRes, err := BackupPartition(name, true)
		if err != nil {
			return nil, fmt.Errorf("auto-backup before flashing failed: %w", err)
		}
		backupFilename = bakRes.Filename
	}

	// Write reader to temporary file while calculating SHA256
	tmpFile, err := os.CreateTemp("", "bfr_flash_*.img")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp flash file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	hasher := sha256.New()
	mw := io.MultiWriter(tmpFile, hasher)
	written, err := io.Copy(mw, reader)
	_ = tmpFile.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to write uploaded image: %w", err)
	}
	if size > 0 && written != size {
		return nil, fmt.Errorf("upload size mismatch: expected %d, got %d", size, written)
	}

	info, errStat := os.Stat(tmpPath)
	if errStat != nil {
		return nil, fmt.Errorf("failed to inspect temporary flash file: %w", errStat)
	}
	imgSize := info.Size()

	if partSize > 0 && imgSize > partSize {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("image size (%s) exceeds target partition size (%s)", FormatBytesHuman(imgSize), FormatBytesHuman(partSize))
	}

	shaHex := hex.EncodeToString(hasher.Sum(nil))

	// Write to block device
	flashed := false
	if devFile, errOpen := os.OpenFile(devPath, os.O_WRONLY, 0600); errOpen == nil {
		src, errRead := os.Open(tmpPath)
		if errRead == nil {
			if _, errWrite := io.Copy(devFile, src); errWrite == nil {
				flashed = true
			}
			_ = src.Close()
		}
		_ = devFile.Close()
	}

	if !flashed {
		cmd := fmt.Sprintf("dd if='%s' of='%s' bs=4096 status=none", tmpPath, devPath)
		out, errDd := config.ExecSuTimeout(60*time.Second, cmd)
		if errDd != nil {
			return nil, fmt.Errorf("dd flash command failed: %v, output: %s", errDd, string(out))
		}
	}

	logger.Get().Infof("partition", "Flashed partition %s from image (sha256=%s)", name, shaHex)

	return &FlashResult{
		Success:       true,
		PartitionName: name,
		BackupFile:    backupFilename,
		SHA256:        shaHex,
		Message:       fmt.Sprintf("Partition %s successfully flashed", name),
	}, nil
}
