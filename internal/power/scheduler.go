package power

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type ScheduleMode string

const (
	ScheduleModeDisabled ScheduleMode = "disabled"
	ScheduleModeInterval ScheduleMode = "interval"
	ScheduleModeClock    ScheduleMode = "clock"
)

type ScheduleConfig struct {
	Enabled       bool         `json:"enabled"`
	Mode          ScheduleMode `json:"mode"`           // "interval" or "clock"
	IntervalHours int          `json:"interval_hours"` // e.g. 2, 4, 6, 8, 12, 24
	ClockTime     string       `json:"clock_time"`     // "04:00" (HH:MM 24-hr)
}

type ScheduleStatus struct {
	Config          ScheduleConfig `json:"config"`
	UptimeSeconds   int64          `json:"uptime_seconds"`
	NextRebootInSec int64          `json:"next_reboot_in_sec"` // -1 if disabled
	NextRebootTime  string         `json:"next_reboot_time"`   // formatted "2006-01-02 15:04:05", empty if disabled
	Description     string         `json:"description"`
}

var (
	startTime = time.Now()
)

type Scheduler struct {
	mu                   sync.RWMutex
	config               ScheduleConfig
	notifier             func(msg string)
	customUptimeProvider func() int64
	customRebootExecutor func() error
	stopChan             chan struct{}
	running              bool
	lastTriggerUptime    int64
	lastTriggerClockDay  int
}

var (
	schedulerInstance *Scheduler
	schedulerOnce     sync.Once
)

func GetScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		schedulerInstance = &Scheduler{
			config: ScheduleConfig{
				Enabled:       false,
				Mode:          ScheduleModeDisabled,
				IntervalHours: 4,
				ClockTime:     "04:00",
			},
			stopChan: make(chan struct{}),
		}
		schedulerInstance.LoadConfig()
	})
	return schedulerInstance
}

func getScheduleConfigPath() string {
	return config.GetPersistentFilePath("power_schedule.json")
}

func (s *Scheduler) LoadConfig() ScheduleConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := getScheduleConfigPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return s.config
	}

	var loaded ScheduleConfig
	if err := json.Unmarshal(data, &loaded); err == nil {
		s.config = loaded
	}
	return s.config
}

func (s *Scheduler) SaveConfig(cfg ScheduleConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = cfg
	p := getScheduleConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func (s *Scheduler) GetConfig() ScheduleConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func validateScheduleConfig(cfg ScheduleConfig) error {
	switch cfg.Mode {
	case ScheduleModeDisabled:
		// Always valid
	case ScheduleModeInterval:
		if cfg.IntervalHours < 1 {
			return fmt.Errorf("interval_hours must be at least 1")
		}
	case ScheduleModeClock:
		parts := strings.Split(cfg.ClockTime, ":")
		if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 2 {
			return fmt.Errorf("clock_time must be in HH:MM format (24-hour)")
		}
		h, errH := strconv.Atoi(parts[0])
		m, errM := strconv.Atoi(parts[1])
		if errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return fmt.Errorf("clock_time must be valid 24-hour format between 00:00 and 23:59")
		}
	default:
		return fmt.Errorf("invalid mode: must be 'disabled', 'interval', or 'clock'")
	}
	return nil
}

func (s *Scheduler) UpdateConfig(cfg ScheduleConfig) error {
	if err := validateScheduleConfig(cfg); err != nil {
		return err
	}

	if !cfg.Enabled {
		cfg.Mode = ScheduleModeDisabled
	} else if cfg.Mode == ScheduleModeDisabled {
		cfg.Enabled = false
	}

	return s.SaveConfig(cfg)
}

func (s *Scheduler) getSystemUptime() int64 {
	if s.customUptimeProvider != nil {
		return s.customUptimeProvider()
	}

	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if upFloat, err := strconv.ParseFloat(fields[0], 64); err == nil && upFloat >= 0 {
				return int64(upFloat)
			}
		}
	}

	// Fallback to process uptime
	return int64(time.Since(startTime).Seconds())
}

func (s *Scheduler) GetStatus() ScheduleStatus {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	uptime := s.getSystemUptime()
	status := ScheduleStatus{
		Config:          cfg,
		UptimeSeconds:   uptime,
		NextRebootInSec: -1,
		NextRebootTime:  "",
		Description:     "Auto reboot is disabled",
	}

	if !cfg.Enabled || cfg.Mode == ScheduleModeDisabled {
		return status
	}

	now := time.Now()

	if cfg.Mode == ScheduleModeInterval {
		intervalSec := int64(cfg.IntervalHours) * 3600
		if intervalSec <= 0 {
			intervalSec = 4 * 3600
		}
		rem := intervalSec - (uptime % intervalSec)
		if rem <= 0 {
			rem = intervalSec
		}
		status.NextRebootInSec = rem
		targetTime := now.Add(time.Duration(rem) * time.Second)
		status.NextRebootTime = targetTime.Format("2006-01-02 15:04:05")
		status.Description = fmt.Sprintf("Rebooting every %d hour(s). Next reboot in %s", cfg.IntervalHours, formatDurationSec(rem))
		return status
	}

	if cfg.Mode == ScheduleModeClock {
		parts := strings.Split(cfg.ClockTime, ":")
		if len(parts) == 2 {
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			targetToday := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
			target := targetToday
			if !now.Before(targetToday) {
				target = targetToday.Add(24 * time.Hour)
			}
			remSec := int64(target.Sub(now).Seconds())
			if remSec < 0 {
				remSec = 0
			}
			status.NextRebootInSec = remSec
			status.NextRebootTime = target.Format("2006-01-02 15:04:05")
			status.Description = fmt.Sprintf("Rebooting daily at %s. Next reboot in %s", cfg.ClockTime, formatDurationSec(remSec))
			return status
		}
	}

	return status
}

func formatDurationSec(sec int64) string {
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	m := sec / 60
	h := m / 60
	m = m % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func (s *Scheduler) SetCustomUptimeProvider(fn func() int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customUptimeProvider = fn
}

func (s *Scheduler) SetCustomRebootExecutor(fn func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customRebootExecutor = fn
}

func (s *Scheduler) CheckAndTriggerReboot() bool {
	return s.checkAndTriggerReboot()
}

func (s *Scheduler) SetRebootNotifier(fn func(msg string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifier = fn
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	go s.runLoop()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopChan)
}

func (s *Scheduler) runLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndTriggerReboot()
		}
	}
}

func (s *Scheduler) checkAndTriggerReboot() bool {
	st := s.GetStatus()
	if !st.Config.Enabled || st.Config.Mode == ScheduleModeDisabled {
		return false
	}

	// If next reboot is within 15 seconds or overdue
	if st.NextRebootInSec >= 0 && st.NextRebootInSec <= 15 {
		s.mu.Lock()
		uptime := st.UptimeSeconds
		now := time.Now()

		if st.Config.Mode == ScheduleModeInterval {
			intervalSec := int64(st.Config.IntervalHours) * 3600
			if uptime-s.lastTriggerUptime < intervalSec/2 && s.lastTriggerUptime != 0 {
				s.mu.Unlock()
				return false
			}
			s.lastTriggerUptime = uptime
		} else if st.Config.Mode == ScheduleModeClock {
			yearDay := now.YearDay()
			if s.lastTriggerClockDay == yearDay {
				s.mu.Unlock()
				return false
			}
			s.lastTriggerClockDay = yearDay
		}

		notifier := s.notifier
		executor := s.customRebootExecutor
		s.mu.Unlock()

		msg := fmt.Sprintf("🔄 *[Auto Reboot]* Perangkat akan reboot sekarang sesuai jadwal (%s)...", st.Description)
		if notifier != nil {
			notifier(msg)
			time.Sleep(2 * time.Second)
		}

		logger.Get().Infof("Power", "Executing scheduled reboot (%s)", st.Description)

		if executor != nil {
			_ = executor()
		} else {
			_ = Execute(ActionReboot)
		}
		return true
	}

	return false
}
