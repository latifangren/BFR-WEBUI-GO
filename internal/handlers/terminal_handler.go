package handlers

import (
	"net/http"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/terminal"
)

type TerminalHandler struct {
	authMgr *auth.Manager
}

func NewTerminalHandler(authMgr *auth.Manager) *TerminalHandler {
	return &TerminalHandler{authMgr: authMgr}
}

func (h *TerminalHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	token := h.authMgr.GetTokenFromRequest(r)
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if !h.authMgr.ValidateSession(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	logger.Get().Infof("Terminal", "Web terminal session started from %s", r.RemoteAddr)
	defer logger.Get().Infof("Terminal", "Web terminal session closed from %s", r.RemoteAddr)
	terminal.HandleWebsocket(w, r)
}
