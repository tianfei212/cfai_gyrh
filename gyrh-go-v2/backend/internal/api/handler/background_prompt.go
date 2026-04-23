package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/core/llm/qwen"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
	"gyrh-go-v2/backend/pkg/httpx"

	_ "golang.org/x/image/webp"
)

// BackgroundPromptHandler 提供背景图提示词模板 CRUD 能力。
type BackgroundPromptHandler struct {
	repo           *db.BackgroundPromptRepo
	storageService storage.StorageService
	qwenAdvisor    qwen.Advisor
}

// NewBackgroundPromptHandler 创建背景图提示词处理器。
func NewBackgroundPromptHandler(repo *db.BackgroundPromptRepo, storageService storage.StorageService, qwenAdvisor qwen.Advisor) *BackgroundPromptHandler {
	return &BackgroundPromptHandler{
		repo:           repo,
		storageService: storageService,
		qwenAdvisor:    qwenAdvisor,
	}
}

// List 返回背景图提示词模板列表。
func (h *BackgroundPromptHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePage(r)

	items, err := h.repo.List(limit, offset)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询背景图提示词模板失败"))
		return
	}

	total, err := h.repo.Count()
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "统计背景图提示词模板失败"))
		return
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, h.toBackgroundPromptItem(r.Context(), item))
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]any{
		"items":  result,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}))
}

// Get 返回背景图提示词模板详情。
func (h *BackgroundPromptHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景图提示词模板 ID"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "背景图提示词模板不存在"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toBackgroundPromptItem(r.Context(), item)))
}

// Create 创建背景图提示词模板。
func (h *BackgroundPromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                   string `json:"name"`
		GeminiPrompt           string `json:"gemini_prompt"`
		GeminiNegativePrompt   string `json:"gemini_negative_prompt"`
		GeminiPromptZH         string `json:"gemini_prompt_zh"`
		GeminiNegativePromptZH string `json:"gemini_negative_prompt_zh"`
		WanPrompt              string `json:"wan_prompt"`
		WanNegativePrompt      string `json:"wan_negative_prompt"`
		WanPromptZH            string `json:"wan_prompt_zh"`
		WanNegativePromptZH    string `json:"wan_negative_prompt_zh"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "name 不能为空"))
		return
	}
	if strings.TrimSpace(req.GeminiPrompt) == "" && strings.TrimSpace(req.WanPrompt) == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "gemini_prompt 和 wan_prompt 不能同时为空"))
		return
	}
	if _, err := h.repo.GetByName(req.Name); err == nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "同名背景图提示词模板已存在"))
		return
	}

	id, err := h.repo.Create(
		req.Name,
		req.GeminiPrompt,
		req.GeminiNegativePrompt,
		req.GeminiPromptZH,
		req.GeminiNegativePromptZH,
		req.WanPrompt,
		req.WanNegativePrompt,
		req.WanPromptZH,
		req.WanNegativePromptZH,
		"",
		"",
		0,
		0,
	)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "创建背景图提示词模板失败"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取新建背景图提示词模板失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toBackgroundPromptItem(r.Context(), item)))
}

// Update 更新背景图提示词模板。
func (h *BackgroundPromptHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, parseErr := parseIDFromVars(r)
	if parseErr != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景图提示词模板 ID"))
		return
	}

	var req struct {
		Name                   *string `json:"name"`
		GeminiPrompt           *string `json:"gemini_prompt"`
		GeminiNegativePrompt   *string `json:"gemini_negative_prompt"`
		GeminiPromptZH         *string `json:"gemini_prompt_zh"`
		GeminiNegativePromptZH *string `json:"gemini_negative_prompt_zh"`
		WanPrompt              *string `json:"wan_prompt"`
		WanNegativePrompt      *string `json:"wan_negative_prompt"`
		WanPromptZH            *string `json:"wan_prompt_zh"`
		WanNegativePromptZH    *string `json:"wan_negative_prompt_zh"`
	}
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "name 不能为空"))
			return
		}
		req.Name = &trimmed
	}

	if req.Name != nil {
		existing, lookupErr := h.repo.GetByName(*req.Name)
		if lookupErr == nil && existing.ID != id {
			httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "同名背景图提示词模板已存在"))
			return
		}
	}

	patch := db.BackgroundPromptPatch{
		Name:                   req.Name,
		GeminiPrompt:           req.GeminiPrompt,
		GeminiNegativePrompt:   req.GeminiNegativePrompt,
		GeminiPromptZH:         req.GeminiPromptZH,
		GeminiNegativePromptZH: req.GeminiNegativePromptZH,
		WanPrompt:              req.WanPrompt,
		WanNegativePrompt:      req.WanNegativePrompt,
		WanPromptZH:            req.WanPromptZH,
		WanNegativePromptZH:    req.WanNegativePromptZH,
	}
	if err := h.repo.Update(id, patch); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "更新背景图提示词模板失败"))
		return
	}

	item, getErr := h.repo.GetByID(id)
	if getErr != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取更新后的背景图提示词模板失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toBackgroundPromptItem(r.Context(), item)))
}

// Delete 删除背景图提示词模板。
func (h *BackgroundPromptHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景图提示词模板 ID"))
		return
	}

	if err := h.repo.Delete(id); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "删除背景图提示词模板失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]any{"deleted": id}))
}

// SuggestDefaults 基于背景图生成 Gemini/Wan 的默认提示词建议。
func (h *BackgroundPromptHandler) SuggestDefaults(w http.ResponseWriter, r *http.Request) {
	if h.qwenAdvisor == nil || h.storageService == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "Qwen 建议服务未初始化"))
		return
	}

	var req struct {
		Background        string `json:"background"`
		BackgroundAssetID string `json:"background_asset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	backgroundAssetID, cleanup, err := h.resolveBackgroundAssetID(r.Context(), req.Background, req.BackgroundAssetID)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, err.Error()))
		return
	}
	defer cleanup()

	result, err := h.qwenAdvisor.SuggestFromAsset(r.Context(), backgroundAssetID)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "生成背景图提示词建议失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

// Import 导入背景图并自动生成提示词。
func (h *BackgroundPromptHandler) Import(w http.ResponseWriter, r *http.Request) {
	if h.qwenAdvisor == nil || h.storageService == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "Qwen 建议服务未初始化"))
		return
	}

	var req struct {
		Image string `json:"image"` // Base64 编码的图像
		Name  string `json:"name"`  // 可选，名称
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "image 不能为空"))
		return
	}

	data, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "image 不是合法的 Base64"))
		return
	}
	imageWidth, imageHeight, err := decodeImageSize(data)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "image 不是合法的图片数据"))
		return
	}

	// 1. 保存图像到存储
	ctx := r.Context()
	logger.Info("开始导入背景图, 大小: %d bytes, 文件名: %s, width=%d, height=%d", len(data), req.Name, imageWidth, imageHeight)
	assetID, err := h.storageService.SaveWithKind(ctx, data, fmt.Sprintf("imported_bg_%d.png", time.Now().UnixNano()), storage.SaveKindAsset)
	if err != nil {
		logger.Error("保存背景图到存储失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "保存背景图失败"))
		return
	}
	logger.Info("保存背景图成功, assetID: %s", assetID)

	imageURL, err := h.storageService.GetImageURL(ctx, assetID)
	if err != nil {
		logger.Error("获取背景图URL失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取背景图URL失败"))
		return
	}
	logger.Info("获取背景图URL成功, URL: %s", imageURL)

	// 2. 调用 Qwen 建议提示词
	logger.Info("开始调用 Qwen 建议提示词, assetID: %s", assetID)
	result, err := h.qwenAdvisor.SuggestFromAsset(ctx, assetID)
	if err != nil {
		logger.Error("Qwen 生成背景图提示词失败: %v", err)
		// 记录错误，但可以继续创建空的提示词
		// 这里选择直接返回错误，保证导入体验完整
		_ = h.storageService.Delete(context.Background(), assetID)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "生成背景图提示词失败: "+err.Error()))
		return
	}
	logger.Info("Qwen 建议提示词成功")

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("Imported-%s", time.Now().Format("20060102-150405"))
	}

	// 3. 保存到数据库
	logger.Info("开始将背景图提示词保存到数据库, name: %s", name)
	id, err := h.repo.Create(
		name,
		result.GeminiPromptEN,
		result.GeminiNegativePromptEN,
		result.GeminiPromptZH,
		result.GeminiNegativePromptZH,
		result.WanPromptEN,
		result.WanNegativePromptEN,
		result.WanPromptZH,
		result.WanNegativePromptZH,
		assetID,
		"", // 不在数据库中保存临时 OSS URL，读取时动态获取
		imageWidth,
		imageHeight,
	)
	if err != nil {
		logger.Error("保存背景图提示词到数据库失败: %v", err)
		_ = h.storageService.Delete(context.Background(), assetID)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "保存背景图记录失败"))
		return
	}
	logger.Info("保存背景图提示词到数据库成功, id: %d", id)

	item, err := h.repo.GetByID(id)
	if err != nil {
		logger.Error("获取新建背景图失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取新建背景图失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toBackgroundPromptItem(r.Context(), item)))
}

// SyncEnglish 根据中文提示词同步英文版本。
func (h *BackgroundPromptHandler) SyncEnglish(w http.ResponseWriter, r *http.Request) {
	if h.qwenAdvisor == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "Qwen 建议服务未初始化"))
		return
	}

	var req qwen.SyncEnglishInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	if strings.TrimSpace(req.GeminiPromptZH) == "" &&
		strings.TrimSpace(req.GeminiNegativePromptZH) == "" &&
		strings.TrimSpace(req.WanPromptZH) == "" &&
		strings.TrimSpace(req.WanNegativePromptZH) == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "至少需要提供一项中文提示词"))
		return
	}

	result, err := h.qwenAdvisor.SyncEnglish(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "同步英文提示词失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *BackgroundPromptHandler) toBackgroundPromptItem(ctx context.Context, item *db.BackgroundPrompt) map[string]any {
	imageURL := item.ImageURL
	if item.ImageAssetID != "" && h.storageService != nil {
		if url, err := h.storageService.GetImageURL(ctx, item.ImageAssetID); err == nil && url != "" {
			imageURL = url
		}
	}

	return map[string]any{
		"id":                        item.ID,
		"name":                      item.Name,
		"gemini_prompt":             item.GeminiPrompt,
		"gemini_negative_prompt":    item.GeminiNegativePrompt,
		"gemini_prompt_zh":          item.GeminiPromptZH,
		"gemini_negative_prompt_zh": item.GeminiNegativePromptZH,
		"wan_prompt":                item.WanPrompt,
		"wan_negative_prompt":       item.WanNegativePrompt,
		"wan_prompt_zh":             item.WanPromptZH,
		"wan_negative_prompt_zh":    item.WanNegativePromptZH,
		"image_asset_id":            item.ImageAssetID,
		"image_url":                 imageURL,
		"image_width":               item.ImageWidth,
		"image_height":              item.ImageHeight,
		"created_at":                item.CreatedAt.Format(time.RFC3339),
		"updated_at":                item.UpdatedAt.Format(time.RFC3339),
	}
}

func decodeImageSize(data []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func (h *BackgroundPromptHandler) resolveBackgroundAssetID(ctx context.Context, backgroundBase64 string, backgroundAssetID string) (string, func(), error) {
	if strings.TrimSpace(backgroundAssetID) != "" {
		return strings.TrimSpace(backgroundAssetID), func() {}, nil
	}
	if strings.TrimSpace(backgroundBase64) == "" {
		return "", nil, fmt.Errorf("background 或 background_asset_id 必须提供一个")
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(backgroundBase64))
	if err != nil {
		return "", nil, fmt.Errorf("background 不是合法的 Base64")
	}

	assetID, err := h.storageService.Save(ctx, data, fmt.Sprintf("background_suggest_%d.png", time.Now().UnixNano()))
	if err != nil {
		return "", nil, fmt.Errorf("保存背景图失败")
	}

	cleanup := func() {
		_ = h.storageService.Delete(context.Background(), assetID)
	}
	return assetID, cleanup, nil
}
