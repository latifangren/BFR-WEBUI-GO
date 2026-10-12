package handlers

import "net/http"

func registerSystemRoutes(rc *routeContext) {
	// Sysinfo URLs
	rc.mux.HandleFunc("/api/stats", rc.wrap(HandleSysinfo, true))
	rc.mux.HandleFunc("/api/sysinfo", rc.wrap(HandleSysinfo, true))
	rc.mux.HandleFunc("/api/sysinfo/governor", rc.wrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			HandleGovernorStatus(w, r)
		} else {
			HandleGovernorSet(w, r)
		}
	}, true))

	// Power URLs
	rc.mux.HandleFunc("/api/power", rc.wrap(HandlePower, true))
	rc.mux.HandleFunc("/api/power/schedule", rc.wrap(HandlePowerSchedule, true))

	// Smart Charger URLs
	rc.mux.HandleFunc("/api/charger/config", rc.wrap(HandleChargerConfig, true))
	rc.mux.HandleFunc("/api/charger/toggle", rc.wrap(HandleChargerToggle, true))
	rc.mux.HandleFunc("/api/charger/probe", rc.wrap(HandleChargerProbe, true))

	// System Modules URLs
	rc.mux.HandleFunc("/api/modules", rc.wrap(HandleModulesList, true))
	rc.mux.HandleFunc("/api/modules/toggle", rc.wrap(HandleModulesToggle, true))
	rc.mux.HandleFunc("/api/modules/install", rc.wrap(HandleModulesInstall, true))

	// System Backup URLs
	rc.mux.HandleFunc("/api/backup/export", rc.wrap(HandleBackupExport, true))
	rc.mux.HandleFunc("/api/backup/import", rc.wrap(HandleBackupImport, true))
	rc.mux.HandleFunc("/api/backup/cloud/config", rc.wrap(HandleCloudBackupConfig, true))
	rc.mux.HandleFunc("/api/backup/cloud/sync", rc.wrap(HandleCloudBackupSync, true))

	// Partition Management URLs
	rc.mux.HandleFunc("/api/partitions", rc.wrap(HandleListPartitions, true))
	rc.mux.HandleFunc("/api/partitions/backup", rc.wrap(HandleBackupPartition, true))
	rc.mux.HandleFunc("/api/partitions/backup-group", rc.wrap(HandleBackupGroup, true))
	rc.mux.HandleFunc("/api/partitions/backups", rc.wrap(HandleListBackups, true))
	rc.mux.HandleFunc("/api/partitions/download", rc.wrap(HandleDownloadBackup, true))
	rc.mux.HandleFunc("/api/partitions/backup/delete", rc.wrap(HandleDeleteBackup, true))
	rc.mux.HandleFunc("/api/partitions/flash", rc.wrap(HandleFlashPartition, true))

	// Theme Wallpaper URLs (list/file stay public so the login screen can preview
	// wallpapers; upload/delete require authentication to prevent disk-fill abuse)
	rc.mux.HandleFunc("/api/theme/wallpapers", rc.wrap(HandleThemeWallpapers, false))
	rc.mux.HandleFunc("/api/theme/wallpapers/upload", rc.wrap(HandleThemeWallpaperUpload, true))
	rc.mux.HandleFunc("/api/theme/wallpapers/file", rc.wrap(HandleThemeWallpaperFile, false))
	rc.mux.HandleFunc("/api/theme/wallpapers/delete", rc.wrap(HandleThemeWallpaperDelete, true))

	// System Logs URLs
	rc.mux.HandleFunc("/api/logs", rc.wrap(HandleLogs, true))
	rc.mux.HandleFunc("/api/logs/clear", rc.wrap(HandleLogsClear, true))
	rc.mux.HandleFunc("/api/logs/logcat/stream", rc.logcatH.HandleWS)
}
