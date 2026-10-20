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
	cfgStoreMu sync.RWMutex
	cachedCfg  *RouterConfig
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
	}
}

// LoadRouterConfig reads persisted router configuration.
func LoadRouterConfig() (*RouterConfig, error) {
	cfgStoreMu.Lock()
	defer cfgStoreMu.Unlock()

	if cachedCfg != nil {
		copyCfg := *cachedCfg
		return &copyCfg, nil
	}

	p := getRouterConfigPath()
	data, err := os.ReadFile(p)
	if err != nil {
		def := DefaultRouterConfig()
		cachedCfg = &def
		return &def, nil
	}

	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var loaded RouterConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		def := DefaultRouterConfig()
		cachedCfg = &def
		return &def, nil
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

	cachedCfg = &loaded
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
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := config.WriteFileAtomic(p, data, 0644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	cachedCfg = cfg
	logger.Get().Infof("hotspot", "Persistently saved router configuration (mode: %s, gateway: %s)", cfg.GatewayMode, cfg.GatewayIP)
	return nil
}
