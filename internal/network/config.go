package network

import (
	"encoding/json"
	"os"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type NetworkSettings struct {
	TTLEnabled   bool   `json:"ttl_enabled"`
	TargetTTL    int    `json:"target_ttl"`
	DNSEnabled   bool   `json:"dns_enabled"`
	DNSPrimary   string `json:"dns_primary"`
	DNSSecondary string `json:"dns_secondary"`
}

func GetNetworkSettingsPath() string {
	return config.GetPersistentFilePath("network_config.json")
}

func LoadNetworkSettings() (NetworkSettings, error) {
	var settings NetworkSettings
	path := GetNetworkSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return NetworkSettings{
			TargetTTL: GetDefaultTTL(),
		}, nil
	}
	err = json.Unmarshal(data, &settings)
	if settings.TargetTTL <= 0 {
		settings.TargetTTL = GetDefaultTTL()
	}
	return settings, err
}

func SaveNetworkSettings(settings NetworkSettings) error {
	path := GetNetworkSettingsPath()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteFileAtomic(path, data, 0644)
}

// ApplyAllTweaks applies persistent network settings (called by --apply-tweaks on boot).
func ApplyAllTweaks() error {
	settings, err := LoadNetworkSettings()
	if err != nil {
		logger.Get().Warnf("network", "Failed to load network settings on boot: %v", err)
		return nil
	}

	if settings.TTLEnabled && settings.TargetTTL > 0 {
		logger.Get().Infof("network", "Applying persistent TTL on boot: %d", settings.TargetTTL)
		_ = SetTTLSpoofSDK(true, settings.TargetTTL)
	}

	if settings.DNSEnabled && settings.DNSPrimary != "" {
		logger.Get().Infof("network", "Applying persistent DNS on boot: %s / %s", settings.DNSPrimary, settings.DNSSecondary)
		_ = SetDNS(settings.DNSPrimary, settings.DNSSecondary)
	}

	return nil
}
