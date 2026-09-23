package partition

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

var partitionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var radioEFSKeywords = []string{
	"modemst1", "modemst2", "modem", "fsg", "fsc", "sec_efs", "efs",
	"nvram", "nvdata", "nvcfg", "protect1", "protect2",
	"md1img", "md1dsp", "spmfw", "radio", "apdp", "msadp",
}

var bootRecoveryKeywords = []string{
	"init_boot", "vendor_boot", "boot", "recovery", "dtbo",
	"vbmeta_system", "vbmeta_vendor", "vbmeta_product", "vbmeta",
	"abl", "xbl", "tz", "hyp", "rpm", "devcfg", "uefisecapp",
}

var systemOSKeywords = []string{
	"system_ext", "system", "vendor", "product", "odm",
	"userdata", "cust", "metadata", "cache", "super",
}

// ParsePartitionSlot separates the base partition name and slot ("a", "b", or "").
func ParsePartitionSlot(name string) (base string, slot string) {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "_a") {
		return name[:len(name)-2], "a"
	}
	if strings.HasSuffix(lower, "_b") {
		return name[:len(name)-2], "b"
	}
	return name, ""
}

// CategorizePartition assigns a category and critical flag to a partition based on its name.
func CategorizePartition(name string) (category string, isCritical bool) {
	base, _ := ParsePartitionSlot(name)
	b := strings.ToLower(base)

	// Exact matches checked first
	for _, k := range radioEFSKeywords {
		if b == k {
			return "radio_efs", true
		}
	}
	for _, k := range bootRecoveryKeywords {
		if b == k {
			return "boot_recovery", true
		}
	}
	for _, k := range systemOSKeywords {
		if b == k {
			return "system_os", false
		}
	}

	// Substring matches
	for _, k := range radioEFSKeywords {
		if strings.Contains(b, k) {
			return "radio_efs", true
		}
	}
	for _, k := range bootRecoveryKeywords {
		if strings.Contains(b, k) {
			return "boot_recovery", true
		}
	}
	for _, k := range systemOSKeywords {
		if strings.Contains(b, k) {
			return "system_os", false
		}
	}

	return "other", false
}

// FormatBytesHuman converts byte count to human-readable format.
func FormatBytesHuman(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FindBlockDeviceRoot finds the Android block device by-name root path.
func FindBlockDeviceRoot() (string, error) {
	if env := os.Getenv("BFR_BLOCK_DEV_ROOT"); env != "" {
		if info, err := os.Stat(env); err == nil && info.IsDir() {
			return env, nil
		}
	}

	for _, c := range []string{"/dev/block/by-name", "/dev/block/bootdevice/by-name", "/dev/block/mapper"} {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}

	for _, p := range []string{"/dev/block/platform/*/by-name", "/dev/block/platform/*/*/by-name"} {
		if matches, err := filepath.Glob(p); err == nil {
			for _, m := range matches {
				if info, err := os.Stat(m); err == nil && info.IsDir() {
					return m, nil
				}
			}
		}
	}
	return "", nil
}

// GetSlotInfo detects A/B slot info from getprop or /proc/cmdline.
func GetSlotInfo() SystemSlotInfo {
	slot := ""
	if out, err := exec.Command("getprop", "ro.boot.slot_suffix").Output(); err == nil {
		slot = strings.TrimSpace(string(out))
	}
	if slot == "" {
		if outSu, errSu := config.ExecSuTimeout(time.Second, "getprop ro.boot.slot_suffix"); errSu == nil {
			slot = strings.TrimSpace(string(outSu))
		}
	}
	if slot == "" {
		if data, err := os.ReadFile("/proc/cmdline"); err == nil {
			for _, f := range strings.Fields(string(data)) {
				if strings.HasPrefix(f, "androidboot.slot_suffix=") {
					slot = strings.TrimPrefix(f, "androidboot.slot_suffix=")
					break
				} else if strings.HasPrefix(f, "androidboot.slot=") {
					slot = strings.TrimPrefix(f, "androidboot.slot=")
					break
				}
			}
		}
	}

	slot = strings.TrimPrefix(strings.TrimSpace(slot), "_")
	root, _ := FindBlockDeviceRoot()
	var total int
	if root != "" {
		if entries, err := os.ReadDir(root); err == nil {
			total = len(entries)
		}
	}

	return SystemSlotInfo{
		CurrentSlot:     slot,
		IsABDevice:      slot == "a" || slot == "b",
		BlockDeviceRoot: root,
		TotalPartitions: total,
	}
}

// getBlockDevSizeBytes reads partition size from sysfs or blockdev.
func getBlockDevSizeBytes(paths ...string) int64 {
	if len(paths) == 0 {
		return 0
	}
	canonicalPath := paths[0]
	devicePath := paths[0]
	if len(paths) > 1 {
		devicePath = paths[1]
	}

	for _, p := range []string{canonicalPath, devicePath} {
		devName := filepath.Base(p)
		sysPath := filepath.Join("/sys/class/block", devName, "size")
		if data, err := os.ReadFile(sysPath); err == nil {
			if sectors, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil && sectors > 0 {
				return sectors * 512
			}
		}
	}

	cmdPath := canonicalPath
	if cmdPath == "" {
		cmdPath = devicePath
	}
	if out, err := exec.Command("blockdev", "--getsize64", cmdPath).Output(); err == nil {
		if s, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil && s > 0 {
			return s
		}
	}
	if outSu, errSu := config.ExecSuTimeout(time.Second, fmt.Sprintf("blockdev --getsize64 '%s'", cmdPath)); errSu == nil {
		if s, err := strconv.ParseInt(strings.TrimSpace(string(outSu)), 10, 64); err == nil && s > 0 {
			return s
		}
	}
	return 0
}

// ListPartitions scans block device root and returns all discovered partitions with slot info.
func ListPartitions() ([]PartitionInfo, SystemSlotInfo, error) {
	slotInfo := GetSlotInfo()
	root := slotInfo.BlockDeviceRoot
	if root == "" {
		return []PartitionInfo{}, slotInfo, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, slotInfo, fmt.Errorf("failed to read block devices: %w", err)
	}

	var partitions []PartitionInfo
	for _, entry := range entries {
		name := entry.Name()
		devPath := filepath.Join(root, name)
		canonicalPath, errEval := filepath.EvalSymlinks(devPath)
		if errEval != nil {
			canonicalPath = devPath
		}
		_, slot := ParsePartitionSlot(name)
		category, isCritical := CategorizePartition(name)
		sizeBytes := getBlockDevSizeBytes(canonicalPath, devPath)

		partitions = append(partitions, PartitionInfo{
			Name:          name,
			DevicePath:    devPath,
			CanonicalPath: canonicalPath,
			SizeBytes:     sizeBytes,
			SizeHuman:     FormatBytesHuman(sizeBytes),
			Category:      category,
			IsCritical:    isCritical,
			Slot:          slot,
			IsActiveSlot:  slot != "" && slot == slotInfo.CurrentSlot,
		})
	}

	sort.Slice(partitions, func(i, j int) bool {
		return partitions[i].Name < partitions[j].Name
	})

	slotInfo.TotalPartitions = len(partitions)
	return partitions, slotInfo, nil
}

// FindPartitionDevice finds the block device path for a given partition name.
func FindPartitionDevice(name string) (string, error) {
	if !partitionNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid partition name: %s", name)
	}
	root, err := FindBlockDeviceRoot()
	if err != nil || root == "" {
		return "", fmt.Errorf("block device root directory not found")
	}
	p := filepath.Join(root, name)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("partition device not found: %s", name)
	}
	return p, nil
}
