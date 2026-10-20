package hotspot_test

import (
	"testing"

	"bfr-webui-go/internal/hotspot"
)

func TestRouterStatus_AppliedRequiresLiveTetherInterface(t *testing.T) {
	// When no tether interface is active, RouterStatus must report
	// applied = false and apply_state = "inactive", even if a static
	// gateway configuration was previously saved.
	st := hotspot.GetRouterStatus()
	names := hotspot.GetActiveInterfaceNames()
	if len(names) == 0 {
		if st.Applied {
			t.Errorf("expected applied=false when no tether interface is up, got %v", st.Applied)
		}
		if st.ApplyState != hotspot.ApplyStateInactive {
			t.Errorf("expected apply_state=%s, got %s", hotspot.ApplyStateInactive, st.ApplyState)
		}
	}
}

func TestRouterStatus_AutoModeIsNotPending(t *testing.T) {
	// In Auto mode Android owns the gateway, DHCP and DNS. BFR owns no
	// kernel object, so the status must never be reported as "pending",
	// which would wrongly imply a failed apply.
	st := hotspot.GetRouterStatus()
	if st.DesiredMode != "auto" {
		t.Skipf("configured mode is %q, not auto", st.DesiredMode)
	}
	names := hotspot.GetActiveInterfaceNames()
	if len(names) == 0 {
		t.Skip("no tether interface is up")
	}
	if st.ApplyState == hotspot.ApplyStatePending {
		t.Fatalf("auto mode must not report apply_state=pending, got %s", st.ApplyState)
	}
	if st.ApplyState != hotspot.ApplyStateAuto {
		t.Fatalf("expected apply_state=%s in auto mode, got %s", hotspot.ApplyStateAuto, st.ApplyState)
	}
	if !st.Applied {
		t.Error("expected applied=true in auto mode while a tether interface is up")
	}
}

func TestRouterStatus_ClientGatewayIsNotTheBFRStaticAddress(t *testing.T) {
	// The client-facing gateway comes from Android's DHCP server. It must
	// never be the static address BFR installs, because clients never
	// receive that one.
	st := hotspot.GetRouterStatus()
	if st.ClientGateway == "" {
		t.Skip("no client gateway reported")
	}
	if st.ObservedGateway != "" && st.ClientGateway == st.ObservedGateway {
		t.Errorf("client gateway %q must not be the BFR static address", st.ClientGateway)
	}
}

func TestSubnetCIDRFrom_ArbitraryPrefixes(t *testing.T) {
	// SubnetCIDRFrom must correctly compute the network boundary for any
	// netmask, including non-/24 subnets and gateways that sit in non-zero
	// blocks.
	cases := []struct {
		name string
		gw   string
		mask string
		want string
	}{
		{
			name: "standard /24",
			gw:   "192.168.1.1",
			mask: "255.255.255.0",
			want: "192.168.1.0/24",
		},
		{
			name: "large /16",
			gw:   "10.0.5.1",
			mask: "255.255.0.0",
			want: "10.0.0.0/16",
		},
		{
			name: "non-zero block /25 (upper half)",
			gw:   "192.168.1.200",
			mask: "255.255.255.128",
			want: "192.168.1.128/25",
		},
		{
			name: "zero block /25 (lower half)",
			gw:   "192.168.1.10",
			mask: "255.255.255.128",
			want: "192.168.1.0/25",
		},
		{
			name: "supernet /23 non-zero third octet",
			gw:   "10.1.3.1",
			mask: "255.255.254.0",
			want: "10.1.2.0/23",
		},
		{
			name: "point-to-point /30",
			gw:   "192.168.1.5",
			mask: "255.255.255.252",
			want: "192.168.1.4/30",
		},
		{
			name: "small /28 non-zero block",
			gw:   "192.168.1.45",
			mask: "255.255.255.240",
			want: "192.168.1.32/28",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hotspot.SubnetCIDRFrom(tc.gw, tc.mask)
			if got != tc.want {
				t.Fatalf("SubnetCIDRFrom(%s, %s) = %s, want %s", tc.gw, tc.mask, got, tc.want)
			}
		})
	}
}

func TestRouterConfigValidation_AcceptsNon24Prefixes(t *testing.T) {
	// A valid /25 configuration with both pool ends in the same block
	// must pass validation.
	cfg := hotspot.RouterConfig{
		GatewayMode:       "static",
		GatewayIP:         "192.168.1.129",
		SubnetMask:        "255.255.255.128",
		DHCPStart:         "192.168.1.130",
		DHCPEnd:           "192.168.1.250",
		LeaseTime:         "12h",
		DNSPrimary:        "1.1.1.1",
		DNSSecondary:      "8.8.8.8",
		Band:              "any",
		UpstreamInterface: "auto",
	}
	if err := hotspot.ValidateRouterConfig(&cfg); err != nil {
		t.Fatalf("expected valid /25 config, got: %v", err)
	}
}

func TestRouterConfigValidation_RejectsPoolOutsideBlock(t *testing.T) {
	// A /25 configuration where DHCPStart is in the lower block
	// (0-127) while the gateway is in the upper block (128-255) must fail.
	cfg := hotspot.RouterConfig{
		GatewayMode:       "static",
		GatewayIP:         "192.168.1.129",
		SubnetMask:        "255.255.255.128",
		DHCPStart:         "192.168.1.10", // wrong half!
		DHCPEnd:           "192.168.1.250",
		LeaseTime:         "12h",
		DNSPrimary:        "1.1.1.1",
		DNSSecondary:      "8.8.8.8",
		Band:              "any",
		UpstreamInterface: "auto",
	}
	if err := hotspot.ValidateRouterConfig(&cfg); err == nil {
		t.Fatal("expected validation failure when DHCPStart is in different block")
	}
}
