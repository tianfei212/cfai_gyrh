package qwen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
)

const defaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"

// Advisor 提供基于 Qwen 的背景图提示词建议与英文同步能力。
type Advisor interface {
	SuggestFromAsset(ctx context.Context, backgroundAssetID string) (*SuggestionResult, error)
	SyncEnglish(ctx context.Context, input SyncEnglishInput) (*SyncEnglishResult, error)
}

type advisor struct {
	storage    storage.StorageService
	promptRepo *db.LLMPromptTemplateRepo
	apiKey     string
	model      string
	baseURL    string
}

// SuggestionResult 表示背景图提示词建议结果。
type SuggestionResult struct {
	GeminiPromptZH         string `json:"gemini_prompt_zh"`
	GeminiPromptEN         string `json:"gemini_prompt_en"`
	GeminiNegativePromptZH string `json:"gemini_negative_prompt_zh"`
	GeminiNegativePromptEN string `json:"gemini_negative_prompt_en"`
	WanPromptZH            string `json:"wan_prompt_zh"`
	WanPromptEN            string `json:"wan_prompt_en"`
	WanNegativePromptZH    string `json:"wan_negative_prompt_zh"`
	WanNegativePromptEN    string `json:"wan_negative_prompt_en"`
}

// SyncEnglishInput 表示需要同步成英文的中文提示词。
type SyncEnglishInput struct {
	GeminiPromptZH         string `json:"gemini_prompt_zh"`
	GeminiNegativePromptZH string `json:"gemini_negative_prompt_zh"`
	WanPromptZH            string `json:"wan_prompt_zh"`
	WanNegativePromptZH    string `json:"wan_negative_prompt_zh"`
}

// SyncEnglishResult 表示同步后的英文提示词。
type SyncEnglishResult struct {
	GeminiPromptEN         string `json:"gemini_prompt_en"`
	GeminiNegativePromptEN string `json:"gemini_negative_prompt_en"`
	WanPromptEN            string `json:"wan_prompt_en"`
	WanNegativePromptEN    string `json:"wan_negative_prompt_en"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// NewAdvisor 创建 Qwen 建议服务。
func NewAdvisor(storageService storage.StorageService, promptRepo *db.LLMPromptTemplateRepo, model string, apiKey string) Advisor {
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	return &advisor{
		storage:    storageService,
		promptRepo: promptRepo,
		apiKey:     apiKey,
		model:      model,
		baseURL:    defaultBaseURL,
	}
}

// SuggestFromAsset 基于背景图生成 Gemini/Wan 的中英双语提示词建议。
func (a *advisor) SuggestFromAsset(ctx context.Context, backgroundAssetID string) (*SuggestionResult, error) {
	imageURL, err := a.storage.GetForModelUpload(ctx, backgroundAssetID, storage.ProviderQwen)
	if err != nil {
		return nil, fmt.Errorf("准备背景图失败: %w", err)
	}

	prompt, err := a.loadPromptTemplate(db.TemplateKeyQwenBackgroundSuggest)
	if err != nil {
		return nil, err
	}

	content, err := a.chat(ctx, []chatMessage{
		{
			Role:    "system",
			Content: "你是严谨的 JSON 输出助手，只返回合法 JSON。",
		},
		{
			Role: "user",
			Content: []map[string]any{
				{
					"type": "image_url",
					"image_url": map[string]string{
						"url": imageURL,
					},
				},
				{
					"type": "text",
					"text": prompt,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var result SuggestionResult
	if err := decodeJSONContent(content, &result); err != nil {
		return nil, fmt.Errorf("解析 Qwen 建议结果失败: %w", err)
	}
	return &result, nil
}

// SyncEnglish 根据当前中文提示词生成对应英文版本。
func (a *advisor) SyncEnglish(ctx context.Context, input SyncEnglishInput) (*SyncEnglishResult, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("序列化中文提示词失败: %w", err)
	}

	basePrompt, err := a.loadPromptTemplate(db.TemplateKeyQwenSyncEnglish)
	if err != nil {
		return nil, err
	}
	prompt := basePrompt + "\n\n输入 JSON：\n" + string(payload)

	content, err := a.chat(ctx, []chatMessage{
		{
			Role:    "system",
			Content: "你是严谨的 JSON 输出助手，只返回合法 JSON。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	})
	if err != nil {
		return nil, err
	}

	var result SyncEnglishResult
	if err := decodeJSONContent(content, &result); err != nil {
		return nil, fmt.Errorf("解析英文同步结果失败: %w", err)
	}
	return &result, nil
}

func (a *advisor) loadPromptTemplate(templateKey string) (string, error) {
	if a.promptRepo == nil {
		return "", fmt.Errorf("Qwen Prompt 模板仓库未初始化")
	}

	item, err := a.promptRepo.GetByKey(templateKey)
	if err != nil {
		return "", fmt.Errorf("读取 Qwen Prompt 模板失败: %w", err)
	}
	return strings.TrimSpace(item.Content), nil
}

func (a *advisor) chat(ctx context.Context, messages []chatMessage) (string, error) {
	if strings.TrimSpace(a.apiKey) == "" {
		return "", fmt.Errorf("DashScope API Key 未配置")
	}

	reqBody := chatCompletionRequest{
		Model:       a.model,
		Messages:    messages,
		Temperature: 0.2,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化 Qwen 请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("创建 Qwen 请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Qwen 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Qwen 响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Qwen 响应异常，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(body, &completion); err != nil {
		return "", fmt.Errorf("解析 Qwen 响应失败: %w", err)
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("Qwen 返回空响应")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	logger.Debug("Qwen 返回内容: %s", content)
	return content, nil
}

func decodeJSONContent(raw string, out any) error {
	content := strings.TrimSpace(raw)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return fmt.Errorf("未找到 JSON 对象")
	}

	if err := json.Unmarshal([]byte(content[start:end+1]), out); err != nil {
		return err
	}
	return nil
}
