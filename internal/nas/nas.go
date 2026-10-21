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
	"golang.org/x/net/webdav"
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
	Active       bool     `json:"active"`
	SharePath    string   `json:"share_path"`
	URL          string   `json:"url"`
	Protocol     string   `json:"protocol"`
	Port         int      `json:"port"`
	ReadOnly     bool     `json:"read_only"`
	AuthRequired bool     `json:"auth_required"`
	IPs          []string `json:"ips"`
	PrimaryIP    string   `json:"primary_ip"`
	MountURLs    []string `json:"mount_urls"`
	StorageUsed  string   `json:"storage_used"`
	StorageTotal string   `json:"storage_total"`
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
			Port:         85,
			ReadOnly:     true,
			AuthRequired: true,
			Username:     "",
			Password:     "",
			Protocol:     "webdav",
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

// GetLocalIPs scans active network interfaces and returns non-loopback IPv4 addresses.
func GetLocalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	seen := make(map[string]bool)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				ipStr := ip4.String()
				if !seen[ipStr] {
					seen[ipStr] = true
					ips = append(ips, ipStr)
				}
			}
		}
	}
	return ips
}

// GetPrimaryIP selects the primary local IPv4 address, prioritizing Wi-Fi, hotspot, and Ethernet interfaces.
func GetPrimaryIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	prefOrder := []string{"wlan0", "ap0", "wlan1", "rndis0", "eth0", "usb0", "tun0"}
	type ifaceIP struct {
		name string
		ip   string
	}
	var list []ifaceIP

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				list = append(list, ifaceIP{name: iface.Name, ip: ip4.String()})
			}
		}
	}

	for _, pref := range prefOrder {
		for _, item := range list {
			if strings.EqualFold(item.name, pref) {
				return item.ip
			}
		}
	}

	if len(list) > 0 {
		return list[0].ip
	}
	return "127.0.0.1"
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
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(buf, &fields); err == nil {
				if _, exists := fields["read_only"]; !exists {
					cfg.ReadOnly = true
				}
				if _, exists := fields["auth_required"]; !exists {
					cfg.AuthRequired = true
				}
			}
			if cfg.Port <= 0 {
				cfg.Port = 85
			}
			if cfg.SharePath == "" {
				cfg.SharePath = "/sdcard"
			}
			if cfg.Protocol == "" {
				cfg.Protocol = "webdav"
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
		cfg.Port = 85
	}
	if cfg.SharePath == "" {
		cfg.SharePath = "/sdcard"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "webdav"
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

	if err := config.WriteFileAtomic(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "nas.json" {
			_ = config.WriteFileAtomic("nas.json", buf, 0644)
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
	if cfg.AuthRequired && (strings.TrimSpace(cfg.Username) == "" || cfg.Password == "") {
		return fmt.Errorf("NAS authentication requires a non-empty username and password")
	}

	_ = m.StopNAS()

	m.mu.Lock()
	defer m.mu.Unlock()

	cleanPath := filepath.Clean(cfg.SharePath)
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("shared path does not exist: %s", cleanPath)
	}

	var baseHandler http.Handler
	if strings.EqualFold(cfg.Protocol, "http") {
		fileServer := http.FileServer(http.Dir(cleanPath))
		baseHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Allow", "GET, HEAD, OPTIONS")
				w.WriteHeader(http.StatusOK)
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	} else {
		baseHandler = &webdav.Handler{
			FileSystem: webdav.Dir(cleanPath),
			LockSystem: webdav.NewMemLS(),
		}
	}

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

		// Read-Only mode restriction check: reject mutation methods
		if cfg.ReadOnly {
			switch strings.ToUpper(r.Method) {
			case "PUT", "POST", "DELETE", "MKCOL", "PROPPATCH", "MOVE", "COPY", "LOCK":
				http.Error(w, "Read-Only mode enabled", http.StatusForbidden)
				return
			}
		}

		baseHandler.ServeHTTP(w, r)
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

	primaryIP := GetPrimaryIP()
	m.server = server
	m.listener = listener
	m.active = true
	m.activeURL = fmt.Sprintf("http://%s:%d", primaryIP, cfg.Port)

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

	ips := GetLocalIPs()
	primaryIP := GetPrimaryIP()

	mountURLs := make([]string, 0, len(ips))
	for _, ip := range ips {
		mountURLs = append(mountURLs, fmt.Sprintf("http://%s:%d", ip, cfg.Port))
	}
	if len(mountURLs) == 0 {
		mountURLs = append(mountURLs, fmt.Sprintf("http://127.0.0.1:%d", cfg.Port))
	}

	url := m.activeURL
	if url == "" && cfg.Enabled {
		url = fmt.Sprintf("http://%s:%d", primaryIP, cfg.Port)
	}

	return NASStatus{
		Active:       m.active,
		SharePath:    cfg.SharePath,
		URL:          url,
		Protocol:     cfg.Protocol,
		Port:         cfg.Port,
		ReadOnly:     cfg.ReadOnly,
		AuthRequired: cfg.AuthRequired,
		IPs:          ips,
		PrimaryIP:    primaryIP,
		MountURLs:    mountURLs,
		StorageUsed:  used,
		StorageTotal: total,
	}
}
