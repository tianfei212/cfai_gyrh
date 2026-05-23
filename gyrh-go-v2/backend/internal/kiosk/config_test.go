package kiosk

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadConfigAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kiosk-client.yaml")
	if err := os.WriteFile(path, []byte("url: \"http://127.0.0.1:9913/admin_viewer\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.URL != "http://127.0.0.1:9913/admin_viewer" {
		t.Fatalf("URL = %q", cfg.URL)
	}
	if cfg.AutoRestart != true {
		t.Fatalf("AutoRestart = %v, want true", cfg.AutoRestart)
	}
	if cfg.RestartDelay != 3*time.Second {
		t.Fatalf("RestartDelay = %v, want 3s", cfg.RestartDelay)
	}
	if cfg.CloseChromeOnExit != true {
		t.Fatalf("CloseChromeOnExit = %v, want true", cfg.CloseChromeOnExit)
	}
	if cfg.UserDataDir != "runtime/chrome-profile" {
		t.Fatalf("UserDataDir = %q", cfg.UserDataDir)
	}
}

func TestLoadConfigRejectsMissingURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kiosk-client.yaml")
	if err := os.WriteFile(path, []byte("auto_restart: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig returned nil error for missing URL")
	}
}

func TestBuildChromeArgsUsesKioskMode(t *testing.T) {
	cfg := Config{
		URL:         "https://example.com/viewer",
		UserDataDir: `C:\gyrh\runtime\chrome-profile`,
	}

	args := BuildChromeArgs(cfg)
	want := []string{
		"--kiosk",
		"--user-data-dir=C:\\gyrh\\runtime\\chrome-profile",
		"--no-first-run",
		"--disable-translate",
		"--disable-infobars",
		"--disable-session-crashed-bubble",
		"--overscroll-history-navigation=0",
		"https://example.com/viewer",
	}

	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}
