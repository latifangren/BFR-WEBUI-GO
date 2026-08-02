package config

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// envOrDefault returns env value or fallback if not set.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// BFR_SU_BIN override for root shell invocation.
var SUBin = func() string {
	defaultBin := "su"
	if os.Getuid() == 0 {
		defaultBin = "sh"
	}
	return envOrDefault("BFR_SU_BIN", defaultBin)
}()

// ExecSuContext executes a command string with root privileges using context cancellation.
func ExecSuContext(ctx context.Context, cmdStr string) ([]byte, error) {
	return exec.CommandContext(ctx, SUBin, "-c", cmdStr).CombinedOutput()
}

// ExecSuTimeout executes a command string with root privileges using a hard timeout duration.
func ExecSuTimeout(d time.Duration, cmdStr string) ([]byte, error) {
	if d <= 0 {
		d = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return ExecSuContext(ctx, cmdStr)
}

// BFR_MODULE_DIR override for Magisk module installation directory.
var ModuleDir = envOrDefault("BFR_MODULE_DIR", "/data/adb/modules/bfr_webui_go")

// BFR_CLASH_API override for Clash controller REST API.
var ClashAPI = envOrDefault("BFR_CLASH_API", "http://127.0.0.1:9090")

// BFR_SMS_DB override for SMS telephony database path.
var SMSDb = envOrDefault("BFR_SMS_DB", "")

// BFR_LEASES_FILE override for dnsmasq leases file.
var LeasesFile = envOrDefault("BFR_LEASES_FILE", "/data/misc/dhcp/dnsmasq.leases")

// BFR_ALLOWED_DIRS list (comma-separated) for FileManager base directory boundaries.
var AllowedDirs = func() []string {
	if val := os.Getenv("BFR_ALLOWED_DIRS"); val != "" {
		var parts []string
		for _, s := range strings.Split(val, ",") {
			parts = append(parts, strings.TrimSpace(s))
		}
		return parts
	}
	return []string{"/sdcard", "/storage", "/data/adb", "/data/local/tmp", "/data/system"}
}()
