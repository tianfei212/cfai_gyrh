package handler

import (
	"encoding/json"
	"net/http"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/pkg/httpx"
)

// StylePromptHandler 风格转换提示词处理器
type StylePromptHandler struct {
	repo *db.StylePromptRepo
}

// NewStylePromptHandler 创建处理器
func NewStylePromptHandler(repo *db.StylePromptRepo) *StylePromptHandler {
	return &StylePromptHandler{repo: repo}
}

// List 获取所有风格提示词
func (h *StylePromptHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"
	
	var list []*db.StylePrompt
	var err error
	
	if activeOnly {
		list, err = h.repo.ListActive()
	} else {
		list, err = h.repo.List()
	}

	if err != nil {
		logger.Error("获取风格提示词列表失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取数据失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

// Get 获取单个风格提示词
func (h *StylePromptHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的ID"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "记录不存在"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

// CreateRequest 创建/更新请求参数
type StylePromptRequest struct {
	Name           string `json:"name"`
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt"`
	IsActive       bool   `json:"is_active"`
}

// Create 创建风格提示词
func (h *StylePromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req StylePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的请求参数"))
		return
	}

	if req.Name == "" || req.Prompt == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "名称和提示词不能为空"))
		return
	}

	id, err := h.repo.Create(req.Name, req.Prompt, req.NegativePrompt, req.IsActive)
	if err != nil {
		logger.Error("创建风格提示词失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "创建失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"success": true,
		"id":      id,
	}))
}

// Update 更新风格提示词
func (h *StylePromptHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的ID"))
		return
	}

	var req StylePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的请求参数"))
		return
	}

	if req.Name == "" || req.Prompt == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "名称和提示词不能为空"))
		return
	}

	if err := h.repo.Update(id, req.Name, req.Prompt, req.NegativePrompt, req.IsActive); err != nil {
		logger.Error("更新风格提示词失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "更新失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"success": true,
	}))
}

// Delete 删除风格提示词
func (h *StylePromptHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的ID"))
		return
	}

	if err := h.repo.Delete(id); err != nil {
		logger.Error("删除风格提示词失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "删除失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"success": true,
	}))
}
