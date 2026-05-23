package config

import (
	"testing"
)

func TestDefaultConfigIncludesHelpper302(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Helpper302.Enabled {
		t.Fatalf("Helpper302 should be enabled by default")
	}
	if cfg.Helpper302.Provider != "302-gpt-image" {
		t.Fatalf("Provider = %q, want 302-gpt-image", cfg.Helpper302.Provider)
	}
	if cfg.Helpper302.BaseURL != "https://api.302.ai" {
		t.Fatalf("BaseURL = %q, want https://api.302.ai", cfg.Helpper302.BaseURL)
	}
	if cfg.Helpper302.ModelName != "gpt-image-2" {
		t.Fatalf("ModelName = %q, want gpt-image-2", cfg.Helpper302.ModelName)
	}
}

func TestApplyEnvOverridesHelpper302(t *testing.T) {
	t.Setenv("GYRH_302_HELPER_ENABLED", "false")
	t.Setenv("GYRH_302_HELPER_BASE_URL", "https://example.test")
	t.Setenv("GYRH_302_HELPER_MODEL_NAME", "gpt-image-1.5")
	t.Setenv("GYRH_302_HELPER_MAX_WAIT_SECONDS", "12")

	cfg := DefaultConfig()
	if err := applyEnvOverrides(cfg); err != nil {
		t.Fatalf("applyEnvOverrides returned error: %v", err)
	}
	if cfg.Helpper302.Enabled {
		t.Fatalf("Helpper302.Enabled should be false from env")
	}
	if cfg.Helpper302.BaseURL != "https://example.test" {
		t.Fatalf("BaseURL = %q", cfg.Helpper302.BaseURL)
	}
	if cfg.Helpper302.ModelName != "gpt-image-1.5" {
		t.Fatalf("ModelName = %q", cfg.Helpper302.ModelName)
	}
	if cfg.Helpper302.MaxWaitSeconds != 12 {
		t.Fatalf("MaxWaitSeconds = %d, want 12", cfg.Helpper302.MaxWaitSeconds)
	}
}

func TestApplyEnvOverridesFrontendAuth(t *testing.T) {
	t.Setenv("GYRH_FRONTEND_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GYRH_FRONTEND_AUTH_TOKEN_TTL_MINUTES", "15")
	t.Setenv("GYRH_FRONTEND_AUTH_ADMIN_USERNAME", "admin")
	t.Setenv("GYRH_FRONTEND_AUTH_ADMIN_PASSWORD", "1234@#")
	t.Setenv("GYRH_FRONTEND_AUTH_PSHOW_USERNAME", "pshow")
	t.Setenv("GYRH_FRONTEND_AUTH_PSHOW_PASSWORD", "a1B2@#")

	cfg := DefaultConfig()
	if err := applyEnvOverrides(cfg); err != nil {
		t.Fatalf("applyEnvOverrides returned error: %v", err)
	}
	if cfg.FrontendAuth.JWTSecret != "test-secret" {
		t.Fatalf("JWTSecret = %q", cfg.FrontendAuth.JWTSecret)
	}
	if cfg.FrontendAuth.TokenTTLMinutes != 15 {
		t.Fatalf("TokenTTLMinutes = %d, want 15", cfg.FrontendAuth.TokenTTLMinutes)
	}
	if len(cfg.FrontendAuth.Users) != 2 {
		t.Fatalf("users len = %d, want 2", len(cfg.FrontendAuth.Users))
	}
	if cfg.FrontendAuth.Users[0].Username != "admin" || cfg.FrontendAuth.Users[0].Role != "admin" {
		t.Fatalf("admin user = %+v", cfg.FrontendAuth.Users[0])
	}
	if cfg.FrontendAuth.Users[1].Username != "pshow" || cfg.FrontendAuth.Users[1].Role != "pshow" {
		t.Fatalf("pshow user = %+v", cfg.FrontendAuth.Users[1])
	}
}

func TestApplyEnvOverridesFrontendAuthRejectsWeakPassword(t *testing.T) {
	t.Setenv("GYRH_FRONTEND_AUTH_ADMIN_USERNAME", "admin")
	t.Setenv("GYRH_FRONTEND_AUTH_ADMIN_PASSWORD", "123456")

	cfg := DefaultConfig()
	if err := applyEnvOverrides(cfg); err == nil {
		t.Fatal("applyEnvOverrides should reject password without two special characters")
	}
}
