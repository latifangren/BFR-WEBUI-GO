package efs

import (
	"fmt"
	"strings"
	"time"
)

// IsItemPath checks if a path belongs to known Qualcomm NV item file directories.
func IsItemPath(p string) bool {
	return strings.HasPrefix(p, "/nv/item_files/") ||
		strings.HasPrefix(p, "/nv/reg_files/") ||
		strings.HasPrefix(p, "/cgps/nv/item_files/") ||
		strings.HasPrefix(p, "/sd/")
}

// EfsEntry represents a file, directory, symlink, or item file inside modem EFS2.
type EfsEntry struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Type      string    `json:"type"` // "file", "dir", "link", "item", "unknown"
	Mode      int32     `json:"mode"`
	Size      int64     `json:"size"`
	Nlink     int32     `json:"nlink,omitempty"`
	Atime     time.Time `json:"atime,omitempty"`
	Mtime     time.Time `json:"mtime,omitempty"`
	Ctime     time.Time `json:"ctime,omitempty"`
	Target    string    `json:"target,omitempty"`     // Symlink target
	EntryType int32     `json:"entry_type,omitempty"` // Raw entry type reported by baseband
}

// StatFSInfo holds filesystem capacity metrics for the modem's internal EFS storage.
type StatFSInfo struct {
	Path        string `json:"path"`
	BlockSize   uint32 `json:"block_size"`
	TotalBlocks uint32 `json:"total_blocks"`
	FreeBlocks  uint32 `json:"free_blocks"`
	TotalBytes  uint64 `json:"total_bytes"`
	FreeBytes   uint64 `json:"free_bytes"`
	UsedBytes   uint64 `json:"used_bytes"`
	UsedPercent int    `json:"used_percent"`
}

// DiagStats contains transport metrics for the QRTR DIAG connection.
type DiagStats struct {
	RxFrames  uint64 `json:"rx_frames"`
	RxDropped uint64 `json:"rx_dropped"`
	Transport string `json:"transport"`
	Subsys    uint8  `json:"subsys"`
}

// Status represents the overall state of the EFS2 subsystem.
type Status struct {
	Connected   bool        `json:"connected"`
	IsQualcomm  bool        `json:"is_qualcomm"`
	ReadOnly    bool        `json:"readonly"`
	Transport   string      `json:"transport"`
	Subsys      uint8       `json:"subsys"`
	UnlockedSPC bool        `json:"unlocked_spc"`
	StatFS      *StatFSInfo `json:"statfs,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// NVItemData represents a single Qualcomm NV item read or write result.
type NVItemData struct {
	Item      uint16 `json:"item"`
	Index     int    `json:"index,omitempty"`
	DataHex   string `json:"data_hex"`
	Status    uint16 `json:"status"`
	StatusStr string `json:"status_str"`
}

// NVStatusStr returns a human-readable interpretation of an NV error code.
func NVStatusStr(s uint16) string {
	switch s {
	case 0:
		return "OK"
	case 1:
		return "Busy"
	case 2:
		return "Bad Command"
	case 3:
		return "NV Full"
	case 4:
		return "Failed"
	case 5:
		return "Not Active"
	case 6:
		return "Bad Parameter"
	case 7:
		return "Read-Only (Unlock Required)"
	case 8:
		return "Not Defined"
	default:
		return fmt.Sprintf("Unknown Error (%d)", s)
	}
}

// FileReadResult represents the content and format of a file read from EFS.
type FileReadResult struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	IsBinary    bool   `json:"is_binary"`
	TextContent string `json:"text_content,omitempty"`
	DataHex     string `json:"data_hex,omitempty"`
	DataBase64  string `json:"data_base64,omitempty"`
}

// WriteOptions specifies settings for EFS file writing operations.
type WriteOptions struct {
	ForceItem int  `json:"force_item"` // -1: auto (use IsItemPath), 0: regular file, 1: item file
	Backup    bool `json:"backup"`     // save old file content before overwrite if file already exists
	Verify    bool `json:"verify"`     // read-back verification comparing written data
}

// WriteResult contains execution metrics and outcome of an EFS write operation.
type WriteResult struct {
	Path         string `json:"path"`
	Size         int    `json:"size"`
	IsItem       bool   `json:"is_item"`
	BackedUp     bool   `json:"backed_up"`
	BackupID     string `json:"backup_id,omitempty"`
	Verified     bool   `json:"verified"`
	SSRTriggered bool   `json:"ssr_triggered,omitempty"`
}
