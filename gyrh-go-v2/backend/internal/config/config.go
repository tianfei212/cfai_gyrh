package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 全局配置结构体。
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Matting MattingConfig `yaml:"matting"`
	Storage StorageConfig `yaml:"storage"`
	Skill   SkillConfig   `yaml:"skill"`
	Models  ModelConfig   `yaml:"models"`
	AliOSS  AliOSSConfig  `yaml:"alioss"`
	Import  ImportConfig  `yaml:"import"`
	Gallery GalleryConfig `yaml:"gallery"`
	Logger  LoggerConfig  `yaml:"logger"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
}

// MattingConfig 抠图配置。
type MattingConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Model     string `yaml:"model"`
	ModelPath string `yaml:"model_path"`
	ServerURL string `yaml:"server_url"`
}

// StorageConfig 存储配置。
type StorageConfig struct {
	Mode string `yaml:"mode"`

	LocalPath string `yaml:"local_path"`

	OssEndpoint     string `yaml:"oss_endpoint"`
	OssBucket       string `yaml:"oss_bucket"`
	OssBucketPrefix string `yaml:"oss_bucket_prefix"`
	OssAccessKey    string `yaml:"oss_access_key"`
	OssSecretKey    string `yaml:"oss_secret_key"`

	DashScopeAPIKey string `yaml:"dashscope_api_key"`
}

// SkillConfig Skill 配置。
type SkillConfig struct {
	UseDatabase         bool   `yaml:"use_database"`
	LocalPath           string `yaml:"local_path"`
	ImportTriggerFile   string `yaml:"import_trigger_file"`
	WatchIntervalSecond int    `yaml:"watch_interval_seconds"`
}

// ModelConfig 大模型名称配置。
type ModelConfig struct {
	Gemini string `yaml:"gemini"`
	Wan    string `yaml:"wan"`
	Qwen   string `yaml:"qwen"`
}

// AliOSSConfig aliOSS 二进制服务配置。
type AliOSSConfig struct {
	Enabled           bool   `yaml:"enabled"`
	AutoStart         bool   `yaml:"auto_start"`
	BinaryPath        string `yaml:"binary_path"`
	Port              int    `yaml:"port"`
	OpenAIAPIKey      string `yaml:"openai_api_key"`
	HealthWaitSeconds int    `yaml:"health_wait_seconds"`
}

// ImportConfig 导入相关配置。
type ImportConfig struct {
	Enabled             bool   `yaml:"enabled"`
	GeneratedDir        string `yaml:"generated_dir"`
	WatchIntervalSecond int    `yaml:"watch_interval_seconds"`
}

// GalleryConfig 图库配置。
type GalleryConfig struct {
	ExternalURL string `yaml:"external_url"`
}

// LoggerConfig 日志配置。
type LoggerConfig struct {
	Level  string `yaml:"level"`
	Path   string `yaml:"path"`
	MaxAge int    `yaml:"max_age"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  60,
			WriteTimeout: 60,
		},
		Matting: MattingConfig{
			Enabled:   true,
			Model:     "mediapipe",
			ModelPath: "./backend/models/mediapipe/selfie_segmentation_landscape.tflite",
			ServerURL: "http://127.0.0.1:5000",
		},
		Storage: StorageConfig{
			Mode:            "local",
			LocalPath:       "./backend/data/generated",
			OssEndpoint:     "http://127.0.0.1:18080",
			OssBucketPrefix: "images_data/",
			DashScopeAPIKey: "",
		},
		Skill: SkillConfig{
			UseDatabase:         true,
			LocalPath:           "./configs/skills",
			ImportTriggerFile:   "./configs/SKILL导入",
			WatchIntervalSecond: 5,
		},
		Models: ModelConfig{
			Gemini: "gemini-1.5-flash",
			Wan:    "wanx-plus",
			Qwen:   "qwen3.6-plus",
		},
		AliOSS: AliOSSConfig{
			Enabled:           true,
			AutoStart:         true,
			BinaryPath:        "./bin/oss-cli",
			Port:              18080,
			HealthWaitSeconds: 15,
		},
		Import: ImportConfig{
			Enabled:             true,
			GeneratedDir:        "./backend/data/generated_import",
			WatchIntervalSecond: 10,
		},
		Gallery: GalleryConfig{
			ExternalURL: "",
		},
		Logger: LoggerConfig{
			Level:  "info",
			Path:   "./backend/logs",
			MaxAge: 30,
		},
	}
}

// Load 加载配置文件并应用环境变量覆盖。
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	rootDir, err := FindProjectRoot()
	if err != nil {
		return nil, err
	}

	yamlPath := configPath
	if !filepath.IsAbs(yamlPath) {
		yamlPath = filepath.Join(rootDir, configPath)
	}

	_ = loadDotEnv(filepath.Join(rootDir, ".env.local"))

	if err := loadYAML(yamlPath, cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	if err := applyEnvOverrides(cfg); err != nil {
		return nil, fmt.Errorf("应用环境变量失败: %w", err)
	}

	resolveConfigPaths(cfg, rootDir)

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// loadYAML 读取 YAML 配置。
func loadYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}
	return nil
}

// FindProjectRoot 向上查找项目根目录。
func FindProjectRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前目录失败: %w", err)
	}

	dir := current
	for {
		cfgPath := filepath.Join(dir, "configs", "config.yaml")
		backendPath := filepath.Join(dir, "backend")
		if fileExists(cfgPath) && dirExists(backendPath) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("未找到项目根目录")
		}
		dir = parent
	}
}

// GetConfigPath 获取默认配置路径。
func GetConfigPath() string {
	return "configs/config.yaml"
}

func applyEnvOverrides(cfg *Config) error {
	if v := os.Getenv("GYRH_SERVER_PORT"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_SERVER_PORT 无效: %s", v)
		}
		cfg.Server.Port = value
	}
	if v := os.Getenv("GYRH_SERVER_READ_TIMEOUT"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_SERVER_READ_TIMEOUT 无效: %s", v)
		}
		cfg.Server.ReadTimeout = value
	}
	if v := os.Getenv("GYRH_SERVER_WRITE_TIMEOUT"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_SERVER_WRITE_TIMEOUT 无效: %s", v)
		}
		cfg.Server.WriteTimeout = value
	}

	if v := os.Getenv("GYRH_MATTING_ENABLED"); v != "" {
		cfg.Matting.Enabled = isTrue(v)
	}
	if v := os.Getenv("GYRH_MATTING_MODEL"); v != "" {
		cfg.Matting.Model = v
	}
	if v := os.Getenv("GYRH_MATTING_MODEL_PATH"); v != "" {
		cfg.Matting.ModelPath = v
	}
	if v := os.Getenv("GYRH_MATTING_SERVER_URL"); v != "" {
		cfg.Matting.ServerURL = v
	}

	if v := os.Getenv("GYRH_STORAGE_MODE"); v != "" {
		cfg.Storage.Mode = v
	}
	if v := os.Getenv("GYRH_STORAGE_LOCAL_PATH"); v != "" {
		cfg.Storage.LocalPath = v
	}
	if v := firstEnv("GYRH_STORAGE_OSS_ENDPOINT", "OSS_ENDPOINT"); v != "" {
		cfg.Storage.OssEndpoint = v
	}
	if v := firstEnv("GYRH_STORAGE_OSS_BUCKET", "OSS_BUCKET"); v != "" {
		cfg.Storage.OssBucket = v
	}
	if v := firstEnv("GYRH_STORAGE_OSS_BUCKET_PREFIX", "OSS_BUCKET_PREFIX"); v != "" {
		cfg.Storage.OssBucketPrefix = v
	}
	if v := firstEnv("GYRH_STORAGE_OSS_ACCESS_KEY", "OSS_ACCESS_KEY_ID"); v != "" {
		cfg.Storage.OssAccessKey = v
	}
	if v := firstEnv("GYRH_STORAGE_OSS_SECRET_KEY", "OSS_ACCESS_KEY_SECRET"); v != "" {
		cfg.Storage.OssSecretKey = v
	}
	if v := firstEnv("GYRH_STORAGE_DASHSCOPE_API_KEY", "DASHSCOPE_API_KEY"); v != "" {
		cfg.Storage.DashScopeAPIKey = v
	}

	if v := os.Getenv("GYRH_SKILL_USE_DATABASE"); v != "" {
		cfg.Skill.UseDatabase = isTrue(v)
	}
	if v := os.Getenv("GYRH_SKILL_LOCAL_PATH"); v != "" {
		cfg.Skill.LocalPath = v
	}
	if v := os.Getenv("GYRH_SKILL_IMPORT_TRIGGER_FILE"); v != "" {
		cfg.Skill.ImportTriggerFile = v
	}
	if v := os.Getenv("GYRH_SKILL_WATCH_INTERVAL_SECONDS"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_SKILL_WATCH_INTERVAL_SECONDS 无效: %s", v)
		}
		cfg.Skill.WatchIntervalSecond = value
	}

	if v := os.Getenv("GYRH_MODEL_GEMINI"); v != "" {
		cfg.Models.Gemini = v
	}
	if v := os.Getenv("GYRH_MODEL_WAN"); v != "" {
		cfg.Models.Wan = v
	}
	if v := os.Getenv("GYRH_MODEL_QWEN"); v != "" {
		cfg.Models.Qwen = v
	}

	if v := os.Getenv("GYRH_ALIOSS_ENABLED"); v != "" {
		cfg.AliOSS.Enabled = isTrue(v)
	}
	if v := os.Getenv("GYRH_ALIOSS_AUTO_START"); v != "" {
		cfg.AliOSS.AutoStart = isTrue(v)
	}
	if v := os.Getenv("GYRH_ALIOSS_BINARY_PATH"); v != "" {
		cfg.AliOSS.BinaryPath = v
	}
	if v := os.Getenv("GYRH_ALIOSS_PORT"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_ALIOSS_PORT 无效: %s", v)
		}
		cfg.AliOSS.Port = value
	}
	if v := firstEnv("GYRH_ALIOSS_OPENAI_API_KEY", "OPENAI_API_KEY"); v != "" {
		cfg.AliOSS.OpenAIAPIKey = v
	}
	if v := os.Getenv("GYRH_ALIOSS_HEALTH_WAIT_SECONDS"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_ALIOSS_HEALTH_WAIT_SECONDS 无效: %s", v)
		}
		cfg.AliOSS.HealthWaitSeconds = value
	}

	if v := os.Getenv("GYRH_IMPORT_ENABLED"); v != "" {
		cfg.Import.Enabled = isTrue(v)
	}
	if v := os.Getenv("GYRH_IMPORT_GENERATED_DIR"); v != "" {
		cfg.Import.GeneratedDir = v
	}
	if v := os.Getenv("GYRH_IMPORT_WATCH_INTERVAL_SECONDS"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_IMPORT_WATCH_INTERVAL_SECONDS 无效: %s", v)
		}
		cfg.Import.WatchIntervalSecond = value
	}

	if v := os.Getenv("GYRH_GALLERY_EXTERNAL_URL"); v != "" {
		cfg.Gallery.ExternalURL = v
	}

	if v := os.Getenv("GYRH_LOGGER_LEVEL"); v != "" {
		cfg.Logger.Level = v
	}
	if v := os.Getenv("GYRH_LOGGER_PATH"); v != "" {
		cfg.Logger.Path = v
	}
	if v := os.Getenv("GYRH_LOGGER_MAX_AGE"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_LOGGER_MAX_AGE 无效: %s", v)
		}
		cfg.Logger.MaxAge = value
	}

	return nil
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port 无效: %d", cfg.Server.Port)
	}
	if cfg.Storage.Mode != "local" && cfg.Storage.Mode != "oss" {
		return fmt.Errorf("storage.mode 仅支持 local 或 oss")
	}
	if cfg.Storage.LocalPath == "" {
		return fmt.Errorf("storage.local_path 不能为空")
	}
	if cfg.Skill.LocalPath == "" {
		return fmt.Errorf("skill.local_path 不能为空")
	}
	if strings.TrimSpace(cfg.Models.Gemini) == "" {
		return fmt.Errorf("models.gemini 不能为空")
	}
	if strings.TrimSpace(cfg.Models.Wan) == "" {
		return fmt.Errorf("models.wan 不能为空")
	}
	if strings.TrimSpace(cfg.Models.Qwen) == "" {
		return fmt.Errorf("models.qwen 不能为空")
	}
	if cfg.AliOSS.Port <= 0 || cfg.AliOSS.Port > 65535 {
		return fmt.Errorf("alioss.port 无效: %d", cfg.AliOSS.Port)
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.Logger.Level] {
		return fmt.Errorf("logger.level 无效: %s", cfg.Logger.Level)
	}
	if cfg.Skill.WatchIntervalSecond <= 0 {
		cfg.Skill.WatchIntervalSecond = 5
	}
	if cfg.Import.WatchIntervalSecond <= 0 {
		cfg.Import.WatchIntervalSecond = 10
	}
	if cfg.AliOSS.HealthWaitSeconds <= 0 {
		cfg.AliOSS.HealthWaitSeconds = 15
	}
	return nil
}

func resolveConfigPaths(cfg *Config, rootDir string) {
	cfg.Matting.ModelPath = resolvePath(rootDir, cfg.Matting.ModelPath)
	cfg.Storage.LocalPath = resolvePath(rootDir, cfg.Storage.LocalPath)
	cfg.Skill.LocalPath = resolvePath(rootDir, cfg.Skill.LocalPath)
	cfg.Skill.ImportTriggerFile = resolvePath(rootDir, cfg.Skill.ImportTriggerFile)
	cfg.AliOSS.BinaryPath = resolvePath(rootDir, cfg.AliOSS.BinaryPath)
	cfg.Import.GeneratedDir = resolvePath(rootDir, cfg.Import.GeneratedDir)
	cfg.Logger.Path = resolvePath(rootDir, cfg.Logger.Path)

	if cfg.Storage.Mode == "oss" && cfg.Storage.OssEndpoint == "" && cfg.AliOSS.Port > 0 {
		cfg.Storage.OssEndpoint = fmt.Sprintf("http://127.0.0.1:%d", cfg.AliOSS.Port)
	}
}

func resolvePath(rootDir, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(rootDir, path)
}

func isTrue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "1" || lower == "true" || lower == "yes" || lower == "on"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
