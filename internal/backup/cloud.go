package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type CloudConfig struct {
	Enabled       bool      `json:"enabled"`
	Provider      string    `json:"provider"` // "webdav"
	URL           string    `json:"url"`
	Username      string    `json:"username"`
	Password      string    `json:"password"`
	IntervalHours int       `json:"interval_hours"`
	LastSync      time.Time `json:"last_sync"`
}

type CloudManager struct {
	mu       sync.RWMutex
	config   CloudConfig
	dataPath string
}

var (
	cloudMgr  *CloudManager
	cloudOnce sync.Once
)

func getCloudStoragePath() string {
	return config.GetPersistentFilePath("cloud_config.json")
}

func GetCloudManager() *CloudManager {
	cloudOnce.Do(func() {
		cloudMgr = &CloudManager{
			dataPath: getCloudStoragePath(),
			config: CloudConfig{
				Enabled:       false,
				Provider:      "webdav",
				URL:           "",
				Username:      "",
				Password:      "",
				IntervalHours: 24,
			},
		}
		cloudMgr.loadConfig()
		cloudMgr.startSyncLoop()
	})
	return cloudMgr
}

func (m *CloudManager) loadConfig() {
	data, err := os.ReadFile(m.dataPath)
	if err != nil && m.dataPath != "cloud_config.json" {
		data, err = os.ReadFile("cloud_config.json")
	}
	if err == nil {
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		var cfg CloudConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			if cfg.IntervalHours <= 0 {
				cfg.IntervalHours = 24
			}
			if cfg.Provider == "" {
				cfg.Provider = "webdav"
			}
			m.config = cfg
		}
	}
}

func (m *CloudManager) saveConfigLocked() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(m.dataPath)
	_ = os.MkdirAll(dir, 0755)
	return config.WriteFileAtomic(m.dataPath, data, 0644)
}

func (m *CloudManager) GetConfig() CloudConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *CloudManager) SaveConfig(cfg CloudConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.IntervalHours <= 0 {
		cfg.IntervalHours = 24
	}
	if cfg.Provider == "" {
		cfg.Provider = "webdav"
	}
	cfg.LastSync = m.config.LastSync

	m.config = cfg
	return m.saveConfigLocked()
}

func CreateTarGzBackup() ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	searchDirs := []string{config.ModuleDir, "."}
	files := []string{
		"charger_config.json",
		"ssh_config.json",
		"telegram_config.json",
		"tweaks.json",
		"vnstat_data.json",
	}

	for _, fname := range files {
		for _, dir := range searchDirs {
			p := filepath.Join(dir, fname)
			data, err := os.ReadFile(p)
			if err == nil {
				hdr := &tar.Header{
					Name:    fname,
					Mode:    0644,
					Size:    int64(len(data)),
					ModTime: time.Now(),
				}
				if err := tw.WriteHeader(hdr); err == nil {
					_, _ = tw.Write(data)
				}
				break
			}
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *CloudManager) SyncBackup() error {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if cfg.URL == "" {
		return fmt.Errorf("cloud backup URL is empty")
	}

	tarData, err := CreateTarGzBackup()
	if err != nil {
		return fmt.Errorf("failed to create backup archive: %w", err)
	}

	targetURL := strings.TrimSpace(cfg.URL)
	if strings.HasSuffix(targetURL, "/") {
		fileName := fmt.Sprintf("bfr_backup_%s.tar.gz", time.Now().Format("20060102_150405"))
		targetURL = targetURL + fileName
	}

	req, err := http.NewRequest(http.MethodPut, targetURL, bytes.NewBuffer(tarData))
	if err != nil {
		return fmt.Errorf("failed to create WebDAV request: %w", err)
	}

	if cfg.Username != "" || cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	req.Header.Set("Content-Type", "application/gzip")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Get().Errorf("backup", "WebDAV cloud sync failed: %v", err)
		return fmt.Errorf("WebDAV upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Get().Errorf("backup", "WebDAV cloud sync returned status code %d", resp.StatusCode)
		return fmt.Errorf("WebDAV server returned HTTP %d", resp.StatusCode)
	}

	m.mu.Lock()
	m.config.LastSync = time.Now()
	_ = m.saveConfigLocked()
	m.mu.Unlock()

	logger.Get().Infof("backup", "Cloud backup sync completed successfully to %s", targetURL)
	return nil
}

func (m *CloudManager) startSyncLoop() {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			m.mu.RLock()
			enabled := m.config.Enabled
			intervalHours := m.config.IntervalHours
			lastSync := m.config.LastSync
			m.mu.RUnlock()

			if !enabled || intervalHours <= 0 {
				continue
			}

			if time.Since(lastSync) >= time.Duration(intervalHours)*time.Hour {
				_ = m.SyncBackup()
			}
		}
	}()
}

func init() {
	GetCloudManager()
}
