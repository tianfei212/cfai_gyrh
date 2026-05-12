package app

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

// Run 加载配置、初始化依赖并启动 HTTP 服务。
// 该函数是后端服务的组合根，负责把配置、数据库、存储、模型、Handler 和路由串联起来。
func Run(ctx context.Context) error {
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
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
		return fmt.Errorf("准备 aliOSS 运行配置失败: %w", prepareErr)
	}

	database, err := db.NewDB(filepath.Join(cfg.Storage.LocalPath, "..", "gyrh.db"))
	if err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	defer database.Close()

	backgroundOSSConfig := cfg.AliOSS
	backgroundOSSConfig.Port = cfg.AliOSS.Port
	aliOSSManager := oss.NewManager(&backgroundOSSConfig, backgroundAgentConfigPath)
	if startErr := aliOSSManager.Start(ctx); startErr != nil {
		return fmt.Errorf("启动 aliOSS 素材服务失败: %w", startErr)
	}

	generatedOSSConfig := cfg.AliOSS
	generatedOSSConfig.Port = cfg.AliOSS.GeneratedPort
	generatedOSSManager := oss.NewManager(&generatedOSSConfig, generatedAgentConfigPath)
	if startErr := generatedOSSManager.Start(ctx); startErr != nil {
		return fmt.Errorf("启动 aliOSS 服务失败: %w", startErr)
	}

	storageService, err := storage.NewStorageService(cfg)
	if err != nil {
		return fmt.Errorf("初始化存储服务失败: %w", err)
	}

	imageRepo := db.NewImageRepo(database)
	referenceRepo := db.NewReferenceRepo(database)
	skillRepo := db.NewSkillRepo(database)
	llmPromptTemplateRepo := db.NewLLMPromptTemplateRepo(database)
	backgroundPromptRepo := db.NewBackgroundPromptRepo(database)
	stylePromptRepo := db.NewStylePromptRepo(database)
	rewriteTaskRepo := db.NewRewriteTaskRepo(database)

	if seedErr := qwen.EnsureDefaultTemplates(llmPromptTemplateRepo, cfg.Skill.LocalPath); seedErr != nil {
		return fmt.Errorf("初始化 Qwen 默认 Prompt 模板失败: %w", seedErr)
	}

	llmService, err := llm.NewService(cfg, storageService, skillRepo, backgroundPromptRepo)
	if err != nil {
		return fmt.Errorf("初始化 LLM 服务失败: %w", err)
	}
	requestTimeout := time.Duration(cfg.Models.HTTPTimeoutMinutes) * time.Minute
	qwenAdvisor := qwen.NewAdvisor(storageService, llmPromptTemplateRepo, cfg.Models.Qwen, cfg.Storage.DashScopeAPIKey, requestTimeout)

	bootstrapService := bootstrap.New(cfg, skillRepo, imageRepo, storageService)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := bootstrapService.Initialize(runCtx); err != nil {
		return fmt.Errorf("执行启动初始化失败: %w", err)
	}
	bootstrapService.StartWatchers(runCtx)

	imageHandler := handler.NewImageHandler(imageRepo, backgroundPromptRepo, stylePromptRepo, storageService, llmService, rewriteTaskRepo)
	referenceHandler := handler.NewReferenceHandler(referenceRepo, storageService)
	skillHandler := handler.NewSkillHandler(skillRepo)
	llmPromptTemplateHandler := handler.NewLLMPromptTemplateHandler(llmPromptTemplateRepo)
	backgroundPromptHandler := handler.NewBackgroundPromptHandler(backgroundPromptRepo, storageService, qwenAdvisor)
	stylePromptHandler := handler.NewStylePromptHandler(stylePromptRepo)

	router := mux.NewRouter()
	api.RegisterRoutes(router, imageHandler, referenceHandler, skillHandler, llmPromptTemplateHandler, backgroundPromptHandler, stylePromptHandler, &middleware.AuthConfig{
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

	waitForShutdown(runCtx, cancel, server, aliOSSManager, generatedOSSManager)
	return nil
}

type stoppableManager interface {
	Stop(ctx context.Context) error
}

// waitForShutdown 等待系统退出信号，并按顺序关闭 HTTP 服务和外部子进程。
func waitForShutdown(ctx context.Context, cancel context.CancelFunc, server *http.Server, managers ...stoppableManager) {
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
			logger.Error("关闭外部服务失败: %v", err)
		}
	}

	logger.Info("服务已优雅关闭")
}

// parseLogLevel 将配置文件中的日志级别文本转换为内部日志等级。
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
