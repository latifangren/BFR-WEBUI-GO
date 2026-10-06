package proxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// HotspotProxyConfig represents BFM ap.list.cfg configuration.
type HotspotProxyConfig struct {
	ProxyHotspot bool     `json:"proxy_hotspot"` // Whether hotspot/tethering is proxied
	ApList       []string `json:"ap_list"`       // Explicit interfaces (ap0, wlan1, rndis0)
}

func GetApListPath() string {
	if custom := os.Getenv("BFR_BOX_BASE"); custom != "" {
		return filepath.Join(custom, "ap.list.cfg")
	}
	base := GetBoxBasePath()
	candidates := []string{
		base + "/ap.list.cfg",
		"/data/adb/box/ap.list.cfg",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return base + "/ap.list.cfg"
}

// LoadHotspotProxyConfig reads ap.list.cfg.
func LoadHotspotProxyConfig() (HotspotProxyConfig, error) {
	cfg := HotspotProxyConfig{
		ProxyHotspot: true,
		ApList:       []string{"ap0", "wlan1", "wlan0", "rndis0"},
	}

	path := GetApListPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}

	var list []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "proxy_hotspot:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cfg.ProxyHotspot = strings.EqualFold(strings.TrimSpace(parts[1]), "true")
			}
			continue
		}
		list = append(list, line)
	}

	if len(list) > 0 {
		cfg.ApList = list
	}

	return cfg, nil
}

// SaveHotspotProxyConfig writes ap.list.cfg.
func SaveHotspotProxyConfig(cfg HotspotProxyConfig) error {
	path := GetApListPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil && config.SUBin != "" && os.Getenv("CI") == "" {
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("mkdir -p %s", dir)).Run()
	}

	var sb strings.Builder
	sb.WriteString("# Box for Magisk Hotspot / SoftAP Proxy Configuration\n")
	sb.WriteString(fmt.Sprintf("proxy_hotspot:%t\n\n", cfg.ProxyHotspot))
	sb.WriteString("# Interfaces to route:\n")
	for _, ifc := range cfg.ApList {
		ifc = strings.TrimSpace(ifc)
		if ifc != "" {
			sb.WriteString(ifc + "\n")
		}
	}

	content := sb.String()
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		if config.SUBin != "" && os.Getenv("CI") == "" {
			cmd := fmt.Sprintf("cat << 'EOF' > %s\n%sEOF\nchmod 644 %s", path, content, path)
			if out, err := exec.Command(config.SUBin, "-c", cmd).CombinedOutput(); err != nil {
				return fmt.Errorf("failed to save ap.list.cfg: %v, out: %s", err, string(out))
			}
			return nil
		}
		return err
	} else {
		_ = os.Rename(tmpPath, path)
	}

	logger.Get().Infof("proxy", "Updated Hotspot proxy config (proxyHotspot=%t)", cfg.ProxyHotspot)
	return nil
}
