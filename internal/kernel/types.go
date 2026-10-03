package kernel

// CPUCluster represents a group of CPU cores managed by a single cpufreq policy.
type CPUCluster struct {
	PolicyID     int      `json:"policy_id"`
	Cores        string   `json:"cores"` // e.g. "0-3" or "0 1 2 3"
	Label        string   `json:"label"` // e.g. "Little (Efficiency)", "Big (Performance)", "Prime"
	Governor     string   `json:"governor"`
	AvailGovs    []string `json:"available_governors"`
	MinFreq      int64    `json:"min_freq"` // in kHz
	MaxFreq      int64    `json:"max_freq"` // in kHz
	CurFreq      int64    `json:"cur_freq"` // in kHz
	AvailFreqs   []int64  `json:"available_frequencies"`
}

// VMConfig represents kernel virtual memory and sysctl tuning values.
type VMConfig struct {
	Swappiness           int      `json:"swappiness"`
	VFSCachePressure     int      `json:"vfs_cache_pressure"`
	DirtyRatio           int      `json:"dirty_ratio"`
	DirtyBackgroundRatio int      `json:"dirty_background_ratio"`
	TCPCongestion        string   `json:"tcp_congestion"`
	AvailTCPCongestion   []string `json:"available_tcp_congestion"`
}

// ClusterConfig represents user-defined persistent settings for a specific CPU cluster.
type ClusterConfig struct {
	PolicyID int    `json:"policy_id"`
	Governor string `json:"governor,omitempty"`
	MinFreq  int64  `json:"min_freq,omitempty"`
	MaxFreq  int64  `json:"max_freq,omitempty"`
}

// KernelConfig represents the user's persistent kernel tuning configuration.
type KernelConfig struct {
	ActiveProfile        string          `json:"active_profile,omitempty"`
	GlobalGovernor       string          `json:"global_governor,omitempty"`
	Clusters             []ClusterConfig `json:"clusters,omitempty"`
	Swappiness           *int            `json:"swappiness,omitempty"`
	VFSCachePressure     *int            `json:"vfs_cache_pressure,omitempty"`
	DirtyRatio           *int            `json:"dirty_ratio,omitempty"`
	DirtyBackgroundRatio *int            `json:"dirty_background_ratio,omitempty"`
	TCPCongestion        string          `json:"tcp_congestion,omitempty"`
}

// KernelStatus represents the full live state sent to the frontend.
type KernelStatus struct {
	Clusters      []CPUCluster `json:"clusters"`
	VM            VMConfig     `json:"vm"`
	Config        KernelConfig `json:"config"`
	ActiveProfile string       `json:"active_profile"`
	KernelRelease string       `json:"kernel_release"`
}
