package hotspot_test

import (
	"strings"
	"testing"

	"bfr-webui-go/internal/hotspot"
)

func validConfig() hotspot.RouterConfig {
	return hotspot.RouterConfig{
		GatewayMode:       "static",
		GatewayIP:         "192.168.1.1",
		SubnetMask:        "255.255.255.0",
		DHCPStart:         "192.168.1.10",
		DHCPEnd:           "192.168.1.100",
		LeaseTime:         "12h",
		DNSPrimary:        "1.1.1.1",
		DNSSecondary:      "8.8.8.8",
		Band:              "any",
		UpstreamInterface: "auto",
	}
}

func TestValidateRouterConfig_AcceptsValid(t *testing.T) {
	cfg := validConfig()
	if err := hotspot.ValidateRouterConfig(&cfg); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateRouterConfig_RejectsShellInjectionInUpstream(t *testing.T) {
	cfg := validConfig()
	cfg.UpstreamInterface = "x'; id > /data/local/tmp/pwned; echo '"
	err := hotspot.ValidateRouterConfig(&cfg)
	if err == nil {
		t.Fatal("expected validation error for shell injection in upstream_interface")
	}
	if !strings.Contains(err.Error(), "upstream_interface") {
		t.Errorf("expected field-specific error, got %v", err)
	}
}

func TestValidateRouterConfig_RejectsBadIfaceName(t *testing.T) {
	for _, bad := range []string{"wlan0;reboot", "a b", "way-too-long-interface-name", "$(id)"} {
		cfg := validConfig()
		cfg.UpstreamInterface = bad
		if err := hotspot.ValidateRouterConfig(&cfg); err == nil {
			t.Errorf("expected error for upstream_interface %q", bad)
		}
	}
}

func TestValidateRouterConfig_RejectsInvalidNetworkFields(t *testing.T) {
	cases := []struct {
		name  string
		mutet func(*hotspot.RouterConfig)
		field string
	}{
		{"bad gateway ip", func(c *hotspot.RouterConfig) { c.GatewayIP = "999.1.1.1" }, "gateway_ip"},
		{"bad subnet mask", func(c *hotspot.RouterConfig) { c.SubnetMask = "255.0.255.0" }, "subnet_mask"},
		{"bad dhcp start", func(c *hotspot.RouterConfig) { c.DHCPStart = "not-an-ip" }, "dhcp_start"},
		{"bad lease", func(c *hotspot.RouterConfig) { c.LeaseTime = "forever" }, "lease_time"},
		{"bad dns", func(c *hotspot.RouterConfig) { c.DNSPrimary = "1.1.1" }, "dns_primary"},
		{"bad band", func(c *hotspot.RouterConfig) { c.Band = "6" }, "band"},
		{"bad mode", func(c *hotspot.RouterConfig) { c.GatewayMode = "hybrid" }, "gateway_mode"},
		{"dhcp outside subnet", func(c *hotspot.RouterConfig) { c.DHCPStart = "10.0.0.5" }, "dhcp_start"},
		{"dhcp end before start", func(c *hotspot.RouterConfig) { c.DHCPEnd = "192.168.1.5" }, "dhcp_end"},
		{"dhcp equals gateway", func(c *hotspot.RouterConfig) { c.DHCPStart = "192.168.1.1" }, "dhcp_start"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			tc.mutet(&cfg)
			err := hotspot.ValidateRouterConfig(&cfg)
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("expected error mentioning %q, got %v", tc.field, err)
			}
		})
	}
}

func TestSubnetCIDRFrom(t *testing.T) {
	cases := []struct {
		gw   string
		mask string
		want string
	}{
		{"192.168.1.1", "255.255.255.0", "192.168.1.0/24"},
		{"10.0.0.1", "255.255.0.0", "10.0.0.0/16"},
		{"172.16.5.1", "255.255.255.128", "172.16.5.0/25"},
		// Non-zero block gateway: 192.168.1.200 with /25 belongs to 192.168.1.128/25.
		{"192.168.1.200", "255.255.255.128", "192.168.1.128/25"},
		// /23 gateway must not truncate to the /24 base address.
		{"10.1.2.3", "255.255.254.0", "10.1.2.0/23"},
		// /30 gateway is a two-host point-to-point.
		{"10.0.0.1", "255.255.255.252", "10.0.0.0/30"},
		// /28 gateway: 192.168.1.17 with /28 belongs to 192.168.1.16/28.
		{"192.168.1.17", "255.255.255.240", "192.168.1.16/28"},
	}
	for _, tc := range cases {
		if got := hotspot.SubnetCIDRFrom(tc.gw, tc.mask); got != tc.want {
			t.Errorf("SubnetCIDRFrom(%s, %s) = %s, want %s", tc.gw, tc.mask, got, tc.want)
		}
	}
}

func TestNormalizeRouterConfig(t *testing.T) {
	cfg := hotspot.RouterConfig{}
	hotspot.NormalizeRouterConfig(&cfg)
	if cfg.GatewayMode != "auto" {
		t.Errorf("expected default gateway_mode auto, got %s", cfg.GatewayMode)
	}
	if cfg.Band != "any" {
		t.Errorf("expected default band any, got %s", cfg.Band)
	}
	if cfg.UpstreamInterface != "auto" {
		t.Errorf("expected default upstream auto, got %s", cfg.UpstreamInterface)
	}
}
