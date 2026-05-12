package model

import (
	"context"

	"gyrh-go-v2/backend/internal/core/llm"
	domainmodel "gyrh-go-v2/backend/internal/domain/model"
)

// Router 在新分层架构中适配旧 LLM Service，便于逐步迁移各 provider 客户端。
type Router struct {
	service llm.Service
}

// NewRouter 创建模型路由适配器。
func NewRouter(service llm.Service) *Router {
	return &Router{service: service}
}

// Compose 调用旧 LLM Service 完成图像融合生成。
func (r *Router) Compose(ctx context.Context, req domainmodel.ComposeRequest) (*domainmodel.ComposeResult, error) {
	images := make([]llm.ImageInput, 0, len(req.Images))
	for _, item := range req.Images {
		images = append(images, llm.ImageInput{
			Type:    llm.ImageType(item.Type),
			AssetID: item.AssetID,
		})
	}
	result, err := r.service.Compose(ctx, llm.ComposeParams{
		Provider:           req.Provider,
		StylePrompt:        req.StylePrompt,
		BackgroundPromptID: req.BackgroundPromptID,
		Images:             images,
	})
	if err != nil {
		return nil, err
	}
	return &domainmodel.ComposeResult{
		ImageURL: result.ImageURL,
		Base64:   result.Base64,
		Status:   result.Status,
	}, nil
}
