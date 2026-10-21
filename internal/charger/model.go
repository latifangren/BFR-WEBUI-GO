package charger

import "time"

// SwitchNode represents a single sysfs/proc node and its corresponding ON/OFF control values.
type SwitchNode struct {
	Path     string `json:"path"`
	OnValue  string `json:"on_value"`
	OffValue string `json:"off_value"`
}

// SwitchProfile represents an ACC charging control profile, containing one or more nodes.
type SwitchProfile struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Nodes             []SwitchNode `json:"nodes"`
	IsBypass          bool         `json:"is_bypass"`
	IsPercentageLimit bool         `json:"is_percentage_limit,omitempty"`
}

// Config represents the persistent configuration for the smart charger limiter.
type Config struct {
	Enabled        bool   `json:"enabled"`
	StartPercent   int    `json:"start_percent"`
	StopPercent    int    `json:"stop_percent"`
	SelectedSwitch string `json:"selected_switch,omitempty"`
	CustomPath     string `json:"custom_path,omitempty"`
	CustomOn       string `json:"custom_on,omitempty"`
	CustomOff      string `json:"custom_off,omitempty"`
}

// BatteryInfo represents detailed hardware battery metrics read from sysfs.
type BatteryInfo struct {
	Capacity    int     `json:"capacity"`
	Status      string  `json:"status"`
	CurrentNow  float64 `json:"current_now"`  // Current in mA (normalized sign)
	VoltageNow  float64 `json:"voltage_now"`  // Voltage in V
	Temperature float64 `json:"temperature"`  // Temperature in °C
	PowerSupply string  `json:"power_supply"` // Name of primary power supply node
	IsOnline    bool    `json:"is_online"`    // True if external charger/USB power connected
	IsIdle      bool    `json:"is_idle"`      // True if idle bypass active (charging stopped with power connected)
}

// ProbeResult stores the testing result for a single switch profile.
type ProbeResult struct {
	ProfileID     string    `json:"profile_id"`
	Name          string    `json:"name"`
	Success       bool      `json:"success"`
	CurrentBefore float64   `json:"current_before"` // Current before write in mA
	CurrentAfter  float64   `json:"current_after"`  // Current after write in mA
	Message       string    `json:"message"`
	TestedAt      time.Time `json:"tested_at"`
}

// StatusResponse encapsulates current state, hardware status, and available switches.
type StatusResponse struct {
	Config            Config          `json:"config"`
	ActiveProfile     *SwitchProfile  `json:"active_profile,omitempty"`
	DetectedPath      string          `json:"detected_path"`
	DetectedType      string          `json:"detected_type"`
	ChargingDisabled  bool            `json:"charging_disabled"`
	CurrentLevel      int             `json:"current_level"` // Maintained for backward compatibility
	Battery           BatteryInfo     `json:"battery"`
	AvailableSwitches []SwitchProfile `json:"available_switches"`
	ProbeResults      []ProbeResult   `json:"probe_results,omitempty"`
	Logs              []string        `json:"logs"`
}
