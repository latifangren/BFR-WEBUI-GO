package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/handlers"
)

func setupTestAuthHandler(t *testing.T) (*handlers.AuthHandler, *auth.Manager) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	authMgr := auth.NewManager("")
	authHandler := handlers.NewAuthHandler(authMgr)
	return authHandler, authMgr
}

func TestAuthHandler_Status(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	rr := httptest.NewRecorder()

	handler.Status(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp["authenticated"] != false {
		t.Errorf("expected authenticated to be false")
	}
	if resp["is_default_pass"] != true {
		t.Errorf("expected is_default_pass to be true")
	}
}

func TestAuthHandler_Login_SuccessAndFailure(t *testing.T) {
	handler, _ := setupTestAuthHandler(t)

	// Invalid password
	loginBodyBad, _ := json.Marshal(map[string]string{"password": "wrong"})
	reqBad := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginBodyBad))
	rrBad := httptest.NewRecorder()

	handler.Login(rrBad, reqBad)
	if rrBad.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for bad password, got %d", rrBad.Code)
	}

	// Valid default password
	loginBodyOk, _ := json.Marshal(map[string]string{"password": auth.DefaultPassword})
	reqOk := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginBodyOk))
	rrOk := httptest.NewRecorder()

	handler.Login(rrOk, reqOk)
	if rrOk.Code != http.StatusOK {
		t.Errorf("expected status 200 for good password, got %d", rrOk.Code)
	}

	// Check cookie in response
	cookies := rrOk.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != auth.CookieName {
		t.Errorf("expected auth session cookie in response")
	}
}

func TestAuthHandler_RequireAuth(t *testing.T) {
	handler, authMgr := setupTestAuthHandler(t)

	protectedHandler := handler.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// 1. Request without auth -> 401
	req1 := httptest.NewRequest("GET", "/api/protected", nil)
	rr1 := httptest.NewRecorder()
	protectedHandler(rr1, req1)

	if rr1.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr1.Code)
	}

	// 2. Request with valid cookie -> 200
	token, ok, _ := authMgr.Authenticate(auth.DefaultPassword, "127.0.0.1:1234")
	if !ok {
		t.Fatalf("failed to authenticate for session token")
	}

	req2 := httptest.NewRequest("GET", "/api/protected", nil)
	req2.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: token,
	})
	rr2 := httptest.NewRecorder()
	protectedHandler(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr2.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	handler, authMgr := setupTestAuthHandler(t)

	token, _, _ := authMgr.Authenticate(auth.DefaultPassword, "127.0.0.1:1234")

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: token,
	})
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for logout, got %d", rr.Code)
	}

	if authMgr.ValidateSession(token) {
		t.Errorf("expected session token to be invalidated after logout")
	}
}
