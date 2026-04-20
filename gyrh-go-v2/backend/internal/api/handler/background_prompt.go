package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/core/llm/qwen"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/storage"
	"gyrh-go-v2/backend/pkg/httpx"
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
		result = append(result, toBackgroundPromptItem(item))
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

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toBackgroundPromptItem(item)))
}

// Create 创建背景图提示词模板。
func (h *BackgroundPromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                 string `json:"name"`
		GeminiPrompt         string `json:"gemini_prompt"`
		GeminiNegativePrompt string `json:"gemini_negative_prompt"`
		WanPrompt            string `json:"wan_prompt"`
		WanNegativePrompt    string `json:"wan_negative_prompt"`
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
		req.WanPrompt,
		req.WanNegativePrompt,
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

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toBackgroundPromptItem(item)))
}

// Update 更新背景图提示词模板。
func (h *BackgroundPromptHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, parseErr := parseIDFromVars(r)
	if parseErr != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景图提示词模板 ID"))
		return
	}

	var req struct {
		Name                 *string `json:"name"`
		GeminiPrompt         *string `json:"gemini_prompt"`
		GeminiNegativePrompt *string `json:"gemini_negative_prompt"`
		WanPrompt            *string `json:"wan_prompt"`
		WanNegativePrompt    *string `json:"wan_negative_prompt"`
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
		Name:                 req.Name,
		GeminiPrompt:         req.GeminiPrompt,
		GeminiNegativePrompt: req.GeminiNegativePrompt,
		WanPrompt:            req.WanPrompt,
		WanNegativePrompt:    req.WanNegativePrompt,
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

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toBackgroundPromptItem(item)))
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

func toBackgroundPromptItem(item *db.BackgroundPrompt) map[string]any {
	return map[string]any{
		"id":                     item.ID,
		"name":                   item.Name,
		"gemini_prompt":          item.GeminiPrompt,
		"gemini_negative_prompt": item.GeminiNegativePrompt,
		"wan_prompt":             item.WanPrompt,
		"wan_negative_prompt":    item.WanNegativePrompt,
		"created_at":             item.CreatedAt.Format(time.RFC3339),
		"updated_at":             item.UpdatedAt.Format(time.RFC3339),
	}
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
