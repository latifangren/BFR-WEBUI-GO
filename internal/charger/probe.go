package charger

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ProbeProfile tests whether writing a profile's OFF values reduces charging current.
func ProbeProfile(profile SwitchProfile) ProbeResult {
	battBefore := GetBatteryInfo()
	result := ProbeResult{
		ProfileID:     profile.ID,
		Name:          profile.Name,
		CurrentBefore: battBefore.CurrentNow,
		TestedAt:      time.Now(),
	}
	if len(profile.Nodes) == 0 {
		result.Message = "Probe failed: profile has no control nodes"
		return result
	}
	if !battBefore.IsOnline || !strings.EqualFold(battBefore.Status, "Charging") || battBefore.CurrentNow < 100 {
		result.Message = "Probe skipped: connect a charger and ensure the battery is actively charging above 100 mA"
		return result
	}

	origValues := make([]string, len(profile.Nodes))
	for i, node := range profile.Nodes {
		value, err := readSysfs(node.Path)
		if err != nil {
			result.Message = fmt.Sprintf("Probe failed: cannot read %s: %v", node.Path, err)
			return result
		}
		origValues[i] = value
	}

	written := 0
	for i, node := range profile.Nodes {
		if err := writeSysfs(node.Path, node.OffValue); err != nil {
			for j := written - 1; j >= 0; j-- {
				_ = writeSysfs(profile.Nodes[j].Path, origValues[j])
			}
			result.Message = fmt.Sprintf("Probe failed: cannot write %s: %v", node.Path, err)
			return result
		}
		written = i + 1
	}

	time.Sleep(5 * time.Second)
	battAfter := GetBatteryInfo()
	result.CurrentAfter = battAfter.CurrentNow

	var restoreErrors []string
	for i, node := range profile.Nodes {
		if err := writeSysfs(node.Path, origValues[i]); err != nil {
			restoreErrors = append(restoreErrors, fmt.Sprintf("%s: %v", node.Path, err))
		}
	}

	drop := battBefore.CurrentNow - battAfter.CurrentNow
	result.Success = drop >= math.Max(100, battBefore.CurrentNow*0.3) &&
		(battAfter.CurrentNow <= battBefore.CurrentNow*0.7 || battAfter.CurrentNow <= 30)
	result.Message = fmt.Sprintf("Current %0.1f mA -> %0.1f mA (status: %s)", battBefore.CurrentNow, battAfter.CurrentNow, battAfter.Status)
	if result.Success {
		result.Message = "Verified: " + result.Message
	} else {
		result.Message = "No verified current reduction: " + result.Message
	}
	if len(restoreErrors) > 0 {
		result.Success = false
		result.Message += "; restore failed: " + strings.Join(restoreErrors, ", ")
	}
	return result
}

// ProbeAll tests all available hardware charging switches and updates Manager results.
func ProbeAll(m *Manager) []ProbeResult {
	if m != nil {
		m.probeMu.Lock()
		defer m.probeMu.Unlock()
	}
	profiles := GetExistingSwitchProfiles()
	results := make([]ProbeResult, 0, len(profiles))
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
	if m != nil {
		m.probeMu.Lock()
		defer m.probeMu.Unlock()
	}
	var target *SwitchProfile
	for _, p := range GetExistingSwitchProfiles() {
		if p.ID == profileID {
			cp := p
			target = &cp
			break
		}
	}
	if target == nil {
		for _, p := range AllSwitchProfiles {
			if p.ID == profileID && !strings.Contains(p.ID, "siop_level") {
				cp := p
				target = &cp
				break
			}
		}
	}
	if target == nil {
		return ProbeResult{ProfileID: profileID, Name: profileID, Message: fmt.Sprintf("Profile '%s' not found on system", profileID), TestedAt: time.Now()}
	}
	res := ProbeProfile(*target)
	if m != nil {
		m.Log(fmt.Sprintf("Single probe %s: %s (success=%v)", target.Name, res.Message, res.Success))
		m.mu.Lock()
		updated := false
		for i, old := range m.probeResults {
			if old.ProfileID == profileID {
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
