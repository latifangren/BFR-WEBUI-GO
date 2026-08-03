package speedtest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type SpeedtestResult struct {
	PingMs       float64 `json:"ping_ms"`
	JitterMs     float64 `json:"jitter_ms"`
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	ProgressPct  int     `json:"progress_pct"`
	Phase        string  `json:"phase"` // "idle", "ping", "download", "upload", "complete", "error"
	Running      bool    `json:"running"`
	Error        string  `json:"error,omitempty"`
	ClientIP     string  `json:"client_ip"`
	ISP          string  `json:"isp"`
	Location     string  `json:"location"`
	ServerColo   string  `json:"server_colo"`
	ServerName   string  `json:"server_name"`
}

type HistoryEntry struct {
	Timestamp    string  `json:"timestamp"`
	PingMs       float64 `json:"ping"`
	JitterMs     float64 `json:"jitter"`
	DownloadMbps float64 `json:"download"`
	UploadMbps   float64 `json:"upload"`
	ClientIP     string  `json:"client_ip"`
	ISP          string  `json:"isp"`
	ServerName   string  `json:"server_name"`
}

type Manager struct {
	mu         sync.RWMutex
	dataPath   string
	result     SpeedtestResult
	history    []HistoryEntry
	cancelFunc context.CancelFunc
}

var (
	globalManager *Manager
	once          sync.Once

	bufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 32768)
			return &buf
		},
	}
)

func getSpeedtestHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 3 * time.Second}
				if !strings.Contains(address, "127.0.0.1") && !strings.Contains(address, "[::1]") && !strings.Contains(address, "localhost") {
					if conn, err := d.DialContext(ctx, network, address); err == nil {
						return conn, nil
					}
				}
				for _, dnsServer := range []string{"1.1.1.1:53", "8.8.8.8:53", "9.9.9.9:53"} {
					if conn, err := d.DialContext(ctx, "udp", dnsServer); err == nil {
						return conn, nil
					}
				}
				return d.DialContext(ctx, "udp", "1.1.1.1:53")
			},
		},
	}

	tr := &http.Transport{
		DialContext:     dialer.DialContext,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
}

func parseCloudflareTrace(body string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result
}

func getColoCity(colo string) string {
	colo = strings.ToUpper(strings.TrimSpace(colo))
	iataMap := map[string]string{
		"CGK": "Jakarta",
		"SUB": "Surabaya",
		"DPS": "Bali",
		"KNO": "Medan",
		"BPN": "Balikpapan",
		"UPG": "Makassar",
		"SIN": "Singapore",
		"KUL": "Kuala Lumpur",
		"BKK": "Bangkok",
		"MNL": "Manila",
		"HKG": "Hong Kong",
		"NRT": "Tokyo",
		"HND": "Tokyo",
		"ICN": "Seoul",
		"TPE": "Taipei",
		"SYD": "Sydney",
		"MEL": "Melbourne",
		"LAX": "Los Angeles",
		"SFO": "San Francisco",
		"SEA": "Seattle",
		"ORD": "Chicago",
		"JFK": "New York",
		"LHR": "London",
		"FRA": "Frankfurt",
		"AMS": "Amsterdam",
	}
	if city, ok := iataMap[colo]; ok {
		return city
	}
	return colo
}

func fetchClientAndISPInfo(ctx context.Context, client *http.Client) (clientIP, isp, location, colo, serverName string) {
	// 1. Fetch Cloudflare trace
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://1.1.1.1/cdn-cgi/trace", nil)
	if err == nil {
		resp, err := client.Do(req)
		if err == nil {
			bodyBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				kv := parseCloudflareTrace(string(bodyBytes))
				clientIP = kv["ip"]
				colo = kv["colo"]
			}
		}
	}

	// 2. Fetch ISP and location info via ip-api.com
	reqApi, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip-api.com/json/?fields=query,isp,org,city,country,countryCode", nil)
	if err == nil {
		respApi, err := client.Do(reqApi)
		if err == nil {
			var apiRes struct {
				Query       string `json:"query"`
				ISP         string `json:"isp"`
				Org         string `json:"org"`
				City        string `json:"city"`
				Country     string `json:"country"`
				CountryCode string `json:"countryCode"`
			}
			if json.NewDecoder(respApi.Body).Decode(&apiRes) == nil {
				_ = respApi.Body.Close()
				if clientIP == "" {
					clientIP = apiRes.Query
				}
				if apiRes.ISP != "" {
					isp = apiRes.ISP
				} else if apiRes.Org != "" {
					isp = apiRes.Org
				}
				if apiRes.City != "" && apiRes.CountryCode != "" {
					location = fmt.Sprintf("%s, %s", apiRes.City, apiRes.CountryCode)
				} else if apiRes.Country != "" {
					location = apiRes.Country
				}
			} else {
				_ = respApi.Body.Close()
			}
		}
	}

	// 3. Fallback to ipinfo.io if ISP or location is missing
	if isp == "" || location == "" {
		reqInfo, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipinfo.io/json", nil)
		if err == nil {
			respInfo, err := client.Do(reqInfo)
			if err == nil {
				var infoRes struct {
					IP      string `json:"ip"`
					City    string `json:"city"`
					Country string `json:"country"`
					Org     string `json:"org"`
				}
				if json.NewDecoder(respInfo.Body).Decode(&infoRes) == nil {
					_ = respInfo.Body.Close()
					if clientIP == "" {
						clientIP = infoRes.IP
					}
					if isp == "" && infoRes.Org != "" {
						parts := strings.SplitN(infoRes.Org, " ", 2)
						if len(parts) > 1 {
							isp = parts[1]
						} else {
							isp = infoRes.Org
						}
					}
					if location == "" && infoRes.City != "" {
						location = fmt.Sprintf("%s, %s", infoRes.City, infoRes.Country)
					}
				} else {
					_ = respInfo.Body.Close()
				}
			}
		}
	}

	if clientIP == "" {
		clientIP = "Unknown"
	}
	if isp == "" {
		isp = "Unknown Carrier"
	}
	if location == "" {
		location = "Global"
	}
	if colo == "" {
		colo = "CDN"
	}

	cityName := getColoCity(colo)
	serverName = fmt.Sprintf("Cloudflare Anycast - %s (%s)", cityName, colo)
	return
}

func getStoragePath() string {
	return config.GetPersistentFilePath("speedtest_history.json")
}

func GetManager() *Manager {
	once.Do(func() {
		globalManager = &Manager{
			dataPath: getStoragePath(),
			result: SpeedtestResult{
				Phase: "idle",
			},
			history: []HistoryEntry{},
		}
		globalManager.loadHistory()
	})
	return globalManager
}

func (m *Manager) loadHistory() {
	if data, err := os.ReadFile(m.dataPath); err == nil {
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		var hist []HistoryEntry
		if err := json.Unmarshal(data, &hist); err == nil {
			m.history = hist
		}
	}
}

func (m *Manager) saveHistoryLocked() {
	data, err := json.MarshalIndent(m.history, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(m.dataPath)
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(m.dataPath, data, 0644)
}

func (m *Manager) GetStatus() SpeedtestResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.result
}

func (m *Manager) GetHistory() []HistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.history == nil {
		return []HistoryEntry{}
	}
	hist := make([]HistoryEntry, len(m.history))
	copy(hist, m.history)
	return hist
}

func (m *Manager) updateResult(fn func(r *SpeedtestResult)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(&m.result)
}

func (m *Manager) StartTest() (SpeedtestResult, error) {
	m.mu.Lock()
	if m.result.Running {
		res := m.result
		m.mu.Unlock()
		return res, fmt.Errorf("speedtest is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	m.result = SpeedtestResult{
		Phase:       "ping",
		ProgressPct: 0,
		Running:     true,
	}
	m.mu.Unlock()

	go m.runTest(ctx)
	return m.GetStatus(), nil
}

func (m *Manager) StopTest() SpeedtestResult {
	m.mu.Lock()
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
	m.result.Running = false
	m.result.Phase = "idle"
	m.result.ProgressPct = 0
	res := m.result
	m.mu.Unlock()
	return res
}

func (m *Manager) runTest(ctx context.Context) {
	defer func() {
		m.mu.Lock()
		m.result.Running = false
		m.cancelFunc = nil
		m.mu.Unlock()
	}()

	client := getSpeedtestHTTPClient(8 * time.Second)

	// Phase 1: Ping & Jitter & ISP/Colo Detection (Progress 0% -> 20%)
	m.updateResult(func(r *SpeedtestResult) {
		r.Phase = "ping"
		r.ProgressPct = 5
	})

	clientIP, isp, location, colo, serverName := fetchClientAndISPInfo(ctx, client)
	m.updateResult(func(r *SpeedtestResult) {
		r.ClientIP = clientIP
		r.ISP = isp
		r.Location = location
		r.ServerColo = colo
		r.ServerName = serverName
	})

	pingEndpoints := []string{
		"https://1.1.1.1/cdn-cgi/trace",
		"https://speed.cloudflare.com/__down?bytes=0",
		"http://1.1.1.1/",
	}

	var pings []float64
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		for _, pingURL := range pingEndpoints {
			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
			if err != nil {
				continue
			}
			resp, err := client.Do(req)
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				latency := float64(time.Since(start).Microseconds()) / 1000.0
				pings = append(pings, latency)
				break
			}
		}

		m.updateResult(func(r *SpeedtestResult) {
			r.ProgressPct = 5 + (i+1)*3
		})
		time.Sleep(100 * time.Millisecond)
	}

	if len(pings) > 0 {
		var sum float64
		for _, p := range pings {
			sum += p
		}
		avgPing := sum / float64(len(pings))

		var jitterSum float64
		if len(pings) > 1 {
			for i := 1; i < len(pings); i++ {
				jitterSum += math.Abs(pings[i] - pings[i-1])
			}
			jitterSum /= float64(len(pings) - 1)
		}

		m.updateResult(func(r *SpeedtestResult) {
			r.PingMs = math.Round(avgPing*100) / 100
			r.JitterMs = math.Round(jitterSum*100) / 100
		})
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	// Phase 2: Download Test (Progress 20% -> 60%)
	m.updateResult(func(r *SpeedtestResult) {
		r.Phase = "download"
		r.ProgressPct = 20
	})

	downloadURLs := []string{
		"https://speed.cloudflare.com/__down?bytes=25000000",
		"http://proof.ovh.net/files/10Mb.dat",
		"http://speed.hetzner.de/10MB.bin",
	}

	var downloadedBytes int64
	dlWorkers := 4
	dlDuration := 6 * time.Second
	dlCtx, dlCancel := context.WithTimeout(ctx, dlDuration)
	defer dlCancel()

	var dlWg sync.WaitGroup
	dlStart := time.Now()

	for i := 0; i < dlWorkers; i++ {
		dlWg.Add(1)
		go func(workerID int) {
			defer dlWg.Done()
			bufPtr := bufferPool.Get().(*[]byte)
			defer bufferPool.Put(bufPtr)
			buf := *bufPtr
			urlIdx := workerID % len(downloadURLs)

			for {
				select {
				case <-dlCtx.Done():
					return
				default:
				}

				targetURL := downloadURLs[urlIdx]
				urlIdx = (urlIdx + 1) % len(downloadURLs)

				req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, targetURL, nil)
				if err != nil {
					continue
				}
				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				for {
					n, err := resp.Body.Read(buf)
					if n > 0 {
						atomic.AddInt64(&downloadedBytes, int64(n))
					}
					if err != nil {
						break
					}
					select {
					case <-dlCtx.Done():
						resp.Body.Close()
						return
					default:
					}
				}
				resp.Body.Close()
			}
		}(i)
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case <-dlCtx.Done():
			ticker.Stop()
			goto DL_FINISHED
		case <-ticker.C:
			elapsed := time.Since(dlStart).Seconds()
			if elapsed > 0 {
				bytesRead := atomic.LoadInt64(&downloadedBytes)
				mbps := (float64(bytesRead) * 8) / (elapsed * 1000000)
				pct := 20 + int((elapsed/dlDuration.Seconds())*40)
				if pct > 60 {
					pct = 60
				}
				m.updateResult(func(r *SpeedtestResult) {
					r.DownloadMbps = math.Round(mbps*100) / 100
					r.ProgressPct = pct
				})
			}
		}
	}

DL_FINISHED:
	dlWg.Wait()
	dlElapsed := time.Since(dlStart).Seconds()
	if dlElapsed > 0 {
		finalMbps := (float64(atomic.LoadInt64(&downloadedBytes)) * 8) / (dlElapsed * 1000000)
		m.updateResult(func(r *SpeedtestResult) {
			r.DownloadMbps = math.Round(finalMbps*100) / 100
			r.ProgressPct = 60
		})
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	// Phase 3: Upload Test (Progress 60% -> 100%)
	m.updateResult(func(r *SpeedtestResult) {
		r.Phase = "upload"
		r.ProgressPct = 60
	})

	uploadURL := "https://speed.cloudflare.com/__up"
	payloadSize := 1024 * 1024
	dummyData := make([]byte, payloadSize)
	_, _ = rand.Read(dummyData)

	var uploadedBytes int64
	ulWorkers := 4
	ulDuration := 6 * time.Second
	ulCtx, ulCancel := context.WithTimeout(ctx, ulDuration)
	defer ulCancel()

	var ulWg sync.WaitGroup
	ulStart := time.Now()

	for i := 0; i < ulWorkers; i++ {
		ulWg.Add(1)
		go func() {
			defer ulWg.Done()
			for {
				select {
				case <-ulCtx.Done():
					return
				default:
				}

				req, err := http.NewRequestWithContext(ulCtx, http.MethodPost, uploadURL, bytes.NewReader(dummyData))
				if err != nil {
					return
				}
				req.Header.Set("Content-Type", "application/octet-stream")

				resp, err := client.Do(req)
				if err == nil {
					atomic.AddInt64(&uploadedBytes, int64(payloadSize))
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				} else {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	ulTicker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case <-ulCtx.Done():
			ulTicker.Stop()
			goto UL_FINISHED
		case <-ulTicker.C:
			elapsed := time.Since(ulStart).Seconds()
			if elapsed > 0 {
				bytesSent := atomic.LoadInt64(&uploadedBytes)
				mbps := (float64(bytesSent) * 8) / (elapsed * 1000000)
				pct := 60 + int((elapsed/ulDuration.Seconds())*40)
				if pct > 100 {
					pct = 100
				}
				m.updateResult(func(r *SpeedtestResult) {
					r.UploadMbps = math.Round(mbps*100) / 100
					r.ProgressPct = pct
				})
			}
		}
	}

UL_FINISHED:
	ulWg.Wait()
	ulElapsed := time.Since(ulStart).Seconds()
	m.mu.Lock()
	if ulElapsed > 0 {
		finalUlMbps := (float64(atomic.LoadInt64(&uploadedBytes)) * 8) / (ulElapsed * 1000000)
		if finalUlMbps > 0 {
			m.result.UploadMbps = math.Round(finalUlMbps*100) / 100
		}
	}
	m.result.ProgressPct = 100
	m.result.Phase = "complete"

	entry := HistoryEntry{
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		PingMs:       m.result.PingMs,
		JitterMs:     m.result.JitterMs,
		DownloadMbps: m.result.DownloadMbps,
		UploadMbps:   m.result.UploadMbps,
		ClientIP:     m.result.ClientIP,
		ISP:          m.result.ISP,
		ServerName:   m.result.ServerName,
	}
	m.history = append(m.history, entry)
	if len(m.history) > 20 {
		m.history = m.history[len(m.history)-20:]
	}
	m.saveHistoryLocked()
	m.mu.Unlock()

	logger.Get().Infof("speedtest", "Speedtest completed: Ping=%.1fms, Down=%.2fMbps, Up=%.2fMbps, ISP=%s, Server=%s",
		m.result.PingMs, m.result.DownloadMbps, m.result.UploadMbps, m.result.ISP, m.result.ServerName)
}