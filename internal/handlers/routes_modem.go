package handlers

func registerModemRoutes(rc *routeContext) {
	// ─── Core Cellular & Radio Telemetry URLs ──────────────────────────────────
	rc.mux.HandleFunc("/api/modem/signal", rc.wrap(HandleModemSignal, true))
	rc.mux.HandleFunc("/api/modem/sim/status", rc.wrap(HandleModemSimStatus, true))
	rc.mux.HandleFunc("/api/modem/sim/switch", rc.wrap(HandleModemSimSwitch, true))
	rc.mux.HandleFunc("/api/modem/thermal", rc.wrap(HandleModemThermal, true))
	rc.mux.HandleFunc("/api/modem/cell_scan", rc.wrap(HandleModemCellScan, true))

	// ─── SMS Viewer URLs ───────────────────────────────────────────────────────
	rc.mux.HandleFunc("/api/sms/inbox", rc.wrap(HandleSMSInbox, true))

	// ─── Proprietary Vendor Modem Seam ─────────────────────────────────────────
	registerVendorModemRoutes(rc)
}
