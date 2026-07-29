package power

import (
	"fmt"
	"os/exec"
)

type Action string

const (
	ActionReboot           Action = "reboot"
	ActionPoweroff         Action = "poweroff"
	ActionRebootRecovery   Action = "recovery"
	ActionRebootBootloader Action = "bootloader"
	ActionSoftReboot       Action = "soft_reboot"
)

func Execute(action Action) error {
	var cmd *exec.Cmd

	switch action {
	case ActionReboot:
		cmd = exec.Command("su", "-c", "svc power reboot || reboot")
	case ActionPoweroff:
		cmd = exec.Command("su", "-c", "svc power shutdown || reboot -p")
	case ActionRebootRecovery:
		cmd = exec.Command("su", "-c", "reboot recovery")
	case ActionRebootBootloader:
		cmd = exec.Command("su", "-c", "reboot bootloader || reboot fastboot")
	case ActionSoftReboot:
		cmd = exec.Command("su", "-c", "setprop ctl.restart zygote")
	default:
		return fmt.Errorf("unknown power action: %s", action)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute %s: %v, output: %s", action, err, string(output))
	}
	return nil
}
