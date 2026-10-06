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

// BoxSettings represents Box for Magisk (BFM) configuration settings.ini.
type BoxSettings struct {
	BinName            string `json:"bin_name"`            // "mihomo", "clash", "sing-box", "xray", "v2fly", "hysteria"
	NetworkMode        string `json:"network_mode"`        // "tproxy", "redirect", "mixed", "enhance", "tun"
	IPv6               bool   `json:"ipv6"`                // true / false
	TproxyPort         string `json:"tproxy_port"`         // default "9898"
	RedirPort          string `json:"redir_port"`          // default "9797"
	CgroupMemcg        bool   `json:"cgroup_memcg"`        // true / false
	MemcgLimit         string `json:"memcg_limit"`         // "100M", "200M"
	RunCrontab         bool   `json:"run_crontab"`         // true / false
	UpdateGeo          bool   `json:"update_geo"`          // true / false
	UpdateSubscription bool   `json:"update_subscription"` // true / false
	Quic               string `json:"quic"`                // "enable" / "disable"
	BoxDir             string `json:"box_dir"`             // "/data/adb/box"
}

// GetSettingsPath returns the path to BFM settings.ini.
func GetSettingsPath() string {
	candidates := []string{
		boxBasePath + "/settings.ini",
		"/data/adb/box/settings.ini",
		"/data/adb/modules/box4magisk/settings.ini",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return boxBasePath + "/settings.ini"
}

// LoadBoxSettings parses settings.ini into BoxSettings struct.
func LoadBoxSettings() (BoxSettings, error) {
	s := BoxSettings{
		BinName:     "mihomo",
		NetworkMode: "tproxy",
		IPv6:        false,
		TproxyPort:  "9898",
		RedirPort:   "9797",
		CgroupMemcg: false,
		MemcgLimit:  "150M",
		Quic:        "enable",
		BoxDir:      boxBasePath,
	}

	path := GetSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback to checking running binary or default
		if _, ok := checkRunning("sing-box"); ok {
			s.BinName = "sing-box"
		}
		return s, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		switch key {
		case "bin_name":
			if val != "" {
				s.BinName = val
			}
		case "network_mode":
			if val != "" {
				s.NetworkMode = val
			}
		case "ipv6":
			s.IPv6 = strings.EqualFold(val, "true")
		case "tproxy_port":
			if val != "" {
				s.TproxyPort = val
			}
		case "redir_port":
			if val != "" {
				s.RedirPort = val
			}
		case "cgroup_memcg":
			s.CgroupMemcg = strings.EqualFold(val, "true")
		case "memcg_limit":
			if val != "" {
				s.MemcgLimit = val
			}
		case "run_crontab":
			s.RunCrontab = strings.EqualFold(val, "true")
		case "update_geo":
			s.UpdateGeo = strings.EqualFold(val, "true")
		case "update_subscription":
			s.UpdateSubscription = strings.EqualFold(val, "true")
		case "quic":
			if val != "" {
				s.Quic = val
			}
		case "box_dir":
			if val != "" {
				s.BoxDir = val
			}
		}
	}

	return s, nil
}

// SaveBoxSettings updates or creates settings.ini in BFM directory.
func SaveBoxSettings(settings BoxSettings) error {
	path := GetSettingsPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("mkdir -p %s", dir)).Run()
	}

	// Read existing file to preserve comments and extra variables
	var lines []string
	keysSet := make(map[string]bool)

	if data, err := os.ReadFile(path); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			rawLine := scanner.Text()
			trimmed := strings.TrimSpace(rawLine)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				lines = append(lines, rawLine)
				continue
			}

			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				switch key {
				case "bin_name":
					lines = append(lines, fmt.Sprintf(`bin_name="%s"`, settings.BinName))
					keysSet["bin_name"] = true
				case "network_mode":
					lines = append(lines, fmt.Sprintf(`network_mode="%s"`, settings.NetworkMode))
					keysSet["network_mode"] = true
				case "ipv6":
					lines = append(lines, fmt.Sprintf(`ipv6="%t"`, settings.IPv6))
					keysSet["ipv6"] = true
				case "tproxy_port":
					lines = append(lines, fmt.Sprintf(`tproxy_port="%s"`, settings.TproxyPort))
					keysSet["tproxy_port"] = true
				case "redir_port":
					lines = append(lines, fmt.Sprintf(`redir_port="%s"`, settings.RedirPort))
					keysSet["redir_port"] = true
				case "cgroup_memcg":
					lines = append(lines, fmt.Sprintf(`cgroup_memcg="%t"`, settings.CgroupMemcg))
					keysSet["cgroup_memcg"] = true
				case "memcg_limit":
					lines = append(lines, fmt.Sprintf(`memcg_limit="%s"`, settings.MemcgLimit))
					keysSet["memcg_limit"] = true
				case "quic":
					lines = append(lines, fmt.Sprintf(`quic="%s"`, settings.Quic))
					keysSet["quic"] = true
				default:
					lines = append(lines, rawLine)
				}
			} else {
				lines = append(lines, rawLine)
			}
		}
	} else {
		// New file template
		lines = append(lines,
			"#!/system/bin/sh",
			`settings="/data/adb/box/settings.ini"`,
			fmt.Sprintf(`bin_name="%s"`, settings.BinName),
			fmt.Sprintf(`network_mode="%s"`, settings.NetworkMode),
			fmt.Sprintf(`ipv6="%t"`, settings.IPv6),
			fmt.Sprintf(`tproxy_port="%s"`, settings.TproxyPort),
			fmt.Sprintf(`redir_port="%s"`, settings.RedirPort),
			fmt.Sprintf(`cgroup_memcg="%t"`, settings.CgroupMemcg),
			fmt.Sprintf(`memcg_limit="%s"`, settings.MemcgLimit),
			fmt.Sprintf(`quic="%s"`, settings.Quic),
			`box_user_group="root:net_admin"`,
			fmt.Sprintf(`box_dir="%s"`, settings.BoxDir),
		)
		keysSet["bin_name"] = true
		keysSet["network_mode"] = true
		keysSet["ipv6"] = true
	}

	// Append any unset keys
	if !keysSet["bin_name"] {
		lines = append(lines, fmt.Sprintf(`bin_name="%s"`, settings.BinName))
	}
	if !keysSet["network_mode"] {
		lines = append(lines, fmt.Sprintf(`network_mode="%s"`, settings.NetworkMode))
	}
	if !keysSet["ipv6"] {
		lines = append(lines, fmt.Sprintf(`ipv6="%t"`, settings.IPv6))
	}

	content := strings.Join(lines, "\n") + "\n"
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		// Fallback via su root
		cmd := fmt.Sprintf("cat << 'EOF' > %s\n%sEOF\nchmod 644 %s", path, content, path)
		if out, err := exec.Command(config.SUBin, "-c", cmd).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to save settings: %v, out: %s", err, string(out))
		}
		return nil
	}

	_ = os.Rename(tmpPath, path)
	logger.Get().Infof("proxy", "Updated Box settings (core: %s, mode: %s, ipv6: %t)", settings.BinName, settings.NetworkMode, settings.IPv6)
	return nil
}

// ApplyIptablesRenew reloads transparent iptables rules in BFM.
func ApplyIptablesRenew() error {
	cmdStr := fmt.Sprintf("if [ -f %s/scripts/box.iptables ]; then %s/scripts/box.iptables renew 2>/dev/null || %s/scripts/box.iptables enable; fi", boxBasePath, boxBasePath, boxBasePath)
	out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
	if err != nil {
		logger.Get().Warnf("proxy", "iptables renew warning: %v, out: %s", err, string(out))
		return err
	}
	return nil
}
