//go:build !linux && !android

package nas

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getStorageStats(path string) (usedStr, totalStr string) {
	cleanPath := filepath.Clean(path)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "df", "-k", cleanPath)
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[len(lines)-1])
			if len(fields) >= 4 {
				totalKb, _ := strconv.ParseUint(fields[1], 10, 64)
				usedKb, _ := strconv.ParseUint(fields[2], 10, 64)

				totalGB := float64(totalKb*1024) / (1024 * 1024 * 1024)
				usedGB := float64(usedKb*1024) / (1024 * 1024 * 1024)

				return fmt.Sprintf("%.2f GB", usedGB), fmt.Sprintf("%.2f GB", totalGB)
			}
		}
	}
	return "0.00 GB", "0.00 GB"
}
