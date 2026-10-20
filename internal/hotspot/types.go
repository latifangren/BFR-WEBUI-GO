package hotspot

import "regexp"

var (
	// H-6: SSID and Passphrase validation regexes
	reHotspotSSID       = regexp.MustCompile(`^[a-zA-Z0-9 _\-]{1,32}$`)
	reHotspotPassphrase = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{}|:<>,.?/~]{8,63}$`)
)

// ConnectedClient represents a client device connected via Wi-Fi or tethering.
type ConnectedClient struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Device    string `json:"device"`
	State     string `json:"state"`
	Interface string `json:"interface,omitempty"`
}

// HotspotStatus provides comprehensive status of all tethering channels.
type HotspotStatus struct {
	Enabled          bool     `json:"enabled"`
	SSID             string   `json:"ssid"`
	SecurityType     string   `json:"security_type,omitempty"` // "wpa2" or "open"
	WifiEnabled      bool     `json:"wifi_enabled"`
	UsbEnabled       bool     `json:"usb_enabled"`
	EthernetEnabled  bool     `json:"ethernet_enabled"`
	ActiveInterfaces []string `json:"active_interfaces"`
	GatewayIP        string   `json:"gateway_ip,omitempty"`
	Subnet           string   `json:"subnet,omitempty"`
}

type TetheringControlRequest struct {
	Type     string `json:"type"` // "wifi", "usb", "ethernet", or "all"
	Enable   bool   `json:"enable"`
	SSID     string `json:"ssid,omitempty"`
	Password string `json:"password,omitempty"`
	Security string `json:"security,omitempty"` // "wpa2" or "open"
	Band     string `json:"band,omitempty"`     // "any", "2", "5"
}

// RouterConfig defines user-configurable routing, DHCP, DNS and Gateway settings.
type RouterConfig struct {
	GatewayMode       string `json:"gateway_mode"`       // "auto" or "static"
	GatewayIP         string `json:"gateway_ip"`         // e.g. "192.168.1.1" or "192.168.42.1"
	SubnetMask        string `json:"subnet_mask"`        // e.g. "255.255.255.0"
	DHCPStart         string `json:"dhcp_start"`         // e.g. "192.168.1.10"
	DHCPEnd           string `json:"dhcp_end"`           // e.g. "192.168.1.100"
	LeaseTime         string `json:"lease_time"`         // e.g. "12h"
	DNSPrimary        string `json:"dns_primary"`        // e.g. "1.1.1.1"
	DNSSecondary      string `json:"dns_secondary"`      // e.g. "8.8.8.8"
	Band              string `json:"band"`               // "any", "2", "5"
	UpstreamInterface string `json:"upstream_interface"` // "auto", "wlan0", "rmnet_data0", "eth0"
}

// RouterStatus separates desired (persisted) configuration from observed
// (kernel-verified) runtime state.
type RouterStatus struct {
	// Desired state (persisted configuration)
	DesiredMode     string `json:"desired_mode"`
	DesiredGateway  string `json:"desired_gateway"`
	DesiredSubnet   string `json:"desired_subnet"`
	DesiredBand     string `json:"desired_band"`
	DesiredUpstream string `json:"desired_upstream"`
	DHCPManagedByOS bool   `json:"dhcp_managed_by_os"`

	// Apply state explains whether the desired state is currently in effect.
	//
	// "inactive": no tether interface is up, so nothing can be applied.
	// "auto": Android manages the gateway, DHCP and DNS. BFR owns no state.
	// "applied": the BFR-owned objects are present in the kernel and verified.
	// "pending": desired state is saved but not present in the kernel.
	// "failed": the last apply attempt errored.
	ApplyState string `json:"apply_state"`

	// ClientGateway is the gateway address Android's DHCP server actually
	// advertises to clients. It is the address clients use, independent of
	// any static address BFR installs.
	ClientGateway string `json:"client_gateway,omitempty"`
	// ClientSubnet is the downstream subnet Android hands out to clients.
	ClientSubnet string `json:"client_subnet,omitempty"`
	// GatewayEffective is true when the gateway clients receive matches the
	// desired static gateway. It stays false while Android's DHCP server
	// advertises a different gateway.
	GatewayEffective bool `json:"gateway_effective"`

	// Observed state (read back from the kernel)
	Applied         bool   `json:"applied"`
	ObservedGateway string `json:"observed_gateway"`
	ObservedSubnet  string `json:"observed_subnet"`
	ObservedIface   string `json:"observed_iface"`
	UpstreamTable   string `json:"upstream_table"`
	AddressPresent  bool   `json:"address_present"`
	RuleToSubnet    bool   `json:"rule_to_subnet_present"`
	RuleFromAP      bool   `json:"rule_from_ap_present"`
	LastApplyError  string `json:"last_apply_error,omitempty"`
	LastApplyTime   string `json:"last_apply_time,omitempty"`
}

// Apply state values reported by RouterStatus.ApplyState.
const (
	// ApplyStateInactive means no tether interface is up.
	ApplyStateInactive = "inactive"
	// ApplyStateAuto means Android manages the gateway, DHCP and DNS.
	ApplyStateAuto = "auto"
	// ApplyStateApplied means the BFR-owned objects are verified in the kernel.
	ApplyStateApplied = "applied"
	// ApplyStatePending means the desired state is saved but not applied.
	ApplyStatePending = "pending"
	// ApplyStateFailed means the last apply attempt failed.
	ApplyStateFailed = "failed"
)
