package llm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gyrh-go-v2/backend/internal/config"
	"gyrh-go-v2/backend/internal/core/llm/gemini"
	"gyrh-go-v2/backend/internal/core/llm/wan"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
)

// ImageType 表示参与模型生成的图片类型。
type ImageType string

const (
	ImageTypeCharacter  ImageType = "character"
	ImageTypeBackground ImageType = "background"
	ImageTypeUpper      ImageType = "upper"
	ImageTypeLower      ImageType = "lower"
	ImageTypeDress      ImageType = "dress"
	ImageTypeAccessory  ImageType = "accessory"
	ImageTypeHeadwear   ImageType = "headwear"
	ImageTypeFootwear   ImageType = "footwear"
)

// ImageInput 表示一张输入图片。
type ImageInput struct {
	Type    ImageType
	AssetID string
}

// ComposeParams 表示融合请求参数。
type ComposeParams struct {
	Provider string
	Prompt   string
	Images   []ImageInput
}

// ComposeResult 表示融合结果。
type ComposeResult struct {
	ImageURL string
	Base64   string
	Error    error
}

// Service 定义大模型服务能力。
type Service interface {
	Compose(ctx context.Context, params ComposeParams) (*ComposeResult, error)
	LoadSkill(ctx context.Context, provider string) (string, error)
	BuildPrompt(ctx context.Context, provider, userPrompt string) (string, error)
}

type service struct {
	cfg            *config.Config
	storageService storage.StorageService
	skillRepo      *db.SkillRepo
	geminiHandler  *gemini.GeminiHandler
	wanHandler     *wan.WanHandler
}

// NewService 创建大模型服务。
func NewService(cfg *config.Config, storageService storage.StorageService, skillRepo *db.SkillRepo) (Service, error) {
	svc := &service{
		cfg:            cfg,
		storageService: storageService,
		skillRepo:      skillRepo,
		geminiHandler:  gemini.NewGeminiHandler(storageService),
		wanHandler:     wan.NewWanHandler(storageService),
	}
	logger.Info("LLM 服务初始化完成")
	return svc, nil
}

// Compose 执行图像融合。
func (s *service) Compose(ctx context.Context, params ComposeParams) (*ComposeResult, error) {
	provider := normalizeProvider(params.Provider)
	prompt, err := s.BuildPrompt(ctx, provider, params.Prompt)
	if err != nil {
		return nil, err
	}

	switch provider {
	case "wan":
		result, err := s.wanHandler.Compose(ctx, &wan.ComposeParams{
			CharacterImage:  firstImageByType(params.Images, ImageTypeCharacter),
			BackgroundImage: firstImageByType(params.Images, ImageTypeBackground),
			ReferenceImages: convertWanReferences(params.Images),
			UserPrompt:      params.Prompt,
		}, prompt)
		if err != nil {
			return nil, err
		}
		return &ComposeResult{
			ImageURL: result.ImageURL,
			Base64:   result.Base64,
			Error:    result.Error,
		}, nil
	default:
		result, err := s.geminiHandler.Compose(ctx, &gemini.ComposeParams{
			CharacterImage:  firstImageByType(params.Images, ImageTypeCharacter),
			BackgroundImage: firstImageByType(params.Images, ImageTypeBackground),
			ReferenceImages: convertGeminiReferences(params.Images),
			UserPrompt:      params.Prompt,
		}, prompt)
		if err != nil {
			return nil, err
		}
		return &ComposeResult{
			ImageURL: result.ImageURL,
			Base64:   result.Base64,
			Error:    result.Error,
		}, nil
	}
}

// LoadSkill 加载 Skill。
func (s *service) LoadSkill(ctx context.Context, provider string) (string, error) {
	if s.cfg.Skill.UseDatabase && s.skillRepo != nil {
		skill, err := s.skillRepo.GetActive(provider)
		if err == nil {
			return skill.Content, nil
		}
		logger.Warn("从数据库加载 Skill 失败，回退本地文件: %v", err)
	}

	return loadLocalSkill(s.cfg.Skill.LocalPath, provider)
}

// BuildPrompt 组合 Skill 与用户提示词。
func (s *service) BuildPrompt(ctx context.Context, provider, userPrompt string) (string, error) {
	skill, err := s.LoadSkill(ctx, provider)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(userPrompt) == "" {
		return skill, nil
	}
	return skill + "\n\n## 用户附加提示词\n" + userPrompt, nil
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "google", "gemini":
		return "google"
	case "wan", "aliwan", "tongyi":
		return "wan"
	default:
		return "google"
	}
}

func loadLocalSkill(basePath, provider string) (string, error) {
	candidates := []string{provider + ".md"}
	if provider == "google" {
		candidates = append(candidates, "gemini.md")
	}
	for _, name := range candidates {
		path := filepath.Join(basePath, name)
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("读取本地 Skill 文件失败: provider=%s", provider)
}

func firstImageByType(inputs []ImageInput, imageType ImageType) string {
	for _, item := range inputs {
		if item.Type == imageType {
			return item.AssetID
		}
	}
	return ""
}

func convertGeminiReferences(inputs []ImageInput) map[gemini.ImageType]string {
	result := make(map[gemini.ImageType]string)
	for _, item := range inputs {
		switch item.Type {
		case ImageTypeUpper:
			result[gemini.ImageTypeUpper] = item.AssetID
		case ImageTypeLower:
			result[gemini.ImageTypeLower] = item.AssetID
		case ImageTypeDress:
			result[gemini.ImageTypeDress] = item.AssetID
		case ImageTypeAccessory:
			result[gemini.ImageTypeAccessory] = item.AssetID
		case ImageTypeHeadwear:
			result[gemini.ImageTypeHeadwear] = item.AssetID
		case ImageTypeFootwear:
			result[gemini.ImageTypeFootwear] = item.AssetID
		}
	}
	return result
}

func convertWanReferences(inputs []ImageInput) map[wan.ImageType]string {
	result := make(map[wan.ImageType]string)
	for _, item := range inputs {
		switch item.Type {
		case ImageTypeUpper:
			result[wan.ImageTypeUpper] = item.AssetID
		case ImageTypeLower:
			result[wan.ImageTypeLower] = item.AssetID
		case ImageTypeDress:
			result[wan.ImageTypeDress] = item.AssetID
		case ImageTypeAccessory:
			result[wan.ImageTypeAccessory] = item.AssetID
		case ImageTypeHeadwear:
			result[wan.ImageTypeHeadwear] = item.AssetID
		case ImageTypeFootwear:
			result[wan.ImageTypeFootwear] = item.AssetID
		}
	}
	return result
}

// ValidateImages 校验图片顺序和内容。
func ValidateImages(images []ImageInput) error {
	if len(images) == 0 {
		return fmt.Errorf("至少需要一张输入图片")
	}
	return nil
}
