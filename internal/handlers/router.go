package handlers

import (
	"compress/gzip"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"bfr-webui-go/frontend"
	"bfr-webui-go/internal/auth"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer  *gzip.Writer
	written bool
}

func (g *gzipResponseWriter) initGzip() {
	if !g.written {
		g.written = true
		g.Header().Del("Content-Length")
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Add("Vary", "Accept-Encoding")
		g.Writer = gzip.NewWriter(g.ResponseWriter)
	}
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	if statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		g.ResponseWriter.WriteHeader(statusCode)
		return
	}
	g.initGzip()
	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	g.initGzip()
	return g.Writer.Write(b)
}

func (g *gzipResponseWriter) Flush() {
	if g.Writer != nil {
		_ = g.Writer.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) Close() {
	if g.Writer != nil {
		_ = g.Writer.Close()
	}
}

// GzipMiddleware wraps an http.Handler to compress responses with gzip.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.ToLower(r.URL.Path)
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") ||
			strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket") ||
			strings.HasPrefix(path, "/api/proxy/logs") ||
			strings.HasSuffix(path, ".jpg") ||
			strings.HasSuffix(path, ".jpeg") ||
			strings.HasSuffix(path, ".png") ||
			strings.HasSuffix(path, ".webp") ||
			strings.HasSuffix(path, ".gif") ||
			strings.HasSuffix(path, ".svg") ||
			strings.HasSuffix(path, ".zip") {
			next.ServeHTTP(w, r)
			return
		}
		gzw := &gzipResponseWriter{ResponseWriter: w}
		defer gzw.Close()
		next.ServeHTTP(gzw, r)
	})
}

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
			if r.URL.Path != "/api/files/upload" && r.URL.Path != "/api/modules/install" && r.URL.Path != "/api/tunnel/upload" && r.URL.Path != "/api/partitions/flash" {
				r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
			}
		}
		next(w, r)
	}
}

type routeContext struct {
	mux     *http.ServeMux
	authMgr *auth.Manager
	wrap    func(h http.HandlerFunc, isProtected bool) http.HandlerFunc
	authH   *AuthHandler
	termH   *TerminalHandler
	scrcpyH *ScrcpyHandler
	logcatH *LogcatHandler
}

func RegisterRoutes(mux *http.ServeMux, authMgr *auth.Manager) {
	distFS, err := frontend.GetFS()
	if err != nil {
		log.Fatalf("Failed to initialize frontend filesystem: %v", err)
	}

	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		log.Fatalf("Failed to read embedded frontend index.html: %v", err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	// SPA Route Handler: Serve static assets if file exists, otherwise serve index.html with no-cache
	mux.HandleFunc("/", securityHeaders(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath != "" && reqPath != "index.html" {
			if f, err := distFS.Open(reqPath); err == nil {
				_ = f.Close()
				if strings.HasPrefix(r.URL.Path, "/assets/") {
					w.Header().Set("Cache-Control", "no-cache, must-revalidate")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexHTML)
	}))

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

	rc := &routeContext{
		mux:     mux,
		authMgr: authMgr,
		wrap:    wrap,
		authH:   authH,
		termH:   termH,
		scrcpyH: scrcpyH,
		logcatH: logcatH,
	}

	// Modular route registration by domain
	registerServiceRoutes(rc)
	registerSystemRoutes(rc)
	registerNetworkRoutes(rc)
	registerModemRoutes(rc)
}
