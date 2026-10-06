package handlers

func registerNetworkRoutes(rc *routeContext) {
	// Ping, DNS, TTL URLs
	rc.mux.HandleFunc("/api/network/ping", rc.wrap(HandlePing, true))
	rc.mux.HandleFunc("/api/network/dns", rc.wrap(HandleDNS, true))
	rc.mux.HandleFunc("/api/network/dns/reset", rc.wrap(HandleDNSReset, true))
	rc.mux.HandleFunc("/api/network/ttl", rc.wrap(HandleTTL, true))
	// SoftAP / Hotspot URLs
	rc.mux.HandleFunc("/api/hotspot/status", rc.wrap(HandleHotspotStatus, true))
	rc.mux.HandleFunc("/api/hotspot/control", rc.wrap(HandleHotspotControl, true))
	rc.mux.HandleFunc("/api/hotspot/clients", rc.wrap(HandleHotspotClients, true))
	rc.mux.HandleFunc("/api/hotspot/mac-filter", rc.wrap(HandleHotspotMACFilter, true))

	// Proxy / Clash / Mihomo URLs
	rc.mux.HandleFunc("/api/proxy/status", rc.wrap(HandleProxyStatus, true))
	rc.mux.HandleFunc("/api/proxy/control", rc.wrap(HandleProxyControl, true))
	rc.mux.HandleFunc("/api/proxy/settings", rc.wrap(HandleProxySettings, true))
	rc.mux.HandleFunc("/api/proxy/apps", rc.wrap(HandleProxyApps, true))
	rc.mux.HandleFunc("/api/proxy/hotspot", rc.wrap(HandleProxyHotspot, true))
	rc.mux.HandleFunc("/api/proxy/subs", rc.wrap(HandleProxySubs, true))
	rc.mux.HandleFunc("/api/proxy/nodes", rc.wrap(HandleProxyNodes, true))
	rc.mux.HandleFunc("/api/proxy/nodes/select", rc.wrap(HandleProxyNodeSelect, true))
	rc.mux.HandleFunc("/api/proxy/delay", rc.wrap(HandleProxyDelay, true))
	rc.mux.HandleFunc("/api/proxy/logs", rc.wrap(HandleProxyLogs, true))
	rc.mux.HandleFunc("/api/proxy/watchdog", rc.wrap(HandleProxyWatchdog, true))
	rc.mux.HandleFunc("/api/proxy/config", rc.wrap(HandleProxyConfig, true))

	// QoS Bandwidth Shaping URLs
	rc.mux.HandleFunc("/api/qos/status", rc.wrap(HandleQoSStatus, true))
	rc.mux.HandleFunc("/api/qos/apply", rc.wrap(HandleQoSApply, true))
	rc.mux.HandleFunc("/api/qos/clear", rc.wrap(HandleQoSClear, true))

	// Vnstat Bandwidth Monitoring URLs
	rc.mux.HandleFunc("/api/vnstat/stats", rc.wrap(HandleVnstatStats, true))
	rc.mux.HandleFunc("/api/vnstat/reset", rc.wrap(HandleVnstatReset, true))

	// Speedtest URLs
	rc.mux.HandleFunc("/api/speedtest/start", rc.wrap(HandleSpeedtestStart, true))
	rc.mux.HandleFunc("/api/speedtest/status", rc.wrap(HandleSpeedtestStatus, true))
	rc.mux.HandleFunc("/api/speedtest/stop", rc.wrap(HandleSpeedtestStop, true))
	rc.mux.HandleFunc("/api/speedtest/history", rc.wrap(HandleSpeedtestHistory, true))

	// Cloudflare / Remote Tunnel URLs
	rc.mux.HandleFunc("/api/tunnel/status", rc.wrap(HandleTunnelStatus, true))
	rc.mux.HandleFunc("/api/tunnel/start", rc.wrap(HandleTunnelStart, true))
	rc.mux.HandleFunc("/api/tunnel/stop", rc.wrap(HandleTunnelStop, true))
	rc.mux.HandleFunc("/api/tunnel/upload", rc.wrap(HandleTunnelUploadBinary, true))
}
