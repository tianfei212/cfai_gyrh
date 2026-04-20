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

	if prepareErr := prepareAliOSSRuntimeFiles(cfg); prepareErr != nil {
		logger.Fatal("准备 aliOSS 运行配置失败: %v", prepareErr)
	}

	database, err := db.NewDB(filepath.Join(cfg.Storage.LocalPath, "..", "gyrh.db"))
	if err != nil {
		logger.Fatal("初始化数据库失败: %v", err)
	}
	defer database.Close()

	aliOSSManager := oss.NewManager(&cfg.AliOSS)
	if startErr := aliOSSManager.Start(context.Background()); startErr != nil {
		logger.Fatal("启动 aliOSS 服务失败: %v", startErr)
	}

	storageService, err := storage.NewStorageService(cfg)
	if err != nil {
		logger.Fatal("初始化存储服务失败: %v", err)
	}

	imageRepo := db.NewImageRepo(database)
	referenceRepo := db.NewReferenceRepo(database)
	skillRepo := db.NewSkillRepo(database)

	llmService, err := llm.NewService(cfg, storageService, skillRepo)
	if err != nil {
		logger.Fatal("初始化 LLM 服务失败: %v", err)
	}

	bootstrapService := bootstrap.New(cfg, skillRepo, imageRepo, storageService)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := bootstrapService.Initialize(ctx); err != nil {
		logger.Fatal("执行启动初始化失败: %v", err)
	}
	bootstrapService.StartWatchers(ctx)

	imageHandler := handler.NewImageHandler(imageRepo, storageService, llmService, cfg)
	referenceHandler := handler.NewReferenceHandler(referenceRepo, storageService)
	skillHandler := handler.NewSkillHandler(skillRepo)

	router := mux.NewRouter()
	api.RegisterRoutes(router, imageHandler, referenceHandler, skillHandler, &middleware.AuthConfig{
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

	waitForShutdown(ctx, cancel, server, aliOSSManager)
}

func waitForShutdown(ctx context.Context, cancel context.CancelFunc, server *http.Server, aliOSSManager *oss.Manager) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("关闭 HTTP 服务失败: %v", err)
	}
	if err := aliOSSManager.Stop(shutdownCtx); err != nil {
		logger.Error("关闭 aliOSS 服务失败: %v", err)
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

func prepareAliOSSRuntimeFiles(cfg *config.Config) error {
	rootDir, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	configPath := filepath.Join(rootDir, "configs", "alioss-agent.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	if cfg.Storage.OssEndpoint != "" {
		_ = os.Setenv("OSS_ENDPOINT", cfg.Storage.OssEndpoint)
	}
	if cfg.Storage.OssBucket != "" {
		_ = os.Setenv("OSS_BUCKET", cfg.Storage.OssBucket)
	}
	if cfg.Storage.OssBucketPrefix != "" {
		_ = os.Setenv("OSS_BUCKET_PREFIX", cfg.Storage.OssBucketPrefix)
	}
	if cfg.AliOSS.OpenAIAPIKey != "" {
		_ = os.Setenv("OPENAI_API_KEY", cfg.AliOSS.OpenAIAPIKey)
	}

	content := fmt.Sprintf(`oss:
  endpoint: %q
  bucket_name: %q
  bucket_prefix: %q

server:
  port: %d
  link_expire_seconds: 3600
  openai_api_key: ""
`, cfg.Storage.OssEndpoint, cfg.Storage.OssBucket, cfg.Storage.OssBucketPrefix, cfg.AliOSS.Port)

	return os.WriteFile(configPath, []byte(content), 0644)
}
