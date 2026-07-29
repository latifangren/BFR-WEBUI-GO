package network

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

type TweaksConfig struct {
	LTECarrierAggregation bool `json:"lte_carrier_aggregation"`
	TCPBufferOptimization bool `json:"tcp_buffer_optimization"`
	BBR2CongestionControl bool `json:"bbr2_congestion_control"`
	SysctlBuffersOpt      bool `json:"sysctl_buffers_opt"`
	DalvikResponsiveness  bool `json:"dalvik_responsiveness"`
	SettingsGlobalTweaks  bool `json:"settings_global_tweaks"`
	TTLSpoofing           bool `json:"ttl_spoofing"`
	PacketSteeringRPS     bool `json:"packet_steering_rps"`
	MTUTuning             bool `json:"mtu_tuning"`
}

func GetConfigPath() string {
	// Try Magisk folder first, fallback to current dir
	paths := []string{
		"/data/adb/modules/bfr_webui_go/tweaks.json",
		"./tweaks.json",
		"tweaks.json",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "tweaks.json"
}

func LoadTweaks() (TweaksConfig, error) {
	var cfg TweaksConfig
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		// Return default config
		return TweaksConfig{
			LTECarrierAggregation: true,
			TCPBufferOptimization: true,
			BBR2CongestionControl: true,
			SysctlBuffersOpt:      true,
			DalvikResponsiveness:  true,
			SettingsGlobalTweaks:  true,
			TTLSpoofing:           true,
			PacketSteeringRPS:     true,
			MTUTuning:             true,
		}, nil
	}

	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func SaveTweaks(cfg TweaksConfig) error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ApplyAllTweaks() error {
	cfg, err := LoadTweaks()
	if err != nil {
		return err
	}

	if cfg.SettingsGlobalTweaks {
		_ = exec.Command("su", "-c", "settings put global adb_enabled 1").Run()
		_ = exec.Command("su", "-c", "settings put global window_animation_scale 0.1").Run()
		_ = exec.Command("su", "-c", "settings put global transition_animation_scale 0.1").Run()
		_ = exec.Command("su", "-c", "settings put global animator_duration_scale 0.1").Run()
		_ = exec.Command("su", "-c", "settings put global gprs_detach_timer 30").Run()
	}

	if cfg.LTECarrierAggregation {
		_ = exec.Command("su", "-c", "settings put global lte_ca_config 1").Run()
		_ = exec.Command("su", "-c", "setprop gsm.lte.ca.support 1").Run()
		_ = exec.Command("su", "-c", "setprop persist.radio.lte_ca 1").Run()
		_ = exec.Command("su", "-c", "setprop vendor.radio.lte_ca 1").Run()
	}

	if cfg.SysctlBuffersOpt {
		_ = SetSysctl("net.core.rmem_max", "67108864")
		_ = SetSysctl("net.core.wmem_max", "67108864")
		_ = SetSysctl("net.core.rmem_default", "33554432")
		_ = SetSysctl("net.core.wmem_default", "33554432")
		_ = SetSysctl("net.core.somaxconn", "1024")
		_ = SetSysctl("net.ipv4.tcp_rmem", "4096 87380 67108864")
		_ = SetSysctl("net.ipv4.tcp_wmem", "4096 65536 67108864")
		_ = SetSysctl("net.ipv4.tcp_tw_reuse", "1")
		_ = SetSysctl("net.ipv4.tcp_fin_timeout", "15")
		_ = SetSysctl("net.ipv4.tcp_max_syn_backlog", "4096")
		_ = SetSysctl("net.ipv4.tcp_keepalive_time", "300")
		_ = SetSysctl("net.ipv4.tcp_keepalive_intvl", "15")
		_ = SetSysctl("net.ipv4.tcp_keepalive_probes", "5")
		_ = SetSysctl("net.ipv4.tcp_timestamps", "1")
		_ = SetSysctl("net.ipv4.tcp_sack", "1")
		_ = SetSysctl("net.ipv4.tcp_window_scaling", "1")
	}

	if cfg.BBR2CongestionControl {
		_ = SetSysctl("net.core.default_qdisc", "fq")
		errBbr := SetSysctl("net.ipv4.tcp_congestion_control", "bbr2")
		if errBbr != nil {
			_ = SetSysctl("net.ipv4.tcp_congestion_control", "bbr")
		}
	}

	if cfg.TTLSpoofing {
		_ = SetTTLSpoofSDK(true, 0)
	} else {
		_ = SetTTLSpoofSDK(false, 0)
	}

	// Apply Per-Interface tweaks
	if cfg.MTUTuning || cfg.PacketSteeringRPS {
		ifaces, err := GetInterfaces()
		if err == nil {
			for _, ifc := range ifaces {
				name := ifc.Name
				if strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "rmnet") || strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "rndis") {
					if cfg.MTUTuning {
						_ = SetInterfaceConfig(name, 1500, 5000)
					}
					if cfg.PacketSteeringRPS {
						_ = ConfigureRPS(name, "f")
					}
				}
			}
		}
	}

	if cfg.DalvikResponsiveness {
		_ = exec.Command("su", "-c", "setprop dalvik.vm.heaptargetutilization 0.75").Run()
		_ = exec.Command("su", "-c", "setprop dalvik.vm.heapgrowthlimit 256m").Run()
		_ = exec.Command("su", "-c", "setprop dalvik.vm.heapsize 512m").Run()
	}

	return nil
}
