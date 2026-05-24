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
	"net/url"
	"strconv"
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
	repo               *db.BackgroundPromptRepo
	storageService     storage.StorageService
	qwenAdvisor        qwen.Advisor
	galleryExternalURL string
	categoryRepo       *db.BackgroundCategoryRepo
}

// NewBackgroundPromptHandler 创建背景图提示词处理器。
// galleryExternalURL 来自配置 gallery.external_url；当同步请求未带 api_url 时用作默认远端图库接口地址。
func NewBackgroundPromptHandler(repo *db.BackgroundPromptRepo, storageService storage.StorageService, qwenAdvisor qwen.Advisor, galleryExternalURL string, categoryRepo *db.BackgroundCategoryRepo) *BackgroundPromptHandler {
	return &BackgroundPromptHandler{
		repo:               repo,
		storageService:     storageService,
		qwenAdvisor:        qwenAdvisor,
		galleryExternalURL: strings.TrimSpace(galleryExternalURL),
		categoryRepo:       categoryRepo,
	}
}

type remoteSyncRequest struct {
	APIURL string `json:"api_url"`
}

type remoteMediaResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []remoteMediaItem `json:"items"`
	} `json:"data"`
}

type remoteMediaItem struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Dimensions string `json:"dimensions"`
	Prompt     string `json:"prompt"`
}

type remoteSyncFailure struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type remoteSyncResult struct {
	Imported int                 `json:"imported"`
	Skipped  int                 `json:"skipped"`
	Failed   int                 `json:"failed"`
	Failures []remoteSyncFailure `json:"failures"`
}

func parseOptionalInt64Query(r *http.Request, key string) (int64, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return parsed, nil
}

// List 返回背景图提示词模板列表。
func (h *BackgroundPromptHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePage(r)

	categoryID, err := parseOptionalInt64Query(r, "category_id")
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景分类 ID"))
		return
	}

	var items []*db.BackgroundPrompt
	if categoryID > 0 {
		items, err = h.repo.ListByCategory(categoryID, limit, offset)
	} else {
		items, err = h.repo.List(limit, offset)
	}
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询背景图提示词模板失败"))
		return
	}

	var total int64
	if categoryID > 0 {
		total, err = h.repo.CountByCategory(categoryID)
	} else {
		total, err = h.repo.Count()
	}
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "统计背景图提示词模板失败"))
		return
	}

	categoriesByBackgroundID, err := h.listCategoriesByBackgroundIDs(items)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询背景图提示词分类失败"))
		return
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, h.toBackgroundPromptItemWithCategories(r.Context(), item, categoriesByBackgroundID[item.ID]))
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

// ListCategories 返回背景图提示词模板绑定的分类列表。
func (h *BackgroundPromptHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景图提示词模板 ID"))
		return
	}
	if h.categoryRepo == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "背景分类服务未初始化"))
		return
	}

	if _, err := h.repo.GetByID(id); err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "背景图提示词模板不存在"))
		return
	}

	items, err := h.categoryRepo.ListByBackgroundID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询背景图提示词分类失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(items))
}

// UpdateCategories 替换背景图提示词模板绑定的分类列表。
func (h *BackgroundPromptHandler) UpdateCategories(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景图提示词模板 ID"))
		return
	}
	if h.categoryRepo == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "背景分类服务未初始化"))
		return
	}

	var req struct {
		CategoryIDs []int64 `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	if _, err := h.repo.GetByID(id); err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "背景图提示词模板不存在"))
		return
	}

	if err := h.categoryRepo.ReplaceBackgroundBindings(id, req.CategoryIDs); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "更新背景图提示词分类失败"))
		return
	}

	items, err := h.categoryRepo.ListByBackgroundID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询背景图提示词分类失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(items))
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
		GPTPrompt              string `json:"gpt_prompt"`
		GPTNegativePrompt      string `json:"gpt_negative_prompt"`
		GPTPromptZH            string `json:"gpt_prompt_zh"`
		GPTNegativePromptZH    string `json:"gpt_negative_prompt_zh"`
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
	if strings.TrimSpace(req.GeminiPrompt) == "" && strings.TrimSpace(req.WanPrompt) == "" && strings.TrimSpace(req.GPTPrompt) == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "gemini_prompt、wan_prompt 和 gpt_prompt 不能同时为空"))
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
		req.GPTPrompt,
		req.GPTNegativePrompt,
		req.GPTPromptZH,
		req.GPTNegativePromptZH,
		"",
		"",
		0,
		0,
	)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "创建背景图提示词模板失败"))
		return
	}
	if err := h.ensureDefaultCategoryBinding(id); err != nil {
		h.deletePromptAfterBindingFailure(id)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "绑定默认背景分类失败"))
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
		GPTPrompt              *string `json:"gpt_prompt"`
		GPTNegativePrompt      *string `json:"gpt_negative_prompt"`
		GPTPromptZH            *string `json:"gpt_prompt_zh"`
		GPTNegativePromptZH    *string `json:"gpt_negative_prompt_zh"`
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
		GPTPrompt:              req.GPTPrompt,
		GPTNegativePrompt:      req.GPTNegativePrompt,
		GPTPromptZH:            req.GPTPromptZH,
		GPTNegativePromptZH:    req.GPTNegativePromptZH,
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
		result.GPTPromptEN,
		result.GPTNegativePromptEN,
		result.GPTPromptZH,
		result.GPTNegativePromptZH,
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
	if err := h.ensureDefaultCategoryBinding(id); err != nil {
		logger.Error("绑定默认背景分类失败: %v", err)
		h.deletePromptAfterBindingFailure(id)
		_ = h.storageService.Delete(context.Background(), assetID)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "绑定默认背景分类失败"))
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

// SyncRemote 同步远端图库当前页背景图，不触发提示词反推。
func (h *BackgroundPromptHandler) SyncRemote(w http.ResponseWriter, r *http.Request) {
	if h.storageService == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "存储服务未初始化"))
		return
	}

	var req remoteSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	apiURL := strings.TrimSpace(req.APIURL)
	if apiURL == "" {
		apiURL = h.galleryExternalURL
	}
	if apiURL == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "未配置远端图库地址：请在 configs/config.yaml 的 gallery.external_url 中填写完整图库接口 URL，或在请求体中提供 api_url"))
		return
	}
	if err := validateRemoteAPIURL(apiURL); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, err.Error()))
		return
	}

	remoteResp, err := fetchRemoteMedia(r.Context(), apiURL)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadGateway, httpx.Error(1, err.Error()))
		return
	}

	result := h.importRemoteMediaItems(r.Context(), apiURL, remoteResp.Data.Items)
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
		strings.TrimSpace(req.WanNegativePromptZH) == "" &&
		strings.TrimSpace(req.GPTPromptZH) == "" &&
		strings.TrimSpace(req.GPTNegativePromptZH) == "" {
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
	imageAssetID := strings.TrimSpace(item.ImageAssetID)
	if imageAssetID != "" {
		if url, err := h.storageService.GetImageURL(ctx, imageAssetID); err == nil && url != "" {
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
		"gpt_prompt":                item.GPTPrompt,
		"gpt_negative_prompt":       item.GPTNegativePrompt,
		"gpt_prompt_zh":             item.GPTPromptZH,
		"gpt_negative_prompt_zh":    item.GPTNegativePromptZH,
		"image_asset_id":            item.ImageAssetID,
		"image_url":                 imageURL,
		"image_proxy_url":           backgroundPromptProxyURL(imageAssetID),
		"image_width":               item.ImageWidth,
		"image_height":              item.ImageHeight,
		"created_at":                item.CreatedAt.Format(time.RFC3339),
		"updated_at":                item.UpdatedAt.Format(time.RFC3339),
	}
}

func backgroundPromptProxyURL(assetID string) string {
	if assetID == "" {
		return ""
	}
	return "/api/v1/images/view?asset_id=" + url.QueryEscape(assetID)
}

func (h *BackgroundPromptHandler) toBackgroundPromptItemWithCategories(ctx context.Context, item *db.BackgroundPrompt, categories []*db.BackgroundCategory) map[string]any {
	result := h.toBackgroundPromptItem(ctx, item)
	if categories == nil {
		categories = []*db.BackgroundCategory{}
	}
	result["categories"] = categories
	return result
}

func (h *BackgroundPromptHandler) listCategoriesByBackgroundIDs(items []*db.BackgroundPrompt) (map[int64][]*db.BackgroundCategory, error) {
	result := make(map[int64][]*db.BackgroundCategory, len(items))
	for _, item := range items {
		result[item.ID] = []*db.BackgroundCategory{}
	}
	if len(items) == 0 || h.categoryRepo == nil {
		return result, nil
	}

	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return h.categoryRepo.ListByBackgroundIDs(ids)
}

func (h *BackgroundPromptHandler) ensureDefaultCategoryBinding(backgroundID int64) error {
	if h.categoryRepo == nil {
		return nil
	}
	return h.categoryRepo.EnsureDefaultBindingForBackground(backgroundID)
}

func (h *BackgroundPromptHandler) deletePromptAfterBindingFailure(backgroundID int64) {
	if err := h.repo.Delete(backgroundID); err != nil {
		logger.Error("清理背景图提示词失败: %v", err)
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

func validateRemoteAPIURL(apiURL string) error {
	if strings.TrimSpace(apiURL) == "" {
		return fmt.Errorf("api_url 不能为空")
	}
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("api_url 不是合法 URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("api_url 只支持 http/https")
	}
	return nil
}

func fetchRemoteMedia(ctx context.Context, apiURL string) (*remoteMediaResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建远端图库请求失败")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求远端图库失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("远端图库状态码异常: %d", resp.StatusCode)
	}

	var remoteResp remoteMediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&remoteResp); err != nil {
		return nil, fmt.Errorf("解析远端图库响应失败")
	}
	return &remoteResp, nil
}

func (h *BackgroundPromptHandler) importRemoteMediaItems(ctx context.Context, apiURL string, items []remoteMediaItem) remoteSyncResult {
	result := remoteSyncResult{
		Failures: []remoteSyncFailure{},
	}

	for _, item := range items {
		remoteID := strings.TrimSpace(item.ID)
		if remoteID == "" {
			result.Failed++
			result.Failures = append(result.Failures, remoteSyncFailure{ID: "", Reason: "远端 id 为空"})
			continue
		}

		name := "remote:" + remoteID
		if _, err := h.repo.GetByName(name); err == nil {
			result.Skipped++
			continue
		}

		if err := h.importRemoteMediaItem(ctx, apiURL, name, item); err != nil {
			result.Failed++
			result.Failures = append(result.Failures, remoteSyncFailure{ID: remoteID, Reason: err.Error()})
			continue
		}
		result.Imported++
	}

	return result
}

func (h *BackgroundPromptHandler) importRemoteMediaItem(ctx context.Context, apiURL string, name string, item remoteMediaItem) error {
	imageURL, err := resolveRemoteMediaURL(apiURL, item.URL)
	if err != nil {
		return err
	}

	imageData, err := downloadRemoteImage(ctx, imageURL)
	if err != nil {
		return err
	}

	imageWidth, imageHeight, decodeErr := decodeImageSize(imageData)
	if decodeErr != nil {
		imageWidth, imageHeight = parseRemoteDimensions(item.Dimensions)
		if imageWidth == 0 || imageHeight == 0 {
			return fmt.Errorf("远端图片不是合法图片数据")
		}
	}

	assetID, err := h.storageService.SaveWithKind(ctx, imageData, fmt.Sprintf("remote_bg_%s.png", sanitizeRemoteID(item.ID)), storage.SaveKindAsset)
	if err != nil {
		return fmt.Errorf("保存背景图失败")
	}

	prompt := strings.TrimSpace(item.Prompt)
	id, err := h.repo.Create(
		name,
		"",
		"",
		prompt,
		"",
		"",
		"",
		prompt,
		"",
		"",
		"",
		prompt,
		"",
		assetID,
		"",
		imageWidth,
		imageHeight,
	)
	if err != nil {
		_ = h.storageService.Delete(context.Background(), assetID)
		return fmt.Errorf("保存背景图记录失败")
	}
	if err := h.ensureDefaultCategoryBinding(id); err != nil {
		h.deletePromptAfterBindingFailure(id)
		_ = h.storageService.Delete(context.Background(), assetID)
		return fmt.Errorf("绑定默认背景分类失败")
	}

	return nil
}

func downloadRemoteImage(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建远端图片请求失败")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载远端图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("远端图片状态码异常: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("读取远端图片失败")
	}
	return buf.Bytes(), nil
}

func resolveRemoteMediaURL(apiURL string, mediaURL string) (string, error) {
	mediaURL = strings.TrimSpace(mediaURL)
	if mediaURL == "" {
		return "", fmt.Errorf("远端图片 URL 为空")
	}

	parsedMedia, err := url.Parse(mediaURL)
	if err != nil {
		return "", fmt.Errorf("远端图片 URL 无效")
	}
	if parsedMedia.IsAbs() {
		if parsedMedia.Scheme != "http" && parsedMedia.Scheme != "https" {
			return "", fmt.Errorf("远端图片 URL 只支持 http/https")
		}
		return parsedMedia.String(), nil
	}

	parsedAPI, err := url.Parse(strings.TrimSpace(apiURL))
	if err != nil || parsedAPI.Scheme == "" || parsedAPI.Host == "" {
		return "", fmt.Errorf("远端 API URL 无效")
	}
	if parsedAPI.Scheme != "http" && parsedAPI.Scheme != "https" {
		return "", fmt.Errorf("远端 API URL 只支持 http/https")
	}

	return parsedAPI.ResolveReference(parsedMedia).String(), nil
}

func parseRemoteDimensions(dimensions string) (int, int) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(dimensions)), "x")
	if len(parts) != 2 {
		return 0, 0
	}

	width, widthErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, heightErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0
	}
	return width, height
}

func sanitizeRemoteID(id string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "?", "_", "&", "_", "=", "_")
	cleaned := strings.TrimSpace(replacer.Replace(id))
	if cleaned == "" {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return cleaned
}
