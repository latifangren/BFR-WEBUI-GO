//go:build linux || android

package sysinfo

import "syscall"

func getDiskPartitions() []DiskPartition {
	var disks []DiskPartition
	targets := []string{"/data", "/system", "/sdcard"}

	for _, target := range targets {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(target, &stat); err == nil {
			total := stat.Blocks * uint64(stat.Bsize)
			free := stat.Bfree * uint64(stat.Bsize)
			used := total - free

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
	return disks
}
