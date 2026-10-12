package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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

// maxSubscriptionBytes caps a downloaded subscription file at 10MB.
const maxSubscriptionBytes = 10 << 20

// validateSubscriptionURL allows only https URLs whose host is not a
// loopback, private, link-local or unspecified address. This keeps the
// daemon from being used as an SSRF primitive against on-device services.
func validateSubscriptionURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("only https subscription URLs are allowed")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("URL missing host")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("subscription host must not be a local address")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("subscription host must not be a local/private address")
		}
	}
	return nil
}

// UpdateSubscriptionNow triggers subscription update via box.tool subs or direct HTTP download.
func UpdateSubscriptionNow(targetURL string) (string, error) {
	// 1. Try box.tool if present in BFM scripts
	boxTool := GetBoxBasePath() + "/scripts/box.tool"
	if _, err := os.Stat(boxTool); err == nil && config.SUBin != "" && os.Getenv("CI") == "" {
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

	if err := validateSubscriptionURL(targetURL); err != nil {
		return "", err
	}

	destConfig := DetectConfigPath()
	if destConfig == "" {
		destConfig = GetBoxBasePath() + "/clash/config.yaml"
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSubscriptionBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) < 100 {
		return "", fmt.Errorf("downloaded file too small or invalid")
	}
	if len(body) > maxSubscriptionBytes {
		return "", fmt.Errorf("downloaded file too large (max 10MB)")
	}

	dir := filepath.Dir(destConfig)
	_ = os.MkdirAll(dir, 0755)

	// Write atomically via temp file + rename. There is intentionally no
	// shell fallback here: interpolating remote content into a shell
	// heredoc would be a remote-code-execution primitive.
	tmpPath := destConfig + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0644); err != nil {
		return "", fmt.Errorf("write error: %w", err)
	}
	if err := os.Rename(tmpPath, destConfig); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("write error: %w", err)
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
	boxTool := GetBoxBasePath() + "/scripts/box.tool"
	if _, err := os.Stat(boxTool); err == nil && config.SUBin != "" && os.Getenv("CI") == "" {
		cmdStr := fmt.Sprintf("%s -c '%s geox'", config.SUBin, boxTool)
		out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
		if err == nil {
			logger.Get().Infof("proxy", "GeoX assets updated via box.tool")
			return string(out), nil
		}
	}

	// 2. Direct download of GeoIP/GeoSite via curl / wget
	downloadDir := GetBoxBasePath() + "/clash"
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
