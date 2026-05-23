package kiosk

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath          = "kiosk-client.yaml"
	DefaultUserDataDir         = "runtime/chrome-profile"
	DefaultRestartDelaySeconds = 3
)

// Config controls the Windows kiosk client shell.
type Config struct {
	URL                   string   `yaml:"url"`
	ChromePath            string   `yaml:"chrome_path"`
	UserDataDir           string   `yaml:"user_data_dir"`
	AutoRestart           bool     `yaml:"auto_restart"`
	RestartDelaySeconds   int      `yaml:"restart_delay_seconds"`
	CloseChromeOnExit     bool     `yaml:"close_chrome_on_exit"`
	AdditionalChromeFlags []string `yaml:"additional_chrome_flags"`

	RestartDelay time.Duration `yaml:"-"`
}

// LoadConfig reads the kiosk client config and applies production defaults.
func LoadConfig(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取 kiosk 配置失败: %w", err)
	}

	cfg := Config{
		AutoRestart:         true,
		RestartDelaySeconds: DefaultRestartDelaySeconds,
		CloseChromeOnExit:   true,
		UserDataDir:         DefaultUserDataDir,
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("解析 kiosk 配置失败: %w", err)
	}

	cfg.URL = strings.TrimSpace(cfg.URL)
	cfg.ChromePath = strings.TrimSpace(cfg.ChromePath)
	cfg.UserDataDir = strings.TrimSpace(cfg.UserDataDir)
	if cfg.URL == "" {
		return Config{}, fmt.Errorf("kiosk 配置缺少 url")
	}
	if cfg.UserDataDir == "" {
		cfg.UserDataDir = DefaultUserDataDir
	}
	if cfg.RestartDelaySeconds <= 0 {
		cfg.RestartDelaySeconds = DefaultRestartDelaySeconds
	}
	cfg.RestartDelay = time.Duration(cfg.RestartDelaySeconds) * time.Second

	return cfg, nil
}

// BuildChromeArgs returns the fixed Chrome flags for the exhibition kiosk shell.
func BuildChromeArgs(cfg Config) []string {
	args := []string{
		"--kiosk",
		"--user-data-dir=" + cfg.UserDataDir,
		"--no-first-run",
		"--disable-translate",
		"--disable-infobars",
		"--disable-session-crashed-bubble",
		"--overscroll-history-navigation=0",
	}
	args = append(args, cfg.AdditionalChromeFlags...)
	args = append(args, cfg.URL)
	return args
}
