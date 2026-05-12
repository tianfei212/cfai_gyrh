package model

import "context"

const (
	// ProviderGemini 表示连接 Google Gemini 图像生成模型。
	ProviderGemini = "google"
	// ProviderWan 表示连接 DashScope Wan 图像生成模型。
	ProviderWan = "wan"
	// ProviderQwen 表示连接 Qwen 文本理解和提示词同步模型。
	ProviderQwen = "qwen"
	// Provider302GPTImage 表示连接 302.ai GPT Image 图片编辑模型。
	Provider302GPTImage = "302-gpt-image"
)

// ComposeRequest 描述应用层向图像模型发起的一次融合生成请求。
type ComposeRequest struct {
	Provider           string
	StylePrompt        string
	BackgroundPromptID int64
	Images             []ImageInput
}

// ComposeResult 描述图像模型生成完成后的结果。
type ComposeResult struct {
	ImageURL string
	Base64   string
	Status   string
}

// ImageInput 表示传给模型的一张输入图片。
type ImageInput struct {
	Type    string
	AssetID string
}

// Provider 定义所有图像模型客户端需要满足的统一能力。
type Provider interface {
	Compose(ctx context.Context, req ComposeRequest) (*ComposeResult, error)
}
