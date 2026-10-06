package proxy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// SubsConfig represents Subscription configuration.
type SubsConfig struct {
	SubscriptionURLs []string `json:"subscription_urls"`
	AutoUpdate       bool     `json:"auto_update"`
	IntervalCron     string   `json:"interval_cron"`
	LastUpdated      string   `json:"last_updated"`
}

// GetSubsConfig loads subscription URLs from settings.ini or clash config.
func GetSubsConfig() SubsConfig {
	cfg := SubsConfig{
		SubscriptionURLs: make([]string, 0),
		AutoUpdate:       false,
		IntervalCron:     "0 0,6,12,18 * * *",
	}

	settings, err := LoadBoxSettings()
	if err == nil {
		cfg.AutoUpdate = settings.UpdateSubscription
	}

	// Try reading subscription URLs from settings.ini text
	path := GetSettingsPath()
	if data, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "subscription_url_clash=") {
				val := strings.TrimPrefix(line, "subscription_url_clash=")
				val = strings.Trim(val, `()"`)
				for _, u := range strings.Fields(val) {
					u = strings.Trim(u, `"'`)
					if strings.HasPrefix(u, "http") {
						cfg.SubscriptionURLs = append(cfg.SubscriptionURLs, u)
					}
				}
			}
		}
	}

	return cfg
}

// UpdateSubscriptionNow triggers subscription update via box.tool subs or direct HTTP download.
func UpdateSubscriptionNow(targetURL string) (string, error) {
	// 1. Try box.tool if present in BFM scripts
	boxTool := boxBasePath + "/scripts/box.tool"
	if _, err := os.Stat(boxTool); err == nil {
		cmdStr := fmt.Sprintf("%s -c '%s subs'", config.SUBin, boxTool)
		out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
		if err == nil {
			logger.Get().Infof("proxy", "Subscription updated via box.tool")
			return string(out), nil
		}
	}

	// 2. Direct HTTP download into config.yaml if URL provided
	if targetURL == "" {
		cfg := GetSubsConfig()
		if len(cfg.SubscriptionURLs) > 0 {
			targetURL = cfg.SubscriptionURLs[0]
		}
	}

	if targetURL == "" {
		return "", fmt.Errorf("no subscription URL configured")
	}

	destConfig := DetectConfigPath()
	if destConfig == "" {
		destConfig = boxBasePath + "/clash/config.yaml"
	}

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "ClashMeta/1.18.0 Mihomo/1.18.0 BFR/1.2.7")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) < 100 {
		return "", fmt.Errorf("downloaded file too small or invalid")
	}

	dir := filepath.Dir(destConfig)
	_ = os.MkdirAll(dir, 0755)

	tmpPath := destConfig + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0644); err != nil {
		cmd := fmt.Sprintf("cat << 'EOF' > %s\n%sEOF\nchmod 644 %s", destConfig, string(body), destConfig)
		if out, err := exec.Command(config.SUBin, "-c", cmd).CombinedOutput(); err != nil {
			return "", fmt.Errorf("write error: %v, out: %s", err, string(out))
		}
	} else {
		_ = os.Rename(tmpPath, destConfig)
	}

	logger.Get().Infof("proxy", "Successfully downloaded subscription from %s (%d bytes)", targetURL, len(body))

	// Trigger core restart to apply updated config
	go func() {
		_ = ControlService("restart")
	}()

	return fmt.Sprintf("Successfully updated config from %s (%d bytes). Restarting proxy...", targetURL, len(body)), nil
}

// UpdateGeoXNow updates GeoIP and GeoSite database files.
func UpdateGeoXNow() (string, error) {
	// 1. Try box.tool geox
	boxTool := boxBasePath + "/scripts/box.tool"
	if _, err := os.Stat(boxTool); err == nil {
		cmdStr := fmt.Sprintf("%s -c '%s geox'", config.SUBin, boxTool)
		out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
		if err == nil {
			logger.Get().Infof("proxy", "GeoX assets updated via box.tool")
			return string(out), nil
		}
	}

	// 2. Direct download of GeoIP/GeoSite via curl / wget
	downloadDir := boxBasePath + "/clash"
	_ = os.MkdirAll(downloadDir, 0755)

	cmdStr := fmt.Sprintf(
		"curl -sL --connect-timeout 20 -o %s/Country.mmdb https://github.com/Dreamacro/maxmind-geoip/releases/latest/download/Country.mmdb 2>/dev/null || wget -q -O %s/Country.mmdb https://github.com/Dreamacro/maxmind-geoip/releases/latest/download/Country.mmdb",
		downloadDir, downloadDir,
	)
	out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("geox download notice: %v, out: %s", err, string(out))
	}

	logger.Get().Infof("proxy", "GeoX database updated successfully")
	return "GeoX database (Country.mmdb) updated successfully.", nil
}
