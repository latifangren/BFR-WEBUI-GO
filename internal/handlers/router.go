package handlers

import (
	"html/template"
	"io/fs"
	"net/http"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/web"
)

func RegisterRoutes(mux *http.ServeMux, authMgr *auth.Manager) {
	tmpl, tmplErr := template.New("index.html").ParseFS(web.Files, "index.html", "templates/*.html")

	subFS, err := fs.Sub(web.Files, ".")
	if err == nil {
		fileServer := http.FileServer(http.FS(subFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" && r.URL.Path != "/index.html" {
				fileServer.ServeHTTP(w, r)
				return
			}
			if tmplErr == nil && tmpl != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if err := tmpl.Execute(w, nil); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	authH := NewAuthHandler(authMgr)
	termH := NewTerminalHandler(authMgr)
	scrcpyH := NewScrcpyHandler(authMgr)

	// Auth URLs
	mux.HandleFunc("/api/auth/status", authH.Status)
	mux.HandleFunc("/api/auth/login", authH.Login)
	mux.HandleFunc("/api/auth/logout", authH.Logout)

	// Middleware protection wrapper
	protected := authH.RequireAuth

	// Sysinfo URLs
	mux.HandleFunc("/api/stats", protected(HandleSysinfo))
	mux.HandleFunc("/api/sysinfo", protected(HandleSysinfo))

	// Power URLs
	mux.HandleFunc("/api/power", protected(HandlePower))

	// Network URLs
	mux.HandleFunc("/api/network/tweaks", protected(HandleNetworkTweaks))
	mux.HandleFunc("/api/network/ping", protected(HandlePing))
	mux.HandleFunc("/api/network/dns", protected(HandleDNS))
	mux.HandleFunc("/api/network/rps", protected(HandleRPS))
	mux.HandleFunc("/api/network/ttl", protected(HandleTTL))

	// Proxy URLs
	mux.HandleFunc("/api/proxy/status", protected(HandleProxyStatus))
	mux.HandleFunc("/api/proxy/control", protected(HandleProxyControl))
	mux.HandleFunc("/api/proxy/logs", protected(HandleProxyLogs))
	mux.HandleFunc("/api/proxy/watchdog", protected(HandleProxyWatchdog))
	mux.HandleFunc("/api/proxy/config", protected(HandleProxyConfig))

	// Files URLs
	mux.HandleFunc("/api/files/list", protected(HandleFilesList))
	mux.HandleFunc("/api/files/read", protected(HandleFilesRead))
	mux.HandleFunc("/api/files/save", protected(HandleFilesSave))
	mux.HandleFunc("/api/files/upload", protected(HandleFilesUpload))
	mux.HandleFunc("/api/files/download", protected(HandleFilesDownload))
	mux.HandleFunc("/api/files/delete", protected(HandleFilesDelete))
	mux.HandleFunc("/api/files/mkdir", protected(HandleFilesMkdir))
	mux.HandleFunc("/api/files/rename", protected(HandleFilesRename))
	mux.HandleFunc("/api/files/create", protected(HandleFilesCreate))

	// Hotspot URLs
	mux.HandleFunc("/api/hotspot/status", protected(HandleHotspotStatus))
	mux.HandleFunc("/api/hotspot/control", protected(HandleHotspotControl))
	mux.HandleFunc("/api/hotspot/clients", protected(HandleHotspotClients))

	// Vnstat URLs
	mux.HandleFunc("/api/vnstat/stats", protected(HandleVnstatStats))
	mux.HandleFunc("/api/vnstat/reset", protected(HandleVnstatReset))

	// Smart Charger URLs
	mux.HandleFunc("/api/charger/config", protected(HandleChargerConfig))
	mux.HandleFunc("/api/charger/toggle", protected(HandleChargerToggle))

	// SMS Viewer URLs
	mux.HandleFunc("/api/sms/inbox", protected(HandleSMSInbox))

	// Remote Screen Scrcpy WS
	mux.HandleFunc("/api/scrcpy/ws", scrcpyH.HandleWS)

	// Terminal WS (WS connection does auth verification inside handle method)
	mux.HandleFunc("/api/terminal/ws", termH.HandleWS)
}
