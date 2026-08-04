//go:build !linux && !android

package sysinfo

import (
	"strconv"
	"strings"
	"time"
)

func getDiskPartitions() []DiskPartition {
	var disks []DiskPartition
	targets := []string{"/data", "/system", "/sdcard"}

	for _, target := range targets {
		out := runCmdTimeout(2*time.Second, "df", "-k", target)
		if out == "" {
			continue
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[len(lines)-1])
			if len(fields) >= 5 {
				totalKb, _ := strconv.ParseUint(fields[1], 10, 64)
				usedKb, _ := strconv.ParseUint(fields[2], 10, 64)
				freeKb, _ := strconv.ParseUint(fields[3], 10, 64)

				total := totalKb * 1024
				used := usedKb * 1024
				free := freeKb * 1024

				var pct float64
				if total > 0 {
					pct = (float64(used) / float64(total)) * 100
				}

				disks = append(disks, DiskPartition{
					Path:    target,
					Total:   total,
					Free:    free,
					Used:    used,
					UsedPct: pct,
				})
			}
		}
	}
	return disks
}
