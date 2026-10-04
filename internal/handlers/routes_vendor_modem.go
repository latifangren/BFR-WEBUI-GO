package handlers

// registerVendorModemRoutes registers proprietary vendor modem controls (Qualcomm, Samsung, Shannon, EFS).
// In the dev branch, this registers all vendor baseband endpoints.
// In the release/no-modem branch, this can be swapped or stubbed without touching routes_modem.go.
func registerVendorModemRoutes(rc *routeContext) {
	// Generic Band Locking & AT URLs
	rc.mux.HandleFunc("/api/modem/bands", rc.wrap(HandleModemBands, true))
	rc.mux.HandleFunc("/api/modem/at", rc.wrap(HandleModemAT, true))
	rc.mux.HandleFunc("/api/modem/reset", rc.wrap(HandleModemReset, true))
	rc.mux.HandleFunc("/api/modem/bands/reset", rc.wrap(HandleModemReset, true))
	rc.mux.HandleFunc("/api/modem/cell_lock", rc.wrap(HandleModemCellLock, true))
	rc.mux.HandleFunc("/api/modem/cell_lock/clear", rc.wrap(HandleModemCellLockClear, true))
	rc.mux.HandleFunc("/api/modem/plmn_lock", rc.wrap(HandleModemPlmnLock, true))
	rc.mux.HandleFunc("/api/modem/hardware_bands", rc.wrap(HandleModemHardwareBands, true))

	// Samsung RIL URLs
	rc.mux.HandleFunc("/api/samsung/status", rc.wrap(HandleSamsungStatus, true))
	rc.mux.HandleFunc("/api/samsung/bands", rc.wrap(HandleSamsungBands, true))
	rc.mux.HandleFunc("/api/samsung/lock", rc.wrap(HandleSamsungLock, true))
	rc.mux.HandleFunc("/api/samsung/auto", rc.wrap(HandleSamsungAuto, true))
	rc.mux.HandleFunc("/api/samsung/sim/switch", rc.wrap(HandleSamsungSimSwitch, true))
	rc.mux.HandleFunc("/api/samsung/thermal", rc.wrap(HandleSamsungThermal, true))

	// Google Tensor Shannon Modem URLs
	rc.mux.HandleFunc("/api/shannon/status", rc.wrap(HandleShannonStatus, true))
	rc.mux.HandleFunc("/api/shannon/lock", rc.wrap(HandleShannonLock, true))
	rc.mux.HandleFunc("/api/shannon/reset", rc.wrap(HandleShannonReset, true))
	rc.mux.HandleFunc("/api/shannon/ca/status", rc.wrap(HandleShannonCAStatus, true))
	rc.mux.HandleFunc("/api/shannon/ca/document", rc.wrap(HandleShannonCAGetDocument, true))
	rc.mux.HandleFunc("/api/shannon/ca/load", rc.wrap(HandleShannonCALoadFile, true))
	rc.mux.HandleFunc("/api/shannon/ca/upload", rc.wrap(HandleShannonCAUpload, true))
	rc.mux.HandleFunc("/api/shannon/ca/save", rc.wrap(HandleShannonCASaveFile, true))
	rc.mux.HandleFunc("/api/shannon/ca/download", rc.wrap(HandleShannonCADownload, true))
	rc.mux.HandleFunc("/api/shannon/ca/validate", rc.wrap(HandleShannonCAValidate, true))
	rc.mux.HandleFunc("/api/shannon/ca/autogen", rc.wrap(HandleShannonCAAutoGen, true))
	rc.mux.HandleFunc("/api/shannon/ca/prune", rc.wrap(HandleShannonCAPrune, true))
	rc.mux.HandleFunc("/api/shannon/ca/clear", rc.wrap(HandleShannonCAClear, true))

	// Google Tensor Shannon Modem 5G NR & EN-DC URLs
	rc.mux.HandleFunc("/api/shannon/nr/status", rc.wrap(HandleShannonNRStatus, true))
	rc.mux.HandleFunc("/api/shannon/nr/document", rc.wrap(HandleShannonNRGetDocument, true))
	rc.mux.HandleFunc("/api/shannon/nr/load", rc.wrap(HandleShannonNRLoadFile, true))
	rc.mux.HandleFunc("/api/shannon/nr/upload", rc.wrap(HandleShannonNRUpload, true))
	rc.mux.HandleFunc("/api/shannon/nr/demo", rc.wrap(HandleShannonNRDemo, true))
	rc.mux.HandleFunc("/api/shannon/nr/download", rc.wrap(HandleShannonNRDownload, true))
	rc.mux.HandleFunc("/api/shannon/nr/clear", rc.wrap(HandleShannonNRClear, true))

	// Qualcomm Snapdragon QMI URLs
	rc.mux.HandleFunc("/api/qualcomm/bands", rc.wrap(HandleModemBands, true))
	rc.mux.HandleFunc("/api/qualcomm/bands/reset", rc.wrap(HandleModemReset, true))
	rc.mux.HandleFunc("/api/qualcomm/reset", rc.wrap(HandleModemReset, true))
	rc.mux.HandleFunc("/api/qualcomm/cell_lock", rc.wrap(HandleModemCellLock, true))
	rc.mux.HandleFunc("/api/qualcomm/cell_lock/clear", rc.wrap(HandleModemCellLockClear, true))
	rc.mux.HandleFunc("/api/qualcomm/cell_scan", rc.wrap(HandleModemCellScan, true))
	rc.mux.HandleFunc("/api/qualcomm/plmn_lock", rc.wrap(HandleModemPlmnLock, true))
	rc.mux.HandleFunc("/api/qualcomm/hardware_bands", rc.wrap(HandleModemHardwareBands, true))

	// Qualcomm Snapdragon EFS2 & DIAG URLs
	rc.mux.HandleFunc("/api/efs/status", rc.wrap(HandleEfsStatus, true))
	rc.mux.HandleFunc("/api/efs/connect", rc.wrap(HandleEfsConnect, true))
	rc.mux.HandleFunc("/api/efs/disconnect", rc.wrap(HandleEfsDisconnect, true))
	rc.mux.HandleFunc("/api/efs/readonly", rc.wrap(HandleEfsReadOnly, true))
	rc.mux.HandleFunc("/api/efs/list", rc.wrap(HandleEfsList, true))
	rc.mux.HandleFunc("/api/efs/read", rc.wrap(HandleEfsRead, true))
	rc.mux.HandleFunc("/api/efs/write", rc.wrap(HandleEfsWrite, true))
	rc.mux.HandleFunc("/api/efs/mkdir", rc.wrap(HandleEfsMkdir, true))
	rc.mux.HandleFunc("/api/efs/unlink", rc.wrap(HandleEfsUnlink, true))
	rc.mux.HandleFunc("/api/efs/rename", rc.wrap(HandleEfsRename, true))
	rc.mux.HandleFunc("/api/efs/chmod", rc.wrap(HandleEfsChmod, true))
	rc.mux.HandleFunc("/api/efs/nv/items", rc.wrap(HandleEfsNVItems, true))
	rc.mux.HandleFunc("/api/efs/nv/read", rc.wrap(HandleEfsNVRead, true))
	rc.mux.HandleFunc("/api/efs/nv/write", rc.wrap(HandleEfsNVWrite, true))
	rc.mux.HandleFunc("/api/efs/nv/backup", rc.wrap(HandleEfsNVBackup, true))
	rc.mux.HandleFunc("/api/efs/nv/restore", rc.wrap(HandleEfsNVRestore, true))
	rc.mux.HandleFunc("/api/efs/nv/format", rc.wrap(HandleEfsNVFormat, true))
	rc.mux.HandleFunc("/api/efs/spc", rc.wrap(HandleEfsSPC, true))
	rc.mux.HandleFunc("/api/efs/ssr", rc.wrap(HandleEfsSSR, true))
	rc.mux.HandleFunc("/api/efs/backup", rc.wrap(HandleEfsBackup, true))
	rc.mux.HandleFunc("/api/efs/carrier-policy/parse", rc.wrap(HandleEfsParseCarrierPolicy, true))
	rc.mux.HandleFunc("/api/efs/imei", rc.wrap(HandleEfsIMEIRead, true))
	rc.mux.HandleFunc("/api/efs/imei/write", rc.wrap(HandleEfsIMEIWrite, true))
}
