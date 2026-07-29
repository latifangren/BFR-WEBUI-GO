package handlers

import (
	"net/http"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/scrcpy"
)

type ScrcpyHandler struct {
	authMgr *auth.Manager
}

func NewScrcpyHandler(authMgr *auth.Manager) *ScrcpyHandler {
	return &ScrcpyHandler{authMgr: authMgr}
}

func (h *ScrcpyHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	token := h.authMgr.GetTokenFromRequest(r)
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if !h.authMgr.ValidateSession(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scrcpy.HandleWebsocket(w, r)
}
