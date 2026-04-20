package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/pkg/httpx"
)

// LLMPromptTemplateHandler 提供大模型提示词模板 CRUD 能力。
type LLMPromptTemplateHandler struct {
	repo *db.LLMPromptTemplateRepo
}

// NewLLMPromptTemplateHandler 创建处理器。
func NewLLMPromptTemplateHandler(repo *db.LLMPromptTemplateRepo) *LLMPromptTemplateHandler {
	return &LLMPromptTemplateHandler{repo: repo}
}

// List 返回模板列表。
func (h *LLMPromptTemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePage(r)
	items, err := h.repo.List(limit, offset)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询大模型提示词模板失败"))
		return
	}
	total, err := h.repo.Count()
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "统计大模型提示词模板失败"))
		return
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, toLLMPromptTemplateItem(item))
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]any{
		"items":  result,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}))
}

// Get 返回模板详情。
func (h *LLMPromptTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的大模型提示词模板 ID"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "大模型提示词模板不存在"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toLLMPromptTemplateItem(item)))
}

// Create 创建模板。
func (h *LLMPromptTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		TemplateKey string `json:"template_key"`
		Content     string `json:"content"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.TemplateKey = strings.TrimSpace(req.TemplateKey)
	req.Content = strings.TrimSpace(req.Content)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" || req.TemplateKey == "" || req.Content == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "name、template_key、content 不能为空"))
		return
	}
	if _, err := h.repo.GetByKey(req.TemplateKey); err == nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "同名 template_key 已存在"))
		return
	}

	id, err := h.repo.Create(req.Name, req.TemplateKey, req.Content, req.Description)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "创建大模型提示词模板失败"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取新建模板失败"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toLLMPromptTemplateItem(item)))
}

// Update 更新模板。
func (h *LLMPromptTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, parseErr := parseIDFromVars(r)
	if parseErr != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的大模型提示词模板 ID"))
		return
	}

	var req struct {
		Name        *string `json:"name"`
		TemplateKey *string `json:"template_key"`
		Content     *string `json:"content"`
		Description *string `json:"description"`
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
	if req.TemplateKey != nil {
		trimmed := strings.TrimSpace(*req.TemplateKey)
		if trimmed == "" {
			httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "template_key 不能为空"))
			return
		}
		existing, lookupErr := h.repo.GetByKey(trimmed)
		if lookupErr == nil && existing.ID != id {
			httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "同名 template_key 已存在"))
			return
		}
		req.TemplateKey = &trimmed
	}
	if req.Content != nil {
		trimmed := strings.TrimSpace(*req.Content)
		if trimmed == "" {
			httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "content 不能为空"))
			return
		}
		req.Content = &trimmed
	}
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		req.Description = &trimmed
	}

	if updateErr := h.repo.Update(id, db.LLMPromptTemplatePatch{
		Name:        req.Name,
		TemplateKey: req.TemplateKey,
		Content:     req.Content,
		Description: req.Description,
	}); updateErr != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "更新大模型提示词模板失败"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取更新后的模板失败"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toLLMPromptTemplateItem(item)))
}

// Delete 删除模板。
func (h *LLMPromptTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的大模型提示词模板 ID"))
		return
	}
	if err := h.repo.Delete(id); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "删除大模型提示词模板失败"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]any{"deleted": id}))
}

func toLLMPromptTemplateItem(item *db.LLMPromptTemplate) map[string]any {
	return map[string]any{
		"id":           item.ID,
		"name":         item.Name,
		"template_key": item.TemplateKey,
		"content":      item.Content,
		"description":  item.Description,
		"created_at":   item.CreatedAt.Format(time.RFC3339),
		"updated_at":   item.UpdatedAt.Format(time.RFC3339),
	}
}
