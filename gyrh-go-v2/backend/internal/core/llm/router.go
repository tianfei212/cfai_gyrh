package llm

import (
	"context"
	"errors"
	"fmt"
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
	Status   string
	Error    error
}

type resolvedPrompt struct {
	Prompt         string
	NegativePrompt string
	BackgroundName string
	ImageWidth     int
	ImageHeight    int
}

// Service 定义大模型服务能力。
type Service interface {
	Compose(ctx context.Context, params ComposeParams) (*ComposeResult, error)
}

type service struct {
	cfg                  *config.Config
	skillRepo            *db.SkillRepo
	backgroundPromptRepo *db.BackgroundPromptRepo
	storageService       storage.StorageService
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
		storageService:       storageService,
		geminiHandler:        gemini.NewGeminiHandler(storageService, cfg.Models.Gemini, requestTimeout),
		wanHandler:           wan.NewWanHandler(storageService, cfg.Models.Wan, requestTimeout),
	}
	logger.Info("图像生成服务初始化完成")
	return svc, nil
}

// ErrBackgroundPromptNotFound 表示请求中引用的背景图模板不存在。
var ErrBackgroundPromptNotFound = errors.New("背景图提示词模板不存在")

// Compose 执行图像融合。
func (s *service) Compose(ctx context.Context, params ComposeParams) (*ComposeResult, error) {
	provider := normalizeProvider(params.Provider)
	resolved, err := s.buildPrompt(ctx, provider, params)
	if err != nil {
		return nil, err
	}

	logger.Debug("========== Compose Request ==========")
	logger.Debug("Provider: %s", provider)
	logger.Debug("BackgroundPromptID: %d", params.BackgroundPromptID)
	logger.Debug("Images Count: %d", len(params.Images))
	logger.Debug("Prompt: \n%s", resolved.Prompt)
	logger.Debug("NegativePrompt: \n%s", resolved.NegativePrompt)
	logger.Debug("=====================================")

	switch provider {
	case "wan":
		result, err := s.wanHandler.Compose(ctx, &wan.ComposeParams{
			CharacterImage:   firstImageByType(params.Images, ImageTypeCharacter),
			BackgroundImage:  firstImageByType(params.Images, ImageTypeBackground),
			ReferenceImages:  convertWanReferences(params.Images),
			BackgroundWidth:  resolved.ImageWidth,
			BackgroundHeight: resolved.ImageHeight,
		}, resolved.Prompt, resolved.NegativePrompt)
		if err != nil {
			logger.Error("Wan Compose Error: %v", err)
			return nil, err
		}

		logger.Debug("========== Wan Compose Response ==========")
		if result.Error != nil {
			logger.Debug("Result Error: %v", result.Error)
		} else {
			logger.Debug("Result ImageURL: %s", result.ImageURL)
			logger.Debug("Result Base64 Length: %d", len(result.Base64))
		}
		logger.Debug("========================================")

		return &ComposeResult{
			ImageURL: result.ImageURL,
			Base64:   result.Base64,
			Status:   result.Status,
			Error:    result.Error,
		}, nil
	default:
		result, err := s.geminiHandler.Compose(ctx, &gemini.ComposeParams{
			CharacterImage:  firstImageByType(params.Images, ImageTypeCharacter),
			BackgroundImage: firstImageByType(params.Images, ImageTypeBackground),
			ReferenceImages: convertGeminiReferences(params.Images),
		}, composeGeminiPrompt(resolved.Prompt, resolved.NegativePrompt))
		if err != nil {
			logger.Error("Gemini Compose Error: %v", err)
			return nil, err
		}

		logger.Debug("========== Gemini Compose Response ==========")
		if result.Error != nil {
			logger.Debug("Result Error: %v", result.Error)
		} else {
			logger.Debug("Result ImageURL: %s", result.ImageURL)
			logger.Debug("Result Base64 Length: %d", len(result.Base64))
		}
		logger.Debug("============================================")

		return &ComposeResult{
			ImageURL: result.ImageURL,
			Base64:   result.Base64,
			Status:   "succeeded",
			Error:    result.Error,
		}, nil
	}
}

// buildPrompt 根据背景模板和风格控制词生成最终 Prompt。
func (s *service) buildPrompt(ctx context.Context, provider string, params ComposeParams) (*resolvedPrompt, error) {
	resolved := &resolvedPrompt{
		Prompt: strings.TrimSpace(params.StylePrompt),
	}

	if hasImageType(params.Images, ImageTypeBackground) && params.BackgroundPromptID > 0 {
		item, err := s.loadBackgroundPromptItem(ctx, params.BackgroundPromptID)
		if err != nil {
			return nil, err
		}
		resolved.BackgroundName = item.Name
		resolved.ImageWidth = item.ImageWidth
		resolved.ImageHeight = item.ImageHeight

		switch provider {
		case "wan":
			resolved.Prompt = strings.TrimSpace(item.WanPrompt)
			resolved.NegativePrompt = strings.TrimSpace(item.WanNegativePrompt)
			logger.Debug("背景提示词来源: background_prompts.id=%d, field=wan_prompt/wan_negative_prompt, name=%s, width=%d, height=%d", params.BackgroundPromptID, item.Name, item.ImageWidth, item.ImageHeight)
		default:
			resolved.Prompt = strings.TrimSpace(item.GeminiPrompt)
			resolved.NegativePrompt = strings.TrimSpace(item.GeminiNegativePrompt)
			logger.Debug("背景提示词来源: background_prompts.id=%d, field=gemini_prompt/gemini_negative_prompt, name=%s, width=%d, height=%d", params.BackgroundPromptID, item.Name, item.ImageWidth, item.ImageHeight)
		}
	}

	return resolved, nil
}

// loadBackgroundPromptItem 加载背景图提示词模板。
func (s *service) loadBackgroundPromptItem(ctx context.Context, backgroundPromptID int64) (*db.BackgroundPrompt, error) {
	_ = ctx
	if s.backgroundPromptRepo == nil {
		return nil, fmt.Errorf("背景图提示词仓库未初始化")
	}

	item, err := s.backgroundPromptRepo.GetByID(backgroundPromptID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrBackgroundPromptNotFound, backgroundPromptID)
	}
	return item, nil
}

func composeGeminiPrompt(prompt, negativePrompt string) string {
	prompt = strings.TrimSpace(prompt)
	negativePrompt = strings.TrimSpace(negativePrompt)
	if prompt == "" {
		return ""
	}
	if negativePrompt == "" {
		return prompt
	}
	return prompt + "\n\nNegative prompt: " + negativePrompt
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
