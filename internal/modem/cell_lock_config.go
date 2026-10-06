package modem

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type CellLockConfig struct {
	Enabled bool   `json:"enabled"`
	Slot    int    `json:"slot"`
	Type    string `json:"type"` // "lte"
	EARFCN  uint32 `json:"earfcn"`
	PCI     uint16 `json:"pci"`
}

func GetCellLockConfigPath() string {
	return config.GetPersistentFilePath("cell_lock.json")
}

func LoadCellLockConfig() (CellLockConfig, error) {
	var cfg CellLockConfig
	path := GetCellLockConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func SaveCellLockConfig(cfg CellLockConfig) error {
	path := GetCellLockConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteFileAtomic(path, data, 0644)
}

func ClearSavedCellLockConfig() error {
	path := GetCellLockConfigPath()
	_ = os.Remove(path)
	return SaveCellLockConfig(CellLockConfig{
		Enabled: false,
	})
}

// ApplySavedCellLock is called during device boot to automatically re-apply persistent cell lock.
func ApplySavedCellLock() error {
	cfg, err := LoadCellLockConfig()
	if err != nil || !cfg.Enabled || cfg.EARFCN == 0 {
		return nil
	}

	logger.Get().Infof("Modem", "Restoring persistent cell lock: Slot=%d EARFCN=%d PCI=%d", cfg.Slot, cfg.EARFCN, cfg.PCI)

	// Retry loop for modem readiness during early boot (up to 4 attempts)
	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		err := ApplyLTECellLock(cfg.Slot, cfg.EARFCN, cfg.PCI)
		if err == nil {
			logger.Get().Infof("Modem", "Persistent cell lock applied successfully on attempt %d", attempt)
			return nil
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}

	logger.Get().Warnf("Modem", "Failed to apply persistent cell lock after boot attempts: %v", lastErr)
	return fmt.Errorf("cell lock boot apply failed: %w", lastErr)
}
