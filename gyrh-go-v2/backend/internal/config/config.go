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
	Server     ServerConfig     `yaml:"server"`
	Matting    MattingConfig    `yaml:"matting"`
	Storage    StorageConfig    `yaml:"storage"`
	Skill      SkillConfig      `yaml:"skill"`
	Models     ModelConfig      `yaml:"models"`
	AliOSS     AliOSSConfig     `yaml:"alioss"`
	Helpper302 Helpper302Config `yaml:"helpper302"`
	Import     ImportConfig     `yaml:"import"`
	Gallery    GalleryConfig    `yaml:"gallery"`
	Logger     LoggerConfig     `yaml:"logger"`
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
	Gemini             string `yaml:"gemini"`
	Wan                string `yaml:"wan"`
	Qwen               string `yaml:"qwen"`
	HTTPTimeoutMinutes int    `yaml:"http_timeout_minutes"`
}

// AliOSSConfig aliOSS 二进制服务配置。
type AliOSSConfig struct {
	Enabled                bool   `yaml:"enabled"`
	AutoStart              bool   `yaml:"auto_start"`
	BinaryPath             string `yaml:"binary_path"`
	Port                   int    `yaml:"port"`
	GeneratedPort          int    `yaml:"generated_port"`
	Endpoint               string `yaml:"endpoint"`
	BucketName             string `yaml:"bucket_name"`
	BackgroundBucketPrefix string `yaml:"background_bucket_prefix"`
	GeneratedBucketPrefix  string `yaml:"generated_bucket_prefix"`
	OpenAIAPIKey           string `yaml:"openai_api_key"`
	HealthWaitSeconds      int    `yaml:"health_wait_seconds"`
}

// Helpper302Config 302Helpper 插件配置。
type Helpper302Config struct {
	Enabled             bool   `yaml:"enabled"`
	BaseURL             string `yaml:"base_url"`
	Provider            string `yaml:"provider"`
	ModelName           string `yaml:"model_name"`
	Mode                string `yaml:"mode"`
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	MaxWaitSeconds      int    `yaml:"max_wait_seconds"`
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
			DashScopeAPIKey: "",
		},
		Skill: SkillConfig{
			UseDatabase:         true,
			LocalPath:           "./configs/skills",
			ImportTriggerFile:   "./configs/SKILL导入",
			WatchIntervalSecond: 5,
		},
		Models: ModelConfig{
			Gemini:             "gemini-1.5-flash",
			Wan:                "wanx-plus",
			Qwen:               "qwen3.6-plus",
			HTTPTimeoutMinutes: 20,
		},
		AliOSS: AliOSSConfig{
			Enabled:                true,
			AutoStart:              true,
			BinaryPath:             "./bin/oss-cli",
			Port:                   18080,
			GeneratedPort:          18081,
			BackgroundBucketPrefix: "images_data/",
			GeneratedBucketPrefix:  "gyrh_images_data/",
			HealthWaitSeconds:      15,
		},
		Helpper302: Helpper302Config{
			Enabled:             true,
			BaseURL:             "https://api.302.ai",
			Provider:            "302-gpt-image",
			ModelName:           "gpt-image-2",
			Mode:                "async",
			PollIntervalSeconds: 2,
			MaxWaitSeconds:      300,
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
	if err := loadAliOSSAgentConfig(filepath.Join(rootDir, "configs", "alioss-agent.yaml"), cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("加载 aliOSS 配置失败: %w", err)
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
	if v := os.Getenv("GYRH_MODEL_HTTP_TIMEOUT_MINUTES"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_MODEL_HTTP_TIMEOUT_MINUTES 无效: %s", v)
		}
		cfg.Models.HTTPTimeoutMinutes = value
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
	if v := os.Getenv("GYRH_ALIOSS_GENERATED_PORT"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_ALIOSS_GENERATED_PORT 无效: %s", v)
		}
		cfg.AliOSS.GeneratedPort = value
	}
	if v := firstEnv("GYRH_ALIOSS_ENDPOINT", "OSS_ENDPOINT"); v != "" {
		cfg.AliOSS.Endpoint = v
	}
	if v := firstEnv("GYRH_ALIOSS_BUCKET", "OSS_BUCKET"); v != "" {
		cfg.AliOSS.BucketName = v
	}
	if v := os.Getenv("GYRH_ALIOSS_BACKGROUND_BUCKET_PREFIX"); v != "" {
		cfg.AliOSS.BackgroundBucketPrefix = v
	}
	if v := os.Getenv("GYRH_ALIOSS_GENERATED_BUCKET_PREFIX"); v != "" {
		cfg.AliOSS.GeneratedBucketPrefix = v
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

	if v := os.Getenv("GYRH_302_HELPER_ENABLED"); v != "" {
		cfg.Helpper302.Enabled = isTrue(v)
	}
	if v := os.Getenv("GYRH_302_HELPER_BASE_URL"); v != "" {
		cfg.Helpper302.BaseURL = v
	}
	if v := os.Getenv("GYRH_302_HELPER_PROVIDER"); v != "" {
		cfg.Helpper302.Provider = v
	}
	if v := os.Getenv("GYRH_302_HELPER_MODEL_NAME"); v != "" {
		cfg.Helpper302.ModelName = v
	}
	if v := os.Getenv("GYRH_302_HELPER_MODE"); v != "" {
		cfg.Helpper302.Mode = v
	}
	if v := os.Getenv("GYRH_302_HELPER_POLL_INTERVAL_SECONDS"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_302_HELPER_POLL_INTERVAL_SECONDS 无效: %s", v)
		}
		cfg.Helpper302.PollIntervalSeconds = value
	}
	if v := os.Getenv("GYRH_302_HELPER_MAX_WAIT_SECONDS"); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GYRH_302_HELPER_MAX_WAIT_SECONDS 无效: %s", v)
		}
		cfg.Helpper302.MaxWaitSeconds = value
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
	if cfg.Models.HTTPTimeoutMinutes <= 0 {
		return fmt.Errorf("models.http_timeout_minutes 必须大于 0")
	}
	if cfg.Models.HTTPTimeoutMinutes > 20 {
		return fmt.Errorf("models.http_timeout_minutes 不能超过 20 分钟")
	}
	if cfg.AliOSS.Port <= 0 || cfg.AliOSS.Port > 65535 {
		return fmt.Errorf("alioss.port 无效: %d", cfg.AliOSS.Port)
	}
	if cfg.AliOSS.GeneratedPort <= 0 || cfg.AliOSS.GeneratedPort > 65535 {
		return fmt.Errorf("alioss.generated_port 无效: %d", cfg.AliOSS.GeneratedPort)
	}
	if cfg.AliOSS.GeneratedPort == cfg.AliOSS.Port {
		return fmt.Errorf("alioss.generated_port 不能与 alioss.port 相同")
	}
	if cfg.AliOSS.Enabled {
		if strings.TrimSpace(cfg.AliOSS.Endpoint) == "" {
			return fmt.Errorf("alioss.endpoint 不能为空")
		}
		if strings.TrimSpace(cfg.AliOSS.BucketName) == "" {
			return fmt.Errorf("alioss.bucket_name 不能为空")
		}
		if strings.TrimSpace(cfg.AliOSS.BackgroundBucketPrefix) == "" {
			return fmt.Errorf("alioss.background_bucket_prefix 不能为空")
		}
		if strings.TrimSpace(cfg.AliOSS.GeneratedBucketPrefix) == "" {
			return fmt.Errorf("alioss.generated_bucket_prefix 不能为空")
		}
	}
	if cfg.Helpper302.Enabled {
		if strings.TrimSpace(cfg.Helpper302.BaseURL) == "" {
			return fmt.Errorf("helpper302.base_url 不能为空")
		}
		if strings.TrimSpace(cfg.Helpper302.Provider) == "" {
			return fmt.Errorf("helpper302.provider 不能为空")
		}
		if strings.TrimSpace(cfg.Helpper302.ModelName) == "" {
			return fmt.Errorf("helpper302.model_name 不能为空")
		}
		if cfg.Helpper302.Mode != "async" && cfg.Helpper302.Mode != "sync" {
			return fmt.Errorf("helpper302.mode 仅支持 async 或 sync")
		}
		if cfg.Helpper302.PollIntervalSeconds <= 0 {
			cfg.Helpper302.PollIntervalSeconds = 2
		}
		if cfg.Helpper302.MaxWaitSeconds <= 0 {
			cfg.Helpper302.MaxWaitSeconds = 300
		}
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
}

type aliOSSAgentFile struct {
	OSS struct {
		Endpoint              string `yaml:"endpoint"`
		BucketName            string `yaml:"bucket_name"`
		BucketPrefix          string `yaml:"bucket_prefix"`
		GeneratedBucketPrefix string `yaml:"generated_bucket_prefix"`
	} `yaml:"oss"`
	Server struct {
		Port         int    `yaml:"port"`
		OpenAIAPIKey string `yaml:"openai_api_key"`
	} `yaml:"server"`
}

func loadAliOSSAgentConfig(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var agent aliOSSAgentFile
	if err := yaml.Unmarshal(data, &agent); err != nil {
		return fmt.Errorf("解析 aliOSS agent YAML 失败: %w", err)
	}

	if agent.OSS.Endpoint != "" {
		cfg.AliOSS.Endpoint = agent.OSS.Endpoint
	}
	if agent.OSS.BucketName != "" {
		cfg.AliOSS.BucketName = agent.OSS.BucketName
	}
	if agent.OSS.BucketPrefix != "" {
		cfg.AliOSS.BackgroundBucketPrefix = agent.OSS.BucketPrefix
	}
	if agent.OSS.GeneratedBucketPrefix != "" {
		cfg.AliOSS.GeneratedBucketPrefix = agent.OSS.GeneratedBucketPrefix
	}
	if agent.Server.Port > 0 {
		cfg.AliOSS.Port = agent.Server.Port
		if cfg.AliOSS.GeneratedPort <= 0 {
			cfg.AliOSS.GeneratedPort = agent.Server.Port + 1
		}
	}
	if agent.Server.OpenAIAPIKey != "" {
		cfg.AliOSS.OpenAIAPIKey = agent.Server.OpenAIAPIKey
	}
	if cfg.AliOSS.GeneratedBucketPrefix == "" {
		cfg.AliOSS.GeneratedBucketPrefix = "gyrh_images_data/"
	}
	if cfg.AliOSS.GeneratedPort <= 0 && cfg.AliOSS.Port > 0 {
		cfg.AliOSS.GeneratedPort = cfg.AliOSS.Port + 1
	}
	return nil
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
