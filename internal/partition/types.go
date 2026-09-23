package partition

import "time"

// PartitionInfo represents metadata and state of an Android storage partition.
type PartitionInfo struct {
	Name          string `json:"name"`
	DevicePath    string `json:"device_path"`
	CanonicalPath string `json:"canonical_path"`
	SizeBytes     int64  `json:"size_bytes"`
	SizeHuman     string `json:"size_human"`
	Category      string `json:"category"` // "radio_efs", "boot_recovery", "system_os", "other"
	IsCritical    bool   `json:"is_critical"`
	Slot          string `json:"slot"` // "a", "b", or ""
	IsActiveSlot  bool   `json:"is_active_slot"`
}

// SystemSlotInfo describes the device's A/B slot configuration and partition root.
type SystemSlotInfo struct {
	CurrentSlot     string `json:"current_slot"` // "a", "b", or ""
	IsABDevice      bool   `json:"is_ab_device"`
	BlockDeviceRoot string `json:"block_device_root"`
	TotalPartitions int    `json:"total_partitions"`
}

// BackupFile represents an archived partition image file on disk.
type BackupFile struct {
	Filename      string    `json:"filename"`
	Path          string    `json:"path"`
	PartitionName string    `json:"partition_name"`
	SizeBytes     int64     `json:"size_bytes"`
	SizeHuman     string    `json:"size_human"`
	SHA256        string    `json:"sha256"`
	IsCompressed  bool      `json:"is_compressed"`
	CreatedAt     time.Time `json:"created_at"`
}

// BackupResult describes the outcome of a partition backup operation.
type BackupResult struct {
	Success       bool   `json:"success"`
	PartitionName string `json:"partition_name"`
	Filename      string `json:"filename,omitempty"`
	FilePath      string `json:"file_path,omitempty"`
	SizeBytes     int64  `json:"size_bytes,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	Error         string `json:"error,omitempty"`
}

// FlashResult describes the outcome of flashing an image to a partition.
type FlashResult struct {
	Success       bool   `json:"success"`
	PartitionName string `json:"partition_name"`
	BackupFile    string `json:"backup_file,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}
