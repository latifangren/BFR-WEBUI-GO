package hotspot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

var (
	cfgStoreMu    sync.RWMutex
	cachedCfg     *RouterConfig
	cachedCfgPath string
)

func getRouterConfigPath() string {
	return config.GetPersistentFilePath("hotspot_settings.json")
}

// DefaultRouterConfig returns safe defaults for the router suite.
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		GatewayMode:       "auto",
		GatewayIP:         "192.168.1.1",
		SubnetMask:        "255.255.255.0",
		DHCPStart:         "192.168.1.10",
		DHCPEnd:           "192.168.1.100",
		LeaseTime:         "12h",
		DNSPrimary:        "1.1.1.1",
		DNSSecondary:      "8.8.8.8",
		Band:              "any",
		UpstreamInterface: "auto",
		AutoStartWifi:     false,
		AutoStartUSB:      false,
		AutoStartEthernet: false,
		SSID:              "AndroidAP",
		Password:          "12345678",
		SecurityType:      "wpa2",
	}
}

// LoadRouterConfig reads persisted router configuration.
func LoadRouterConfig() (*RouterConfig, error) {
	cfgStoreMu.Lock()
	defer cfgStoreMu.Unlock()

	p := getRouterConfigPath()
	if cachedCfg != nil && cachedCfgPath == p {
		copyCfg := *cachedCfg
		return &copyCfg, nil
	}

	applyAOSPFallback := func(cfg *RouterConfig) {
		if aospSSID, aospPass, aospSec := readAOSPSoftApCredentials(); aospSSID != "" {
			cfg.SSID = aospSSID
			if aospSec != "" {
				cfg.SecurityType = aospSec
			}
			if cfg.SecurityType == "open" {
				cfg.Password = "none"
			} else if aospPass != "" {
				cfg.Password = aospPass
			}
		}
	}

	data, err := os.ReadFile(p)
	if err != nil {
		def := DefaultRouterConfig()
		applyAOSPFallback(&def)
		cachedCfg = &def
		cachedCfgPath = p
		copyCfg := *cachedCfg
		return &copyCfg, nil
	}

	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var loaded RouterConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		def := DefaultRouterConfig()
		applyAOSPFallback(&def)
		cachedCfg = &def
		cachedCfgPath = p
		copyCfg := *cachedCfg
		return &copyCfg, nil
	}

	// Validate fields and fill defaults if empty
	if loaded.GatewayMode == "" {
		loaded.GatewayMode = "auto"
	}
	if loaded.GatewayIP == "" {
		loaded.GatewayIP = "192.168.1.1"
	}
	if loaded.SubnetMask == "" {
		loaded.SubnetMask = "255.255.255.0"
	}
	if loaded.DHCPStart == "" {
		loaded.DHCPStart = "192.168.1.10"
	}
	if loaded.DHCPEnd == "" {
		loaded.DHCPEnd = "192.168.1.100"
	}
	if loaded.LeaseTime == "" {
		loaded.LeaseTime = "12h"
	}
	if loaded.Band == "" {
		loaded.Band = "any"
	}
	if loaded.UpstreamInterface == "" {
		loaded.UpstreamInterface = "auto"
	}
	if loaded.SSID == "" {
		applyAOSPFallback(&loaded)
		if loaded.SSID == "" {
			loaded.SSID = "AndroidAP"
			loaded.Password = "12345678"
			loaded.SecurityType = "wpa2"
		}
	}
	if loaded.SecurityType == "" {
		loaded.SecurityType = "wpa2"
	}
	if loaded.SecurityType == "open" {
		loaded.Password = "none"
	} else if loaded.Password == "" {
		loaded.Password = "12345678"
	}

	cachedCfg = &loaded
	cachedCfgPath = p
	copyCfg := *cachedCfg
	return &copyCfg, nil
}

// SaveRouterConfig saves router configuration atomically to disk.
func SaveRouterConfig(cfg *RouterConfig) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}

	cfgStoreMu.Lock()
	defer cfgStoreMu.Unlock()

	p := getRouterConfigPath()
	toSave := *cfg

	// Preserve existing persisted SoftAP credentials when caller omits them.
	if cachedCfg != nil && cachedCfgPath == p {
		if toSave.SSID == "" {
			toSave.SSID = cachedCfg.SSID
		}
		if toSave.SecurityType == "" {
			toSave.SecurityType = cachedCfg.SecurityType
		}
		if toSave.Password == "" && toSave.SecurityType != "open" {
			toSave.Password = cachedCfg.Password
		}
	}
	if toSave.SSID == "" {
		toSave.SSID = "AndroidAP"
	}
	if toSave.SecurityType == "" {
		toSave.SecurityType = "wpa2"
	}
	if toSave.SecurityType == "open" {
		toSave.Password = "none"
	} else if toSave.Password == "" {
		toSave.Password = "12345678"
	}

	data, err := json.MarshalIndent(&toSave, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := config.WriteFileAtomic(p, data, 0644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	cachedCopy := toSave
	cachedCfg = &cachedCopy
	cachedCfgPath = p
	*cfg = toSave
	logger.Get().Infof("hotspot", "Persistently saved router configuration (mode: %s, gateway: %s)", toSave.GatewayMode, toSave.GatewayIP)
	return nil
}
