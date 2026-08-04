package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
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
		filepath.Join(config.ModuleDir, "tweaks.json"),
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
			LTECarrierAggregation: false,
			TCPBufferOptimization: false,
			BBR2CongestionControl: false,
			SysctlBuffersOpt:      false,
			DalvikResponsiveness:  false,
			SettingsGlobalTweaks:  false,
			TTLSpoofing:           false,
			PacketSteeringRPS:     false,
			MTUTuning:             false,
		}, nil
	}

	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
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

func getTotalRAMBytes() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseUint(fields[1], 10, 64)
				return kb * 1024
			}
		}
	}
	return 0
}

func ApplyAllTweaks() error {
	cfg, err := LoadTweaks()
	if err != nil {
		return err
	}

	var cmds []string

	if cfg.SettingsGlobalTweaks {
		cmds = append(cmds,
			"settings put global adb_enabled 1",
			"settings put global window_animation_scale 0.1",
			"settings put global transition_animation_scale 0.1",
			"settings put global animator_duration_scale 0.1",
			"settings put global gprs_detach_timer 30",
		)
	}

	if cfg.LTECarrierAggregation {
		cmds = append(cmds,
			"settings put global lte_ca_config 1",
			"setprop gsm.lte.ca.support 1",
			"setprop persist.radio.lte_ca 1",
			"setprop vendor.radio.lte_ca 1",
		)
	}

	if cfg.SysctlBuffersOpt {
		totalRAM := getTotalRAMBytes()
		maxBuf := "33554432"
		defBuf := "16777216"
		if totalRAM > 0 && totalRAM < 4*1024*1024*1024 { // < 4GB
			maxBuf = "16777216"
			defBuf = "8388608"
		} else if totalRAM >= 4*1024*1024*1024 { // >= 4GB
			maxBuf = "67108864"
			defBuf = "33554432"
		}
		cmds = append(cmds,
			fmt.Sprintf("sysctl -w net.core.rmem_max=\"%s\"", maxBuf),
			fmt.Sprintf("sysctl -w net.core.wmem_max=\"%s\"", maxBuf),
			fmt.Sprintf("sysctl -w net.core.rmem_default=\"%s\"", defBuf),
			fmt.Sprintf("sysctl -w net.core.wmem_default=\"%s\"", defBuf),
			"sysctl -w net.core.somaxconn=\"1024\"",
			fmt.Sprintf("sysctl -w net.ipv4.tcp_rmem=\"4096 87380 %s\"", maxBuf),
			fmt.Sprintf("sysctl -w net.ipv4.tcp_wmem=\"4096 65536 %s\"", maxBuf),
			"sysctl -w net.ipv4.tcp_tw_reuse=\"1\"",
			"sysctl -w net.ipv4.tcp_fin_timeout=\"15\"",
			"sysctl -w net.ipv4.tcp_max_syn_backlog=\"4096\"",
			"sysctl -w net.ipv4.tcp_keepalive_time=\"300\"",
			"sysctl -w net.ipv4.tcp_keepalive_intvl=\"15\"",
			"sysctl -w net.ipv4.tcp_keepalive_probes=\"5\"",
			"sysctl -w net.ipv4.tcp_timestamps=\"1\"",
			"sysctl -w net.ipv4.tcp_sack=\"1\"",
			"sysctl -w net.ipv4.tcp_window_scaling=\"1\"",
		)
	}

	if cfg.BBR2CongestionControl {
		cmds = append(cmds,
			"sysctl -w net.core.default_qdisc=\"fq\"",
			"sysctl -w net.ipv4.tcp_congestion_control=\"bbr2\" || sysctl -w net.ipv4.tcp_congestion_control=\"bbr\"",
		)
	}

	if cfg.DalvikResponsiveness {
		cmds = append(cmds,
			"setprop dalvik.vm.heaptargetutilization 0.75",
			"setprop dalvik.vm.heapgrowthlimit 256m",
			"setprop dalvik.vm.heapsize 512m",
		)
	}

	// Apply Per-Interface tweaks
	if cfg.MTUTuning || cfg.PacketSteeringRPS {
		ifaces, err := GetInterfaces()
		if err == nil {
			for _, ifc := range ifaces {
				name := ifc.Name
				if strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "rmnet") || strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "rndis") {
					if cfg.MTUTuning {
						cmds = append(cmds, fmt.Sprintf("ip link set %s mtu 1500; ip link set %s txqueuelen 5000", name, name))
					}
					if cfg.PacketSteeringRPS {
						cmds = append(cmds, fmt.Sprintf("echo f > /sys/class/net/%s/queues/rx-0/rps_cpus 2>/dev/null || true", name))
					}
				}
			}
		}
	}

	if len(cmds) > 0 {
		batchCmd := strings.Join(cmds, " ; ")
		_, _ = config.ExecSuTimeout(10*time.Second, batchCmd)
	}

	if cfg.TTLSpoofing {
		_ = SetTTLSpoofSDK(true, 0)
	} else {
		_ = SetTTLSpoofSDK(false, 0)
	}

	return nil
}
