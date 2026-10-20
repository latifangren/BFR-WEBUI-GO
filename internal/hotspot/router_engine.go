package hotspot

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// ownedState records exactly what BFR installed into the kernel so cleanup can
// remove only BFR-owned objects and never touch Android-managed state.
type ownedState struct {
	Iface       string `json:"iface"`
	Address     string `json:"address"`     // e.g. "192.168.1.1/24"
	SubnetCIDR  string `json:"subnet_cidr"` // e.g. "192.168.1.0/24"
	RuleToSub   int    `json:"rule_to_subnet"`
	RuleFromAP  int    `json:"rule_from_ap"`
	AppliedAt   string `json:"applied_at"`
	UpstreamTbl string `json:"upstream_table"`
}

var (
	ownedMu sync.Mutex
	current ownedState
)

// ownedStatePath is where the BFR-owned kernel objects are recorded so cleanup
// still works after the WebUI process restarts.
func ownedStatePath() string {
	return config.GetPersistentFilePath("hotspot_owned_state.json")
}

// loadOwnedState restores the record of BFR-owned kernel objects.
func loadOwnedState() ownedState {
	data, err := os.ReadFile(ownedStatePath())
	if err != nil {
		return ownedState{}
	}
	var st ownedState
	if err := json.Unmarshal(data, &st); err != nil {
		return ownedState{}
	}
	return st
}

// persistOwnedState records the BFR-owned kernel objects on disk.
func persistOwnedState(st ownedState) {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = config.WriteFileAtomic(ownedStatePath(), data, 0644)
}

// ObservedState reports kernel state BFR can verify, not in-process intent.
type ObservedState struct {
	AddressPresent bool   `json:"address_present"`
	RuleToSubnet   bool   `json:"rule_to_subnet_present"`
	RuleFromAP     bool   `json:"rule_from_ap_present"`
	NatRulePresent bool   `json:"nat_rule_present"`
	UpstreamTable  string `json:"upstream_table,omitempty"`
	Iface          string `json:"iface,omitempty"`
	SubnetCIDR     string `json:"subnet_cidr,omitempty"`
}

// rulePriority constants keep a stable identity for cleanup.
const (
	rulePrioToSubnet = 7000
	rulePrioFromAP   = 7010
)

// resolveTetherIface returns the interface name to operate on, or "" if none.
func resolveTetherIface(preferred string) string {
	if preferred != "" {
		return preferred
	}
	names := GetActiveInterfaceNames()
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

// DetectTetherUpstreamTable resolves the routing table Android actually uses for
// tethering traffic.
//
// On Android 14 the tethering stack installs its own policy rule:
//
//	21000: from all iif wlan1 lookup wlan0
//
// That rule is authoritative, so it is read first. Only if it is absent do we
// fall back to a default route that actually has a next hop (Android's
// "dev dummy0" placeholder entries are never a valid upstream).
func DetectTetherUpstreamTable(customIface string) string {
	customIface = NormalizeUpstreamValue(customIface)

	if customIface != "" && customIface != "auto" {
		if tbl, ok := tableForIface(customIface); ok {
			return tbl
		}
		return "main"
	}

	// 1. Android's own tethering rule: "from all iif <tether> lookup <table>".
	if tbl, ok := androidTetheringUpstreamTable(); ok {
		return tbl
	}

	// 2. Fallback: the default route that has an actual next hop.
	out, err := config.ExecSuTimeout(2*time.Second, "ip route show table all 2>/dev/null")
	if err == nil {
		if tbl, ok := pickReachableDefaultTable(string(out)); ok {
			return tbl
		}
	}
	return "main"
}

// androidTetheringUpstreamTable reads the table targeted by Android's own
// "from all iif <tether-iface> lookup <table>" policy rule.
func androidTetheringUpstreamTable() (string, bool) {
	ifaces := GetActiveInterfaceNames()
	if len(ifaces) == 0 {
		return "", false
	}

	out, err := config.ExecSuTimeout(2*time.Second, "ip rule show 2>/dev/null")
	if err != nil {
		return "", false
	}

	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "iif ") || !strings.Contains(line, "lookup ") {
			continue
		}
		matchesTether := false
		for _, ifc := range ifaces {
			if strings.Contains(line, "iif "+ifc+" ") {
				matchesTether = true
				break
			}
		}
		if !matchesTether {
			continue
		}
		if tbl, ok := parseTableField(line); ok {
			return tbl, true
		}
	}

	return "", false
}

func tableForIface(iface string) (string, bool) {
	cmd := fmt.Sprintf("ip route show table all 2>/dev/null | grep ' dev %s ' | head -n 1", iface)
	out, err := config.ExecSuTimeout(2*time.Second, cmd)
	if err != nil {
		return "", false
	}
	return parseTableField(string(out))
}

func parseTableField(out string) (string, bool) {
	fields := strings.Fields(out)
	for i, f := range fields {
		if f == "table" && i+1 < len(fields) {
			return fields[i+1], true
		}
	}
	return "", false
}

// pickReachableDefaultTable returns the table of a default route that has a
// gateway, skipping Android's placeholder "dev dummy0" default entries.
func pickReachableDefaultTable(out string) (string, bool) {
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "default ") {
			continue
		}
		if !strings.Contains(line, "via ") {
			continue
		}
		if tbl, ok := parseTableField(line); ok {
			return tbl, true
		}
	}
	return "", false
}

// ApplyStaticGateway installs only BFR-owned objects for the requested static
// gateway. It never flushes addresses or kills processes owned by Android.
func ApplyStaticGateway(iface string, cfg *RouterConfig) error {
	if cfg == nil || cfg.GatewayMode != "static" {
		return nil
	}
	if err := ValidateRouterConfig(cfg); err != nil {
		return err
	}

	iface = resolveTetherIface(iface)
	if iface == "" {
		return fmt.Errorf("no active tether interface to configure")
	}

	ownedMu.Lock()
	defer ownedMu.Unlock()

	// Remove any previously owned state first so apply is idempotent.
	removeOwnedStateLocked()

	prefix := strconv.Itoa(maskToCIDRFromMask(cfg.SubnetMask))
	subnetCIDR := SubnetCIDRFrom(cfg.GatewayIP, cfg.SubnetMask)
	upstreamTable := DetectTetherUpstreamTable(cfg.UpstreamInterface)

	cmds := []string{
		// Add (never flush) the static gateway address.
		fmt.Sprintf("ip addr replace %s/%s dev %s", cfg.GatewayIP, prefix, iface),
		fmt.Sprintf("ip route replace %s dev %s src %s proto static scope link table main", subnetCIDR, iface, cfg.GatewayIP),
		// Remove the exact previous rule first: "ip rule add" fails with
		// "File exists" when an identical rule is already installed.
		fmt.Sprintf("ip rule del to %s lookup main pref %d 2>/dev/null; ip rule add to %s lookup main pref %d", subnetCIDR, rulePrioToSubnet, subnetCIDR, rulePrioToSubnet),
		fmt.Sprintf("ip rule del from all iif %s lookup %s pref %d 2>/dev/null; ip rule add from all iif %s lookup %s pref %d", iface, upstreamTable, rulePrioFromAP, iface, upstreamTable, rulePrioFromAP),
	}

	if out, err := config.ExecSuTimeout(4*time.Second, strings.Join(cmds, " && ")); err != nil {
		// Roll back partial application.
		rollbackLocked(iface, cfg.GatewayIP, prefix, subnetCIDR, upstreamTable)
		applyErr := fmt.Errorf("failed to install static gateway state: %v, out: %s", err, string(out))
		recordApplyResult(applyErr)
		return applyErr
	}

	current = ownedState{
		Iface:       iface,
		Address:     fmt.Sprintf("%s/%s", cfg.GatewayIP, prefix),
		SubnetCIDR:  subnetCIDR,
		RuleToSub:   rulePrioToSubnet,
		RuleFromAP:  rulePrioFromAP,
		AppliedAt:   time.Now().Format(time.RFC3339),
		UpstreamTbl: upstreamTable,
	}
	persistOwnedState(current)

	recordApplyResult(nil)
	logger.Get().Infof("hotspot", "Static gateway %s installed on %s (upstream table %s)", current.Address, iface, upstreamTable)
	return nil
}

func rollbackLocked(iface, addr, prefix, subnetCIDR, upstreamTable string) {
	cleanupCmd := strings.Join([]string{
		// Delete the exact rules BFR created, matching interface, subnet and
		// table so a foreign rule that happens to share the priority stays.
		fmt.Sprintf("ip rule del to %s lookup main pref %d 2>/dev/null || true", subnetCIDR, rulePrioToSubnet),
		fmt.Sprintf("ip rule del from all iif %s lookup %s pref %d 2>/dev/null || true", iface, upstreamTable, rulePrioFromAP),
		fmt.Sprintf("ip route del %s dev %s src %s proto static scope link table main 2>/dev/null || true", subnetCIDR, iface, addr),
		fmt.Sprintf("ip addr del %s/%s dev %s 2>/dev/null || true", addr, prefix, iface),
	}, "; ")
	_, _ = config.ExecSuTimeout(3*time.Second, cleanupCmd)
	current = ownedState{}
	persistOwnedState(current)
}

// RemoveStaticGateway removes only the objects this feature installed.
func RemoveStaticGateway(iface string) {
	ownedMu.Lock()
	defer ownedMu.Unlock()
	removeOwnedStateLocked()
}

// removeOwnedStateLocked tears down exactly what ApplyStaticGateway created.
func removeOwnedStateLocked() {
	// Fall back to the persisted record so cleanup still works after a
	// WebUI process restart.
	if current.Iface == "" {
		current = loadOwnedState()
	}
	if current.Iface == "" {
		return
	}

	addr, prefix := splitCIDR(current.Address)
	cmds := []string{
		// Match the full rule selectors, not just the priority, so rules
		// owned by other software are never removed.
		fmt.Sprintf("ip rule del to %s lookup main pref %d 2>/dev/null || true", current.SubnetCIDR, current.RuleToSub),
		fmt.Sprintf("ip rule del from all iif %s lookup %s pref %d 2>/dev/null || true", current.Iface, current.UpstreamTbl, current.RuleFromAP),
	}
	if current.SubnetCIDR != "" {
		cmds = append(cmds, fmt.Sprintf("ip route del %s dev %s src %s proto static scope link table main 2>/dev/null || true", current.SubnetCIDR, current.Iface, addr))
	}
	if addr != "" {
		cmds = append(cmds, fmt.Sprintf("ip addr del %s/%s dev %s 2>/dev/null || true", addr, prefix, current.Iface))
	}
	// BFR installs no NAT rule in the Android-managed model; remove defensively
	// only the exact rule that would have been installed for this subnet.
	if current.SubnetCIDR != "" {
		cmds = append(cmds, fmt.Sprintf("iptables -t nat -D POSTROUTING -s %s -j MASQUERADE 2>/dev/null || true", current.SubnetCIDR))
	}

	_, _ = config.ExecSuTimeout(3*time.Second, strings.Join(cmds, "; "))
	logger.Get().Infof("hotspot", "Removed BFR-owned gateway state from %s", current.Iface)
	current = ownedState{}
	persistOwnedState(current)
}

func splitCIDR(cidr string) (string, string) {
	parts := strings.SplitN(cidr, "/", 2)
	if len(parts) != 2 {
		return cidr, ""
	}
	return parts[0], parts[1]
}

// ObserveStaticGateway reads back kernel state so the API can report observed
// facts rather than in-process intent.
//
// Rules are verified by their full selector (interface, subnet, table), not
// by priority alone, so a stale or foreign rule at the same priority cannot
// be mistaken for a correctly applied state.
func ObserveStaticGateway() ObservedState {
	ownedMu.Lock()
	if current.Iface == "" {
		// Fall back to the persisted record so observation survives a restart.
		current = loadOwnedState()
	}
	iface := current.Iface
	addr, prefix := splitCIDR(current.Address)
	subnetCIDR := current.SubnetCIDR
	upstream := current.UpstreamTbl
	ruleToSub := current.RuleToSub
	ruleFromAP := current.RuleFromAP
	ownedMu.Unlock()

	out := ObservedState{Iface: iface, UpstreamTable: upstream, SubnetCIDR: subnetCIDR}
	if iface == "" {
		return out
	}

	if addr != "" {
		out.AddressPresent = addrPresent(iface, addr, prefix)
	}
	if subnetCIDR != "" {
		out.RuleToSubnet = ruleToSubnetPresent(ruleToSub, subnetCIDR)
	}
	out.RuleFromAP = ruleFromAPPresent(ruleFromAP, iface, upstream)

	return out
}

func addrPresent(iface, addr, prefix string) bool {
	out, err := config.ExecSuTimeout(2*time.Second, fmt.Sprintf("ip -o addr show dev %s 2>/dev/null", iface))
	if err != nil {
		return false
	}
	want := addr + "/" + prefix
	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.Contains(line, want+" ") || strings.Contains(line, want+"\t") {
			return true
		}
	}
	return false
}

// ruleToSubnetPresent reports whether the "to <subnet> lookup main" rule
// exists at the given priority and targets the expected subnet.
func ruleToSubnetPresent(pref int, subnetCIDR string) bool {
	for _, line := range ruleLines(pref) {
		tokens := ruleTokens(line)
		if !tokensContain(tokens, "to", subnetCIDR) {
			continue
		}
		if tokensContain(tokens, "lookup", "main") {
			return true
		}
	}
	return false
}

// ruleFromAPPresent reports whether the "from all iif <iface> lookup <table>"
// rule exists at the given priority and targets the expected interface and
// upstream table.
func ruleFromAPPresent(pref int, iface, upstreamTable string) bool {
	if upstreamTable == "" {
		return false
	}
	for _, line := range ruleLines(pref) {
		tokens := ruleTokens(line)
		if !tokensContain(tokens, "iif", iface) {
			continue
		}
		if tokensContain(tokens, "lookup", upstreamTable) {
			return true
		}
	}
	return false
}

// ruleTokens splits a rule line into its selector tokens.
//
// "ip rule show" prints rules as "7010:\tfrom all iif wlan1 lookup wlan0",
// where the table name is the last token and carries no trailing space.
// Matching on tokens rather than on substrings keeps the check correct for
// every field position.
func ruleTokens(line string) []string {
	// Strip the "<pref>:" prefix.
	if idx := strings.Index(line, ":"); idx >= 0 {
		line = line[idx+1:]
	}
	return strings.Fields(line)
}

// tokensContain reports whether the token list holds the given value
// immediately after the given keyword.
func tokensContain(tokens []string, keyword, value string) bool {
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i] == keyword && tokens[i+1] == value {
			return true
		}
	}
	return false
}

// ruleLines returns the normalized rule lines at a given priority.
func ruleLines(pref int) []string {
	out, err := config.ExecSuTimeout(2*time.Second, fmt.Sprintf("ip rule show pref %d 2>/dev/null", pref))
	if err != nil {
		return nil
	}
	var lines []string
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// applyStatus keeps the last runtime apply outcome so the API can report it.
var applyStatus struct {
	sync.Mutex
	err string
	at  string
}

func recordApplyResult(err error) {
	applyStatus.Lock()
	defer applyStatus.Unlock()
	if err != nil {
		applyStatus.err = err.Error()
	} else {
		applyStatus.err = ""
	}
	applyStatus.at = time.Now().Format(time.RFC3339)
}

// GetRouterStatus reports desired configuration and observed kernel state.
func GetRouterStatus() RouterStatus {
	cfg, _ := LoadRouterConfig()

	var st RouterStatus
	if cfg != nil {
		st.DesiredMode = cfg.GatewayMode
		st.DesiredGateway = cfg.GatewayIP
		st.DesiredSubnet = cfg.SubnetMask
		st.DesiredBand = cfg.Band
		st.DesiredUpstream = cfg.UpstreamInterface
	}
	// Android owns DHCP and DNS for its own SoftAP; BFR does not run a second server.
	st.DHCPManagedByOS = true

	observed := ObserveStaticGateway()
	st.ObservedIface = observed.Iface
	st.ObservedGateway = observedGatewayAddress()
	st.ObservedSubnet = observed.SubnetCIDR
	st.UpstreamTable = observed.UpstreamTable
	st.AddressPresent = observed.AddressPresent
	st.RuleToSubnet = observed.RuleToSubnet
	st.RuleFromAP = observed.RuleFromAP

	// The BFR-owned objects are only meaningful while a tether interface is
	// up. With no interface, nothing is applied even if the configuration is
	// saved and the last apply attempt succeeded.
	activeIfaces := GetActiveInterfaceNames()
	tetherUp := len(activeIfaces) > 0
	kernelStatePresent := observed.AddressPresent && observed.RuleToSubnet && observed.RuleFromAP

	// In Auto mode the desired state is "Android manages the gateway", which
	// is satisfied by Android itself and needs no BFR-owned kernel object.
	autoMode := st.DesiredMode != "static"
	st.Applied = tetherUp && (autoMode || kernelStatePresent)

	// The gateway clients actually receive comes from Android's DHCP server,
	// not from the address BFR installs.
	st.ClientGateway, st.ClientSubnet = clientFacingGateway(activeIfaces)
	st.GatewayEffective = st.ClientGateway != "" && st.ClientGateway == st.ObservedGateway

	applyStatus.Lock()
	st.LastApplyError = applyStatus.err
	st.LastApplyTime = applyStatus.at
	applyStatus.Unlock()

	switch {
	case !tetherUp:
		st.ApplyState = ApplyStateInactive
	case autoMode:
		// Android owns the gateway, DHCP and DNS: there is nothing for BFR
		// to apply, so this is not a pending state.
		st.ApplyState = ApplyStateAuto
	case st.LastApplyError != "":
		st.ApplyState = ApplyStateFailed
	case st.Applied:
		st.ApplyState = ApplyStateApplied
	default:
		st.ApplyState = ApplyStatePending
	}

	return st
}

// clientFacingGateway returns the gateway and subnet that Android's DHCP
// server advertises to clients on the tether interface.
//
// Android runs its own DHCP server for the SoftAP downstream, so the address
// it hands out is the one clients use. BFR must report that address, not the
// static one it installs alongside it.
//
// The tethering service reports the exact address it assigned to the
// interface in its LinkProperties, and that is the address used as the
// client-facing gateway. Any address BFR installed itself is excluded,
// because it belongs to a different subnet and is never advertised.
func clientFacingGateway(activeIfaces []string) (gateway string, subnet string) {
	if len(activeIfaces) == 0 {
		return "", ""
	}

	iface := activeIfaces[0]
	if gw, sn, ok := tetheringLinkAddress(iface); ok {
		return gw, sn
	}

	// Fallback: the interface address inside the advertised downstream.
	downstream := tetheringDownstreamSubnet()
	out, err := config.ExecSuTimeout(2*time.Second, fmt.Sprintf("ip -o addr show dev %s 2>/dev/null", iface))
	if err != nil {
		return "", ""
	}

	bfrAddr := observedGatewayAddress()
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f != "inet" || i+1 >= len(fields) {
				continue
			}
			addr, cidrNet, err := net.ParseCIDR(fields[i+1])
			if err != nil {
				continue
			}
			// Skip the address BFR installed: it is not the client gateway.
			if bfrAddr != "" && addr.String() == bfrAddr {
				continue
			}
			// Prefer the address inside the subnet Android advertises.
			if downstream != nil && !cidrNet.Contains(addr) {
				continue
			}
			if gateway == "" {
				gateway = addr.String()
				subnet = cidrNet.String()
			}
		}
	}

	return gateway, subnet
}

// tetheringLinkAddress reads the address Android assigned to the tether
// interface from the tethering service's LinkProperties.
//
// The tethering service logs a line such as:
//
//	OBSERVED LinkProperties update iface=wlan1 state=TETHERED lp={InterfaceName: wlan1 LinkAddresses: [ 10.186.174.146/24 ] ...}
func tetheringLinkAddress(iface string) (gateway string, subnet string, ok bool) {
	out, err := config.ExecSuTimeout(2*time.Second, "dumpsys tethering 2>/dev/null")
	if err != nil {
		return "", "", false
	}

	needle := "iface=" + iface + " state=TETHERED"
	for line := range strings.SplitSeq(string(out), "\n") {
		if !strings.Contains(line, needle) {
			continue
		}
		idx := strings.Index(line, "LinkAddresses: [")
		if idx < 0 {
			continue
		}
		rest := line[idx+len("LinkAddresses: ["):]
		end := strings.Index(rest, "]")
		if end < 0 {
			continue
		}
		first := strings.TrimSpace(strings.Split(rest[:end], ",")[0])
		addr, cidrNet, err := net.ParseCIDR(first)
		if err != nil {
			continue
		}
		return addr.String(), cidrNet.String(), true
	}
	return "", "", false
}

// tetheringDownstreamSubnet reads the subnet Android's tethering service
// hands out to clients. It returns nil when the service does not report one.
func tetheringDownstreamSubnet() *net.IPNet {
	out, err := config.ExecSuTimeout(2*time.Second, "dumpsys tethering 2>/dev/null")
	if err != nil {
		return nil
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		// The downstream shows up as "addDownstream(wlan1, 10.186.174.0/24)".
		idx := strings.Index(line, "addDownstream(")
		if idx < 0 {
			continue
		}
		rest := line[idx+len("addDownstream("):]
		end := strings.Index(rest, ")")
		if end < 0 {
			continue
		}
		parts := strings.Split(rest[:end], ",")
		if len(parts) < 2 {
			continue
		}
		_, cidrNet, err := net.ParseCIDR(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		return cidrNet
	}
	return nil
}

// observedGatewayAddress returns the gateway address BFR recorded, if any.
func observedGatewayAddress() string {
	ownedMu.Lock()
	defer ownedMu.Unlock()
	addr, _ := splitCIDR(current.Address)
	return addr
}