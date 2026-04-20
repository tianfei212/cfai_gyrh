package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/storage"
	"gyrh-go-v2/backend/pkg/httpx"
)

// ReferenceHandler 提供参考图 CRUD 能力。
type ReferenceHandler struct {
	repo    *db.ReferenceRepo
	storage storage.StorageService
}

// NewReferenceHandler 创建参考图处理器。
func NewReferenceHandler(repo *db.ReferenceRepo, storageService storage.StorageService) *ReferenceHandler {
	return &ReferenceHandler{repo: repo, storage: storageService}
}

// List 返回参考图列表。
func (h *ReferenceHandler) List(w http.ResponseWriter, r *http.Request) {
	imageType := r.URL.Query().Get("type")
	limit, offset := parsePage(r)

	var (
		items []*db.ReferenceImage
		err   error
		total int64
	)
	if imageType != "" {
		items, err = h.repo.ListByType(imageType, limit)
		total, _ = h.repo.Count(imageType)
	} else {
		items, err = h.repo.List("", limit, offset)
		total, _ = h.repo.Count("")
	}
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询参考图失败"))
		return
	}

	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		viewURL, _ := h.storage.GetImageURL(r.Context(), item.Path)
		result = append(result, map[string]interface{}{
			"id":         item.ID,
			"name":       item.Name,
			"path":       item.Path,
			"image_type": item.ImageType,
			"view_url":   viewURL,
			"created_at": item.CreatedAt.Format(time.RFC3339),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"items":  result,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}))
}

// View 返回参考图访问地址。
func (h *ReferenceHandler) View(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的参考图 ID"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "参考图不存在"))
		return
	}

	viewURL, err := h.storage.GetImageURL(r.Context(), item.Path)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取参考图地址失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"id":         item.ID,
		"name":       item.Name,
		"image_type": item.ImageType,
		"view_url":   viewURL,
	}))
}

// Upload 上传参考图。
func (h *ReferenceHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "上传体解析失败"))
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "缺少 image 文件"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "读取上传内容失败"))
		return
	}

	imageType := r.FormValue("type")
	if !validReferenceType(imageType) {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "不支持的参考图类型"))
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	assetID, err := h.storage.Save(r.Context(), data, header.Filename)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "保存参考图失败"))
		return
	}

	id, err := h.repo.Create(name, assetID, imageType)
	if err != nil {
		_ = h.storage.Delete(r.Context(), assetID)
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "写入参考图记录失败"))
		return
	}

	viewURL, _ := h.storage.GetImageURL(r.Context(), assetID)
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"id":         id,
		"name":       name,
		"image_type": imageType,
		"path":       assetID,
		"view_url":   viewURL,
	}))
}

// Update 更新参考图元信息。
func (h *ReferenceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的参考图 ID"))
		return
	}

	var req struct {
		Name      string `json:"name"`
		ImageType string `json:"image_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}
	if req.ImageType != "" && !validReferenceType(req.ImageType) {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "不支持的参考图类型"))
		return
	}

	if err := h.repo.Update(id, req.Name, "", req.ImageType); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "更新参考图失败"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取更新后的参考图失败"))
		return
	}

	viewURL, _ := h.storage.GetImageURL(r.Context(), item.Path)
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"id":         item.ID,
		"name":       item.Name,
		"image_type": item.ImageType,
		"path":       item.Path,
		"view_url":   viewURL,
	}))
}

// Delete 删除参考图。
func (h *ReferenceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的参考图 ID"))
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "参考图不存在"))
		return
	}

	if err := h.repo.Delete(id); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "删除参考图失败"))
		return
	}
	_ = h.storage.Delete(r.Context(), item.Path)

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{"deleted": id}))
}

func validReferenceType(value string) bool {
	switch value {
	case "upper", "lower", "dress", "accessory", "headwear", "footwear":
		return true
	default:
		return false
	}
}
