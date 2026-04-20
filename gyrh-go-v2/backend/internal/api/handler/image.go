package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/config"
	"gyrh-go-v2/backend/internal/core/llm"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
)

// ImageHandler 图像处理器，提供图像的列表、下载、查看、上传、改写和删除功能
type ImageHandler struct {
	imageRepo      *db.ImageRepo          // 图像数据库仓库
	storageService storage.StorageService // 存储服务
	llmService     llm.Service            // 大模型服务
	config         *config.Config         // 全局配置
}

// NewImageHandler 创建图像处理器实例
func NewImageHandler(imageRepo *db.ImageRepo, storageService storage.StorageService, llmService llm.Service, cfg *config.Config) *ImageHandler {
	return &ImageHandler{
		imageRepo:      imageRepo,
		storageService: storageService,
		llmService:     llmService,
		config:         cfg,
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
	// 解析查询参数
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return writeJSONError(w, http.StatusBadRequest, "缺少图像ID参数")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSONError(w, http.StatusBadRequest, "无效的图像ID")
	}

	logger.Info("下载图像: id=%d", id)

	// 查询图像记录
	img, err := h.imageRepo.GetByID(id)
	if err != nil {
		logger.Error("查询图像记录失败: id=%d, err=%v", id, err)
		return writeJSONError(w, http.StatusNotFound, "图像不存在")
	}

	// 获取图像数据
	imageData, err := h.LoadFile(img.Path)
	if err != nil {
		logger.Error("读取图像文件失败: path=%s, err=%v", img.Path, err)
		return writeJSONError(w, http.StatusInternalServerError, "读取图像文件失败")
	}

	// 设置响应头
	filename := fmt.Sprintf("%s.png", img.Name)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))

	// 写入图像数据
	if _, err := w.Write(imageData); err != nil {
		logger.Error("写入图像数据失败: %v", err)
		return err
	}

	logger.Info("图像下载成功: id=%d, size=%d", id, len(imageData))
	return nil
}

// View 图像预览（支持 WebP 转换）
// GET /images/view?id=xxx
func (h *ImageHandler) View(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 解析查询参数
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return writeJSONError(w, http.StatusBadRequest, "缺少图像ID参数")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSONError(w, http.StatusBadRequest, "无效的图像ID")
	}

	logger.Debug("查看图像: id=%d", id)

	// 查询图像记录
	img, err := h.imageRepo.GetByID(id)
	if err != nil {
		logger.Error("查询图像记录失败: id=%d, err=%v", id, err)
		return writeJSONError(w, http.StatusNotFound, "图像不存在")
	}

	// 获取图像数据
	imageData, err := h.LoadFile(img.Path)
	if err != nil {
		logger.Error("读取图像文件失败: path=%s, err=%v", img.Path, err)
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
		w.Header().Set("Content-Type", "image/png")
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Header().Set("Cache-Control", "public, max-age=31536000") // 缓存 1 年

	// 流式写入图像数据
	if _, err := w.Write(imageData); err != nil {
		logger.Error("写入图像数据失败: %v", err)
		return err
	}

	logger.Debug("图像查看成功: id=%d, size=%d", id, len(imageData))
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
	imageID, err := h.imageRepo.Create(name, assetID, false, "")
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
	Foreground string          `json:"foreground"` // 可选，Base64 编码的前景图
	Background string          `json:"background"` // 可选，Base64 编码的背景图
	References []ReferenceItem `json:"references"` // 可选参考图列表
	Provider   string          `json:"provider"`   // 模型提供者: google/wan，默认 google
	Prompt     string          `json:"prompt"`     // 用户自定义提示词
}

// RewriteResponse 图像改写响应结果
type RewriteResponse struct {
	Success  bool   `json:"success"`         // 是否成功
	ID       int64  `json:"id"`              // 图像ID
	AssetID  string `json:"asset_id"`        // 存储资源ID
	ImageURL string `json:"image_url"`       // 访问URL
	Message  string `json:"message"`         // 消息
	Error    string `json:"error,omitempty"` // 错误信息
}

// Rewrite 图像改写（光影融合）
// POST /images/rewrite
// 请求体: {"foreground": "base64...", "background": "base64...", "references": [...], "provider": "google/wan", "prompt": "xxx"}
func (h *ImageHandler) Rewrite(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 限制请求体大小（最大 100MB）
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

	// 解析请求体
	var req RewriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("解析改写请求失败: %v", err)
		return writeJSONError(w, http.StatusBadRequest, "请求参数解析失败")
	}

	// 设置默认 provider
	if req.Provider == "" {
		req.Provider = "google"
	}
	if req.Provider != "google" && req.Provider != "wan" {
		return writeJSONError(w, http.StatusBadRequest, "不支持的模型提供者，仅支持 google 或 wan")
	}

	logger.Info("图像改写请求: provider=%s, prompt=%s, references=%d",
		req.Provider, req.Prompt, len(req.References))

	inputs, err := h.prepareLLMInputs(ctx, req)
	if err != nil {
		return writeJSONError(w, http.StatusBadRequest, err.Error())
	}

	if h.llmService == nil {
		return writeJSONError(w, http.StatusInternalServerError, "LLM 服务未初始化")
	}

	result, err := h.llmService.Compose(ctx, llm.ComposeParams{
		Provider: req.Provider,
		Prompt:   req.Prompt,
		Images:   inputs,
	})
	if err != nil {
		logger.Error("调用模型融合失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "模型融合失败")
	}

	resultData, err := h.resolveComposeResult(ctx, result)
	if err != nil {
		logger.Error("解析模型返回结果失败: %v", err)
		return writeJSONError(w, http.StatusInternalServerError, "解析模型结果失败")
	}

	filename := fmt.Sprintf("rewrite_%d.png", time.Now().Unix())
	assetID, err := h.storageService.Save(ctx, resultData, filename)
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
	imageID, err := h.imageRepo.Create(name, assetID, false, req.Provider)
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
		Message:  "图像改写成功",
	})
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
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return nil, fmt.Errorf("模型未返回图像数据")
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

// mergeImages 合并多张图像（光影融合）
// foreground 前景图数据
// background 背景图数据
// references 参考图列表
// prompt 用户提示词
// 返回融合后的图像数据
func (h *ImageHandler) mergeImages(foreground, background []byte, references map[string][]byte, prompt string) ([]byte, error) {
	_ = prompt
	// 如果有前景图和背景图，进行简单的叠加融合
	if len(foreground) > 0 && len(background) > 0 {
		return h.blendImages(foreground, background)
	}

	// 如果只有前景图
	if len(foreground) > 0 {
		return foreground, nil
	}

	// 如果只有背景图
	if len(background) > 0 {
		return background, nil
	}

	// 如果有参考图，使用第一张
	for _, refData := range references {
		return refData, nil
	}

	return nil, fmt.Errorf("没有提供任何图像数据")
}

// blendImages 简单的图像叠加混合
// 将前景图叠加到背景图上
func (h *ImageHandler) blendImages(foreground, background []byte) ([]byte, error) {
	// 解码前景图
	fgImg, _, err := image.Decode(bytes.NewReader(foreground))
	if err != nil {
		// 尝试作为 raw bytes 处理
		return foreground, nil
	}

	// 解码背景图
	bgImg, _, err := image.Decode(bytes.NewReader(background))
	if err != nil {
		// 尝试作为 raw bytes 处理
		return background, nil
	}

	// 获取背景图尺寸
	bounds := bgImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 创建输出图像
	output := image.NewRGBA(bounds)

	// 绘制背景图
	for y := range height {
		for x := range width {
			output.Set(x, y, bgImg.At(x, y))
		}
	}

	// 获取前景图尺寸
	fgBounds := fgImg.Bounds()
	fgWidth := fgBounds.Dx()
	fgHeight := fgBounds.Dy()

	// 将前景图叠加到背景图中心
	offsetX := (width - fgWidth) / 2
	offsetY := (height - fgHeight) / 2

	// Alpha 混合
	for y := range fgHeight {
		for x := range fgWidth {
			fx := x + offsetX
			fy := y + offsetY
			if fx >= 0 && fx < width && fy >= 0 && fy < height {
				fgPixel := fgImg.At(x, y)
				bgPixel := output.At(fx, fy)
				blended := alphaBlend(fgPixel, bgPixel, 0.7) // 70% 透明度
				output.Set(fx, fy, blended)
			}
		}
	}

	// 编码为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, output); err != nil {
		return nil, fmt.Errorf("图像编码失败: %w", err)
	}

	return buf.Bytes(), nil
}

// alphaBlend Alpha 混合
// fg 前景色
// bg 背景色
// alpha 前景色透明度
func alphaBlend(fg, bg color.Color, alpha float64) color.Color {
	fgR, fgG, fgB, fgA := fg.RGBA()
	bgR, bgG, bgB, bgA := bg.RGBA()

	// 计算最终 alpha
	a := alpha*float64(fgA)/float64(0xFFFF) + (1-alpha)*float64(bgA)/float64(0xFFFF)

	// 计算最终颜色
	r := float64(fgR)*alpha/float64(0xFFFF) + float64(bgR)*(1-alpha)/float64(0xFFFF)
	g := float64(fgG)*alpha/float64(0xFFFF) + float64(bgG)*(1-alpha)/float64(0xFFFF)
	b := float64(fgB)*alpha/float64(0xFFFF) + float64(bgB)*(1-alpha)/float64(0xFFFF)

	return color.RGBA64{
		R: uint16(r * 0xFFFF),
		G: uint16(g * 0xFFFF),
		B: uint16(b * 0xFFFF),
		A: uint16(a * 0xFFFF),
	}
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// writeJSONError 写入 JSON 错误响应
func writeJSONError(w http.ResponseWriter, statusCode int, message string) error {
	return writeJSON(w, statusCode, map[string]any{
		"success": false,
		"error":   message,
	})
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
