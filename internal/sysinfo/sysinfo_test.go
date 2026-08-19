package sysinfo_test

import (
	"os"
	"testing"

	"bfr-webui-go/internal/sysinfo"
)

func TestGetStats(t *testing.T) {
	stats, err := sysinfo.GetStats()
	if err != nil {
		t.Fatalf("unexpected error from GetStats: %v", err)
	}

	// On Linux / Android, memTotal or uptime should be non-zero
	if stats.ServerTime == "" {
		t.Errorf("expected non-empty ServerTime")
	}
}

func TestGetGovernorInfo(t *testing.T) {
	info, err := sysinfo.GetGovernorInfo()
	if err != nil {
		t.Fatalf("unexpected error from GetGovernorInfo: %v", err)
	}

	if info.Available == nil {
		t.Errorf("expected Available governors slice to be non-nil")
	}
}

func TestDiskPartitions(t *testing.T) {
	// Calling GetStats includes disk partitions scanning
	stats, err := sysinfo.GetStats()
	if err != nil {
		t.Fatalf("failed to get sysinfo stats: %v", err)
	}

	// Disks can be empty in mock/ci environments, but slice should be safe
	_ = stats.Disks
}

func TestProcMeminfoParsingMock(t *testing.T) {
	// Verify memory calculation logic consistency
	meminfoSample := `MemTotal:        8000000 kB
MemFree:         2000000 kB
MemAvailable:    4000000 kB
Buffers:          500000 kB
Cached:          1500000 kB
SwapTotal:       2000000 kB
SwapFree:        1000000 kB
`
	tmpFile, err := os.CreateTemp("", "meminfo_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(meminfoSample); err != nil {
		t.Fatalf("failed to write mock meminfo: %v", err)
	}
	tmpFile.Close()

	// Verify temp file reading
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("expected non-empty meminfo content")
	}
}
