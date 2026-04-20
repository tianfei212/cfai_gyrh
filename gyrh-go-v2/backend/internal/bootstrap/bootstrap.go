package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/config"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
)

// Service 管理启动期初始化与后台同步任务。
type Service struct {
	cfg            *config.Config
	skillRepo      *db.SkillRepo
	imageRepo      *db.ImageRepo
	storageService storage.StorageService
}

// New 创建启动服务。
func New(cfg *config.Config, skillRepo *db.SkillRepo, imageRepo *db.ImageRepo, storageService storage.StorageService) *Service {
	return &Service{
		cfg:            cfg,
		skillRepo:      skillRepo,
		imageRepo:      imageRepo,
		storageService: storageService,
	}
}

// Initialize 执行一次性的启动初始化。
func (s *Service) Initialize(ctx context.Context) error {
	if s.cfg.Skill.UseDatabase {
		if err := s.seedSkills(); err != nil {
			return err
		}
	}
	if s.cfg.Import.Enabled {
		if err := s.importGeneratedImages(ctx); err != nil {
			return err
		}
	}
	return nil
}

func resolveSkillFilePath(basePath, provider string) (string, error) {
	candidates := []string{provider + ".md"}
	if provider == "google" {
		candidates = append(candidates, "gemini.md")
	}

	for _, name := range candidates {
		path := filepath.Join(basePath, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到 provider=%s 对应的 Skill 文件", provider)
}

// StartWatchers 启动后台监控。
func (s *Service) StartWatchers(ctx context.Context) {
	go s.watchSkillImportFile(ctx)
	go s.watchGeneratedImports(ctx)
}

func (s *Service) seedSkills() error {
	for _, provider := range []string{"google", "wan"} {
		path, err := resolveSkillFilePath(s.cfg.Skill.LocalPath, provider)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取本地 Skill 文件失败: %w", err)
		}
		if err := s.skillRepo.UpsertByName(provider+".md", string(content), provider, true); err != nil {
			return err
		}
	}
	logger.Info("Skill 初始化完成")
	return nil
}

func (s *Service) reloadSkills() error {
	for _, provider := range []string{"google", "wan"} {
		if err := s.skillRepo.DeleteByProvider(provider); err != nil {
			return err
		}
	}
	return s.seedSkills()
}

func (s *Service) watchSkillImportFile(ctx context.Context) {
	if s.cfg.Skill.ImportTriggerFile == "" || !s.cfg.Skill.UseDatabase {
		return
	}

	ticker := time.NewTicker(time.Duration(s.cfg.Skill.WatchIntervalSecond) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := os.Stat(s.cfg.Skill.ImportTriggerFile); err == nil {
				logger.Info("检测到 Skill 导入触发文件，开始刷新数据库中的 Skill")
				if err := s.reloadSkills(); err != nil {
					logger.Error("刷新 Skill 失败: %v", err)
					continue
				}
				_ = os.Remove(s.cfg.Skill.ImportTriggerFile)
			}
		}
	}
}

func (s *Service) watchGeneratedImports(ctx context.Context) {
	if !s.cfg.Import.Enabled {
		return
	}

	ticker := time.NewTicker(time.Duration(s.cfg.Import.WatchIntervalSecond) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.importGeneratedImages(ctx); err != nil {
				logger.Error("批量导入生成图失败: %v", err)
			}
		}
	}
}

func (s *Service) importGeneratedImages(ctx context.Context) error {
	if s.cfg.Import.GeneratedDir == "" {
		return nil
	}

	if err := os.MkdirAll(s.cfg.Import.GeneratedDir, 0755); err != nil {
		return fmt.Errorf("创建生成图导入目录失败: %w", err)
	}

	entries, err := os.ReadDir(s.cfg.Import.GeneratedDir)
	if err != nil {
		return fmt.Errorf("读取生成图导入目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !isImageFile(entry.Name()) {
			continue
		}

		if _, err := s.imageRepo.GetByName(entry.Name()); err == nil {
			continue
		}

		sourcePath := filepath.Join(s.cfg.Import.GeneratedDir, entry.Name())
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			logger.Error("读取待导入图片失败: %v", err)
			continue
		}

		assetID, err := s.storageService.SaveWithKind(ctx, data, entry.Name(), storage.SaveKindGenerated)
		if err != nil {
			logger.Error("保存导入图片失败: %v", err)
			continue
		}

		if _, err := s.imageRepo.Create(entry.Name(), assetID, false, "imported"); err != nil {
			logger.Error("写入导入图片数据库失败: %v", err)
			continue
		}

		archivePath := sourcePath + ".imported"
		_ = os.Rename(sourcePath, archivePath)
		logger.Info("批量导入生成图完成: %s", entry.Name())
	}

	return nil
}

func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}
