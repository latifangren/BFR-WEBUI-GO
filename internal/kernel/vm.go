package kernel

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

// ReadVMConfig reads current kernel virtual memory and networking sysctl properties.
func ReadVMConfig() VMConfig {
	vm := VMConfig{
		Swappiness:           readIntProc("/proc/sys/vm/swappiness", 60),
		VFSCachePressure:     readIntProc("/proc/sys/vm/vfs_cache_pressure", 100),
		DirtyRatio:           readIntProc("/proc/sys/vm/dirty_ratio", 20),
		DirtyBackgroundRatio: readIntProc("/proc/sys/vm/dirty_background_ratio", 5),
		TCPCongestion:        readSysfsTrim("/proc/sys/net/ipv4/tcp_congestion_control"),
	}

	availStr := readSysfsTrim("/proc/sys/net/ipv4/tcp_available_congestion_control")
	if availStr != "" {
		vm.AvailTCPCongestion = strings.Fields(availStr)
	} else {
		vm.AvailTCPCongestion = []string{"cubic", "reno"}
	}

	return vm
}

// ApplyVMConfig applies virtual memory and networking settings to sysctl / procfs.
func ApplyVMConfig(swappiness, vfsPressure, dirtyRatio, dirtyBgRatio *int, tcpCongestion string) error {
	var cmds []string

	if swappiness != nil && *swappiness >= 0 && *swappiness <= 200 {
		cmds = append(cmds, fmt.Sprintf("echo %d > /proc/sys/vm/swappiness", *swappiness))
	}
	if vfsPressure != nil && *vfsPressure >= 0 && *vfsPressure <= 500 {
		cmds = append(cmds, fmt.Sprintf("echo %d > /proc/sys/vm/vfs_cache_pressure", *vfsPressure))
	}
	if dirtyRatio != nil && *dirtyRatio >= 1 && *dirtyRatio <= 100 {
		cmds = append(cmds, fmt.Sprintf("echo %d > /proc/sys/vm/dirty_ratio", *dirtyRatio))
	}
	if dirtyBgRatio != nil && *dirtyBgRatio >= 1 && *dirtyBgRatio <= 100 {
		cmds = append(cmds, fmt.Sprintf("echo %d > /proc/sys/vm/dirty_background_ratio", *dirtyBgRatio))
	}
	if tcpCongestion != "" && isValidIdentifier(tcpCongestion) {
		cmds = append(cmds, fmt.Sprintf("echo %s > /proc/sys/net/ipv4/tcp_congestion_control", tcpCongestion))
	}

	if len(cmds) == 0 {
		return nil
	}

	fullCmd := strings.Join(cmds, " ; ")
	_, err := config.ExecSuTimeout(5*time.Second, fullCmd)
	return err
}

func readIntProc(path string, fallback int) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fallback
	}
	return val
}

func isValidIdentifier(s string) bool {
	for _, ch := range s {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '-') {
			return false
		}
	}
	return len(s) > 0 && len(s) < 32
}
