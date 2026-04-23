package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/core/llm"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
	"gyrh-go-v2/backend/pkg/httpx"
)

// ImageHandler 图像处理器，提供图像的列表、下载、查看、上传、改写和删除功能
type ImageHandler struct {
	imageRepo            *db.ImageRepo            // 图像数据库仓库
	backgroundPromptRepo *db.BackgroundPromptRepo // 背景提示词仓库
	storageService       storage.StorageService   // 存储服务
	llmService           llm.Service              // 大模型服务
}

// NewImageHandler 创建图像处理器实例
func NewImageHandler(imageRepo *db.ImageRepo, backgroundPromptRepo *db.BackgroundPromptRepo, storageService storage.StorageService, llmService llm.Service) *ImageHandler {
	return &ImageHandler{
		imageRepo:            imageRepo,
		backgroundPromptRepo: backgroundPromptRepo,
		storageService:       storageService,
		llmService:           llmService,
	}
}

// ListRequest 列表查询请求参数
type ListRequest struct {
	Limit  int `json:"limit"`  // 限制返回数量，0表示不限制
	Offset int `json:"offset"` // 偏移量，用于分页
}

// ListResponse 列表查询响应结果
type ListResponse struct {
	Success bool                 `json:"success"` // 是否成功
	Images  []*db.GeneratedImage `json:"images"`  // 图像列表
	Total   int64                `json:"total"`   // 总数
	Message string               `json:"message"` // 消息
}

// List 获取图像列表
// GET /images
// Query 参数: limit - 限制数量, offset - 偏移量
func (h *ImageHandler) List(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 解析查询参数
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 0
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	logger.Info("获取图像列表: limit=%d, offset=%d", limit, offset)

	// 查询图像列表
	images, err := h.imageRepo.List(limit, offset)
	if err != nil {
		logger.Error("查询图像列表失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "查询图像列表失败")
	}

	// 获取总数
	total, err := h.imageRepo.Count()
	if err != nil {
		logger.Warn("获取图像总数失败: %v", err)
		total = int64(len(images))
	}

	for _, img := range images {
		if img == nil {
			continue
		}
		if img.AssetID == "" {
			img.AssetID = img.Path
		}
		if img.AssetID == "" {
			continue
		}
		imageURL, urlErr := h.storageService.GetImageURL(ctx, img.AssetID)
		if urlErr != nil {
			logger.Warn("生成历史图像访问 URL 失败: id=%d, asset_id=%s, err=%v", img.ID, img.AssetID, urlErr)
			continue
		}
		img.ImageURL = imageURL
	}

	logger.Info("图像列表查询成功: count=%d, total=%d", len(images), total)

	return writeJSON(w, http.StatusOK, ListResponse{
		Success: true,
		Images:  images,
		Total:   total,
		Message: "获取图像列表成功",
	})
}

// DownloadRequest 图像下载请求参数
type DownloadRequest struct {
	ID int64 `json:"id"` // 图像ID
}

// Download 图像下载
// GET /images/download?id=xxx
func (h *ImageHandler) Download(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	target, statusCode, err := h.resolveImageAccessTarget(r)
	if err != nil {
		return writeJSONError(w, statusCode, err.Error())
	}
	logger.Info("下载图像: asset_id=%s", target.AssetID)

	// 获取图像数据
	imageData, err := h.LoadFile(target.AssetID)
	if err != nil {
		logger.Error("读取图像文件失败: asset_id=%s, err=%v", target.AssetID, err)
		return writeJSONError(w, http.StatusInternalServerError, "读取图像文件失败")
	}

	// 设置响应头
	w.Header().Set("Content-Type", http.DetectContentType(imageData))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", target.Filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))

	// 写入图像数据
	if _, err := w.Write(imageData); err != nil {
		logger.Error("写入图像数据失败: %v", err)
		return err
	}

	logger.Info("图像下载成功: asset_id=%s, size=%d", target.AssetID, len(imageData))
	return nil
}

// View 图像预览（支持 WebP 转换）
// GET /images/view?id=xxx
func (h *ImageHandler) View(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	target, statusCode, err := h.resolveImageAccessTarget(r)
	if err != nil {
		return writeJSONError(w, statusCode, err.Error())
	}
	logger.Debug("查看图像: asset_id=%s", target.AssetID)

	// 获取图像数据
	imageData, err := h.LoadFile(target.AssetID)
	if err != nil {
		logger.Error("读取图像文件失败: asset_id=%s, err=%v", target.AssetID, err)
		return writeJSONError(w, http.StatusInternalServerError, "读取图像文件失败")
	}

	// 检查是否支持 WebP 转换（通过 Accept 头判断）
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "image/webp") {
		// 尝试转换为 WebP 格式
		webpData, err := convertToWebP(imageData)
		if err != nil {
			logger.Warn("WebP 转换失败，使用原始格式: err=%v", err)
		} else {
			imageData = webpData
			w.Header().Set("Content-Type", "image/webp")
		}
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", http.DetectContentType(imageData))
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Header().Set("Cache-Control", "public, max-age=31536000") // 缓存 1 年

	// 流式写入图像数据
	if _, err := w.Write(imageData); err != nil {
		logger.Error("写入图像数据失败: %v", err)
		return err
	}

	logger.Debug("图像查看成功: asset_id=%s, size=%d", target.AssetID, len(imageData))
	return nil
}

// Thumbnail 图像缩略图重定向
// GET /images/thumbnail?url=...&w=200&h=80 或 ?asset_id=...&w=200&h=80
func (h *ImageHandler) Thumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	imageURL := r.URL.Query().Get("url")
	assetIDParam := r.URL.Query().Get("asset_id")
	widthStr := r.URL.Query().Get("w")
	heightStr := r.URL.Query().Get("h")

	logger.Debug("========== 缩略图请求(Thumbnail Handler) ==========")
	logger.Debug("原始查询参数 (Raw Query): %s", r.URL.RawQuery)
	logger.Debug("解析的 url 参数: %s", imageURL)
	logger.Debug("解析的 asset_id 参数: %s", assetIDParam)
	logger.Debug("解析的宽度 w: %s, 高度 h: %s", widthStr, heightStr)

	if imageURL == "" && assetIDParam == "" {
		return writeJSONError(w, http.StatusBadRequest, "缺少url或asset_id参数")
	}

	wInt, _ := strconv.Atoi(widthStr)
	hInt, _ := strconv.Atoi(heightStr)

	if wInt <= 0 || hInt <= 0 {
		// 降级回退
		logger.Debug("由于宽度或高度无效(w=%d, h=%d)，降级回退至原图", wInt, hInt)
		if imageURL != "" {
			http.Redirect(w, r, imageURL, http.StatusFound)
		} else {
			url, _ := h.storageService.GetImageURL(ctx, assetIDParam)
			http.Redirect(w, r, url, http.StatusFound)
		}
		return nil
	}

	var assetID string
	var fallbackURL string

	if assetIDParam != "" {
		assetID = assetIDParam
		fallbackURL, _ = h.storageService.GetImageURL(ctx, assetID)
	} else {
		fallbackURL = imageURL
		u, err := url.Parse(imageURL)
		if err != nil {
			logger.Error("解析 imageURL 失败: %v", err)
			http.Redirect(w, r, fallbackURL, http.StatusFound)
			return nil
		}

		fileID := strings.TrimPrefix(u.Path, "/")
		logger.Debug("从 URL 中解析出 fileID: %s", fileID)

		if !strings.HasPrefix(fileID, "images_data/") {
			logger.Debug("fileID 不以 images_data/ 开头，降级回退至原图")
			http.Redirect(w, r, fallbackURL, http.StatusFound)
			return nil
		}

		kind := storage.SaveKindAsset
		if strings.Contains(fileID, "rewrite_") || strings.Contains(fileID, "_sr_4x") || strings.Contains(fileID, "upload_") || strings.Contains(fileID, "generated_") {
			kind = storage.SaveKindGenerated
		}

		if kind == storage.SaveKindGenerated {
			assetID = "generated:" + fileID
		} else {
			assetID = "asset:" + fileID
		}
	}

	logger.Debug("最终解析出的 assetID: %s", assetID)

	thumbnailURL, err := h.storageService.GetThumbnailURL(ctx, assetID, wInt, hInt)
	if err != nil {
		logger.Warn("获取缩略图 URL 失败: %v, 回退到原图", err)
		http.Redirect(w, r, fallbackURL, http.StatusFound)
		return nil
	}

	logger.Debug("生成的缩略图 URL (thumbnailURL): %s", thumbnailURL)
	logger.Debug("==================================================")

	http.Redirect(w, r, thumbnailURL, http.StatusFound)
	return nil
}

// UploadRequest 图像上传请求参数
type UploadRequest struct {
	Image    string `json:"image"`    // Base64 编码的图像数据
	Filename string `json:"filename"` // 文件名（可选）
	Name     string `json:"name"`     // 图像名称（可选）
}

// UploadResponse 图像上传响应结果
type UploadResponse struct {
	Success  bool   `json:"success"`         // 是否成功
	ID       int64  `json:"id"`              // 图像ID
	AssetID  string `json:"asset_id"`        // 存储资源ID
	ImageURL string `json:"image_url"`       // 访问URL
	Message  string `json:"message"`         // 消息
	Error    string `json:"error,omitempty"` // 错误信息
}

// Upload 图像上传
// POST /images/upload
// 请求体: {"image": "base64...", "filename": "xxx.png", "name": "xxx"}
func (h *ImageHandler) Upload(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 限制请求体大小（最大 50MB）
	r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024)

	// 解析请求体
	var req UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("解析上传请求失败: %v", err)
		return writeJSONError(w, http.StatusBadRequest, "请求参数解析失败")
	}

	// 验证图像数据
	if req.Image == "" {
		return writeJSONError(w, http.StatusBadRequest, "图像数据不能为空")
	}

	// 解码 Base64 图像数据
	imageData, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		logger.Error("Base64 解码失败: %v", err)
		return writeJSONError(w, http.StatusBadRequest, "图像数据格式无效")
	}

	// 生成文件名
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("upload_%d.png", time.Now().Unix())
	}

	logger.Info("上传图像: filename=%s, size=%d", filename, len(imageData))

	// 保存到存储服务
	assetID, err := h.storageService.Save(ctx, imageData, filename)
	if err != nil {
		logger.Error("保存图像失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "保存图像失败")
	}

	// 生成访问 URL
	imageURL, err := h.storageService.GetImageURL(ctx, assetID)
	if err != nil {
		logger.Error("生成访问 URL 失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "生成访问 URL 失败")
	}

	// 图像名称
	name := req.Name
	if name == "" {
		name = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	// 保存到数据库
	imageWidth, imageHeight, err := decodeStoredImageSize(imageData)
	if err != nil {
		logger.Warn("解析上传图像尺寸失败: %v", err)
	}

	imageID, err := h.imageRepo.Create(db.GeneratedImageCreateInput{
		Name:        name,
		Path:        assetID,
		AssetID:     assetID,
		IsUpscale:   false,
		Status:      "uploaded",
		ImageWidth:  imageWidth,
		ImageHeight: imageHeight,
	})
	if err != nil {
		logger.Error("创建数据库记录失败: %v", err)
		// 尝试删除已上传的文件
		_ = h.storageService.Delete(ctx, assetID)
		return writeJSONError(w, http.StatusInternalServerError, "创建图像记录失败")
	}

	logger.Info("图像上传成功: id=%d, asset_id=%s", imageID, assetID)

	return writeJSON(w, http.StatusOK, UploadResponse{
		Success:  true,
		ID:       imageID,
		AssetID:  assetID,
		ImageURL: imageURL,
		Message:  "图像上传成功",
	})
}

// ReferenceItem 参考图项
type ReferenceItem struct {
	Type string `json:"type"` // 参考图类型: upper/lower/left/right
	Data string `json:"data"` // Base64 编码的图像数据
}

// RewriteRequest 图像改写请求参数
type RewriteRequest struct {
	Foreground         string          `json:"foreground"`           // 可选，Base64 编码的前景图
	Background         string          `json:"background"`           // 可选，Base64 编码的背景图
	References         []ReferenceItem `json:"references"`           // 可选参考图列表
	Provider           string          `json:"provider"`             // 模型提供者: google/wan，默认 google
	StylePrompt        string          `json:"style_prompt"`         // 可选，仅用于风格转换的控制提示词
	LegacyPrompt       string          `json:"prompt"`               // 兼容旧字段，仅作为 style_prompt 别名读取
	BackgroundPromptID int64           `json:"background_prompt_id"` // 可选，背景图提示词模板 ID
}

// RewriteResponse 图像改写响应结果
type RewriteResponse struct {
	Success  bool   `json:"success"`         // 是否成功
	ID       int64  `json:"id"`              // 图像ID
	AssetID  string `json:"asset_id"`        // 存储资源ID
	ImageURL string `json:"image_url"`       // 访问URL
	Status   string `json:"status"`          // 生成状态
	Message  string `json:"message"`         // 消息
	Error    string `json:"error,omitempty"` // 错误信息
}

// Rewrite 图像改写（光影融合）
// POST /images/rewrite
// 请求体: {"foreground": "base64...", "background": "base64...", "references": [...], "provider": "google/wan", "background_prompt_id": 1, "style_prompt": "黑白线稿风格"}
func (h *ImageHandler) Rewrite(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 限制请求体大小（最大 100MB）
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

	// 解析请求体
	var req RewriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("解析改写请求失败: %v", err)
		return writeJSONError(w, http.StatusBadRequest, "请求参数解析失败")
	}

	// 详细打印前端请求
	logger.Debug("========== 收到前端 Rewrite 请求 ==========")
	logger.Debug("Provider: %s", req.Provider)
	logger.Debug("BackgroundPromptID: %d", req.BackgroundPromptID)
	logger.Debug("StylePrompt: %s", req.StylePrompt)
	logger.Debug("LegacyPrompt: %s", req.LegacyPrompt)
	logger.Debug("Foreground Base64 Length: %d", len(req.Foreground))
	logger.Debug("Background Base64 Length: %d", len(req.Background))
	logger.Debug("References Count: %d", len(req.References))
	logger.Debug("==========================================")

	// 设置默认 provider
	if req.Provider == "" {
		req.Provider = "google"
	}
	if req.Provider != "google" && req.Provider != "wan" {
		return writeJSONError(w, http.StatusBadRequest, "不支持的模型提供者，仅支持 google 或 wan")
	}

	stylePrompt := req.effectiveStylePrompt()
	hasBackground := strings.TrimSpace(req.Background) != ""

	if hasBackground && req.BackgroundPromptID <= 0 {
		return writeJSONError(w, http.StatusBadRequest, "提供背景图时必须同时提供 background_prompt_id")
	}
	if !hasBackground && req.BackgroundPromptID > 0 {
		return writeJSONError(w, http.StatusBadRequest, "未提供背景图时不能传 background_prompt_id")
	}
	if hasBackground && stylePrompt != "" {
		return writeJSONError(w, http.StatusBadRequest, "背景融合场景不支持 style_prompt")
	}

	logger.Info("图像改写请求: provider=%s, background_prompt_id=%d, has_style_prompt=%t, references=%d",
		req.Provider, req.BackgroundPromptID, stylePrompt != "", len(req.References))

	inputs, err := h.prepareLLMInputs(ctx, req)
	if err != nil {
		return writeJSONError(w, http.StatusBadRequest, err.Error())
	}
	if persistErr := h.persistBackgroundImage(req, inputs); persistErr != nil {
		logger.Error("背景图入库失败: %v", persistErr)
		return writeJSONError(w, http.StatusInternalServerError, "背景图入库失败")
	}

	if h.llmService == nil {
		return writeJSONError(w, http.StatusInternalServerError, "LLM 服务未初始化")
	}

	result, err := h.llmService.Compose(ctx, llm.ComposeParams{
		Provider:           req.Provider,
		StylePrompt:        stylePrompt,
		Images:             inputs,
		BackgroundPromptID: req.BackgroundPromptID,
	})
	if err != nil {
		logger.Error("调用模型融合失败: %v", err)
		if errors.Is(err, llm.ErrBackgroundPromptNotFound) {
			return writeJSONError(w, http.StatusBadRequest, "背景图提示词模板不存在")
		}
		return writeJSONError(w, http.StatusInternalServerError, "模型融合失败")
	}

	resultData, err := h.resolveComposeResult(ctx, result)
	if err != nil {
		logger.Error("解析模型返回结果失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "解析模型结果失败")
	}

	filename := fmt.Sprintf("rewrite_%d.png", time.Now().Unix())
	assetID, err := h.storageService.SaveWithKind(ctx, resultData, filename, storage.SaveKindGenerated)
	if err != nil {
		logger.Error("保存改写图像失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "保存改写图像失败")
	}

	// 生成访问 URL
	imageURL, err := h.storageService.GetImageURL(ctx, assetID)
	if err != nil {
		logger.Error("生成访问 URL 失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "生成访问 URL 失败")
	}

	// 保存到数据库
	name := fmt.Sprintf("rewrite_%d", time.Now().Unix())
	imageWidth, imageHeight, err := decodeStoredImageSize(resultData)
	if err != nil {
		logger.Warn("解析生成图像尺寸失败: %v", err)
	}

	imageID, err := h.imageRepo.Create(db.GeneratedImageCreateInput{
		Name:               name,
		Path:               assetID,
		AssetID:            assetID,
		IsUpscale:          false,
		StyleTransform:     req.Provider,
		Provider:           req.Provider,
		Status:             normalizeRewriteStatus(result.Status),
		BackgroundPromptID: req.BackgroundPromptID,
		ImageWidth:         imageWidth,
		ImageHeight:        imageHeight,
	})
	if err != nil {
		logger.Error("创建数据库记录失败: %v", err)
		_ = h.storageService.Delete(ctx, assetID)
		return writeJSONError(w, http.StatusInternalServerError, "创建图像记录失败")
	}

	logger.Info("图像改写成功: id=%d, asset_id=%s, provider=%s", imageID, assetID, req.Provider)

	return writeJSON(w, http.StatusOK, RewriteResponse{
		Success:  true,
		ID:       imageID,
		AssetID:  assetID,
		ImageURL: imageURL,
		Status:   normalizeRewriteStatus(result.Status),
		Message:  "图像改写成功",
	})
}

func (r RewriteRequest) effectiveStylePrompt() string {
	if strings.TrimSpace(r.StylePrompt) != "" {
		return strings.TrimSpace(r.StylePrompt)
	}
	return strings.TrimSpace(r.LegacyPrompt)
}

func (h *ImageHandler) prepareLLMInputs(ctx context.Context, req RewriteRequest) ([]llm.ImageInput, error) {
	inputs := make([]llm.ImageInput, 0, 2+len(req.References))

	if req.Foreground != "" {
		assetID, err := h.saveBase64Asset(ctx, req.Foreground, "foreground")
		if err != nil {
			return nil, fmt.Errorf("前景图无效: %w", err)
		}
		inputs = append(inputs, llm.ImageInput{Type: llm.ImageTypeCharacter, AssetID: assetID})
	}

	if req.Background != "" {
		assetID, err := h.saveBase64Asset(ctx, req.Background, "background")
		if err != nil {
			return nil, fmt.Errorf("背景图无效: %w", err)
		}
		inputs = append(inputs, llm.ImageInput{Type: llm.ImageTypeBackground, AssetID: assetID})
	}

	for _, ref := range req.References {
		imageType, err := mapReferenceType(ref.Type)
		if err != nil {
			return nil, err
		}
		assetID, err := h.saveBase64Asset(ctx, ref.Data, ref.Type)
		if err != nil {
			return nil, fmt.Errorf("参考图 %s 无效: %w", ref.Type, err)
		}
		inputs = append(inputs, llm.ImageInput{Type: imageType, AssetID: assetID})
	}

	if len(inputs) == 0 {
		return nil, fmt.Errorf("至少需要提供一张图片")
	}
	return inputs, nil
}

func (h *ImageHandler) saveBase64Asset(ctx context.Context, raw, prefix string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_%d.png", prefix, time.Now().UnixNano())
	return h.storageService.Save(ctx, data, filename)
}

func (h *ImageHandler) persistBackgroundImage(req RewriteRequest, inputs []llm.ImageInput) error {
	if req.BackgroundPromptID <= 0 || strings.TrimSpace(req.Background) == "" || h.backgroundPromptRepo == nil {
		return nil
	}

	backgroundAssetID := ""
	for _, input := range inputs {
		if input.Type == llm.ImageTypeBackground {
			backgroundAssetID = input.AssetID
			break
		}
	}
	if backgroundAssetID == "" {
		return nil
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.Background))
	if err != nil {
		return fmt.Errorf("解码背景图失败: %w", err)
	}
	imageWidth, imageHeight, err := decodeImageSize(data)
	if err != nil {
		return fmt.Errorf("解析背景图尺寸失败: %w", err)
	}

	patch := db.BackgroundPromptPatch{
		ImageAssetID: &backgroundAssetID,
		ImageWidth:   &imageWidth,
		ImageHeight:  &imageHeight,
	}
	if err := h.backgroundPromptRepo.Update(req.BackgroundPromptID, patch); err != nil {
		return fmt.Errorf("更新背景图记录失败: %w", err)
	}
	logger.Info("背景图已入库: background_prompt_id=%d, asset_id=%s, size=%dx%d",
		req.BackgroundPromptID, backgroundAssetID, imageWidth, imageHeight)
	return nil
}

func (h *ImageHandler) resolveComposeResult(ctx context.Context, result *llm.ComposeResult) ([]byte, error) {
	if result == nil {
		return nil, fmt.Errorf("模型返回为空")
	}
	if result.Base64 != "" {
		return base64.StdEncoding.DecodeString(result.Base64)
	}
	if result.ImageURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, result.ImageURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "image/*")
		resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return nil, fmt.Errorf("下载模型结果失败: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return io.ReadAll(resp.Body)
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return nil, fmt.Errorf("模型未返回图像数据")
}

func decodeStoredImageSize(data []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func mapReferenceType(value string) (llm.ImageType, error) {
	switch value {
	case "upper":
		return llm.ImageTypeUpper, nil
	case "lower":
		return llm.ImageTypeLower, nil
	case "dress":
		return llm.ImageTypeDress, nil
	case "accessory":
		return llm.ImageTypeAccessory, nil
	case "headwear":
		return llm.ImageTypeHeadwear, nil
	case "footwear":
		return llm.ImageTypeFootwear, nil
	default:
		return "", fmt.Errorf("不支持的参考图类型: %s", value)
	}
}

// DeleteRequest 图像删除请求参数
type DeleteRequest struct {
	ID int64 `json:"id"` // 图像ID
}

// DeleteResponse 图像删除响应结果
type DeleteResponse struct {
	Success bool   `json:"success"` // 是否成功
	Message string `json:"message"` // 消息
}

// Delete 图像删除
// DELETE /images?id=xxx
func (h *ImageHandler) Delete(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 解析查询参数
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return writeJSONError(w, http.StatusBadRequest, "缺少图像ID参数")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSONError(w, http.StatusBadRequest, "无效的图像ID")
	}

	logger.Info("删除图像: id=%d", id)

	// 查询图像记录（确保存在）
	img, err := h.imageRepo.GetByID(id)
	if err != nil {
		logger.Error("查询图像记录失败: id=%d, err=%v", id, err)
		return writeJSONError(w, http.StatusNotFound, "图像不存在")
	}

	// 删除存储文件
	if err := h.storageService.Delete(ctx, img.Path); err != nil {
		logger.Warn("删除存储文件失败: path=%s, err=%v", img.Path, err)
		// 继续删除数据库记录
	}

	// 删除数据库记录
	if err := h.imageRepo.Delete(id); err != nil {
		logger.Error("删除数据库记录失败: id=%d, err=%v", id, err)
		return writeJSONError(w, http.StatusInternalServerError, "删除图像记录失败")
	}

	logger.Info("图像删除成功: id=%d", id)

	return writeJSON(w, http.StatusOK, DeleteResponse{
		Success: true,
		Message: "图像删除成功",
	})
}

// convertToWebP 将 PNG 图像转换为 WebP 格式
// 由于标准库不支持 WebP，这里返回原始数据
// 实际项目中可使用 golang.org/x/image/webp 包
func convertToWebP(pngData []byte) ([]byte, error) {
	// 尝试解码 PNG
	_, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("PNG 解码失败: %w", err)
	}

	// 注意：这里返回原始 PNG 数据，因为标准库不支持 WebP 编码
	// 实际项目中需要引入 golang.org/x/image/webp 包
	// 这里使用简单的方案：直接返回原始数据
	// 更好的方案是使用第三方库进行 WebP 编码

	// 如果需要真正的 WebP 支持，可以安装 webp 库：
	// go get golang.org/x/image/webp
	//
	// img, _, _ := image.Decode(bytes.NewReader(pngData))
	// var buf bytes.Buffer
	// if err := webp.Encode(&buf, img, &webp.Options{Quality: 85}); err != nil {
	//     return nil, err
	// }
	// return buf.Bytes(), nil

	return pngData, nil
}

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, statusCode int, data any) error {
	return httpx.WriteJSON(w, statusCode, httpx.Success(data))
}

// writeJSONError 写入 JSON 错误响应
func writeJSONError(w http.ResponseWriter, statusCode int, message string) error {
	return httpx.WriteJSON(w, statusCode, httpx.Error(1, message))
}

type imageAccessTarget struct {
	AssetID  string
	Filename string
}

func (h *ImageHandler) resolveImageAccessTarget(r *http.Request) (*imageAccessTarget, int, error) {
	if assetID := strings.TrimSpace(r.URL.Query().Get("asset_id")); assetID != "" {
		return &imageAccessTarget{
			AssetID:  assetID,
			Filename: filepath.Base(assetID),
		}, 0, nil
	}

	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	if idStr == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("缺少图像ID参数")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("无效的图像ID")
	}

	img, err := h.imageRepo.GetByID(id)
	if err != nil {
		logger.Error("查询图像记录失败: id=%d, err=%v", id, err)
		return nil, http.StatusNotFound, fmt.Errorf("图像不存在")
	}

	filename := fmt.Sprintf("%s%s", img.Name, filepath.Ext(img.Path))
	if filepath.Ext(filename) == "" {
		filename += ".png"
	}

	return &imageAccessTarget{
		AssetID:  img.Path,
		Filename: filename,
	}, 0, nil
}

func normalizeRewriteStatus(status string) string {
	value := strings.TrimSpace(strings.ToLower(status))
	if value == "" {
		return "succeeded"
	}
	return value
}

// LoadFile 加载文件内容
// path 文件路径或 assetID
func (h *ImageHandler) LoadFile(path string) ([]byte, error) {
	return h.storageService.Read(context.Background(), path)
}

// LoadFileFromStorage 从存储服务加载文件
func (h *ImageHandler) LoadFileFromStorage(assetID string) ([]byte, error) {
	// 使用存储服务的方法
	data, err := h.LoadFile(assetID)
	if err == nil {
		return data, nil
	}
	return nil, err
}
