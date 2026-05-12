package GPT

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/logger"
)

const (
	defaultBaseURL      = "https://api.302.ai"
	defaultModelName    = "gpt-image-2"
	defaultPollInterval = 2 * time.Second
	defaultMaxWait      = 300 * time.Second
)

// Config controls the direct 302.ai GPT Image client.
type Config struct {
	Enabled             bool
	BaseURL             string
	ModelName           string
	PollIntervalSeconds int
	MaxWaitSeconds      int
}

// Client calls 302.ai GPT Image APIs directly.
type Client struct {
	cfg          Config
	httpClient   *http.Client
	pollInterval time.Duration
	maxWait      time.Duration
}

// ComposeRequest contains one GPT Image edit request.
type ComposeRequest struct {
	Prompt          string
	ForegroundImage []byte
	BackgroundImage []byte
}

// ComposeResult contains the generated image bytes and source URL.
type ComposeResult struct {
	Image []byte
	URL   string
}

// NewClient creates a direct 302.ai GPT Image client.
func NewClient(cfg Config) *Client {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	if cfg.ModelName == "" {
		cfg.ModelName = defaultModelName
	}

	pollInterval := time.Duration(cfg.PollIntervalSeconds) * time.Second
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	maxWait := time.Duration(cfg.MaxWaitSeconds) * time.Second
	if maxWait <= 0 {
		maxWait = defaultMaxWait
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: maxWait + 30*time.Second,
		},
		pollInterval: pollInterval,
		maxWait:      maxWait,
	}
}

// Compose submits an async GPT Image edit task, waits for it, and downloads the result.
func (c *Client) Compose(ctx context.Context, req ComposeRequest) (*ComposeResult, error) {
	taskID, err := c.CreateTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.WaitResult(ctx, taskID)
}

// CreateTask submits a direct 302.ai GPT Image edits request.
func (c *Client) CreateTask(ctx context.Context, req ComposeRequest) (string, error) {
	if err := c.validateReady(); err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("302 GPT prompt 不能为空")
	}
	if len(req.ForegroundImage) == 0 {
		return "", fmt.Errorf("302 GPT 需要前景人物图")
	}

	apiKey := providerAPIKey()
	if apiKey == "" {
		return "", fmt.Errorf("未设置 PROVIDER_302_API_KEY，无法直连 302 GPT")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writeImagePart(writer, "image", "foreground.png", req.ForegroundImage); err != nil {
		return "", err
	}
	fields := map[string]string{
		"prompt":         prompt,
		"model":          c.cfg.ModelName,
		"n":              "1",
		"quality":        "high",
		"background":     "auto",
		"output_format":  "png",
		"moderation":     "auto",
		"input_fidelity": "high",
		"size":           "1536x1024",
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return "", fmt.Errorf("写入 302 GPT 字段 %s 失败: %w", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("关闭 302 GPT multipart 失败: %w", err)
	}

	endpoint, err := c.endpoint("/v1/images/edits", url.Values{
		"response_format": []string{"url"},
		"async":           []string{"true"},
	})
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	logger.Info("302 GPT 创建任务: model=%s prompt_len=%d image_bytes=%d", c.cfg.ModelName, len(prompt), len(req.ForegroundImage))
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("调用 302 GPT 创建任务失败: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("302 GPT 创建任务失败: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	taskID, err := extractTaskID(data)
	if err != nil {
		return "", err
	}
	logger.Info("302 GPT 创建任务成功: task_id=%s", taskID)
	return taskID, nil
}

// WaitResult polls 302.ai async_result and downloads the generated image.
func (c *Client) WaitResult(ctx context.Context, taskID string) (*ComposeResult, error) {
	if err := c.validateReady(); err != nil {
		return nil, err
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("302 GPT task_id 不能为空")
	}
	if providerAPIKey() == "" {
		return nil, fmt.Errorf("未设置 PROVIDER_302_API_KEY，无法直连 302 GPT")
	}

	deadline := time.Now().Add(c.maxWait)
	for {
		resultURL, done, err := c.fetchResultURL(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if done {
			image, err := c.downloadImage(ctx, resultURL)
			if err != nil {
				return nil, err
			}
			return &ComposeResult{Image: image, URL: resultURL}, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("302 GPT 任务超时")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.pollInterval):
		}
	}
}

func (c *Client) fetchResultURL(ctx context.Context, taskID string) (string, bool, error) {
	endpoint, err := c.endpoint("/async_result", url.Values{"task_id": []string{taskID}})
	if err != nil {
		return "", false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+providerAPIKey())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("轮询 302 GPT 任务失败: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("轮询 302 GPT 任务失败: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	resultURL, done, err := parseAsyncResult(data)
	if err != nil {
		return "", false, err
	}
	if !done {
		logger.Debug("302 GPT 任务仍在处理中: task_id=%s", taskID)
	}
	return resultURL, done, nil
}

func (c *Client) downloadImage(ctx context.Context, resultURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resultURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 302 GPT 结果失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 302 GPT 结果失败: status=%d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("302 GPT 结果不是图片: %s", contentType)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) validateReady() error {
	if c == nil || !c.cfg.Enabled {
		return fmt.Errorf("302 GPT provider 未启用")
	}
	return nil
}

func (c *Client) endpoint(path string, query url.Values) (string, error) {
	base, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return "", fmt.Errorf("解析 302 GPT base_url 失败: %w", err)
	}
	rel, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	resolved := base.ResolveReference(rel)
	resolved.RawQuery = query.Encode()
	return resolved.String(), nil
}

func writeImagePart(writer *multipart.Writer, fieldName, filename string, data []byte) error {
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		return fmt.Errorf("创建 302 GPT 图片字段失败: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("写入 302 GPT 图片字段失败: %w", err)
	}
	return nil
}

func extractTaskID(data []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("解析 302 GPT 创建任务响应失败: %w", err)
	}
	for _, key := range []string{"task_id", "job_id", "id"} {
		if id := stringify(payload[key]); id != "" {
			return id, nil
		}
	}
	if nested, ok := payload["data"].(map[string]any); ok {
		for _, key := range []string{"task_id", "job_id", "id"} {
			if id := stringify(nested[key]); id != "" {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("302 GPT 未返回 task_id")
}

func parseAsyncResult(data []byte) (string, bool, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false, fmt.Errorf("解析 302 GPT 异步结果失败: %w", err)
	}
	if msg := strings.TrimSpace(stringify(payload["err"])); msg != "" {
		if isPendingMessage(msg) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("302 GPT 上游任务失败: %s", msg)
	}

	statusCode := intValue(payload["status_code"])
	if statusCode != 0 && statusCode != http.StatusOK {
		return "", false, nil
	}
	if resultURL := extractResultURL(payload["data"]); resultURL != "" {
		return resultURL, true, nil
	}
	if statusCode == http.StatusOK {
		return "", false, fmt.Errorf("302 GPT 未返回结果 URL")
	}
	return "", false, nil
}

func isPendingMessage(msg string) bool {
	switch strings.ToLower(strings.TrimSpace(msg)) {
	case "result pending", "pending":
		return true
	default:
		return false
	}
}

func extractResultURL(value any) string {
	if resultURL := stringify(value); strings.HasPrefix(resultURL, "http") {
		return resultURL
	}
	dataMap, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	items, ok := dataMap["data"].([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	resultURL := stringify(first["url"])
	if strings.HasPrefix(resultURL, "http") {
		return resultURL
	}
	return ""
}

func providerAPIKey() string {
	return strings.TrimSpace(os.Getenv("PROVIDER_302_API_KEY"))
}

func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		result, _ := typed.Int64()
		return int(result)
	default:
		return 0
	}
}
