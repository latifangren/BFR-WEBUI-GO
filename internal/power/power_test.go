package power_test

import (
	"testing"

	"bfr-webui-go/internal/power"
)

func TestPowerActionConstants(t *testing.T) {
	if power.ActionReboot != "reboot" {
		t.Errorf("expected ActionReboot to be 'reboot', got %s", power.ActionReboot)
	}
	if power.ActionRebootRecovery != "recovery" {
		t.Errorf("expected ActionRebootRecovery to be 'recovery', got %s", power.ActionRebootRecovery)
	}
	if power.ActionRebootBootloader != "bootloader" {
		t.Errorf("expected ActionRebootBootloader to be 'bootloader', got %s", power.ActionRebootBootloader)
	}
	if power.ActionSoftReboot != "soft_reboot" {
		t.Errorf("expected ActionSoftReboot to be 'soft_reboot', got %s", power.ActionSoftReboot)
	}
}

func TestExecute_InvalidAction(t *testing.T) {
	err := power.Execute("unknown_action")
	if err == nil {
		t.Errorf("expected error for unknown power action")
	}
}
