package terminal_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"bfr-webui-go/internal/terminal"
)

func TestHandleWebsocket_Unauthorized(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Request terminal without token should return 401
	req := httptest.NewRequest("GET", "/ws/terminal", nil)
	rr := httptest.NewRecorder()

	terminal.HandleWebsocket(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for unauthenticated terminal request, got %d", rr.Code)
	}
}

func TestConfigureTerminalCmd_EnvironmentAndWorkingDir(t *testing.T) {
	cmd := exec.Command("sh")
	terminal.ConfigureTerminalCmd(cmd)

	if cmd.Dir == "" {
		t.Errorf("expected cmd.Dir to be set, got empty")
	}

	requiredVars := map[string]string{
		"HOME":      "/data/adb",
		"TERM":      "xterm-256color",
		"COLORTERM": "truecolor",
		"USER":      "root",
		"LOGNAME":   "root",
		"TMPDIR":    "/data/local/tmp",
		"SHELL":     "/system/bin/sh",
		"PATH":      "/data/adb/ap/bin:/data/adb/ksu/bin:/data/adb/magisk:/data/data/com.termux/files/usr/bin:/system/bin:/system/xbin:/vendor/bin:/apex/com.android.runtime/bin",
	}

	envMap := make(map[string]string)
	for _, env := range cmd.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	for key, expectedVal := range requiredVars {
		if val, ok := envMap[key]; !ok {
			t.Errorf("expected env var %s to be set in cmd.Env", key)
		} else if val != expectedVal {
			t.Errorf("expected env var %s=%s, got %s", key, expectedVal, val)
		}
	}
}

func TestResizeMessageParsing(t *testing.T) {
	raw := []byte(`{"type":"resize","cols":120,"rows":40}`)
	var msg struct {
		Type string `json:"type"`
		Cols uint16 `json:"cols"`
		Rows uint16 `json:"rows"`
	}

	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("failed to unmarshal resize message: %v", err)
	}

	if msg.Type != "resize" || msg.Cols != 120 || msg.Rows != 40 {
		t.Errorf("unexpected parsed resize message: %+v", msg)
	}
}
