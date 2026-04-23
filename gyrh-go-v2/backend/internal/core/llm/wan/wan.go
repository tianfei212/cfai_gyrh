package wan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
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
	// BackgroundWidth 背景模板图宽度
	BackgroundWidth int
	// BackgroundHeight 背景模板图高度
	BackgroundHeight int
}

// ComposeResult 光影融合结果
type ComposeResult struct {
	// ImageURL 生成的图像 URL
	ImageURL string
	// Base64 生成的图像 Base64（当使用 Base64 传输时）
	Base64 string
	// Status 任务状态
	Status string
	// Error 错误信息
	Error error
}

// WanHandler 阿里云百炼模型处理器
type WanHandler struct {
	storage storage.StorageService
	apiKey  string
	baseURL string
	model   string
	timeout time.Duration
}

// NewWanHandler 创建 Wan 处理器实例
func NewWanHandler(storageSvc storage.StorageService, model string, timeout time.Duration) *WanHandler {
	return &WanHandler{
		storage: storageSvc,
		apiKey:  os.Getenv("WAN_API_KEY"),
		baseURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation",
		model:   model,
		timeout: timeout,
	}
}

// GetProvider 返回模型提供者标识
func (h *WanHandler) GetProvider() string {
	return Provider
}

// WanRequest 阿里云百炼 API 请求结构
type WanRequest struct {
	Model      string        `json:"model"`
	Input      WanInput      `json:"input"`
	Parameters WanParameters `json:"parameters,omitempty"`
}

// WanInput 输入参数
type WanInput struct {
	Prompt   string       `json:"prompt,omitempty"`
	Messages []WanMessage `json:"messages,omitempty"`
}

// WanMessage 消息结构
type WanMessage struct {
	Role    string       `json:"role"`
	Content []WanContent `json:"content"`
}

// WanContent 内容结构
type WanContent struct {
	Image string `json:"image,omitempty"`
	Text  string `json:"text,omitempty"`
}

// WanParameters 生成参数
type WanParameters struct {
	N              int    `json:"n,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	PromptExtend   bool   `json:"prompt_extend,omitempty"`
	Watermark      bool   `json:"watermark"`
	Size           string `json:"size,omitempty"`
}

// WanResponse 阿里云百炼 API 响应结构
type WanResponse struct {
	Output    WanOutput `json:"output"`
	Usage     Usage     `json:"usage"`
	RequestID string    `json:"request_id"`
}

// WanOutput 输出参数
type WanOutput struct {
	Results    []WanResult `json:"results,omitempty"`
	ImageURL   string      `json:"image_url,omitempty"`
	TaskID     string      `json:"task_id,omitempty"`
	TaskStatus string      `json:"task_status,omitempty"`
	Choices    []WanChoice `json:"choices,omitempty"`
}

// WanResult 结果项
type WanResult struct {
	URL string `json:"url"`
}

// WanChoice 兼容 DashScope 在 output.choices 中返回结果的结构。
type WanChoice struct {
	Message WanMessageResponse `json:"message"`
}

// WanMessageResponse 助手消息结构。
type WanMessageResponse struct {
	Content []WanMessagePart `json:"content"`
	Role    string           `json:"role"`
}

// WanMessagePart 消息内容片段，图片结果会出现在 image 字段。
type WanMessagePart struct {
	Image string `json:"image,omitempty"`
	Type  string `json:"type,omitempty"`
	Text  string `json:"text,omitempty"`
}

// Usage 使用量统计
type Usage struct {
	ImageCount int `json:"image_count"`
}

// Compose 执行光影融合。
// prompt 和 negativePrompt 已经是上层从数据库中整理好的最终提示词。
func (h *WanHandler) Compose(ctx context.Context, params *ComposeParams, prompt, negativePrompt string) (*ComposeResult, error) {
	logger.Info("Wan 光影融合开始: character=%s, background=%s",
		params.CharacterImage, params.BackgroundImage)

	// 组装图像 URL 列表（严格协议顺序）
	imageURLs, err := h.assembleImageURLs(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("组装图像 URL 失败: %w", err)
	}

	var contents []WanContent
	for _, url := range imageURLs {
		contents = append(contents, WanContent{Image: url})
	}
	contents = append(contents, WanContent{Text: prompt})

	sizeValue, sizeLog := resolveOutputSize(params.BackgroundWidth, params.BackgroundHeight)
	wanReq := WanRequest{
		Model: h.model,
		Input: WanInput{
			Messages: []WanMessage{
				{
					Role:    "user",
					Content: contents,
				},
			},
		},
		Parameters: WanParameters{
			N:              1,
			NegativePrompt: strings.TrimSpace(negativePrompt),
			PromptExtend:   true,
			Watermark:      false,
			Size:           sizeValue,
		},
	}
	logger.Info("Wan 输出尺寸策略: %s", sizeLog)

	// 发送请求
	response, err := h.sendRequest(ctx, wanReq)
	if err != nil {
		return nil, fmt.Errorf("Wan API 请求失败: %w", err)
	}

	// 解析响应
	result, err := h.parseResponse(ctx, response)
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
	if len(imageURLs) > 0 {
		logger.Debug("Wan 图1(人物图): %s", imageURLs[0])
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
	if len(imageURLs) > 1 {
		logger.Debug("Wan 图2(场景图): %s", imageURLs[1])
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

	logger.Debug("组装图像 URL 列表成功: count=%d，顺序固定为图1人物、图2场景、后续才是参考图", len(imageURLs))
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
	return sanitizeURL(uploadURL), nil
}

// sendRequest 发送请求到 Wan API
func (h *WanHandler) sendRequest(ctx context.Context, req WanRequest) (*WanResponse, error) {
	url := h.baseURL

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	logger.Debug("Wan Request JSON: %s", string(jsonData))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.apiKey)

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

	logger.Debug("Wan Response JSON: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var wanResp WanResponse
	if err := json.Unmarshal(body, &wanResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &wanResp, nil
}

// parseResponse 解析 Wan 响应。
func (h *WanHandler) parseResponse(_ context.Context, resp *WanResponse) (*ComposeResult, error) {
	var imageURL string
	if len(resp.Output.Results) > 0 {
		imageURL = resp.Output.Results[0].URL
	} else if len(resp.Output.Choices) > 0 {
		for _, choice := range resp.Output.Choices {
			for _, part := range choice.Message.Content {
				if strings.TrimSpace(part.Image) != "" {
					imageURL = part.Image
					break
				}
			}
			if imageURL != "" {
				break
			}
		}
	} else if resp.Output.ImageURL != "" {
		imageURL = resp.Output.ImageURL
	}
	imageURL = sanitizeURL(imageURL)

	if imageURL == "" {
		return nil, fmt.Errorf("Wan 返回空图像 URL, resp=%+v", resp)
	}

	return &ComposeResult{
		ImageURL: imageURL,
		Base64:   "", // Wan 使用 URL 方式
		Status:   normalizeTaskStatus(resp.Output.TaskStatus),
	}, nil
}

func sanitizeURL(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, "`")
	value = strings.TrimSpace(value)
	return value
}

func normalizeTaskStatus(status string) string {
	value := strings.TrimSpace(strings.ToLower(status))
	if value == "" {
		return "succeeded"
	}
	return value
}

func resolveOutputSize(width, height int) (string, string) {
	const (
		defaultSize = "2K"
		minPixels   = 768 * 768
		maxPixels   = 2048 * 2048
		maxRatio    = 8.0
	)

	if width <= 0 || height <= 0 {
		return defaultSize, "background_size=unknown, request_size=2K(fallback)"
	}

	ratio := float64(width) / float64(height)
	if ratio < 1.0/maxRatio || ratio > maxRatio {
		return defaultSize, fmt.Sprintf("background_size=%dx%d, request_size=2K(fallback: aspect ratio out of range)", width, height)
	}

	targetWidth := width
	targetHeight := height
	pixels := width * height

	switch {
	case pixels > maxPixels:
		scale := math.Sqrt(float64(maxPixels) / float64(pixels))
		targetWidth = maxInt(1, int(math.Round(float64(width)*scale)))
		targetHeight = maxInt(1, int(math.Round(float64(height)*scale)))
	case pixels < minPixels:
		scale := math.Sqrt(float64(minPixels) / float64(pixels))
		targetWidth = maxInt(1, int(math.Round(float64(width)*scale)))
		targetHeight = maxInt(1, int(math.Round(float64(height)*scale)))
	}

	sizeValue := fmt.Sprintf("%d*%d", targetWidth, targetHeight)
	if targetWidth == width && targetHeight == height {
		return sizeValue, fmt.Sprintf("background_size=%dx%d, request_size=%s(exact)", width, height, sizeValue)
	}
	return sizeValue, fmt.Sprintf("background_size=%dx%d, request_size=%s(scaled)", width, height, sizeValue)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
