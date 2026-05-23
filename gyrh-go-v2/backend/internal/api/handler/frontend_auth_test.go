package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	frontendauthapp "gyrh-go-v2/backend/internal/application/frontendauth"
	"gyrh-go-v2/backend/internal/config"
)

func testFrontendAuthHandler() *FrontendAuthHandler {
	return NewFrontendAuthHandler(frontendauthapp.NewService(config.FrontendAuthConfig{
		JWTSecret:       "test-secret",
		TokenTTLMinutes: 15,
		Users: []config.FrontendUserConfig{
			{Username: "admin", Password: "123456", Role: "admin"},
			{Username: "pshow", Password: "a1B2c3", Role: "pshow"},
		},
	}, ""))
}

func TestFrontendAuthLoginReadsHOMEHeaders(t *testing.T) {
	h := testFrontendAuthHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	req.Header.Set("HOME1", "admin")
	req.Header.Set("HOME2", "123456")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Username  string `json:"username"`
			Role      string `json:"role"`
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Username != "admin" || resp.Data.Role != "admin" || resp.Data.Token == "" {
		t.Fatalf("login data = %+v", resp.Data)
	}
	if resp.Data.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("expires_at = %d, want future timestamp", resp.Data.ExpiresAt)
	}
	if cookie := rec.Result().Cookies()[0]; cookie.Name != FrontendAuthCookieName || cookie.Value == "" {
		t.Fatalf("auth cookie = %+v", cookie)
	}
}

func TestFrontendAuthLoginRejectsWrongPassword(t *testing.T) {
	h := testFrontendAuthHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	req.Header.Set("HOME1", "admin")
	req.Header.Set("HOME2", "badpwd")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body=%s", rec.Code, rec.Body.String())
	}
}

func TestFrontendAuthSessionValidatesBearerToken(t *testing.T) {
	h := testFrontendAuthHandler()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	loginReq.Header.Set("HOME1", "pshow")
	loginReq.Header.Set("HOME2", "a1B2c3")
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/frontend-auth/session", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Data.Token)
	rec := httptest.NewRecorder()

	h.Session(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if resp.Data.Username != "pshow" || resp.Data.Role != "pshow" {
		t.Fatalf("session data = %+v", resp.Data)
	}
}

func TestFrontendAuthSessionValidatesCookieToken(t *testing.T) {
	h := testFrontendAuthHandler()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	loginReq.Header.Set("HOME1", "admin")
	loginReq.Header.Set("HOME2", "123456")
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/frontend-auth/session", nil)
	for _, cookie := range loginRec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()

	h.Session(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !h.HasValidSession(req) {
		t.Fatal("HasValidSession should accept auth cookie")
	}
}

func TestFrontendAuthRequireSessionRejectsMissingSession(t *testing.T) {
	h := testFrontendAuthHandler()
	called := false
	next := h.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/view", nil)
	rec := httptest.NewRecorder()
	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if called {
		t.Fatal("next handler should not be called without session")
	}
}

func TestFrontendAuthRequireSessionAllowsCookieSession(t *testing.T) {
	h := testFrontendAuthHandler()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	loginReq.Header.Set("HOME1", "admin")
	loginReq.Header.Set("HOME2", "123456")
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	next := h.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/view", nil)
	for _, cookie := range loginRec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestFrontendAuthReloadsDotEnvOnLogin(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.local")
	writeFrontendAuthEnv(t, envPath, "old@#1", "ps@#01")
	h := NewFrontendAuthHandler(frontendauthapp.NewService(config.FrontendAuthConfig{}, dir))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	req.Header.Set("HOME1", "admin")
	req.Header.Set("HOME2", "old@#1")
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("old password login status = %d, body=%s", rec.Code, rec.Body.String())
	}

	writeFrontendAuthEnv(t, envPath, "new@#1", "ps@#02")
	req = httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	req.Header.Set("HOME1", "admin")
	req.Header.Set("HOME2", "new@#1")
	rec = httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("new password login status = %d, body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
	req.Header.Set("HOME1", "admin")
	req.Header.Set("HOME2", "old@#1")
	rec = httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("old password status after reload = %d, want 401", rec.Code)
	}
}

func TestFrontendAuthRandomLoginLoad1000RPS(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.local")
	writeFrontendAuthEnv(t, envPath, "3010@#", "SCqz@#")
	h := NewFrontendAuthHandler(frontendauthapp.NewService(config.FrontendAuthConfig{}, dir))

	type credential struct {
		username string
		password string
		wantCode int
	}
	credentials := []credential{
		{username: "admin", password: "3010@#", wantCode: http.StatusOK},
		{username: "pshow", password: "SCqz@#", wantCode: http.StatusOK},
		{username: "admin", password: "bad@#1", wantCode: http.StatusUnauthorized},
		{username: "pshow", password: "bad@#2", wantCode: http.StatusUnauthorized},
	}

	const (
		requestsPerSecond = 1000
		seconds           = 3
	)
	var failures atomic.Int64
	var serverErrors atomic.Int64
	var wg sync.WaitGroup
	rng := rand.New(rand.NewSource(42))
	ticker := time.NewTicker(time.Second / requestsPerSecond)
	defer ticker.Stop()

	for i := 0; i < requestsPerSecond*seconds; i++ {
		<-ticker.C
		cred := credentials[rng.Intn(len(credentials))]
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/frontend-auth/login", nil)
			req.Header.Set("HOME1", cred.username)
			req.Header.Set("HOME2", cred.password)
			rec := httptest.NewRecorder()
			h.Login(rec, req)
			if rec.Code >= http.StatusInternalServerError {
				serverErrors.Add(1)
			}
			if rec.Code != cred.wantCode {
				failures.Add(1)
			}
		}()
	}
	wg.Wait()

	if serverErrors.Load() != 0 {
		t.Fatalf("server errors = %d, want 0", serverErrors.Load())
	}
	if failures.Load() != 0 {
		t.Fatalf("unexpected login responses = %d, want 0", failures.Load())
	}
}

func writeFrontendAuthEnv(t *testing.T, path, adminPassword, pshowPassword string) {
	t.Helper()
	content := "GYRH_FRONTEND_AUTH_JWT_SECRET=test-secret\n" +
		"GYRH_FRONTEND_AUTH_TOKEN_TTL_MINUTES=15\n" +
		"GYRH_FRONTEND_AUTH_ADMIN_USERNAME=admin\n" +
		"GYRH_FRONTEND_AUTH_ADMIN_PASSWORD=" + adminPassword + "\n" +
		"GYRH_FRONTEND_AUTH_PSHOW_USERNAME=pshow\n" +
		"GYRH_FRONTEND_AUTH_PSHOW_PASSWORD=" + pshowPassword + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write env: %v", err)
	}
}
