package power

import "testing"

func TestInternalSchedulerCustomUptime(t *testing.T) {
	s := &Scheduler{
		customUptimeProvider: func() int64 {
			return 12345
		},
	}
	uptime := s.getSystemUptime()
	if uptime != 12345 {
		t.Errorf("expected 12345, got %d", uptime)
	}

	// fallback when customUptimeProvider is nil
	s2 := &Scheduler{}
	uptime2 := s2.getSystemUptime()
	t.Logf("system uptime: %d", uptime2)
}

func TestInternalCheckAndTriggerReboot(t *testing.T) {
	s := &Scheduler{
		config: ScheduleConfig{
			Enabled: false,
			Mode:    ScheduleModeDisabled,
		},
	}
	if s.checkAndTriggerReboot() {
		t.Error("expected false when disabled")
	}
}

func TestInternalTriggerRebootExecution(t *testing.T) {
	rebootExecuted := false
	notified := false

	s := &Scheduler{
		config: ScheduleConfig{
			Enabled:       true,
			Mode:          ScheduleModeInterval,
			IntervalHours: 1,
		},
		customUptimeProvider: func() int64 {
			return 3590 // 10 seconds remaining <= 15
		},
		customRebootExecutor: func() error {
			rebootExecuted = true
			return nil
		},
		notifier: func(msg string) {
			notified = true
		},
	}

	triggered := s.checkAndTriggerReboot()
	if !triggered {
		t.Error("expected reboot to be triggered")
	}
	if !rebootExecuted {
		t.Error("expected customRebootExecutor to be called")
	}
	if !notified {
		t.Error("expected notifier to be called")
	}
}

func TestFormatDurationSec(t *testing.T) {
	if formatDurationSec(30) != "30s" {
		t.Errorf("expected 30s, got %s", formatDurationSec(30))
	}
	if formatDurationSec(120) != "2m" {
		t.Errorf("expected 2m, got %s", formatDurationSec(120))
	}
	if formatDurationSec(3700) != "1h 1m" {
		t.Errorf("expected 1h 1m, got %s", formatDurationSec(3700))
	}
}

func TestExecuteActions(t *testing.T) {
	actions := []Action{
		ActionReboot,
		ActionPoweroff,
		ActionRebootRecovery,
		ActionRebootBootloader,
		ActionSoftReboot,
	}
	for _, a := range actions {
		_ = Execute(a)
	}
}
