package kernel

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type Manager struct {
	mu         sync.Mutex
	configPath string
	cachedCfg  KernelConfig
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager returns the singleton Kernel Manager instance.
func GetManager() *Manager {
	once.Do(func() {
		cfgPath := config.GetPersistentFilePath("kernel_config.json")
		instance = &Manager{
			configPath: cfgPath,
		}
		_ = instance.loadConfig()
	})
	return instance
}

func (m *Manager) loadConfig() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		m.cachedCfg = KernelConfig{}
		return nil
	}
	return json.Unmarshal(data, &m.cachedCfg)
}

func (m *Manager) saveConfigLocked() error {
	data, err := json.MarshalIndent(m.cachedCfg, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteFileAtomic(m.configPath, data, 0644)
}

// GetStatus returns the current live hardware state and stored configuration.
func (m *Manager) GetStatus() (KernelStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	clusters, _ := ReadCPUClusters()
	vm := ReadVMConfig()

	kernelRelease := readSysfsTrim("/proc/sys/kernel/osrelease")
	if kernelRelease == "" {
		kernelRelease = "Linux"
	}

	return KernelStatus{
		Clusters:      clusters,
		VM:            vm,
		Config:        m.cachedCfg,
		ActiveProfile: m.cachedCfg.ActiveProfile,
		KernelRelease: kernelRelease,
	}, nil
}

// ApplyProfile activates a pre-configured profile and saves it to persistent storage.
func (m *Manager) ApplyProfile(profileName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	profile, ok := AvailableProfiles[strings.ToLower(profileName)]
	if !ok {
		return fmt.Errorf("unknown profile: %s", profileName)
	}

	// 1. Apply global governor
	if err := ApplyGlobalGovernor(profile.Governor); err != nil {
		logger.Get().Warnf("kernel", "Failed to apply profile governor: %v", err)
	}

	// 2. Apply VM parameters
	swap := profile.Swappiness
	vfs := profile.VFSCachePressure
	dr := profile.DirtyRatio
	dbr := profile.DirtyBackgroundRatio
	if err := ApplyVMConfig(&swap, &vfs, &dr, &dbr, ""); err != nil {
		logger.Get().Warnf("kernel", "Failed to apply profile VM config: %v", err)
	}

	// 3. Update persistent config
	m.cachedCfg.ActiveProfile = profile.Name
	m.cachedCfg.GlobalGovernor = profile.Governor
	m.cachedCfg.Swappiness = &swap
	m.cachedCfg.VFSCachePressure = &vfs
	m.cachedCfg.DirtyRatio = &dr
	m.cachedCfg.DirtyBackgroundRatio = &dbr

	if err := m.saveConfigLocked(); err != nil {
		logger.Get().Warnf("kernel", "Failed to save kernel config: %v", err)
	}

	logger.Get().Infof("kernel", "Applied profile: %s", profile.Label)
	return nil
}

// SetClusterConfig updates and persists CPU cluster governor and frequency bounds.
func (m *Manager) SetClusterConfig(policyID int, governor string, minFreq, maxFreq int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := ApplyClusterSettings(policyID, governor, minFreq, maxFreq); err != nil {
		return err
	}

	// Update persistent cluster list
	found := false
	for i, c := range m.cachedCfg.Clusters {
		if c.PolicyID == policyID {
			m.cachedCfg.Clusters[i].Governor = governor
			m.cachedCfg.Clusters[i].MinFreq = minFreq
			m.cachedCfg.Clusters[i].MaxFreq = maxFreq
			found = true
			break
		}
	}
	if !found {
		m.cachedCfg.Clusters = append(m.cachedCfg.Clusters, ClusterConfig{
			PolicyID: policyID,
			Governor: governor,
			MinFreq:  minFreq,
			MaxFreq:  maxFreq,
		})
	}

	// Custom cluster config overrides named profile
	m.cachedCfg.ActiveProfile = "custom"
	_ = m.saveConfigLocked()

	logger.Get().Infof("kernel", "Updated cluster policy%d: gov=%s min=%d max=%d", policyID, governor, minFreq, maxFreq)
	return nil
}

// SetVMConfig updates and persists VM and TCP sysctl parameters.
func (m *Manager) SetVMConfig(swappiness, vfsPressure, dirtyRatio, dirtyBgRatio *int, tcpCongestion string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := ApplyVMConfig(swappiness, vfsPressure, dirtyRatio, dirtyBgRatio, tcpCongestion); err != nil {
		return err
	}

	if swappiness != nil {
		m.cachedCfg.Swappiness = swappiness
	}
	if vfsPressure != nil {
		m.cachedCfg.VFSCachePressure = vfsPressure
	}
	if dirtyRatio != nil {
		m.cachedCfg.DirtyRatio = dirtyRatio
	}
	if dirtyBgRatio != nil {
		m.cachedCfg.DirtyBackgroundRatio = dirtyBgRatio
	}
	if tcpCongestion != "" {
		m.cachedCfg.TCPCongestion = tcpCongestion
	}

	m.cachedCfg.ActiveProfile = "custom"
	_ = m.saveConfigLocked()

	logger.Get().Infof("kernel", "Updated VM sysctl parameters")
	return nil
}

// ApplySavedConfig applies all persistent settings upon system boot.
func (m *Manager) ApplySavedConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_ = m.loadConfig()
	cfg := m.cachedCfg

	// If named profile is set, apply profile
	if cfg.ActiveProfile != "" && cfg.ActiveProfile != "custom" {
		if profile, ok := AvailableProfiles[strings.ToLower(cfg.ActiveProfile)]; ok {
			_ = ApplyGlobalGovernor(profile.Governor)
			swap := profile.Swappiness
			vfs := profile.VFSCachePressure
			dr := profile.DirtyRatio
			dbr := profile.DirtyBackgroundRatio
			_ = ApplyVMConfig(&swap, &vfs, &dr, &dbr, cfg.TCPCongestion)
			logger.Get().Infof("kernel", "Restored persistent kernel profile at boot: %s", profile.Label)
			return nil
		}
	}

	// Otherwise apply granular cluster & VM settings
	if cfg.GlobalGovernor != "" {
		_ = ApplyGlobalGovernor(cfg.GlobalGovernor)
	}

	for _, c := range cfg.Clusters {
		_ = ApplyClusterSettings(c.PolicyID, c.Governor, c.MinFreq, c.MaxFreq)
	}

	_ = ApplyVMConfig(cfg.Swappiness, cfg.VFSCachePressure, cfg.DirtyRatio, cfg.DirtyBackgroundRatio, cfg.TCPCongestion)
	logger.Get().Infof("kernel", "Restored persistent custom kernel config at boot")
	return nil
}
