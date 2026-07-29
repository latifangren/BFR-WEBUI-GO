package vnstat

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type InterfaceStat struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type TotalStat struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
	Total   uint64 `json:"total"`
}

type PeriodStats struct {
	Total      TotalStat                `json:"total"`
	Interfaces map[string]InterfaceStat `json:"interfaces"`
}

type StatsResponse struct {
	Daily       PeriodStats `json:"daily"`
	Monthly     PeriodStats `json:"monthly"`
	LastUpdated time.Time   `json:"last_updated"`
}

type PersistentData struct {
	LastDay   string                   `json:"last_day"`
	LastMonth string                   `json:"last_month"`
	Daily     map[string]InterfaceStat `json:"daily"`
	Monthly   map[string]InterfaceStat `json:"monthly"`
	Updated   time.Time                `json:"updated"`
}

type rawDevStat struct {
	rxBytes uint64
	txBytes uint64
}

type Tracker struct {
	mu          sync.RWMutex
	dataPath    string
	lastDay     string
	lastMonth   string
	daily       map[string]InterfaceStat
	monthly     map[string]InterfaceStat
	lastRaw     map[string]rawDevStat
	lastUpdated time.Time
}

var (
	globalTracker *Tracker
	once          sync.Once
)

func getStoragePath() string {
	magiskDir := "/data/adb/modules/bfr_webui_go"
	magiskPath := filepath.Join(magiskDir, "vnstat_data.json")
	if _, err := os.Stat(magiskDir); err == nil {
		return magiskPath
	}
	if _, err := os.Stat(magiskPath); err == nil {
		return magiskPath
	}
	if _, err := os.Stat("vnstat_data.json"); err == nil {
		return "vnstat_data.json"
	}
	return magiskPath
}

func newTracker() *Tracker {
	t := &Tracker{
		dataPath: getStoragePath(),
		daily:    make(map[string]InterfaceStat),
		monthly:  make(map[string]InterfaceStat),
		lastRaw:  make(map[string]rawDevStat),
	}
	t.load()
	return t
}

func GetTracker() *Tracker {
	once.Do(func() {
		globalTracker = newTracker()
		globalTracker.start()
	})
	return globalTracker
}

func init() {
	GetTracker()
}

func (t *Tracker) load() {
	buf, err := os.ReadFile(t.dataPath)
	if err != nil && t.dataPath != "vnstat_data.json" {
		buf, err = os.ReadFile("vnstat_data.json")
		if err == nil {
			t.dataPath = "vnstat_data.json"
		}
	}
	if err == nil {
		var pd PersistentData
		if err := json.Unmarshal(buf, &pd); err == nil {
			t.lastDay = pd.LastDay
			t.lastMonth = pd.LastMonth
			if pd.Daily != nil {
				t.daily = pd.Daily
			}
			if pd.Monthly != nil {
				t.monthly = pd.Monthly
			}
			if !pd.Updated.IsZero() {
				t.lastUpdated = pd.Updated
			}
		}
	}
}

func (t *Tracker) saveLocked() {
	data := PersistentData{
		LastDay:   t.lastDay,
		LastMonth: t.lastMonth,
		Daily:     t.daily,
		Monthly:   t.monthly,
		Updated:   time.Now(),
	}
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(t.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.dataPath = "vnstat_data.json"
	}
	if err := os.WriteFile(t.dataPath, buf, 0644); err != nil {
		if t.dataPath != "vnstat_data.json" {
			t.dataPath = "vnstat_data.json"
			_ = os.WriteFile(t.dataPath, buf, 0644)
		}
	}
}

func parseProcNetDev() map[string]rawDevStat {
	res := make(map[string]rawDevStat)
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return res
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" || iface == "" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) >= 9 {
			rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
			txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
			res[iface] = rawDevStat{
				rxBytes: rxBytes,
				txBytes: txBytes,
			}
		}
	}
	return res
}

func (t *Tracker) checkRolloverLocked(now time.Time) {
	today := now.Format("2006-01-02")
	thisMonth := now.Format("2006-01")

	if t.lastDay != today {
		t.daily = make(map[string]InterfaceStat)
		t.lastDay = today
	}

	if t.lastMonth != thisMonth {
		t.monthly = make(map[string]InterfaceStat)
		t.lastMonth = thisMonth
	}
}

func (t *Tracker) updateLocked() {
	now := time.Now()
	t.checkRolloverLocked(now)

	currentDevStats := parseProcNetDev()

	for iface, curr := range currentDevStats {
		prev, exists := t.lastRaw[iface]
		if !exists {
			t.lastRaw[iface] = curr
			continue
		}

		var deltaRX, deltaTX uint64
		if curr.rxBytes >= prev.rxBytes {
			deltaRX = curr.rxBytes - prev.rxBytes
		} else {
			deltaRX = curr.rxBytes
		}

		if curr.txBytes >= prev.txBytes {
			deltaTX = curr.txBytes - prev.txBytes
		} else {
			deltaTX = curr.txBytes
		}

		t.lastRaw[iface] = curr

		if deltaRX > 0 || deltaTX > 0 {
			d := t.daily[iface]
			d.RxBytes += deltaRX
			d.TxBytes += deltaTX
			t.daily[iface] = d

			m := t.monthly[iface]
			m.RxBytes += deltaRX
			m.TxBytes += deltaTX
			t.monthly[iface] = m
		}
	}

	t.lastUpdated = now
	t.saveLocked()
}

func (t *Tracker) start() {
	t.mu.Lock()
	t.updateLocked()
	t.mu.Unlock()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			t.mu.Lock()
			t.updateLocked()
			t.mu.Unlock()
		}
	}()
}

func (t *Tracker) GetStats() StatsResponse {
	t.mu.Lock()
	t.updateLocked()
	res := t.getStatsLocked()
	t.mu.Unlock()
	return res
}

func (t *Tracker) ResetStats() StatsResponse {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.lastDay = now.Format("2006-01-02")
	t.lastMonth = now.Format("2006-01")
	t.daily = make(map[string]InterfaceStat)
	t.monthly = make(map[string]InterfaceStat)
	t.lastRaw = parseProcNetDev()
	t.lastUpdated = now
	t.saveLocked()

	return t.getStatsLocked()
}

func (t *Tracker) getStatsLocked() StatsResponse {
	var resp StatsResponse
	resp.Daily.Interfaces = make(map[string]InterfaceStat)
	resp.Monthly.Interfaces = make(map[string]InterfaceStat)

	var dRx, dTx, mRx, mTx uint64

	for k, v := range t.daily {
		resp.Daily.Interfaces[k] = v
		dRx += v.RxBytes
		dTx += v.TxBytes
	}
	resp.Daily.Total = TotalStat{
		RxBytes: dRx,
		TxBytes: dTx,
		Total:   dRx + dTx,
	}

	for k, v := range t.monthly {
		resp.Monthly.Interfaces[k] = v
		mRx += v.RxBytes
		mTx += v.TxBytes
	}
	resp.Monthly.Total = TotalStat{
		RxBytes: mRx,
		TxBytes: mTx,
		Total:   mRx + mTx,
	}

	resp.LastUpdated = t.lastUpdated
	return resp
}

func GetStats() StatsResponse {
	return GetTracker().GetStats()
}

func ResetStats() StatsResponse {
	return GetTracker().ResetStats()
}
