package proxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// AppProxyConfig represents BFM package.list.cfg configuration.
type AppProxyConfig struct {
	Mode     string   `json:"mode"`     // "blacklist" or "whitelist"
	Packages []string `json:"packages"` // List of package names
}

// InstalledApp represents an Android application detected on device.
type InstalledApp struct {
	PackageName string `json:"package_name"`
	Label       string `json:"label"`
	IsSystem    bool   `json:"is_system"`
	IsProxy     bool   `json:"is_proxy"` // true if currently enabled in package list
}

func GetPackageConfigPath() string {
	if custom := os.Getenv("BFR_BOX_BASE"); custom != "" {
		return filepath.Join(custom, "package.list.cfg")
	}
	base := GetBoxBasePath()
	candidates := []string{
		base + "/package.list.cfg",
		"/data/adb/box/package.list.cfg",
		"/data/adb/modules/box4magisk/package.list.cfg",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return base + "/package.list.cfg"
}

// LoadAppProxyConfig reads package.list.cfg into AppProxyConfig struct.
func LoadAppProxyConfig() (AppProxyConfig, error) {
	cfg := AppProxyConfig{
		Mode:     "blacklist",
		Packages: make([]string, 0),
	}

	path := GetPackageConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "mode:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				m := strings.TrimSpace(parts[1])
				if m == "whitelist" || m == "blacklist" {
					cfg.Mode = m
				}
			}
			continue
		}

		// Clean up user id prefixes like "0:com.termux" -> "com.termux"
		pkg := line
		if idx := strings.Index(pkg, ":"); idx != -1 {
			pkg = pkg[idx+1:]
		}
		pkg = strings.TrimSpace(pkg)
		if pkg != "" && strings.Contains(pkg, ".") {
			cfg.Packages = append(cfg.Packages, pkg)
		}
	}

	return cfg, nil
}

// SaveAppProxyConfig writes package.list.cfg to disk and triggers BFM inotify/iptables update.
func SaveAppProxyConfig(cfg AppProxyConfig) error {
	path := GetPackageConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil && config.SUBin != "" && os.Getenv("CI") == "" {
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("mkdir -p %s", dir)).Run()
	}

	mode := "blacklist"
	if cfg.Mode == "whitelist" {
		mode = "whitelist"
	}

	var sb strings.Builder
	sb.WriteString("# Box for Magisk Package Configuration\n")
	sb.WriteString(fmt.Sprintf("mode:%s\n\n", mode))
	sb.WriteString("# Packages to proxy or bypass:\n")

	// Sort packages alphabetically
	sort.Strings(cfg.Packages)
	seen := make(map[string]bool)
	for _, p := range cfg.Packages {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			sb.WriteString(p + "\n")
		}
	}

	content := sb.String()
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		if config.SUBin != "" && os.Getenv("CI") == "" {
			cmd := fmt.Sprintf("cat << 'EOF' > %s\n%sEOF\nchmod 644 %s", path, content, path)
			if out, err := exec.Command(config.SUBin, "-c", cmd).CombinedOutput(); err != nil {
				return fmt.Errorf("failed to save package config: %v, out: %s", err, string(out))
			}
			return nil
		}
		return err
	} else {
		_ = os.Rename(tmpPath, path)
	}

	logger.Get().Infof("proxy", "Saved App proxy rules (%s with %d packages)", mode, len(cfg.Packages))

	// Notify BFM to reload package list rules
	go func() {
		_ = ApplyIptablesRenew()
	}()

	return nil
}

// GetInstalledApps lists Android applications on the device.
func GetInstalledApps() ([]InstalledApp, error) {
	activeCfg, _ := LoadAppProxyConfig()
	activeMap := make(map[string]bool)
	for _, p := range activeCfg.Packages {
		activeMap[p] = true
	}

	// 1. Fetch 3rd party user packages first (most relevant for proxying)
	outUser, _ := exec.Command("pm", "list", "packages", "-3").Output()
	userMap := make(map[string]bool)
	for _, line := range strings.Split(string(outUser), "\n") {
		pkg := strings.TrimPrefix(strings.TrimSpace(line), "package:")
		if pkg != "" {
			userMap[pkg] = true
		}
	}

	// 2. Fetch all packages
	outAll, err := exec.Command("pm", "list", "packages").Output()
	if err != nil && len(userMap) == 0 {
		return nil, fmt.Errorf("failed to list packages: %v", err)
	}

	var apps []InstalledApp
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(outAll), "\n") {
		pkg := strings.TrimPrefix(strings.TrimSpace(line), "package:")
		if pkg == "" || seen[pkg] {
			continue
		}
		seen[pkg] = true

		isSystem := !userMap[pkg]
		label := humanizePackageName(pkg)

		apps = append(apps, InstalledApp{
			PackageName: pkg,
			Label:       label,
			IsSystem:    isSystem,
			IsProxy:     activeMap[pkg],
		})
	}

	// Sort user apps first, then alphabetically by label
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].IsSystem != apps[j].IsSystem {
			return !apps[i].IsSystem // user apps first
		}
		return apps[i].Label < apps[j].Label
	})

	return apps, nil
}

// humanizePackageName generates a readable label from package name.
func humanizePackageName(pkg string) string {
	parts := strings.Split(pkg, ".")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if len(last) > 1 {
			return strings.ToUpper(last[:1]) + last[1:]
		}
		return last
	}
	return pkg
}
