package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Provider           string
	StylePrompt        string
	Images             []ImageInput
	BackgroundPromptID int64
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
}

type service struct {
	cfg                  *config.Config
	skillRepo            *db.SkillRepo
	backgroundPromptRepo *db.BackgroundPromptRepo
	geminiHandler        *gemini.GeminiHandler
	wanHandler           *wan.WanHandler
}

// NewService 创建大模型服务。
func NewService(cfg *config.Config, storageService storage.StorageService, skillRepo *db.SkillRepo, backgroundPromptRepo *db.BackgroundPromptRepo) (Service, error) {
	requestTimeout := time.Duration(cfg.Models.HTTPTimeoutMinutes) * time.Minute

	svc := &service{
		cfg:                  cfg,
		skillRepo:            skillRepo,
		backgroundPromptRepo: backgroundPromptRepo,
		geminiHandler:        gemini.NewGeminiHandler(storageService, cfg.Models.Gemini, requestTimeout),
		wanHandler:           wan.NewWanHandler(storageService, cfg.Models.Wan, requestTimeout),
	}
	logger.Info("LLM 服务初始化完成")
	return svc, nil
}

// ErrBackgroundPromptNotFound 表示请求中引用的背景图模板不存在。
var ErrBackgroundPromptNotFound = errors.New("背景图提示词模板不存在")

// Compose 执行图像融合。
func (s *service) Compose(ctx context.Context, params ComposeParams) (*ComposeResult, error) {
	provider := normalizeProvider(params.Provider)
	prompt, err := s.buildPrompt(ctx, provider, params)
	if err != nil {
		return nil, err
	}

	switch provider {
	case "wan":
		result, err := s.wanHandler.Compose(ctx, &wan.ComposeParams{
			CharacterImage:  firstImageByType(params.Images, ImageTypeCharacter),
			BackgroundImage: firstImageByType(params.Images, ImageTypeBackground),
			ReferenceImages: convertWanReferences(params.Images),
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

// loadSkill 加载 Skill。
func (s *service) loadSkill(_ context.Context, provider string) (string, error) {
	if s.cfg.Skill.UseDatabase && s.skillRepo != nil {
		skill, err := s.skillRepo.GetActive(provider)
		if err == nil {
			return skill.Content, nil
		}
		logger.Warn("从数据库加载 Skill 失败，回退本地文件: %v", err)
	}

	return loadLocalSkill(s.cfg.Skill.LocalPath, provider)
}

// buildPrompt 根据 Skill、背景模板和风格控制词生成最终 Prompt。
func (s *service) buildPrompt(ctx context.Context, provider string, params ComposeParams) (string, error) {
	skill, err := s.loadSkill(ctx, provider)
	if err != nil {
		return "", err
	}

	var (
		backgroundPrompt         string
		backgroundNegativePrompt string
	)
	if hasImageType(params.Images, ImageTypeBackground) && params.BackgroundPromptID > 0 {
		backgroundPrompt, backgroundNegativePrompt, err = s.loadBackgroundPrompt(ctx, provider, params.BackgroundPromptID)
		if err != nil {
			return "", err
		}
	}

	return composePromptDocument(skill, backgroundPrompt, backgroundNegativePrompt, params.StylePrompt), nil
}

// loadBackgroundPrompt 加载指定 provider 对应的背景图提示词模板。
func (s *service) loadBackgroundPrompt(ctx context.Context, provider string, backgroundPromptID int64) (string, string, error) {
	_ = ctx
	if s.backgroundPromptRepo == nil {
		return "", "", fmt.Errorf("背景图提示词仓库未初始化")
	}

	item, err := s.backgroundPromptRepo.GetByID(backgroundPromptID)
	if err != nil {
		return "", "", fmt.Errorf("%w: id=%d", ErrBackgroundPromptNotFound, backgroundPromptID)
	}

	switch provider {
	case "wan":
		return strings.TrimSpace(item.WanPrompt), strings.TrimSpace(item.WanNegativePrompt), nil
	default:
		return strings.TrimSpace(item.GeminiPrompt), strings.TrimSpace(item.GeminiNegativePrompt), nil
	}
}

// composePromptDocument 按固定结构组装最终 Prompt，避免多层重复拼接。
func composePromptDocument(skill, backgroundPrompt, backgroundNegativePrompt, stylePrompt string) string {
	sections := make([]string, 0, 4)

	if skill = strings.TrimSpace(skill); skill != "" {
		sections = append(sections, "[Skill]\n"+skill+"\n[/Skill]")
	}
	if backgroundPrompt = strings.TrimSpace(backgroundPrompt); backgroundPrompt != "" {
		sections = append(sections, "[BackgroundPrompt]\n"+backgroundPrompt+"\n[/BackgroundPrompt]")
	}
	if backgroundNegativePrompt = strings.TrimSpace(backgroundNegativePrompt); backgroundNegativePrompt != "" {
		sections = append(sections, "[BackgroundNegativePrompt]\n请避免在背景中出现以下内容：\n"+backgroundNegativePrompt+"\n[/BackgroundNegativePrompt]")
	}
	if stylePrompt = strings.TrimSpace(stylePrompt); stylePrompt != "" {
		sections = append(sections, "[StylePrompt]\n"+stylePrompt+"\n[/StylePrompt]")
	}

	return strings.Join(sections, "\n\n")
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

func hasImageType(inputs []ImageInput, imageType ImageType) bool {
	return firstImageByType(inputs, imageType) != ""
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
