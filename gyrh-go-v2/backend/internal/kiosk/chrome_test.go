package kiosk

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindChromeUsesConfiguredPath(t *testing.T) {
	dir := t.TempDir()
	chromePath := filepath.Join(dir, "chrome.exe")
	if err := os.WriteFile(chromePath, []byte("fake chrome"), 0o755); err != nil {
		t.Fatalf("write fake chrome: %v", err)
	}

	got, err := FindChrome(chromePath, nil)
	if err != nil {
		t.Fatalf("FindChrome returned error: %v", err)
	}
	if got != chromePath {
		t.Fatalf("FindChrome = %q, want %q", got, chromePath)
	}
}

func TestFindChromeSearchesWindowsInstallLocations(t *testing.T) {
	dir := t.TempDir()
	programFiles := filepath.Join(dir, "ProgramFiles")
	localAppData := filepath.Join(dir, "LocalAppData")
	want := filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatalf("mkdir fake chrome dir: %v", err)
	}
	if err := os.WriteFile(want, []byte("fake chrome"), 0o755); err != nil {
		t.Fatalf("write fake chrome: %v", err)
	}

	env := map[string]string{
		"PROGRAMFILES":       programFiles,
		"PROGRAMFILES(X86)":  filepath.Join(dir, "ProgramFilesX86"),
		"LOCALAPPDATA":       localAppData,
		"ProgramW6432":       filepath.Join(dir, "ProgramW6432"),
		"GOOGLE_CHROME_SHIM": "",
	}
	got, err := findChromeWithEnv("", env, func(string) (string, error) {
		return "", os.ErrNotExist
	})
	if err != nil {
		t.Fatalf("findChromeWithEnv returned error: %v", err)
	}
	if got != want {
		t.Fatalf("findChromeWithEnv = %q, want %q", got, want)
	}
}

func TestFindChromeFallsBackToPathLookup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a fake lookup path independent of Windows executable resolution")
	}

	want := filepath.Join(t.TempDir(), "google-chrome")
	got, err := findChromeWithEnv("", nil, func(name string) (string, error) {
		if name == "google-chrome" {
			return want, nil
		}
		return "", os.ErrNotExist
	})
	if err != nil {
		t.Fatalf("findChromeWithEnv returned error: %v", err)
	}
	if got != want {
		t.Fatalf("findChromeWithEnv = %q, want %q", got, want)
	}
}
