package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/filemanager"
	"bfr-webui-go/internal/network"
	"bfr-webui-go/internal/power"
	"bfr-webui-go/internal/proxy"
	"bfr-webui-go/internal/sysinfo"
	"bfr-webui-go/internal/terminal"
	"bfr-webui-go/web"
)

type loginRequest struct {
	Password string `json:"password"`
}

type powerRequest struct {
	Action string `json:"action"`
}

type sysctlRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ttlRequest struct {
	Enable bool `json:"enable"`
	TTL    int  `json:"ttl"`
}

type interfaceRequest struct {
	Interface  string `json:"interface"`
	MTU        int    `json:"mtu"`
	TxQueueLen int    `json:"txqueuelen"`
}

type dnsRequest struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}

type pingRequest struct {
	Host  string `json:"host"`
	Count int    `json:"count"`
}

type proxyControlRequest struct {
	Action string `json:"action"`
	Mode   string `json:"mode"`
}

type fileSaveRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type filePathRequest struct {
	Path string `json:"path"`
}

func main() {
	port := flag.String("port", "8080", "HTTP server port")
	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	authPass := os.Getenv("BFR_PASSWORD")
	authMgr := auth.NewManager(authPass)

	subFS, err := fs.Sub(web.Files, ".")
	if err != nil {
		log.Fatalf("Failed to load embedded files: %v", err)
	}

	fileServer := http.FileServer(http.FS(subFS))
	mux := http.NewServeMux()

	// Static files / WebUI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			fileServer.ServeHTTP(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	// Auth APIs
	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		token := authMgr.GetTokenFromRequest(r)
		isAuth := authMgr.ValidateSession(token)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": isAuth,
		})
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid request body",
			})
			return
		}

		token, ok := authMgr.Authenticate(req.Password)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid password",
			})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(auth.SessionDuration),
			SameSite: http.SameSiteLaxMode,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	})

	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := authMgr.GetTokenFromRequest(r)
		authMgr.Logout(token)

		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	})

	// Protected API Middleware
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := authMgr.GetTokenFromRequest(r)
			if !authMgr.ValidateSession(token) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Unauthorized",
				})
				return
			}
			next(w, r)
		}
	}

	// 1. Sysinfo Endpoint
	mux.HandleFunc("/api/stats", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		stats, err := sysinfo.GetStats()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	}))

	mux.HandleFunc("/api/sysinfo", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		stats, err := sysinfo.GetStats()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	}))

	// 2. Power Endpoint
	mux.HandleFunc("/api/power", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req powerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
			return
		}

		act := power.Action(strings.ToLower(req.Action))
		go func() {
			time.Sleep(500 * time.Millisecond)
			if err := power.Execute(act); err != nil {
				log.Printf("Power execution error: %v", err)
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Executing action: %s", act),
		})
	}))

	// 3. Network Tweaks Endpoint
	mux.HandleFunc("/api/network/tweaks", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			ifaces, _ := network.GetInterfaces()
			ttlStatus := network.GetTTLSpoofStatus()
			tcpCongestion, _ := network.GetSysctl("net.ipv4.tcp_congestion_control")
			tcpFastOpen, _ := network.GetSysctl("net.ipv4.tcp_fastopen")

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"interfaces":     ifaces,
				"ttl_spoof":      ttlStatus,
				"tcp_congestion": tcpCongestion,
				"tcp_fastopen":   tcpFastOpen,
				"preset_dns":     network.PresetDNS,
			})
			return
		}

		if r.Method == http.MethodPost {
			action := r.URL.Query().Get("action")
			switch action {
			case "sysctl":
				var req sysctlRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					err := network.SetSysctl(req.Key, req.Value)
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
					return
				}
			case "ttl":
				var req ttlRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					err := network.SetTTLSpoof(req.Enable, req.TTL)
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
					return
				}
			case "interface":
				var req interfaceRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					err := network.SetInterfaceConfig(req.Interface, req.MTU, req.TxQueueLen)
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
					return
				}
			case "dns":
				var req dnsRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					err := network.SetDNS(req.Primary, req.Secondary)
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid action"})
		}
	}))

	// 4. Ping Endpoint
	mux.HandleFunc("/api/network/ping", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req pingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Host == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid host"})
			return
		}
		out, err := sysinfo.RunPing(req.Host, req.Count)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"output":  out,
			"success": err == nil,
		})
	}))

	// 5. Proxy Endpoints
	mux.HandleFunc("/api/proxy/status", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		cores := proxy.DetectCores()
		mode := proxy.GetMode()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"cores": cores,
			"mode":  mode,
		})
	}))

	mux.HandleFunc("/api/proxy/control", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req proxyControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid payload"})
			return
		}

		if req.Mode != "" {
			err := proxy.SetMode(req.Mode)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
			return
		}

		if req.Action != "" {
			err := proxy.ControlService(req.Action)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "No action specified"})
	}))

	mux.HandleFunc("/api/proxy/logs", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		proxy.StreamLogs(w, r)
	}))

	// 6. Terminal WebSocket Endpoint
	mux.HandleFunc("/api/terminal/ws", func(w http.ResponseWriter, r *http.Request) {
		token := authMgr.GetTokenFromRequest(r)
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if !authMgr.ValidateSession(token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		terminal.HandleWebsocket(w, r)
	})

	// 7. File Manager Endpoints
	mux.HandleFunc("/api/files/list", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		dirPath := r.URL.Query().Get("path")
		files, currentPath, err := filemanager.ListDirectory(dirPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path":  currentPath,
			"files": files,
		})
	}))

	mux.HandleFunc("/api/files/read", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Query().Get("path")
		content, err := filemanager.ReadFile(filePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    filePath,
			"content": content,
		})
	}))

	mux.HandleFunc("/api/files/save", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req fileSaveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
			return
		}
		err := filemanager.SaveFile(req.Path, req.Content)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   fmt.Sprintf("%v", err),
		})
	}))

	mux.HandleFunc("/api/files/upload", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseMultipartForm(32 << 20) // 32MB max
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to parse form"})
			return
		}
		dirPath := r.FormValue("path")
		file, header, err := r.FormFile("file")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "No file uploaded"})
			return
		}
		defer file.Close()

		err = filemanager.UploadFile(dirPath, header.Filename, file)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   fmt.Sprintf("%v", err),
		})
	}))

	mux.HandleFunc("/api/files/download", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Query().Get("path")
		filemanager.DownloadFile(filePath, w, r)
	}))

	mux.HandleFunc("/api/files/delete", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req filePathRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid path"})
			return
		}
		err := filemanager.DeletePath(req.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   fmt.Sprintf("%v", err),
		})
	}))

	mux.HandleFunc("/api/files/mkdir", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req filePathRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid directory path"})
			return
		}
		err := filemanager.CreateDir(req.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   fmt.Sprintf("%v", err),
		})
	}))

	addr := ":" + *port
	log.Printf("BFR WEBUI GO starting on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
