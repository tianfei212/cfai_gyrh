package kiosk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindChrome resolves Google Chrome from explicit config, standard Windows
// install directories, and finally PATH.
func FindChrome(configuredPath string, env map[string]string) (string, error) {
	return findChromeWithEnv(configuredPath, env, exec.LookPath)
}

func findChromeWithEnv(configuredPath string, env map[string]string, lookPath func(string) (string, error)) (string, error) {
	if path := strings.TrimSpace(configuredPath); path != "" {
		if fileExists(path) {
			return path, nil
		}
		return "", fmt.Errorf("配置的 chrome_path 不存在: %s", path)
	}

	for _, candidate := range chromeCandidates(env) {
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	for _, name := range []string{"chrome.exe", "chrome", "google-chrome", "google-chrome-stable"} {
		if path, err := lookPath(name); err == nil && strings.TrimSpace(path) != "" {
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到 Google Chrome，请安装 Chrome 或在 kiosk-client.yaml 配置 chrome_path")
}

func chromeCandidates(env map[string]string) []string {
	baseDirs := []string{
		envValue(env, "PROGRAMFILES"),
		envValue(env, "ProgramW6432"),
		envValue(env, "PROGRAMFILES(X86)"),
		envValue(env, "LOCALAPPDATA"),
	}

	var candidates []string
	seen := make(map[string]bool)
	for _, baseDir := range baseDirs {
		if strings.TrimSpace(baseDir) == "" {
			continue
		}
		path := filepath.Join(baseDir, "Google", "Chrome", "Application", "chrome.exe")
		if !seen[path] {
			candidates = append(candidates, path)
			seen[path] = true
		}
	}
	return candidates
}

func envValue(env map[string]string, key string) string {
	if env != nil {
		return env[key]
	}
	return os.Getenv(key)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
