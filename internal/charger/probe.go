package charger

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ProbeProfile tests a single switch profile to verify if writing its OFF value halts or reduces charging current.
func ProbeProfile(profile SwitchProfile) ProbeResult {
	battBefore := GetBatteryInfo()
	currBefore := battBefore.CurrentNow

	// 1. Read existing values to guarantee safe non-destructive restoration
	origValues := make([]string, len(profile.Nodes))
	for i, node := range profile.Nodes {
		val, _ := readSysfs(node.Path)
		origValues[i] = val
	}

	// 2. Write OFF value to all nodes in the profile
	for _, node := range profile.Nodes {
		_ = writeSysfs(node.Path, node.OffValue)
	}

	// 3. Wait 1.5 seconds for kernel charging IC and hardware fuel gauge to settle
	time.Sleep(1500 * time.Millisecond)

	// 4. Read battery status and current after probe
	battAfter := GetBatteryInfo()
	currAfter := battAfter.CurrentNow

	// 5. Success evaluation:
	// - Current dropped by more than 100 mA: (currBefore - currAfter) > 100
	// - Current dropped into idle threshold: math.Abs(currAfter) <= 30 when currBefore > 100
	// - Status changed to Not charging, Discharging, or Idle
	currentDropped := (currBefore - currAfter) > 100.0
	idleReached := currBefore > 100.0 && math.Abs(currAfter) <= 30.0
	statusHalted := strings.EqualFold(battAfter.Status, "Not charging") ||
		strings.EqualFold(battAfter.Status, "Discharging") ||
		strings.EqualFold(battAfter.Status, "Idle") ||
		battAfter.IsIdle

	success := currentDropped || idleReached || statusHalted

	// 6. Non-destructive restore: return nodes to pre-probe state
	for i, node := range profile.Nodes {
		restoreVal := node.OnValue
		if origValues[i] != "" {
			restoreVal = origValues[i]
		}
		_ = writeSysfs(node.Path, restoreVal)
	}

	var msg string
	if success {
		msg = fmt.Sprintf("Success: Current dropped from %.1f mA to %.1f mA (Status: %s -> %s)",
			currBefore, currAfter, battBefore.Status, battAfter.Status)
	} else {
		msg = fmt.Sprintf("No change detected: Current %.1f mA -> %.1f mA (Status: %s)",
			currBefore, currAfter, battAfter.Status)
	}

	return ProbeResult{
		ProfileID:     profile.ID,
		Name:          profile.Name,
		Success:       success,
		CurrentBefore: currBefore,
		CurrentAfter:  currAfter,
		Message:       msg,
		TestedAt:      time.Now(),
	}
}

// ProbeAll tests all available switch profiles on the system and updates Manager results.
func ProbeAll(m *Manager) []ProbeResult {
	profiles := GetExistingSwitchProfiles()
	var results []ProbeResult

	for _, p := range profiles {
		res := ProbeProfile(p)
		results = append(results, res)
		if m != nil {
			m.Log(fmt.Sprintf("Probe %s: %s (success=%v)", p.Name, res.Message, res.Success))
		}
	}

	if m != nil {
		m.mu.Lock()
		m.probeResults = results
		m.mu.Unlock()
	}

	return results
}

// ProbeSingle finds a profile by ID and runs ProbeProfile on it, updating Manager results.
func ProbeSingle(m *Manager, profileID string) ProbeResult {
	var target *SwitchProfile

	// Check existing available profiles first
	for _, p := range GetExistingSwitchProfiles() {
		if p.ID == profileID {
			target = &p
			break
		}
	}
	// Fallback to AllSwitchProfiles
	if target == nil {
		for _, p := range AllSwitchProfiles {
			if p.ID == profileID {
				target = &p
				break
			}
		}
	}

	if target == nil {
		return ProbeResult{
			ProfileID: profileID,
			Name:      profileID,
			Success:   false,
			Message:   fmt.Sprintf("Profile '%s' not found on system", profileID),
			TestedAt:  time.Now(),
		}
	}

	res := ProbeProfile(*target)
	if m != nil {
		m.Log(fmt.Sprintf("Single probe %s: %s (success=%v)", target.Name, res.Message, res.Success))
		m.mu.Lock()
		updated := false
		for i, r := range m.probeResults {
			if r.ProfileID == profileID {
				m.probeResults[i] = res
				updated = true
				break
			}
		}
		if !updated {
			m.probeResults = append(m.probeResults, res)
		}
		m.mu.Unlock()
	}

	return res
}
