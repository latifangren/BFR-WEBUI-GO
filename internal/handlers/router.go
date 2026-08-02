package handlers

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/web"
)

// L-3: securityHeaders middleware sets security headers on responses.
func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next(w, r)
	}
}

// M-1: maxBodySize middleware enforces 1MB max body size on POST/PUT/PATCH requests using http.MaxBytesReader.
func maxBodySize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if r.URL.Path != "/api/files/upload" && r.URL.Path != "/api/modules/install" {
				r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
			}
		}
		next(w, r)
	}
}

func RegisterRoutes(mux *http.ServeMux, authMgr *auth.Manager) {
	var htmlFiles []string
	_ = fs.WalkDir(web.Files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			htmlFiles = append(htmlFiles, path)
		}
		return nil
	})

	tmpl, tmplErr := template.New("index.html").ParseFS(web.Files, htmlFiles...)
	if tmplErr != nil {
		log.Fatalf("Template parse error: %v", tmplErr)
	}

	subFS, err := fs.Sub(web.Files, ".")
	if err == nil {
		fileServer := http.FileServer(http.FS(subFS))
		mux.HandleFunc("/", securityHeaders(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" && r.URL.Path != "/index.html" {
				fileServer.ServeHTTP(w, r)
				return
			}
			if tmpl != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if err := tmpl.Execute(w, nil); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
			fileServer.ServeHTTP(w, r)
		}))
	}

	authH := NewAuthHandler(authMgr)
	termH := NewTerminalHandler(authMgr)
	scrcpyH := NewScrcpyHandler(authMgr)
	logcatH := NewLogcatHandler(authMgr)

	// Helper wrapper combining security headers, max body limit, and optional auth
	wrap := func(h http.HandlerFunc, isProtected bool) http.HandlerFunc {
		fn := h
		if isProtected {
			fn = authH.RequireAuth(fn)
		}
		return securityHeaders(maxBodySize(fn))
	}

	// Auth URLs
	mux.HandleFunc("/api/auth/status", wrap(authH.Status, false))
	mux.HandleFunc("/api/auth/login", wrap(authH.Login, false))
	mux.HandleFunc("/api/auth/logout", wrap(authH.Logout, false))

	// Sysinfo URLs
	mux.HandleFunc("/api/stats", wrap(HandleSysinfo, true))
	mux.HandleFunc("/api/sysinfo", wrap(HandleSysinfo, true))
	mux.HandleFunc("/api/sysinfo/governor", wrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			HandleGovernorStatus(w, r)
		} else {
			HandleGovernorSet(w, r)
		}
	}, true))

	// Power URLs
	mux.HandleFunc("/api/power", wrap(HandlePower, true))

	// Network URLs
	mux.HandleFunc("/api/network/tweaks", wrap(HandleNetworkTweaks, true))
	mux.HandleFunc("/api/network/ping", wrap(HandlePing, true))
	mux.HandleFunc("/api/network/dns", wrap(HandleDNS, true))
	mux.HandleFunc("/api/network/rps", wrap(HandleRPS, true))
	mux.HandleFunc("/api/network/ttl", wrap(HandleTTL, true))

	// Proxy URLs
	mux.HandleFunc("/api/proxy/status", wrap(HandleProxyStatus, true))
	mux.HandleFunc("/api/proxy/control", wrap(HandleProxyControl, true))
	mux.HandleFunc("/api/proxy/logs", wrap(HandleProxyLogs, true))
	mux.HandleFunc("/api/proxy/watchdog", wrap(HandleProxyWatchdog, true))
	mux.HandleFunc("/api/proxy/config", wrap(HandleProxyConfig, true))

	// Files URLs
	mux.HandleFunc("/api/files/list", wrap(HandleFilesList, true))
	mux.HandleFunc("/api/files/read", wrap(HandleFilesRead, true))
	mux.HandleFunc("/api/files/save", wrap(HandleFilesSave, true))
	mux.HandleFunc("/api/files/upload", wrap(HandleFilesUpload, true))
	mux.HandleFunc("/api/files/download", wrap(HandleFilesDownload, true))
	mux.HandleFunc("/api/files/delete", wrap(HandleFilesDelete, true))
	mux.HandleFunc("/api/files/mkdir", wrap(HandleFilesMkdir, true))
	mux.HandleFunc("/api/files/rename", wrap(HandleFilesRename, true))
	mux.HandleFunc("/api/files/create", wrap(HandleFilesCreate, true))
	mux.HandleFunc("/api/files/copy", wrap(HandleFilesCopy, true))
	mux.HandleFunc("/api/files/move", wrap(HandleFilesMove, true))
	mux.HandleFunc("/api/files/batch", wrap(HandleFilesBatch, true))
	mux.HandleFunc("/api/files/permissions", wrap(HandleFilesPermissions, true))
	mux.HandleFunc("/api/files/compress", wrap(HandleFilesCompress, true))
	mux.HandleFunc("/api/files/extract", wrap(HandleFilesExtract, true))
	mux.HandleFunc("/api/files/search", wrap(HandleFilesSearch, true))
	mux.HandleFunc("/api/files/storage", wrap(HandleFilesStorage, true))

	// Hotspot URLs
	mux.HandleFunc("/api/hotspot/status", wrap(HandleHotspotStatus, true))
	mux.HandleFunc("/api/hotspot/control", wrap(HandleHotspotControl, true))
	mux.HandleFunc("/api/hotspot/clients", wrap(HandleHotspotClients, true))

	// Vnstat URLs
	mux.HandleFunc("/api/vnstat/stats", wrap(HandleVnstatStats, true))
	mux.HandleFunc("/api/vnstat/reset", wrap(HandleVnstatReset, true))

	// Smart Charger URLs
	mux.HandleFunc("/api/charger/config", wrap(HandleChargerConfig, true))
	mux.HandleFunc("/api/charger/toggle", wrap(HandleChargerToggle, true))

	// Telegram URLs
	mux.HandleFunc("/api/telegram/status", wrap(HandleTelegramStatus, true))
	mux.HandleFunc("/api/telegram/config", wrap(HandleTelegramConfig, true))
	mux.HandleFunc("/api/telegram/control", wrap(HandleTelegramControl, true))

	// SSH URLs
	mux.HandleFunc("/api/ssh/status", wrap(HandleSSHStatus, true))
	mux.HandleFunc("/api/ssh/config", wrap(HandleSSHConfig, true))
	mux.HandleFunc("/api/ssh/control", wrap(HandleSSHControl, true))

	// Logs URLs
	mux.HandleFunc("/api/logs", wrap(HandleLogs, true))
	mux.HandleFunc("/api/logs/clear", wrap(HandleLogsClear, true))
	mux.HandleFunc("/api/logs/logcat/stream", logcatH.HandleWS)

	// Modules URLs
	mux.HandleFunc("/api/modules", wrap(HandleModulesList, true))
	mux.HandleFunc("/api/modules/toggle", wrap(HandleModulesToggle, true))
	mux.HandleFunc("/api/modules/install", wrap(HandleModulesInstall, true))

	// Backup URLs
	mux.HandleFunc("/api/backup/export", wrap(HandleBackupExport, true))
	mux.HandleFunc("/api/backup/import", wrap(HandleBackupImport, true))
	mux.HandleFunc("/api/backup/cloud/config", wrap(HandleCloudBackupConfig, true))
	mux.HandleFunc("/api/backup/cloud/sync", wrap(HandleCloudBackupSync, true))

	// Speedtest URLs
	mux.HandleFunc("/api/speedtest/start", wrap(HandleSpeedtestStart, true))
	mux.HandleFunc("/api/speedtest/status", wrap(HandleSpeedtestStatus, true))
	mux.HandleFunc("/api/speedtest/stop", wrap(HandleSpeedtestStop, true))
	mux.HandleFunc("/api/speedtest/history", wrap(HandleSpeedtestHistory, true))

	// SMS Viewer URLs
	mux.HandleFunc("/api/sms/inbox", wrap(HandleSMSInbox, true))

	// Docs URL
	mux.HandleFunc("/docs", wrap(func(w http.ResponseWriter, r *http.Request) {
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

	// Remote Screen Scrcpy WS
	mux.HandleFunc("/api/scrcpy/ws", scrcpyH.HandleWS)

	// Terminal WS (WS connection does auth verification inside handle method)
	mux.HandleFunc("/api/terminal/ws", termH.HandleWS)
}
