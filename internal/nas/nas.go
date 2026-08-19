package nas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type NASConfig struct {
	Enabled      bool   `json:"enabled"`
	SharePath    string `json:"share_path"`
	Port         int    `json:"port"`
	ReadOnly     bool   `json:"read_only"`
	AuthRequired bool   `json:"auth_required"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Protocol     string `json:"protocol"` // "webdav", "http"
}

type NASStatus struct {
	Active       bool   `json:"active"`
	SharePath    string `json:"share_path"`
	URL          string `json:"url"`
	Protocol     string `json:"protocol"`
	StorageUsed  string `json:"storage_used"`
	StorageTotal string `json:"storage_total"`
}

type Manager struct {
	mu        sync.RWMutex
	dataPath  string
	config    NASConfig
	active    bool
	server    *http.Server
	listener  net.Listener
	activeURL string
}

var (
	globalManager *Manager
	once          sync.Once
)

func getStoragePath() string {
	return config.GetPersistentFilePath("nas.json")
}

func newManager() *Manager {
	m := &Manager{
		dataPath: getStoragePath(),
		config: NASConfig{
			Enabled:      false,
			SharePath:    "/sdcard",
			Port:         8088,
			ReadOnly:     false,
			AuthRequired: false,
			Username:     "admin",
			Password:     "bfr",
			Protocol:     "http",
		},
	}
	_, _ = m.LoadConfig()
	return m
}

func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager()
	})
	return globalManager
}

func (m *Manager) LoadConfig() (*NASConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	buf, err := os.ReadFile(m.dataPath)
	if err != nil && m.dataPath != "nas.json" {
		buf, err = os.ReadFile("nas.json")
	}

	if err == nil {
		buf = bytes.TrimPrefix(buf, []byte("\xef\xbb\xbf"))
		var cfg NASConfig
		if err := json.Unmarshal(buf, &cfg); err == nil {
			if cfg.Port <= 0 {
				cfg.Port = 8088
			}
			if cfg.SharePath == "" {
				cfg.SharePath = "/sdcard"
			}
			if cfg.Protocol == "" {
				cfg.Protocol = "http"
			}
			m.config = cfg
			return &m.config, nil
		}
	}

	return &m.config, nil
}

func (m *Manager) SaveConfig(cfg *NASConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if cfg.Port <= 0 {
		cfg.Port = 8088
	}
	if cfg.SharePath == "" {
		cfg.SharePath = "/sdcard"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "http"
	}

	m.config = *cfg

	buf, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.dataPath = "nas.json"
	}

	if err := os.WriteFile(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "nas.json" {
			_ = os.WriteFile("nas.json", buf, 0644)
		}
		return err
	}

	return nil
}

func (m *Manager) StopNAS() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = m.server.Shutdown(ctx)
		m.server = nil
	}

	if m.listener != nil {
		_ = m.listener.Close()
		m.listener = nil
	}

	m.active = false
	m.activeURL = ""
	logger.Get().Infof("NAS", "NAS File Server stopped")
	return nil
}

func (m *Manager) StartNAS(cfg NASConfig) error {
	if err := m.SaveConfig(&cfg); err != nil {
		return err
	}

	if !cfg.Enabled {
		return m.StopNAS()
	}

	_ = m.StopNAS()

	m.mu.Lock()
	defer m.mu.Unlock()

	cleanPath := filepath.Clean(cfg.SharePath)
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("shared path does not exist: %s", cleanPath)
	}

	// Prepare FileServer handler
	fsHandler := http.FileServer(http.Dir(cleanPath))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic Auth middleware check
		if cfg.AuthRequired {
			user, pass, ok := r.BasicAuth()
			if !ok || user != cfg.Username || pass != cfg.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="BFR NAS File Server"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Read-Only mode restriction check
		if cfg.ReadOnly {
			method := strings.ToUpper(r.Method)
			if method == "PUT" || method == "POST" || method == "DELETE" || method == "MKCOL" || method == "PROPPATCH" || method == "MOVE" || method == "COPY" {
				http.Error(w, "Read-Only mode enabled", http.StatusForbidden)
				return
			}
		}

		// Support WebDAV OPTIONS query
		if r.Method == "OPTIONS" {
			w.Header().Set("Allow", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PROPFIND, MKCOL")
			w.Header().Set("DAV", "1, 2")
			w.WriteHeader(http.StatusOK)
			return
		}

		fsHandler.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind NAS port %d: %w", cfg.Port, err)
	}

	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
	}

	m.server = server
	m.listener = listener
	m.active = true
	m.activeURL = fmt.Sprintf("http://localhost:%d", cfg.Port)

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Get().Errorf("NAS", "NAS server error: %v", err)
		}
	}()

	logger.Get().Infof("NAS", "NAS File Server started on port %d (path: %s, protocol: %s)", cfg.Port, cleanPath, cfg.Protocol)
	return nil
}

func (m *Manager) GetStatus() NASStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg := m.config
	used, total := getStorageStats(cfg.SharePath)

	url := m.activeURL
	if url == "" && cfg.Enabled {
		url = fmt.Sprintf("http://localhost:%d", cfg.Port)
	}

	return NASStatus{
		Active:       m.active,
		SharePath:    cfg.SharePath,
		URL:          url,
		Protocol:     cfg.Protocol,
		StorageUsed:  used,
		StorageTotal: total,
	}
}
