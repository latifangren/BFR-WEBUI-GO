package hotspot

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Linux IFNAMSIZ - interface names are limited to 15 characters + NUL.
	reIfaceName = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
	// Lease time in dnsmasq notation: 12h, 30m, 1h, infinite
	reLeaseTime = regexp.MustCompile(`^(infinite|[0-9]{1,4}[mhdw])$`)
)

// ValidationError describes a field-specific validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// ValidateIfaceName checks an interface name against the kernel's naming policy.
func ValidateIfaceName(name string) error {
	if name == "" {
		return invalid("interface", "must not be empty")
	}
	if !reIfaceName.MatchString(name) {
		return invalid("interface", "must be 1-15 characters of letters, digits or underscore")
	}
	return nil
}

// ValidateIPv4 checks a dotted-quad IPv4 address.
func ValidateIPv4(field, addr string) error {
	if addr == "" {
		return invalid(field, "must not be empty")
	}
	ip := net.ParseIP(addr)
	if ip == nil || ip.To4() == nil {
		return invalid(field, "must be a valid IPv4 address")
	}
	return nil
}

// ValidateSubnetMask checks a dotted-quad subnet mask for contiguity.
func ValidateSubnetMask(mask string) error {
	if err := ValidateIPv4("subnet_mask", mask); err != nil {
		return err
	}
	ip := net.ParseIP(mask).To4()
	ones, _ := net.IPMask(ip).Size()
	if ones == 0 {
		return invalid("subnet_mask", "must be a contiguous netmask (255.255.255.0 style)")
	}
	return nil
}

// ValidateLeaseTime checks a dnsmasq-style lease duration.
func ValidateLeaseTime(lease string) error {
	if lease == "" {
		return invalid("lease_time", "must not be empty")
	}
	if !reLeaseTime.MatchString(lease) {
		return invalid("lease_time", "must be a duration like 12h, 30m, or infinite")
	}
	return nil
}

// ValidateBand checks the radio band preference.
func ValidateBand(band string) error {
	switch band {
	case "", "any", "2", "5":
		return nil
	}
	return invalid("band", "must be one of: any, 2, 5")
}

// ValidateGatewayMode checks the gateway ownership mode.
func ValidateGatewayMode(mode string) error {
	switch mode {
	case "", "auto", "static":
		return nil
	}
	return invalid("gateway_mode", "must be one of: auto, static")
}

// ValidateSoftApCredentials validates optional persisted SoftAP credentials on RouterConfig.
func ValidateSoftApCredentials(ssid, pass, secType string) error {
	if ssid != "" && !reHotspotSSID.MatchString(ssid) {
		return invalid("ssid", "must be 1-32 alphanumeric characters, spaces, underscores, or hyphens")
	}
	switch secType {
	case "", "wpa2", "open":
	default:
		return invalid("security_type", "must be one of: wpa2, open")
	}
	if secType != "open" && pass != "" && pass != "none" {
		if !reHotspotPassphrase.MatchString(pass) {
			return invalid("password", "must be 8-63 valid characters without shell metacharacters")
		}
	}
	return nil
}

// ValidateRouterConfig validates and normalizes every field that reaches a root shell.
func ValidateRouterConfig(cfg *RouterConfig) error {
	if cfg == nil {
		return invalid("config", "must not be empty")
	}

	if err := ValidateGatewayMode(cfg.GatewayMode); err != nil {
		return err
	}
	if err := ValidateBand(cfg.Band); err != nil {
		return err
	}
	if err := ValidateSoftApCredentials(cfg.SSID, cfg.Password, cfg.SecurityType); err != nil {
		return err
	}

	// Upstream interface is interpolated into a root shell pipeline.
	if cfg.UpstreamInterface != "" && cfg.UpstreamInterface != "auto" {
		if err := ValidateIfaceName(cfg.UpstreamInterface); err != nil {
			return invalid("upstream_interface", "must be 'auto' or a valid interface name (1-15 chars, letters/digits/underscore)")
		}
	}

	if err := ValidateIPv4("gateway_ip", cfg.GatewayIP); err != nil {
		return err
	}
	if err := ValidateSubnetMask(cfg.SubnetMask); err != nil {
		return err
	}
	if err := ValidateIPv4("dhcp_start", cfg.DHCPStart); err != nil {
		return err
	}
	if err := ValidateIPv4("dhcp_end", cfg.DHCPEnd); err != nil {
		return err
	}
	if err := ValidateLeaseTime(cfg.LeaseTime); err != nil {
		return err
	}
	if err := ValidateIPv4("dns_primary", cfg.DNSPrimary); err != nil {
		return err
	}
	if err := ValidateIPv4("dns_secondary", cfg.DNSSecondary); err != nil {
		return err
	}

	// DHCP pool must sit inside the gateway subnet and be ordered.
	gw := net.ParseIP(cfg.GatewayIP).To4()
	start := net.ParseIP(cfg.DHCPStart).To4()
	end := net.ParseIP(cfg.DHCPEnd).To4()
	if !sameSubnet(gw, start, cfg.SubnetMask) {
		return invalid("dhcp_start", "must be inside the gateway subnet")
	}
	if !sameSubnet(gw, end, cfg.SubnetMask) {
		return invalid("dhcp_end", "must be inside the gateway subnet")
	}
	if compareIP(start, end) >= 0 {
		return invalid("dhcp_end", "must be greater than dhcp_start")
	}
	if compareIP(start, gw) == 0 {
		return invalid("dhcp_start", "must not equal the gateway address")
	}
	if compareIP(end, gw) == 0 {
		return invalid("dhcp_end", "must not equal the gateway address")
	}
	return nil
}

func sameSubnet(a, b net.IP, maskStr string) bool {
	maskIP := net.ParseIP(maskStr).To4()
	if maskIP == nil {
		return false
	}
	mask := net.IPMask(maskIP)
	return a.Mask(mask).Equal(b.Mask(mask))
}

func compareIP(a, b net.IP) int {
	ab := a.To4()
	bb := b.To4()
	for i := range 4 {
		if ab[i] != bb[i] {
			return int(ab[i]) - int(bb[i])
		}
	}
	return 0
}

// NormalizeRouterConfig applies safe defaults for empty fields.
func NormalizeRouterConfig(cfg *RouterConfig) {
	if cfg.GatewayMode == "" {
		cfg.GatewayMode = "auto"
	}
	if cfg.Band == "" {
		cfg.Band = "any"
	}
	if cfg.UpstreamInterface == "" {
		cfg.UpstreamInterface = "auto"
	}
	if cfg.LeaseTime == "" {
		cfg.LeaseTime = "12h"
	}
	if cfg.SecurityType != "" {
		cfg.SecurityType = strings.ToLower(strings.TrimSpace(cfg.SecurityType))
	}
	if cfg.SecurityType == "open" {
		cfg.Password = "none"
	}
}

// maskToCIDRFromMask converts a validated netmask into its prefix length.
func maskToCIDRFromMask(maskStr string) int {
	ip := net.ParseIP(maskStr)
	if ip == nil {
		return 24
	}
	ones, _ := net.IPMask(ip.To4()).Size()
	if ones <= 0 || ones > 32 {
		return 24
	}
	return ones
}

// SubnetCIDRFrom returns the network CIDR string for a gateway address and mask.
//
// The network address is computed by masking the gateway, so it is correct
// for any prefix length and for gateways that sit in a non-zero block
// (e.g. 192.168.1.200/25 belongs to 192.168.1.128/25, and a /23 gateway
// belongs to a two-octet-aligned network).
func SubnetCIDRFrom(gw, mask string) string {
	ip := net.ParseIP(gw)
	if ip == nil {
		return "0.0.0.0/0"
	}
	prefix := maskToCIDRFromMask(mask)
	v4 := ip.To4()
	if v4 == nil {
		return "0.0.0.0/0"
	}
	network := v4.Mask(net.CIDRMask(prefix, 32))
	return fmt.Sprintf("%s/%d", network.String(), prefix)
}

// NormalizeUpstreamValue lowercases and trims an upstream interface value.
func NormalizeUpstreamValue(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// ParseIPOctet is a small helper used by validation error messages.
func ParseIPOctet(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}
