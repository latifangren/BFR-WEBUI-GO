package kernel

// ProfileDefinition defines the pre-configured parameters for a kernel tuning profile.
type ProfileDefinition struct {
	Name                 string `json:"name"`
	Label                string `json:"label"`
	Description          string `json:"description"`
	Governor             string `json:"governor"`
	Swappiness           int    `json:"swappiness"`
	VFSCachePressure     int    `json:"vfs_cache_pressure"`
	DirtyRatio           int    `json:"dirty_ratio"`
	DirtyBackgroundRatio int    `json:"dirty_background_ratio"`
}

var AvailableProfiles = map[string]ProfileDefinition{
	"powersave": {
		Name:                 "powersave",
		Label:                "Battery Saver",
		Description:          "Caps CPU frequencies and limits scheduler aggressiveness to extend battery life.",
		Governor:             "powersave",
		Swappiness:           30,
		VFSCachePressure:     120,
		DirtyRatio:           15,
		DirtyBackgroundRatio: 5,
	},
	"balanced": {
		Name:                 "balanced",
		Label:                "Balanced (Default)",
		Description:          "Dynamic Energy-Aware Scheduling (schedutil) with balanced memory paging.",
		Governor:             "schedutil",
		Swappiness:           60,
		VFSCachePressure:     100,
		DirtyRatio:           20,
		DirtyBackgroundRatio: 5,
	},
	"performance": {
		Name:                 "performance",
		Label:                "Maximum Performance",
		Description:          "Locks CPU cores to peak frequency, optimizes memory flushing for zero latency.",
		Governor:             "performance",
		Swappiness:           100,
		VFSCachePressure:     50,
		DirtyRatio:           30,
		DirtyBackgroundRatio: 10,
	},
}
