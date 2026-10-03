package handlers

import "net/http"

func registerServiceRoutes(rc *routeContext) {
	// Auth URLs
	rc.mux.HandleFunc("/api/auth/status", rc.wrap(rc.authH.Status, false))
	rc.mux.HandleFunc("/api/auth/login", rc.wrap(rc.authH.Login, false))
	rc.mux.HandleFunc("/api/auth/logout", rc.wrap(rc.authH.Logout, false))
	rc.mux.HandleFunc("/api/auth/change-password", rc.wrap(rc.authH.ChangePassword, true))

	// File Manager URLs
	rc.mux.HandleFunc("/api/files/list", rc.wrap(HandleFilesList, true))
	rc.mux.HandleFunc("/api/files/upload", rc.wrap(HandleFilesUpload, true))
	rc.mux.HandleFunc("/api/files/download", rc.wrap(HandleFilesDownload, true))
	rc.mux.HandleFunc("/api/files/delete", rc.wrap(HandleFilesDelete, true))
	rc.mux.HandleFunc("/api/files/rename", rc.wrap(HandleFilesRename, true))
	rc.mux.HandleFunc("/api/files/mkdir", rc.wrap(HandleFilesMkdir, true))
	rc.mux.HandleFunc("/api/files/create", rc.wrap(HandleFilesCreate, true))
	rc.mux.HandleFunc("/api/files/read", rc.wrap(HandleFilesRead, true))
	rc.mux.HandleFunc("/api/files/save", rc.wrap(HandleFilesSave, true))
	rc.mux.HandleFunc("/api/files/copy", rc.wrap(HandleFilesCopy, true))
	rc.mux.HandleFunc("/api/files/move", rc.wrap(HandleFilesMove, true))
	rc.mux.HandleFunc("/api/files/batch", rc.wrap(HandleFilesBatch, true))
	rc.mux.HandleFunc("/api/files/permissions", rc.wrap(HandleFilesPermissions, true))
	rc.mux.HandleFunc("/api/files/compress", rc.wrap(HandleFilesCompress, true))
	rc.mux.HandleFunc("/api/files/extract", rc.wrap(HandleFilesExtract, true))
	rc.mux.HandleFunc("/api/files/search", rc.wrap(HandleFilesSearch, true))
	rc.mux.HandleFunc("/api/files/storage", rc.wrap(HandleFilesStorage, true))

	// Custom App Shortcuts URLs
	rc.mux.HandleFunc("/api/shortcuts/list", rc.wrap(HandleShortcutsList, true))
	rc.mux.HandleFunc("/api/shortcuts/save", rc.wrap(HandleShortcutsSave, true))
	rc.mux.HandleFunc("/api/shortcuts/delete", rc.wrap(HandleShortcutsDelete, true))

	// Local NAS File Server URLs
	rc.mux.HandleFunc("/api/nas/status", rc.wrap(HandleNASStatus, true))
	rc.mux.HandleFunc("/api/nas/start", rc.wrap(HandleNASStart, true))
	rc.mux.HandleFunc("/api/nas/stop", rc.wrap(HandleNASStop, true))

	// Telegram URLs
	rc.mux.HandleFunc("/api/telegram/status", rc.wrap(HandleTelegramStatus, true))
	rc.mux.HandleFunc("/api/telegram/config", rc.wrap(HandleTelegramConfig, true))
	rc.mux.HandleFunc("/api/telegram/control", rc.wrap(HandleTelegramControl, true))

	// SSH URLs
	rc.mux.HandleFunc("/api/ssh/status", rc.wrap(HandleSSHStatus, true))
	rc.mux.HandleFunc("/api/ssh/config", rc.wrap(HandleSSHConfig, true))
	rc.mux.HandleFunc("/api/ssh/control", rc.wrap(HandleSSHControl, true))

	// Remote Screen Scrcpy WS
	rc.mux.HandleFunc("/api/scrcpy/ws", rc.scrcpyH.HandleWS)

	// Terminal WS (WS connection does auth verification inside handle method)
	rc.mux.HandleFunc("/api/terminal/ws", rc.termH.HandleWS)

	// API Docs URL
	rc.mux.HandleFunc("/docs", rc.wrap(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!doctype html>
<html>
<head>
  <title>BFR WEBUI GO API Docs</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>body { margin: 0; background: #0f172a; color: #fff; }</style>
</head>
<body>
  <script id="api-reference" data-url="/openapi.json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  <noscript><p style="padding:20px;">OpenAPI specification is available at <a href="/openapi.json" style="color:#60a5fa">/openapi.json</a></p></noscript>
</body>
</html>`
		_, _ = w.Write([]byte(html))
	}, false))
}
