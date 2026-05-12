package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	gpt302 "gyrh-go-v2/backend/internal/302Helpper/GPT"
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
	StartCompose(ctx context.Context, params ComposeParams) (*StartedComposeTask, error)
	WaitComposeResult(ctx context.Context, provider, externalTaskID string) (*ComposeResult, error)
}

type StartedComposeTask struct {
	ExternalTaskID string
}

type service struct {
	cfg                  *config.Config
	skillRepo            *db.SkillRepo
	backgroundPromptRepo *db.BackgroundPromptRepo
	storageService       storage.StorageService
	geminiHandler        *gemini.GeminiHandler
	wanHandler           *wan.WanHandler
	helpper302Client     *gpt302.Client
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
		helpper302Client: gpt302.NewClient(gpt302.Config{
			Enabled:             cfg.Helpper302.Enabled,
			BaseURL:             cfg.Helpper302.BaseURL,
			ModelName:           cfg.Helpper302.ModelName,
			PollIntervalSeconds: cfg.Helpper302.PollIntervalSeconds,
			MaxWaitSeconds:      cfg.Helpper302.MaxWaitSeconds,
		}),
	}
	logger.Info("图像生成服务初始化完成")
	return svc, nil
}

func (s *service) StartCompose(ctx context.Context, params ComposeParams) (*StartedComposeTask, error) {
	provider := normalizeProvider(params.Provider)
	if provider != "302-gpt-image" {
		return nil, fmt.Errorf("provider %s 不支持外部异步任务", provider)
	}
	resolved, err := s.buildPrompt(ctx, provider, params)
	if err != nil {
		return nil, err
	}
	characterAsset := firstImageByType(params.Images, ImageTypeCharacter)
	if characterAsset == "" {
		return nil, fmt.Errorf("302-gpt-image 需要人物图")
	}
	backgroundAsset := firstImageByType(params.Images, ImageTypeBackground)
	if backgroundAsset == "" {
		return nil, fmt.Errorf("302-gpt-image 需要背景图")
	}
	foregroundBytes, err := s.storageService.Read(ctx, characterAsset)
	if err != nil {
		return nil, fmt.Errorf("读取 302-gpt-image 人物图失败: %w", err)
	}
	backgroundBytes, err := s.storageService.Read(ctx, backgroundAsset)
	if err != nil {
		return nil, fmt.Errorf("读取 302-gpt-image 背景图失败: %w", err)
	}
	externalTaskID, err := s.helpper302Client.CreateTask(ctx, gpt302.ComposeRequest{
		Prompt:          resolved.Prompt,
		ForegroundImage: foregroundBytes,
		BackgroundImage: backgroundBytes,
	})
	if err != nil {
		return nil, err
	}
	return &StartedComposeTask{ExternalTaskID: externalTaskID}, nil
}

func (s *service) WaitComposeResult(ctx context.Context, provider, externalTaskID string) (*ComposeResult, error) {
	provider = normalizeProvider(provider)
	if provider != "302-gpt-image" {
		return nil, fmt.Errorf("provider %s 不支持外部异步任务", provider)
	}
	result, err := s.helpper302Client.WaitResult(ctx, externalTaskID)
	if err != nil {
		return nil, err
	}
	return &ComposeResult{
		Base64: base64.StdEncoding.EncodeToString(result.Image),
		Status: "succeeded",
	}, nil
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
	case "302-gpt-image":
		characterAsset := firstImageByType(params.Images, ImageTypeCharacter)
		if characterAsset == "" {
			return nil, fmt.Errorf("302-gpt-image 需要人物图")
		}
		backgroundAsset := firstImageByType(params.Images, ImageTypeBackground)
		if backgroundAsset == "" {
			return nil, fmt.Errorf("302-gpt-image 需要背景图")
		}
		foregroundBytes, err := s.storageService.Read(ctx, characterAsset)
		if err != nil {
			return nil, fmt.Errorf("读取 302-gpt-image 人物图失败: %w", err)
		}
		backgroundBytes, err := s.storageService.Read(ctx, backgroundAsset)
		if err != nil {
			return nil, fmt.Errorf("读取 302-gpt-image 背景图失败: %w", err)
		}
		result, err := s.helpper302Client.Compose(ctx, gpt302.ComposeRequest{
			Prompt:          resolved.Prompt,
			ForegroundImage: foregroundBytes,
			BackgroundImage: backgroundBytes,
		})
		if err != nil {
			logger.Error("302-gpt-image Compose Error: %v", err)
			return nil, err
		}

		logger.Debug("========== 302-gpt-image Compose Response ==========")
		logger.Debug("Result ImageURL: %s", result.URL)
		logger.Debug("Result Bytes Length: %d", len(result.Image))
		logger.Debug("====================================================")

		return &ComposeResult{
			Base64: base64.StdEncoding.EncodeToString(result.Image),
			Status: "succeeded",
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
		case "302-gpt-image":
			resolved.Prompt = strings.TrimSpace(item.GPTPrompt)
			resolved.NegativePrompt = strings.TrimSpace(item.GPTNegativePrompt)
			logger.Debug("背景提示词来源: background_prompts.id=%d, field=gpt_prompt/gpt_negative_prompt, name=%s, width=%d, height=%d", params.BackgroundPromptID, item.Name, item.ImageWidth, item.ImageHeight)
		default:
			resolved.Prompt = strings.TrimSpace(item.GeminiPrompt)
			resolved.NegativePrompt = strings.TrimSpace(item.GeminiNegativePrompt)
			logger.Debug("背景提示词来源: background_prompts.id=%d, field=gemini_prompt/gemini_negative_prompt, name=%s, width=%d, height=%d", params.BackgroundPromptID, item.Name, item.ImageWidth, item.ImageHeight)
		}

		if resolved.Prompt == "" && resolved.NegativePrompt == "" {
			if err := s.useActiveSkillPrompt(provider, resolved); err != nil {
				return nil, err
			}
			logger.Debug("背景提示词为空，回退到 active skill: background_prompts.id=%d, provider=%s", params.BackgroundPromptID, provider)
		}
	}

	if hasImageType(params.Images, ImageTypeBackground) && params.BackgroundPromptID == 0 {
		if err := s.useActiveSkillPrompt(provider, resolved); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}

func (s *service) useActiveSkillPrompt(provider string, resolved *resolvedPrompt) error {
	if s.skillRepo == nil {
		return fmt.Errorf("Skill 仓库未初始化，无法获取默认融合提示词")
	}
	skill, err := s.skillRepo.GetActive(provider)
	if err != nil {
		return fmt.Errorf("未找到当前模型的激活 Skill: provider=%s: %w", provider, err)
	}
	resolved.Prompt = strings.TrimSpace(skill.Content)
	resolved.NegativePrompt = ""
	logger.Debug("背景提示词来源: active skill id=%d, provider=%s, name=%s", skill.ID, provider, skill.Name)
	return nil
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
	outputRequirement := "Output requirements: generate a 16:9 landscape image at 2K resolution. Preserve a cinematic wide composition and do not output square or portrait framing."
	if prompt == "" {
		prompt = outputRequirement
	} else {
		prompt = prompt + "\n\n" + outputRequirement
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
	case "302-gpt-image", "gpt-image", "gpt":
		return "302-gpt-image"
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
