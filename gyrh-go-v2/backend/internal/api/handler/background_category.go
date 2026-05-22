package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/pkg/httpx"
)

// BackgroundCategoryHandler 提供背景图提示词分类 CRUD 能力。
type BackgroundCategoryHandler struct {
	repo *db.BackgroundCategoryRepo
}

// NewBackgroundCategoryHandler 创建背景图提示词分类处理器。
func NewBackgroundCategoryHandler(repo *db.BackgroundCategoryRepo) *BackgroundCategoryHandler {
	return &BackgroundCategoryHandler{repo: repo}
}

type backgroundCategoryRequest struct {
	ParentName string `json:"parent_name"`
	ChildName  string `json:"child_name"`
}

// List 返回背景图提示词分类列表。
func (h *BackgroundCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List()
	if err != nil {
		logger.Error("查询背景分类列表失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询背景分类列表失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(items))
}

// Create 创建背景图提示词分类。
func (h *BackgroundCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeBackgroundCategoryRequest(w, r)
	if !ok {
		return
	}

	id, err := h.repo.Create(req.ParentName, req.ChildName)
	if err != nil {
		logger.Error("创建背景分类失败: %v", err)
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "创建背景分类失败"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		logger.Error("获取新建背景分类失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取新建背景分类失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

// Update 更新背景图提示词分类。
func (h *BackgroundCategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景分类 ID"))
		return
	}

	req, ok := decodeBackgroundCategoryRequest(w, r)
	if !ok {
		return
	}

	if err := h.repo.Update(id, req.ParentName, req.ChildName); err != nil {
		logger.Error("更新背景分类失败: %v", err)
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "更新背景分类失败"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		logger.Error("获取更新后背景分类失败: %v", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取更新后背景分类失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

// Delete 删除背景图提示词分类。
func (h *BackgroundCategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的背景分类 ID"))
		return
	}

	if err := h.repo.Delete(id); err != nil {
		logger.Error("删除背景分类失败: %v", err)
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "删除背景分类失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]any{
		"success": true,
	}))
}

func decodeBackgroundCategoryRequest(w http.ResponseWriter, r *http.Request) (backgroundCategoryRequest, bool) {
	var req backgroundCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return req, false
	}

	req.ParentName = strings.TrimSpace(req.ParentName)
	req.ChildName = strings.TrimSpace(req.ChildName)
	if req.ParentName == "" || req.ChildName == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "parent_name 和 child_name 不能为空"))
		return req, false
	}

	return req, true
}
