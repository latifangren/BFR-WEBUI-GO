package handlers

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os/exec"
	"sync"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
	"github.com/gorilla/websocket"
)

var upgraderLogcat = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type LogcatHandler struct {
	authMgr *auth.Manager
}

func NewLogcatHandler(authMgr *auth.Manager) *LogcatHandler {
	return &LogcatHandler{authMgr: authMgr}
}

func (h *LogcatHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	token := h.authMgr.GetTokenFromRequest(r)
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if !h.authMgr.ValidateSession(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgraderLogcat.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Logcat WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	conn.SetReadLimit(512 * 1024)
	logger.Get().Infof("logcat", "Live logcat stream connected from %s", r.RemoteAddr)
	defer logger.Get().Infof("logcat", "Live logcat stream closed from %s", r.RemoteAddr)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	cmd := exec.CommandContext(ctx, config.SUBin, "-c", "logcat -v time")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to pipe logcat stdout\r\n"))
		return
	}

	if err := cmd.Start(); err != nil {
		cmdFallback := exec.CommandContext(ctx, "logcat", "-v", "time")
		stdout, err = cmdFallback.StdoutPipe()
		if err != nil || cmdFallback.Start() != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to start logcat process\r\n"))
			return
		}
		cmd = cmdFallback
	}

	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	var writeMux sync.Mutex
	writeWS := func(msg string) error {
		writeMux.Lock()
		defer writeMux.Unlock()
		return conn.WriteMessage(websocket.TextMessage, []byte(msg))
	}

	// Incoming message discarder loop (handles close signal)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				break
			}
		}
	}()

	// Read logcat stdout scanner loop
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if err := writeWS(line + "\n"); err != nil {
			break
		}
	}
}
