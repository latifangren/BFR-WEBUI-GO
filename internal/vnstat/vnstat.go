package vnstat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
)

// DayRecord represents daily total bandwidth consumption for a single day.
type DayRecord struct {
	Date    string `json:"date"`
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
	Total   uint64 `json:"total"`
}

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
	Daily        PeriodStats `json:"daily"`
	Yesterday    PeriodStats `json:"yesterday"`
	Weekly       PeriodStats `json:"weekly"`
	Monthly      PeriodStats `json:"monthly"`
	DailyHistory []DayRecord `json:"daily_history"`
	LastUpdated  time.Time   `json:"last_updated"`
}

type PersistentData struct {
	LastDay      string                   `json:"last_day"`
	LastWeek     string                   `json:"last_week"`
	LastMonth    string                   `json:"last_month"`
	Daily        map[string]InterfaceStat `json:"daily"`
	Yesterday    map[string]InterfaceStat `json:"yesterday"`
	Weekly       map[string]InterfaceStat `json:"weekly"`
	Monthly      map[string]InterfaceStat `json:"monthly"`
	DailyHistory []DayRecord              `json:"daily_history"`
	Updated      time.Time                `json:"updated"`
}

type rawDevStat struct {
	rxBytes uint64
	txBytes uint64
}

type Tracker struct {
	mu           sync.RWMutex
	dataPath     string
	lastDay      string
	lastWeek     string
	lastMonth    string
	daily        map[string]InterfaceStat
	yesterday    map[string]InterfaceStat
	weekly       map[string]InterfaceStat
	monthly      map[string]InterfaceStat
	dailyHistory []DayRecord
	lastRaw      map[string]rawDevStat
	lastUpdated  time.Time
}

var (
	globalTracker *Tracker
	once          sync.Once
)

func getStoragePath() string {
	return config.GetPersistentFilePath("vnstat_data.json")
}

func newTracker() *Tracker {
	t := &Tracker{
		dataPath:     getStoragePath(),
		daily:        make(map[string]InterfaceStat),
		yesterday:    make(map[string]InterfaceStat),
		weekly:       make(map[string]InterfaceStat),
		monthly:      make(map[string]InterfaceStat),
		dailyHistory: make([]DayRecord, 0),
		lastRaw:      make(map[string]rawDevStat),
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
			t.lastWeek = pd.LastWeek
			t.lastMonth = pd.LastMonth
			if pd.Daily != nil {
				t.daily = pd.Daily
			}
			if pd.Yesterday != nil {
				t.yesterday = pd.Yesterday
			}
			if pd.Weekly != nil {
				t.weekly = pd.Weekly
			}
			if pd.Monthly != nil {
				t.monthly = pd.Monthly
			}
			if pd.DailyHistory != nil {
				t.dailyHistory = pd.DailyHistory
			}
			if !pd.Updated.IsZero() {
				t.lastUpdated = pd.Updated
			}
		}
	}
}

func (t *Tracker) saveLocked() {
	data := PersistentData{
		LastDay:      t.lastDay,
		LastWeek:     t.lastWeek,
		LastMonth:    t.lastMonth,
		Daily:        t.daily,
		Yesterday:    t.yesterday,
		Weekly:       t.weekly,
		Monthly:      t.monthly,
		DailyHistory: t.dailyHistory,
		Updated:      time.Now(),
	}
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(t.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.dataPath = "vnstat_data.json"
	}
	if err := config.WriteFileAtomic(t.dataPath, buf, 0644); err != nil {
		if t.dataPath != "vnstat_data.json" {
			t.dataPath = "vnstat_data.json"
			_ = config.WriteFileAtomic(t.dataPath, buf, 0644)
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
	year, week := now.ISOWeek()
	thisWeek := fmt.Sprintf("%04d-W%02d", year, week)
	thisMonth := now.Format("2006-01")

	if t.lastDay != today {
		if t.lastDay != "" {
			var dayRx, dayTx uint64
			for _, s := range t.daily {
				dayRx += s.RxBytes
				dayTx += s.TxBytes
			}
			record := DayRecord{
				Date:    t.lastDay,
				RxBytes: dayRx,
				TxBytes: dayTx,
				Total:   dayRx + dayTx,
			}
			t.dailyHistory = append(t.dailyHistory, record)
			if len(t.dailyHistory) > 7 {
				t.dailyHistory = t.dailyHistory[len(t.dailyHistory)-7:]
			}
			t.yesterday = t.daily
		}
		t.daily = make(map[string]InterfaceStat)
		t.lastDay = today
	}

	if t.lastWeek != thisWeek {
		t.weekly = make(map[string]InterfaceStat)
		t.lastWeek = thisWeek
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

			w := t.weekly[iface]
			w.RxBytes += deltaRX
			w.TxBytes += deltaTX
			t.weekly[iface] = w

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
	return t.ResetStatsAt(time.Now())
}

// ResetStatsAt resets all counters and initializes date markers to the given timestamp.
func (t *Tracker) ResetStatsAt(now time.Time) StatsResponse {
	t.mu.Lock()
	defer t.mu.Unlock()

	year, week := now.ISOWeek()
	t.lastDay = now.Format("2006-01-02")
	t.lastWeek = fmt.Sprintf("%04d-W%02d", year, week)
	t.lastMonth = now.Format("2006-01")
	t.daily = make(map[string]InterfaceStat)
	t.yesterday = make(map[string]InterfaceStat)
	t.weekly = make(map[string]InterfaceStat)
	t.monthly = make(map[string]InterfaceStat)
	t.dailyHistory = make([]DayRecord, 0)
	t.lastRaw = parseProcNetDev()
	t.lastUpdated = now
	t.saveLocked()

	return t.getStatsLocked()
}

func computePeriodStats(statsMap map[string]InterfaceStat) PeriodStats {
	var ps PeriodStats
	ps.Interfaces = make(map[string]InterfaceStat)
	var rx, tx uint64
	for k, v := range statsMap {
		ps.Interfaces[k] = v
		rx += v.RxBytes
		tx += v.TxBytes
	}
	ps.Total = TotalStat{
		RxBytes: rx,
		TxBytes: tx,
		Total:   rx + tx,
	}
	return ps
}

func (t *Tracker) getStatsLocked() StatsResponse {
	var resp StatsResponse
	resp.Daily = computePeriodStats(t.daily)
	resp.Yesterday = computePeriodStats(t.yesterday)
	resp.Weekly = computePeriodStats(t.weekly)
	resp.Monthly = computePeriodStats(t.monthly)
	if t.dailyHistory != nil {
		resp.DailyHistory = make([]DayRecord, len(t.dailyHistory))
		copy(resp.DailyHistory, t.dailyHistory)
	} else {
		resp.DailyHistory = make([]DayRecord, 0)
	}

	dailyEmpty := (len(t.daily) == 0 || resp.Daily.Total.Total == 0)
	weeklyEmpty := (len(t.weekly) == 0 || resp.Weekly.Total.Total == 0)
	monthlyEmpty := (len(t.monthly) == 0 || resp.Monthly.Total.Total == 0)

	if dailyEmpty || weeklyEmpty || monthlyEmpty {
		rawStats := parseProcNetDev()
		for iface, raw := range rawStats {
			lower := strings.ToLower(iface)
			isPhysical := strings.HasPrefix(lower, "wlan") ||
				strings.HasPrefix(lower, "ap") ||
				strings.HasPrefix(lower, "softap") ||
				strings.HasPrefix(lower, "rmnet") ||
				strings.HasPrefix(lower, "ccmni") ||
				strings.HasPrefix(lower, "wwan")

			if isPhysical {
				if dailyEmpty {
					resp.Daily.Interfaces[iface] = InterfaceStat{
						RxBytes: raw.rxBytes,
						TxBytes: raw.txBytes,
					}
					resp.Daily.Total.RxBytes += raw.rxBytes
					resp.Daily.Total.TxBytes += raw.txBytes
				}
				if weeklyEmpty {
					resp.Weekly.Interfaces[iface] = InterfaceStat{
						RxBytes: raw.rxBytes,
						TxBytes: raw.txBytes,
					}
					resp.Weekly.Total.RxBytes += raw.rxBytes
					resp.Weekly.Total.TxBytes += raw.txBytes
				}
				if monthlyEmpty {
					resp.Monthly.Interfaces[iface] = InterfaceStat{
						RxBytes: raw.rxBytes,
						TxBytes: raw.txBytes,
					}
					resp.Monthly.Total.RxBytes += raw.rxBytes
					resp.Monthly.Total.TxBytes += raw.txBytes
				}
			}
		}
		resp.Daily.Total.Total = resp.Daily.Total.RxBytes + resp.Daily.Total.TxBytes
		resp.Weekly.Total.Total = resp.Weekly.Total.RxBytes + resp.Weekly.Total.TxBytes
		resp.Monthly.Total.Total = resp.Monthly.Total.RxBytes + resp.Monthly.Total.TxBytes
	}

	resp.LastUpdated = t.lastUpdated
	return resp
}

// CheckRollover manually invokes checkRolloverLocked with a target timestamp (used for testing rollover).
func (t *Tracker) CheckRollover(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.checkRolloverLocked(now)
}

// AddTrafficDelta adds a traffic delta for an interface (used for testing).
func (t *Tracker) AddTrafficDelta(iface string, rx, tx uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	d := t.daily[iface]
	d.RxBytes += rx
	d.TxBytes += tx
	t.daily[iface] = d

	w := t.weekly[iface]
	w.RxBytes += rx
	w.TxBytes += tx
	t.weekly[iface] = w

	m := t.monthly[iface]
	m.RxBytes += rx
	m.TxBytes += tx
	t.monthly[iface] = m
}

// GetSnapshot returns the current statistics without triggering a real-time /proc/net/dev update.
func (t *Tracker) GetSnapshot() StatsResponse {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.getStatsLocked()
}

func GetStats() StatsResponse {
	return GetTracker().GetStats()
}

func GetSnapshot() StatsResponse {
	return GetTracker().GetSnapshot()
}

func ResetStats() StatsResponse {
	return GetTracker().ResetStats()
}
