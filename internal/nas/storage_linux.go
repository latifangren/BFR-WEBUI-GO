//go:build linux || android

package nas

import (
	"fmt"
	"path/filepath"
	"syscall"
)

func getStorageStats(path string) (usedStr, totalStr string) {
	cleanPath := filepath.Clean(path)
	var stat syscall.Statfs_t
	if err := syscall.Statfs(cleanPath, &stat); err == nil {
		totalBytes := stat.Blocks * uint64(stat.Bsize)
		freeBytes := stat.Bfree * uint64(stat.Bsize)
		usedBytes := totalBytes - freeBytes

		totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
		usedGB := float64(usedBytes) / (1024 * 1024 * 1024)

		return fmt.Sprintf("%.2f GB", usedGB), fmt.Sprintf("%.2f GB", totalGB)
	}
	return "0.00 GB", "0.00 GB"
}
