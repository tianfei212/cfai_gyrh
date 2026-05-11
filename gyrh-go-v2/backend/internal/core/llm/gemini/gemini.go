package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
)

// Provider 模型提供者标识
const Provider = "google/gemini"

// ImageType 图像类型枚举
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

// ComposeParams 光影融合参数
type ComposeParams struct {
	// CharacterImage 人物透明通道图（assetID）
	CharacterImage string
	// BackgroundImage 背景图（assetID）
	BackgroundImage string
	// ReferenceImages 参考图映射（按 ImageType 索引）
	ReferenceImages map[ImageType]string
}

// ComposeResult 光影融合结果
type ComposeResult struct {
	// ImageURL 生成的图像 URL
	ImageURL string
	// Base64 生成的图像 Base64（当使用 Base64 传输时）
	Base64 string
	// Error 错误信息
	Error error
}

// GeminiHandler Gemini 模型处理器
type GeminiHandler struct {
	storage  storage.StorageService
	apiKey   string
	endpoint string
	model    string
	timeout  time.Duration
}

// NewGeminiHandler 创建 Gemini 处理器实例
func NewGeminiHandler(storageSvc storage.StorageService, model string, timeout time.Duration) *GeminiHandler {
	return &GeminiHandler{
		storage:  storageSvc,
		apiKey:   os.Getenv("GEMINI_API_KEY"),
		endpoint: "https://generativelanguage.googleapis.com/v1beta/models",
		model:    model,
		timeout:  timeout,
	}
}

// GetProvider 返回模型提供者标识
func (h *GeminiHandler) GetProvider() string {
	return Provider
}

// GeminiRequest Gemini API 请求结构
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Content 对话内容
type Content struct {
	Parts []Part `json:"parts"`
}

// Part 内容部分
type Part struct {
	Text            string      `json:"text,omitempty"`
	InlineData      *InlineData `json:"inline_data,omitempty"`
	InlineDataCamel *InlineData `json:"inlineData,omitempty"`
}

// InlineData Gemini inline image data.
type InlineData struct {
	MimeType      string `json:"mime_type,omitempty"`
	MimeTypeCamel string `json:"mimeType,omitempty"`
	Data          string `json:"data"`
}

// GeminiResponse Gemini API 响应结构
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate 候选回复
type Candidate struct {
	Content Content `json:"content"`
}

// Compose 执行光影融合。
// prompt 已经是上层整理好的最终文本，Gemini 侧不再重复拼装。
func (h *GeminiHandler) Compose(ctx context.Context, params *ComposeParams, prompt string) (*ComposeResult, error) {
	logger.Info("Gemini 光影融合开始: character=%s, background=%s",
		params.CharacterImage, params.BackgroundImage)

	// 组装 Payload（严格协议顺序）
	payload, err := h.assemblePayload(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("组装 Payload 失败: %w", err)
	}

	// 构造 Gemini 请求
	geminiReq := GeminiRequest{
		Contents: []Content{
			{
				Parts: append([]Part{
					{Text: prompt},
				}, payload...),
			},
		},
	}

	// 发送请求
	response, err := h.sendRequest(ctx, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("Gemini API 请求失败: %w", err)
	}

	// 解析响应
	result, err := h.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析 Gemini 响应失败: %w", err)
	}

	logger.Info("Gemini 光影融合完成")
	return result, nil
}

// assemblePayload 组装 Payload（严格协议顺序）
// 1: 人物透明通道图
// 2: 背景图
// 3-8: 参考图（upper/lower/dress/accessory/headwear/footwear）
func (h *GeminiHandler) assemblePayload(ctx context.Context, params *ComposeParams) ([]Part, error) {
	var parts []Part

	// 1. 人物透明通道图
	if params.CharacterImage != "" {
		imagePart, err := h.prepareImagePart(ctx, params.CharacterImage, "image/png")
		if err != nil {
			logger.Warn("准备人物透明通道图失败: %v", err)
		} else {
			parts = append(parts, *imagePart)
		}
	}

	// 2. 背景图
	if params.BackgroundImage != "" {
		imagePart, err := h.prepareImagePart(ctx, params.BackgroundImage, "image/png")
		if err != nil {
			logger.Warn("准备背景图失败: %v", err)
		} else {
			parts = append(parts, *imagePart)
		}
	}

	// 3-8. 参考图（按顺序）
	referenceOrder := []ImageType{
		ImageTypeUpper,
		ImageTypeLower,
		ImageTypeDress,
		ImageTypeAccessory,
		ImageTypeHeadwear,
		ImageTypeFootwear,
	}

	for _, imgType := range referenceOrder {
		if assetID, ok := params.ReferenceImages[imgType]; ok && assetID != "" {
			imagePart, err := h.prepareImagePart(ctx, assetID, "image/png")
			if err != nil {
				logger.Warn("准备参考图失败 [%s]: %v", imgType, err)
			} else {
				parts = append(parts, *imagePart)
			}
		}
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("没有可用的图像数据")
	}

	return parts, nil
}

// prepareImagePart 准备图像 Part
// Gemini 策略：优先 OSS，失败回退 Base64
func (h *GeminiHandler) prepareImagePart(ctx context.Context, assetID string, mimeType string) (*Part, error) {
	source, err := h.storage.GetForModelUpload(ctx, assetID, storage.ProviderGoogle)
	if err == nil && source != "" {
		switch {
		case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
			remoteData, readErr := h.fetchRemoteImage(ctx, source)
			if readErr == nil {
				return encodeImagePart(remoteData, mimeType), nil
			}
			logger.Warn("读取 Gemini 远端图片失败，回退到 Base64: assetID=%s, error=%v", assetID, readErr)
		default:
			return &Part{
				InlineData: &InlineData{
					MimeType: mimeType,
					Data:     source,
				},
			}, nil
		}
	}

	logger.Warn("获取 Gemini 模型上传源失败，回退到 Base64: assetID=%s, error=%v", assetID, err)
	return h.prepareImagePartBase64(ctx, assetID, mimeType)
}

// prepareImagePartBase64 使用 Base64 方式准备图像 Part
func (h *GeminiHandler) prepareImagePartBase64(ctx context.Context, assetID string, mimeType string) (*Part, error) {
	imageData, err := h.storage.Read(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("读取图像失败: %w", err)
	}
	return encodeImagePart(imageData, mimeType), nil
}

func (h *GeminiHandler) fetchRemoteImage(ctx context.Context, sourceURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("读取远端图片失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

func encodeImagePart(imageData []byte, mimeType string) *Part {
	return &Part{
		InlineData: &InlineData{
			MimeType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(imageData),
		},
	}
}

// sendRequest 发送请求到 Gemini API
func (h *GeminiHandler) sendRequest(ctx context.Context, req GeminiRequest) (*GeminiResponse, error) {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", h.endpoint, h.model, h.apiKey)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: h.timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &geminiResp, nil
}

// parseResponse 解析 Gemini 响应
func (h *GeminiHandler) parseResponse(resp *GeminiResponse) (*ComposeResult, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("Gemini 返回空响应")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini 返回空内容")
	}

	// 查找图像数据
	for _, part := range candidate.Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != "" {
			// 返回 Base64 数据
			return &ComposeResult{
				Base64:   part.InlineData.Data,
				ImageURL: "", // Base64 模式下 ImageURL 为空
			}, nil
		}
		if part.InlineDataCamel != nil && part.InlineDataCamel.Data != "" {
			return &ComposeResult{
				Base64:   part.InlineDataCamel.Data,
				ImageURL: "",
			}, nil
		}

		// 查找文本响应（可能是图像 URL）
		if part.Text != "" {
			// 检查是否是 URL
			if strings.HasPrefix(part.Text, "http://") || strings.HasPrefix(part.Text, "https://") {
				return &ComposeResult{
					ImageURL: part.Text,
					Base64:   "",
				}, nil
			}

			// 返回文本内容（可能是错误信息或描述）
			logger.Info("Gemini 返回文本响应: %s", part.Text)
		}
	}

	return nil, fmt.Errorf("Gemini 响应中未找到图像数据")
}
