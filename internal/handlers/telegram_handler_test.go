package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
)

func TestHandleTelegramStatus(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Wrong method
	reqPost := httptest.NewRequest("POST", "/api/telegram/status", nil)
	rrPost := httptest.NewRecorder()
	handlers.HandleTelegramStatus(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST, got %d", rrPost.Code)
	}

	// GET
	reqGet := httptest.NewRequest("GET", "/api/telegram/status", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleTelegramStatus(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("expected 200 for GET, got %d", rrGet.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rrGet.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["has_token"]; !ok {
		t.Errorf("expected has_token in status response")
	}
	if _, ok := resp["has_chat_id"]; !ok {
		t.Errorf("expected has_chat_id in status response")
	}
	if _, ok := resp["running"]; !ok {
		t.Errorf("expected running in status response")
	}
	if _, ok := resp["enabled"]; !ok {
		t.Errorf("expected enabled in status response")
	}
}

func TestHandleTelegramConfig_Get(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	req := httptest.NewRequest("GET", "/api/telegram/config", nil)
	rr := httptest.NewRecorder()
	handlers.HandleTelegramConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedKeys := []string{"config", "running", "bot_name", "enabled", "bot_token", "allowed_chat_ids", "notify_on_boot", "notifications"}
	for _, key := range expectedKeys {
		if _, ok := resp[key]; !ok {
			t.Errorf("expected key %q in GET config response", key)
		}
	}
}

func TestHandleTelegramConfig_PostFlexible(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Save with string chat_id and flat fields
	payload := map[string]interface{}{
		"enabled":        true,
		"bot_token":      "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
		"chat_id":        "987654321",
		"notify_on_boot": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/telegram/config", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handlers.HandleTelegramConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != nil {
		t.Errorf("expected error to be nil, got %v", resp["error"])
	}
	if resp["chat_id"] != "987654321" {
		t.Errorf("expected chat_id '987654321', got %v", resp["chat_id"])
	}
	if _, ok := resp["enabled"]; !ok {
		t.Errorf("expected enabled field in response")
	}

	// Verify GET now returns the chat_id
	reqGet := httptest.NewRequest("GET", "/api/telegram/config", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandleTelegramConfig(rrGet, reqGet)

	var getResp map[string]interface{}
	_ = json.Unmarshal(rrGet.Body.Bytes(), &getResp)
	if getResp["chat_id"] != "987654321" {
		t.Errorf("expected GET config chat_id '987654321', got %v", getResp["chat_id"])
	}

	// Reset chat_id with empty string
	resetPayload := map[string]interface{}{
		"chat_id": "",
	}
	resetBody, _ := json.Marshal(resetPayload)
	reqReset := httptest.NewRequest("POST", "/api/telegram/config", bytes.NewBuffer(resetBody))
	rrReset := httptest.NewRecorder()
	handlers.HandleTelegramConfig(rrReset, reqReset)

	if rrReset.Code != http.StatusOK {
		t.Fatalf("expected 200 for reset, got %d", rrReset.Code)
	}

	var resetResp map[string]interface{}
	_ = json.Unmarshal(rrReset.Body.Bytes(), &resetResp)
	if resetResp["chat_id"] != "" {
		t.Errorf("expected empty string chat_id after reset, got %v", resetResp["chat_id"])
	}
}
