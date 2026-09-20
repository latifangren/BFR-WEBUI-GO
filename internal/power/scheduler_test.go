package power_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"bfr-webui-go/internal/power"
)

func TestScheduler_ConfigValidation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	s := power.GetScheduler()

	// 1. Valid Disabled config
	err := s.UpdateConfig(power.ScheduleConfig{
		Enabled: false,
		Mode:    power.ScheduleModeDisabled,
	})
	if err != nil {
		t.Fatalf("expected valid disabled config, got error: %v", err)
	}

	// 2. Valid Interval config
	err = s.UpdateConfig(power.ScheduleConfig{
		Enabled:       true,
		Mode:          power.ScheduleModeInterval,
		IntervalHours: 6,
	})
	if err != nil {
		t.Fatalf("expected valid interval config, got error: %v", err)
	}
	cfg := s.GetConfig()
	if !cfg.Enabled || cfg.Mode != power.ScheduleModeInterval || cfg.IntervalHours != 6 {
		t.Errorf("unexpected config: %+v", cfg)
	}

	// 3. Invalid Interval config (interval < 1)
	err = s.UpdateConfig(power.ScheduleConfig{
		Enabled:       true,
		Mode:          power.ScheduleModeInterval,
		IntervalHours: 0,
	})
	if err == nil {
		t.Errorf("expected error for interval_hours < 1")
	}

	// 4. Valid Clock config
	err = s.UpdateConfig(power.ScheduleConfig{
		Enabled:   true,
		Mode:      power.ScheduleModeClock,
		ClockTime: "04:30",
	})
	if err != nil {
		t.Fatalf("expected valid clock config, got error: %v", err)
	}

	// 5. Invalid Clock config formats
	invalidClocks := []string{"24:00", "04:60", "4:00", "abc", "", "12", "12:34:56"}
	for _, ic := range invalidClocks {
		err = s.UpdateConfig(power.ScheduleConfig{
			Enabled:   true,
			Mode:      power.ScheduleModeClock,
			ClockTime: ic,
		})
		if err == nil {
			t.Errorf("expected error for invalid clock_time: %s", ic)
		}
	}

	// 6. Invalid Mode
	err = s.UpdateConfig(power.ScheduleConfig{
		Enabled: true,
		Mode:    "random_mode",
	})
	if err == nil {
		t.Errorf("expected error for invalid mode")
	}
}

func TestScheduler_StatusCalculations(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	s := power.GetScheduler()

	// 1. Status when disabled
	_ = s.UpdateConfig(power.ScheduleConfig{
		Enabled: false,
		Mode:    power.ScheduleModeDisabled,
	})
	st := s.GetStatus()
	if st.Config.Enabled {
		t.Errorf("expected enabled=false")
	}
	if st.NextRebootInSec != -1 {
		t.Errorf("expected NextRebootInSec=-1 for disabled, got %d", st.NextRebootInSec)
	}
	if st.NextRebootTime != "" {
		t.Errorf("expected empty NextRebootTime for disabled, got %s", st.NextRebootTime)
	}

	// 2. Status with Interval mode (4 hours = 14400s)
	// Mock uptime: 3600 seconds (1 hour)
	s.SetCustomUptimeProvider(func() int64 {
		return 3600
	})
	_ = s.UpdateConfig(power.ScheduleConfig{
		Enabled:       true,
		Mode:          power.ScheduleModeInterval,
		IntervalHours: 4,
	})
	st = s.GetStatus()
	// Next reboot should be in 14400 - 3600 = 10800s (3 hours)
	if st.NextRebootInSec != 10800 {
		t.Errorf("expected NextRebootInSec=10800, got %d", st.NextRebootInSec)
	}
	if !strings.Contains(st.Description, "4 hour") {
		t.Errorf("expected description to mention 4 hours, got: %s", st.Description)
	}

	// 3. Status with Clock mode
	now := time.Now()
	// Set clock 2 hours in the future
	futureTime := now.Add(2 * time.Hour)
	clockStr := fmt.Sprintf("%02d:%02d", futureTime.Hour(), futureTime.Minute())
	_ = s.UpdateConfig(power.ScheduleConfig{
		Enabled:   true,
		Mode:      power.ScheduleModeClock,
		ClockTime: clockStr,
	})
	st = s.GetStatus()
	if st.NextRebootInSec <= 0 || st.NextRebootInSec > 7300 {
		t.Errorf("expected NextRebootInSec ~7200, got %d", st.NextRebootInSec)
	}
	if st.NextRebootTime == "" {
		t.Errorf("expected non-empty NextRebootTime")
	}
}

func TestScheduler_TriggerCallback(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	s := power.GetScheduler()

	var notifiedMsg string
	s.SetRebootNotifier(func(msg string) {
		notifiedMsg = msg
	})

	var rebootExecuted bool
	s.SetCustomRebootExecutor(func() error {
		rebootExecuted = true
		return nil
	})

	// Interval mode: 2 hours (7200s). Mock uptime: 7195s (5 seconds remaining)
	s.SetCustomUptimeProvider(func() int64 {
		return 7195
	})
	_ = s.UpdateConfig(power.ScheduleConfig{
		Enabled:       true,
		Mode:          power.ScheduleModeInterval,
		IntervalHours: 2,
	})

	triggered := s.CheckAndTriggerReboot()
	if !triggered {
		t.Fatalf("expected reboot to be triggered")
	}
	if notifiedMsg == "" {
		t.Errorf("expected notification to be sent")
	}
	if !rebootExecuted {
		t.Errorf("expected custom reboot executor to be called")
	}
}
