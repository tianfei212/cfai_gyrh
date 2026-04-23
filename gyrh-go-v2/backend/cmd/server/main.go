package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"gyrh-go-v2/backend/internal/api"
	"gyrh-go-v2/backend/internal/api/handler"
	"gyrh-go-v2/backend/internal/api/middleware"
	"gyrh-go-v2/backend/internal/bootstrap"
	"gyrh-go-v2/backend/internal/config"
	"gyrh-go-v2/backend/internal/core/llm"
	"gyrh-go-v2/backend/internal/core/llm/qwen"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/oss"
	"gyrh-go-v2/backend/internal/storage"
)

func main() {
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		panic(err)
	}

	logger.Init(logger.Config{
		Level:     parseLogLevel(cfg.Logger.Level),
		Directory: cfg.Logger.Path,
		MaxDays:   cfg.Logger.MaxAge,
		MaxSize:   100,
	})
	defer logger.Close()

	backgroundAgentConfigPath, generatedAgentConfigPath, prepareErr := prepareAliOSSRuntimeFiles(cfg)
	if prepareErr != nil {
		logger.Fatal("准备 aliOSS 运行配置失败: %v", prepareErr)
	}

	database, err := db.NewDB(filepath.Join(cfg.Storage.LocalPath, "..", "gyrh.db"))
	if err != nil {
		logger.Fatal("初始化数据库失败: %v", err)
	}
	defer database.Close()

	backgroundOSSConfig := cfg.AliOSS
	backgroundOSSConfig.Port = cfg.AliOSS.Port
	aliOSSManager := oss.NewManager(&backgroundOSSConfig, backgroundAgentConfigPath)
	if startErr := aliOSSManager.Start(context.Background()); startErr != nil {
		logger.Fatal("启动 aliOSS 素材服务失败: %v", startErr)
	}

	generatedOSSConfig := cfg.AliOSS
	generatedOSSConfig.Port = cfg.AliOSS.GeneratedPort
	generatedOSSManager := oss.NewManager(&generatedOSSConfig, generatedAgentConfigPath)
	if startErr := generatedOSSManager.Start(context.Background()); startErr != nil {
		logger.Fatal("启动 aliOSS 服务失败: %v", startErr)
	}

	storageService, err := storage.NewStorageService(cfg)
	if err != nil {
		logger.Fatal("初始化存储服务失败: %v", err)
	}

	imageRepo := db.NewImageRepo(database)
	referenceRepo := db.NewReferenceRepo(database)
	skillRepo := db.NewSkillRepo(database)
	llmPromptTemplateRepo := db.NewLLMPromptTemplateRepo(database)
	backgroundPromptRepo := db.NewBackgroundPromptRepo(database)

	if seedErr := qwen.EnsureDefaultTemplates(llmPromptTemplateRepo, cfg.Skill.LocalPath); seedErr != nil {
		logger.Fatal("初始化 Qwen 默认 Prompt 模板失败: %v", seedErr)
	}

	llmService, err := llm.NewService(cfg, storageService, skillRepo, backgroundPromptRepo)
	if err != nil {
		logger.Fatal("初始化 LLM 服务失败: %v", err)
	}
	requestTimeout := time.Duration(cfg.Models.HTTPTimeoutMinutes) * time.Minute
	qwenAdvisor := qwen.NewAdvisor(storageService, llmPromptTemplateRepo, cfg.Models.Qwen, cfg.Storage.DashScopeAPIKey, requestTimeout)

	bootstrapService := bootstrap.New(cfg, skillRepo, imageRepo, storageService)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := bootstrapService.Initialize(ctx); err != nil {
		logger.Fatal("执行启动初始化失败: %v", err)
	}
	bootstrapService.StartWatchers(ctx)

	imageHandler := handler.NewImageHandler(imageRepo, backgroundPromptRepo, storageService, llmService)
	referenceHandler := handler.NewReferenceHandler(referenceRepo, storageService)
	skillHandler := handler.NewSkillHandler(skillRepo)
	llmPromptTemplateHandler := handler.NewLLMPromptTemplateHandler(llmPromptTemplateRepo)
	backgroundPromptHandler := handler.NewBackgroundPromptHandler(backgroundPromptRepo, storageService, qwenAdvisor)

	router := mux.NewRouter()
	api.RegisterRoutes(router, imageHandler, referenceHandler, skillHandler, llmPromptTemplateHandler, backgroundPromptHandler, &middleware.AuthConfig{
		PrivateKeyFetcher: func(publicKey string) string {
			if configuredPublicKey := os.Getenv("GYRH_AUTH_PUBLIC_KEY"); configuredPublicKey != "" && configuredPublicKey == publicKey {
				return os.Getenv("GYRH_AUTH_PRIVATE_KEY")
			}
			envKey := "GYRH_AUTH_PRIVATE_KEY_" + strings.ToUpper(strings.ReplaceAll(publicKey, "-", "_"))
			return os.Getenv(envKey)
		},
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		logger.Info("服务已启动，监听端口: %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务启动失败: %v", err)
		}
	}()

	waitForShutdown(ctx, cancel, server, aliOSSManager, generatedOSSManager)
}

func waitForShutdown(ctx context.Context, cancel context.CancelFunc, server *http.Server, managers ...*oss.Manager) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("关闭 HTTP 服务失败: %v", err)
	}
	for _, manager := range managers {
		if manager == nil {
			continue
		}
		if err := manager.Stop(shutdownCtx); err != nil {
			logger.Error("关闭 aliOSS 服务失败: %v", err)
		}
	}

	logger.Info("服务已优雅关闭")
}

func parseLogLevel(value string) logger.Level {
	switch strings.ToLower(value) {
	case "debug":
		return logger.DebugLevel
	case "warn":
		return logger.WarnLevel
	case "error":
		return logger.ErrorLevel
	default:
		return logger.InfoLevel
	}
}

func prepareAliOSSRuntimeFiles(cfg *config.Config) (string, string, error) {
	rootDir, err := config.FindProjectRoot()
	if err != nil {
		return "", "", err
	}

	backgroundConfigPath := filepath.Join(rootDir, "configs", "alioss-agent.yaml")
	generatedConfigPath := filepath.Join(rootDir, "configs", "alioss-agent-generated.yaml")
	if err := os.MkdirAll(filepath.Dir(backgroundConfigPath), 0755); err != nil {
		return "", "", err
	}

	content := fmt.Sprintf(`oss:
  endpoint: %q
  bucket_name: %q
  bucket_prefix: %q
  generated_bucket_prefix: %q

server:
  port: %d
  link_expire_seconds: 3600
  openai_api_key: %q
`, cfg.AliOSS.Endpoint, cfg.AliOSS.BucketName, cfg.AliOSS.BackgroundBucketPrefix, cfg.AliOSS.GeneratedBucketPrefix, cfg.AliOSS.Port, cfg.AliOSS.OpenAIAPIKey)

	generatedContent := fmt.Sprintf(`oss:
  endpoint: %q
  bucket_name: %q
  bucket_prefix: %q
  generated_bucket_prefix: %q

server:
  port: %d
  link_expire_seconds: 3600
  openai_api_key: %q
`, cfg.AliOSS.Endpoint, cfg.AliOSS.BucketName, cfg.AliOSS.GeneratedBucketPrefix, cfg.AliOSS.GeneratedBucketPrefix, cfg.AliOSS.GeneratedPort, cfg.AliOSS.OpenAIAPIKey)

	if err := os.WriteFile(backgroundConfigPath, []byte(content), 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(generatedConfigPath, []byte(generatedContent), 0644); err != nil {
		return "", "", err
	}
	return backgroundConfigPath, generatedConfigPath, nil
}
