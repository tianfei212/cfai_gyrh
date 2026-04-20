package wan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
)

// Provider 模型提供者标识
const Provider = "wan"

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

// ComposeParams 光影融合参数（Wan 专用）
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

// WanHandler 阿里云百炼模型处理器
type WanHandler struct {
	storage storage.StorageService
	apiKey  string
	baseURL string
	model   string
}

// NewWanHandler 创建 Wan 处理器实例
func NewWanHandler(storageSvc storage.StorageService, model string) *WanHandler {
	return &WanHandler{
		storage: storageSvc,
		apiKey:  os.Getenv("WAN_API_KEY"),
		baseURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation",
		model:   model,
	}
}

// GetProvider 返回模型提供者标识
func (h *WanHandler) GetProvider() string {
	return Provider
}

// WanRequest 阿里云百炼 API 请求结构
type WanRequest struct {
	Model string   `json:"model"`
	Input WanInput `json:"input"`
}

// WanInput 输入参数
type WanInput struct {
	Prompt    string   `json:"prompt"`
	ImageURLs []string `json:"image_urls,omitempty"`
}

// WanResponse 阿里云百炼 API 响应结构
type WanResponse struct {
	Output    WanOutput `json:"output"`
	Usage     Usage     `json:"usage"`
	RequestID string    `json:"request_id"`
}

// WanOutput 输出参数
type WanOutput struct {
	ImageURL string `json:"image_url"`
}

// Usage 使用量统计
type Usage struct {
	ImageCount int `json:"image_count"`
}

// Compose 执行光影融合。
// prompt 已经是上层整理好的最终文本，Wan 侧不再重复拼装。
func (h *WanHandler) Compose(ctx context.Context, params *ComposeParams, prompt string) (*ComposeResult, error) {
	logger.Info("Wan 光影融合开始: character=%s, background=%s",
		params.CharacterImage, params.BackgroundImage)

	// 组装图像 URL 列表（严格协议顺序）
	imageURLs, err := h.assembleImageURLs(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("组装图像 URL 失败: %w", err)
	}

	// 构建请求
	wanReq := WanRequest{
		Model: h.model,
		Input: WanInput{
			Prompt: prompt,
		},
	}

	// 如果有图像 URL，添加到请求中
	if len(imageURLs) > 0 {
		wanReq.Input.ImageURLs = imageURLs
	}

	// 发送请求
	response, err := h.sendRequest(ctx, wanReq)
	if err != nil {
		return nil, fmt.Errorf("Wan API 请求失败: %w", err)
	}

	// 解析响应
	result, err := h.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析 Wan 响应失败: %w", err)
	}

	logger.Info("Wan 光影融合完成")
	return result, nil
}

// assembleImageURLs 组装图像 URL 列表（严格协议顺序）
// 1: 人物透明通道图
// 2: 背景图
// 3-8: 参考图（upper/lower/dress/accessory/headwear/footwear）
// Wan 策略：必须使用 OSS 路径
func (h *WanHandler) assembleImageURLs(ctx context.Context, params *ComposeParams) ([]string, error) {
	var imageURLs []string

	// 1. 人物透明通道图
	if params.CharacterImage != "" {
		url, err := h.getOSSURL(ctx, params.CharacterImage)
		if err != nil {
			logger.Warn("获取人物透明通道图 OSS URL 失败: %v", err)
		} else {
			imageURLs = append(imageURLs, url)
		}
	}

	// 2. 背景图
	if params.BackgroundImage != "" {
		url, err := h.getOSSURL(ctx, params.BackgroundImage)
		if err != nil {
			logger.Warn("获取背景图 OSS URL 失败: %v", err)
		} else {
			imageURLs = append(imageURLs, url)
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
			url, err := h.getOSSURL(ctx, assetID)
			if err != nil {
				logger.Warn("获取参考图 OSS URL 失败 [%s]: %v", imgType, err)
			} else {
				imageURLs = append(imageURLs, url)
			}
		}
	}

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf("没有可用的图像数据")
	}

	logger.Debug("组装图像 URL 列表成功: count=%d", len(imageURLs))
	return imageURLs, nil
}

// getOSSURL 获取图像的 OSS URL
// Wan 策略：必须使用 OSS 路径，不支持 Base64
func (h *WanHandler) getOSSURL(ctx context.Context, assetID string) (string, error) {
	uploadURL, err := h.storage.GetForModelUpload(ctx, assetID, storage.ProviderWan)
	if err != nil {
		return "", fmt.Errorf("获取 OSS 上传路径失败: %w", err)
	}
	logger.Debug("获取 Wan 模型 OSS 路径成功: assetID=%s, url=%s", assetID, uploadURL)
	return uploadURL, nil
}

// sendRequest 发送请求到 Wan API
func (h *WanHandler) sendRequest(ctx context.Context, req WanRequest) (*WanResponse, error) {
	url := h.baseURL

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.apiKey)

	client := &http.Client{Timeout: 180 * time.Second}
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

	var wanResp WanResponse
	if err := json.Unmarshal(body, &wanResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &wanResp, nil
}

// parseResponse 解析 Wan 响应
func (h *WanHandler) parseResponse(resp *WanResponse) (*ComposeResult, error) {
	if resp.Output.ImageURL == "" {
		return nil, fmt.Errorf("Wan 返回空图像 URL")
	}

	return &ComposeResult{
		ImageURL: resp.Output.ImageURL,
		Base64:   "", // Wan 使用 URL 方式
	}, nil
}
