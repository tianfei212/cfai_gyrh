package frontendauth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gyrh-go-v2/backend/internal/config"
)

func TestServiceLoginAndCookieSession(t *testing.T) {
	service := NewService(config.FrontendAuthConfig{
		JWTSecret:       "test-secret",
		TokenTTLMinutes: 15,
		Users: []config.FrontendUserConfig{
			{Username: "admin", Password: "123456", Role: "admin"},
		},
	}, "")

	session, err := service.Login("admin", "123456")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: session.Token})

	got, err := service.SessionFromRequest(req)
	if err != nil {
		t.Fatalf("session from request: %v", err)
	}
	if got.Username != "admin" || got.Role != "admin" {
		t.Fatalf("session = %+v", got)
	}
}

func TestServiceHotReloadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.local")
	writeEnv(t, envPath, "old@#1")
	service := NewService(config.FrontendAuthConfig{}, dir)

	if _, err := service.Login("admin", "old@#1"); err != nil {
		t.Fatalf("old password login: %v", err)
	}
	writeEnv(t, envPath, "new@#1")
	if _, err := service.Login("admin", "new@#1"); err != nil {
		t.Fatalf("new password login: %v", err)
	}
	if _, err := service.Login("admin", "old@#1"); err == nil {
		t.Fatal("old password should fail after hot reload")
	}
}

func TestRealIPPrefersForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "10.0.0.9:1234"
	req.Header.Set("X-Real-IP", "10.0.0.8")
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.8")

	if got := RealIP(req); got != "203.0.113.10" {
		t.Fatalf("RealIP = %q, want 203.0.113.10", got)
	}
}

func TestRealIPFallsBackToRemoteAddrHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "10.0.0.9:1234"

	if got := RealIP(req); got != "10.0.0.9" {
		t.Fatalf("RealIP = %q, want 10.0.0.9", got)
	}
}

func writeEnv(t *testing.T, path, password string) {
	t.Helper()
	content := "GYRH_FRONTEND_AUTH_JWT_SECRET=test-secret\n" +
		"GYRH_FRONTEND_AUTH_TOKEN_TTL_MINUTES=15\n" +
		"GYRH_FRONTEND_AUTH_ADMIN_USERNAME=admin\n" +
		"GYRH_FRONTEND_AUTH_ADMIN_PASSWORD=" + password + "\n" +
		"GYRH_FRONTEND_AUTH_PSHOW_USERNAME=pshow\n" +
		"GYRH_FRONTEND_AUTH_PSHOW_PASSWORD=ps@#01\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write env: %v", err)
	}
}
