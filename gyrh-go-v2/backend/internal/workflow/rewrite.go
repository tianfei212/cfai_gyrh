package workflow

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/oss"
)

// RewriteRequest rewrite 请求参数
type RewriteRequest struct {
	ImageID        int64  `json:"image_id"`        // 图像 ID
	SourceImageURL string `json:"source_image_url"` // 源图像 URL
	Prompt         string `json:"prompt"`          // rewrite 提示词
	ModelProvider  string `json:"model_provider"` // 模型提供者: wan, google
	StyleTransform string `json:"style_transform"` // 风格转换类型
	EnableMatting  bool   `json:"enable_matting"`   // 是否启用本地抠图
}

// RewriteResponse rewrite 响应结果
type RewriteResponse struct {
	Success       bool   `json:"success"`        // 是否成功
	ImageID       int64  `json:"image_id"`        // 生成图像 ID
	ImageURL      string `json:"image_url"`       // 图像访问 URL
	StyleTransform string `json:"style_transform"` // 风格转换类型
	Message       string `json:"message"`         // 消息
	Error         string `json:"error,omitempty"`  // 错误信息
}

// RewriteContext rewrite 工作流上下文数据
type RewriteContext struct {
	Request     *RewriteRequest     // 请求参数
	SourceData  []byte              // 源图像数据
	MattingData []byte              // 抠图后图像数据
	ModelResult *ModelAPIResult     // 模型 API 调用结果
	OssURL      string              // OSS 存储 URL
	DBImageID   int64               // 数据库记录 ID
	OSSClient   oss.OSSClient       // OSS 客户端
	DB          *db.DB              // 数据库连接
}

// ModelAPIResult 模型 API 调用结果
type ModelAPIResult struct {
	ImageURL  string            `json:"image_url"`  // 生成的图像 URL
	ImageData []byte            `json:"image_data"` // 生成的图像数据
	Metadata  map[string]string `json:"metadata"`   // 其他元数据
}

// RewritePipeline rewrite 工作流流水线
type RewritePipeline struct {
	pipeline *Pipeline
	context  *RewriteContext
}

// NewRewritePipeline 创建 rewrite 工作流流水线
func NewRewritePipeline(req *RewriteRequest, ossClient oss.OSSClient, database *db.DB) *RewritePipeline {
	p := &RewritePipeline{
		context: &RewriteContext{
			Request:   req,
			OSSClient: ossClient,
			DB:        database,
		},
	}

	// 构建工作流步骤
	pipeline := NewPipeline("rewrite")

	// Step 1: 本地抠图（可选）
	if req.EnableMatting {
		pipeline.AddStep(p.createMattingStep())
	}

	// Step 2: 调用模型 API
	pipeline.AddStep(p.createModelAPIStep())

	// Step 3: 写入 OSS
	pipeline.AddStep(p.createOssUploadStep())

	// Step 4: 写入 SQLite
	pipeline.AddStep(p.createDBInsertStep())

	// Step 5: 构建响应
	pipeline.AddStep(p.createResponseStep())

	p.pipeline = pipeline
	return p
}

// Execute 执行 rewrite 工作流
func (rp *RewritePipeline) Execute(ctx context.Context) (*RewriteResponse, error) {
	stepCtx := NewStepContext(ctx)

	// 执行流水线
	_, err := rp.pipeline.Execute(stepCtx)
	if err != nil {
		// 构建错误响应
		return &RewriteResponse{
			Success: false,
			Message: "工作流执行失败",
			Error:   err.Error(),
		}, err
	}

	// 构建成功响应
	return &RewriteResponse{
		Success:       true,
		ImageID:       rp.context.DBImageID,
		ImageURL:      rp.context.OssURL,
		StyleTransform: rp.context.Request.StyleTransform,
		Message:       "图像 rewrite 成功",
	}, nil
}

// createMattingStep 创建本地抠图步骤
func (rp *RewritePipeline) createMattingStep() StepInterface {
	return &StepFunc{
		StepBase: NewStepBase("本地抠图"),
		ExecuteFunc: func(ctx *StepContext) (interface{}, error) {
			logger.Info("Step 1: 开始本地抠图处理")

			sourceURL := rp.context.Request.SourceImageURL
			if sourceURL == "" {
				return nil, fmt.Errorf("源图像 URL 为空")
			}

			// 下载源图像
			imageData, err := rp.downloadImage(ctx, sourceURL)
			if err != nil {
				logger.Error("下载源图像失败: %v", err)
				return nil, fmt.Errorf("下载源图像失败: %w", err)
			}

			rp.context.SourceData = imageData
			logger.Info("源图像下载成功，大小: %d bytes", len(imageData))

			// TODO: 调用本地抠图服务进行图像处理
			// 这里暂时跳过，实际项目中应调用 matting 服务
			rp.context.MattingData = imageData

			return imageData, nil
		},
		RollbackFunc: func(ctx *StepContext) error {
			logger.Info("回滚: 清理抠图资源")
			rp.context.MattingData = nil
			return nil
		},
	}
}

// createModelAPIStep 创建模型 API 调用步骤
func (rp *RewritePipeline) createModelAPIStep() StepInterface {
	return &StepFunc{
		StepBase: NewStepBase("模型 API 调用"),
		ExecuteFunc: func(ctx *StepContext) (interface{}, error) {
			logger.Info("Step 2: 调用模型 API")

			// 获取要处理的图像数据（优先使用抠图后的数据）
			imageData := rp.context.SourceData
			if rp.context.MattingData != nil {
				imageData = rp.context.MattingData
			}

			if len(imageData) == 0 {
				return nil, fmt.Errorf("没有可用的图像数据")
			}

			// 调用模型 API
			result, err := rp.callModelAPI(ctx, imageData)
			if err != nil {
				logger.Error("模型 API 调用失败: %v", err)
				return nil, fmt.Errorf("模型 API 调用失败: %w", err)
			}

			rp.context.ModelResult = result
			logger.Info("模型 API 调用成功")

			return result, nil
		},
		RollbackFunc: func(ctx *StepContext) error {
			logger.Info("回滚: 清理模型 API 调用资源")
			rp.context.ModelResult = nil
			return nil
		},
	}
}

// createOssUploadStep 创建 OSS 上传步骤
func (rp *RewritePipeline) createOssUploadStep() StepInterface {
	return &StepFunc{
		StepBase: NewStepBase("OSS 上传"),
		ExecuteFunc: func(ctx *StepContext) (interface{}, error) {
			logger.Info("Step 3: 上传图像到 OSS")

			if rp.context.ModelResult == nil || len(rp.context.ModelResult.ImageData) == 0 {
				return nil, fmt.Errorf("没有可上传的图像数据")
			}

			// 生成文件名
			filename := fmt.Sprintf("rewrite_%d_%d.png", rp.context.Request.ImageID, time.Now().Unix())

			// 上传到 OSS
			ossURL, err := rp.uploadToOSS(ctx, rp.context.ModelResult.ImageData, filename)
			if err != nil {
				logger.Error("OSS 上传失败: %v", err)
				return nil, fmt.Errorf("OSS 上传失败: %w", err)
			}

			rp.context.OssURL = ossURL

			// 添加回滚函数：删除 OSS 上的文件
			ctx.AddRollback(func() error {
				logger.Info("回滚: 删除 OSS 文件: %s", ossURL)
				return rp.deleteFromOSS(ctx, ossURL)
			})

			logger.Info("OSS 上传成功: %s", ossURL)
			return ossURL, nil
		},
		RollbackFunc: func(ctx *StepContext) error {
			logger.Info("回滚: 删除 OSS 上的文件")
			if rp.context.OssURL != "" {
				return rp.deleteFromOSS(ctx, rp.context.OssURL)
			}
			return nil
		},
	}
}

// createDBInsertStep 创建数据库插入步骤
func (rp *RewritePipeline) createDBInsertStep() StepInterface {
	return &StepFunc{
		StepBase: NewStepBase("数据库写入"),
		ExecuteFunc: func(ctx *StepContext) (interface{}, error) {
			logger.Info("Step 4: 写入数据库")

			if rp.context.OssURL == "" {
				return nil, fmt.Errorf("OSS URL 为空，无法创建数据库记录")
			}

			// 获取数据库连接
			imageRepo := db.NewImageRepo(rp.context.DB)

			// 生成图像名称
			name := fmt.Sprintf("rewrite_%d", rp.context.Request.ImageID)

			// 创建数据库记录
			imageID, err := imageRepo.Create(
				name,
				rp.context.OssURL,
				false, // isUpscale
				rp.context.Request.StyleTransform,
			)
			if err != nil {
				logger.Error("创建数据库记录失败: %v", err)
				return nil, fmt.Errorf("创建数据库记录失败: %w", err)
			}

			rp.context.DBImageID = imageID

			// 添加回滚函数：删除数据库记录
			imgRepoForRollback := imageRepo
			ctx.AddRollback(func() error {
				logger.Info("回滚: 删除数据库记录: id=%d", imageID)
				return imgRepoForRollback.Delete(imageID)
			})

			logger.Info("数据库写入成功: id=%d", imageID)
			return imageID, nil
		},
		RollbackFunc: func(ctx *StepContext) error {
			logger.Info("回滚: 删除数据库记录")
			if rp.context.DBImageID > 0 {
				imageRepo := db.NewImageRepo(rp.context.DB)
				return imageRepo.Delete(rp.context.DBImageID)
			}
			return nil
		},
	}
}

// createResponseStep 创建响应构建步骤
func (rp *RewritePipeline) createResponseStep() StepInterface {
	return &StepFunc{
		StepBase: NewStepBase("构建响应"),
		ExecuteFunc: func(ctx *StepContext) (interface{}, error) {
			logger.Info("Step 5: 构建响应数据")

			response := &RewriteResponse{
				Success:       true,
				ImageID:       rp.context.DBImageID,
				ImageURL:      rp.context.OssURL,
				StyleTransform: rp.context.Request.StyleTransform,
				Message:       "图像 rewrite 成功",
			}

			logger.Info("响应构建完成: ImageID=%d, ImageURL=%s", response.ImageID, response.ImageURL)
			return response, nil
		},
		RollbackFunc: nil, // 无需回滚
	}
}

// downloadImage 下载图像
func (rp *RewritePipeline) downloadImage(ctx *StepContext, url string) ([]byte, error) {
	// 使用 HTTP 客户端下载图像
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx.Ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载图像失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载图像失败，状态码: %d", resp.StatusCode)
	}

	// 读取图像数据
	data := make([]byte, 0, resp.ContentLength)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return data, nil
}

// callModelAPI 调用模型 API
func (rp *RewritePipeline) callModelAPI(ctx *StepContext, imageData []byte) (*ModelAPIResult, error) {
	provider := rp.context.Request.ModelProvider
	prompt := rp.context.Request.Prompt

	logger.Info("调用模型 API: provider=%s, prompt=%s", provider, prompt)

	// 根据不同的模型提供者调用不同的 API
	switch provider {
	case "wan":
		return rp.callWanModelAPI(ctx, imageData, prompt)
	case "google":
		return rp.callGoogleModelAPI(ctx, imageData, prompt)
	default:
		return nil, fmt.Errorf("不支持的模型提供者: %s", provider)
	}
}

// callWanModelAPI 调用百炼模型 API
// 注意: 这里需要根据实际的百炼 API 文档来实现
func (rp *RewritePipeline) callWanModelAPI(ctx *StepContext, imageData []byte, prompt string) (*ModelAPIResult, error) {
	// TODO: 实现百炼模型 API 调用
	// 目前返回错误，实际项目中需要根据百炼 API 文档实现
	logger.Warn("百炼模型 API 调用未实现，使用模拟数据")

	// 返回模拟数据用于测试
	return &ModelAPIResult{
		ImageURL:  "",
		ImageData: imageData, // 暂时直接返回原图
		Metadata: map[string]string{
			"provider": "wan",
			"prompt":   prompt,
		},
	}, nil
}

// callGoogleModelAPI 调用 Google 模型 API
// 注意: 这里需要根据实际的 Google AI 文档来实现
func (rp *RewritePipeline) callGoogleModelAPI(ctx *StepContext, imageData []byte, prompt string) (*ModelAPIResult, error) {
	// TODO: 实现 Google 模型 API 调用
	// 目前返回错误，实际项目中需要根据 Google AI 文档实现
	logger.Warn("Google 模型 API 调用未实现，使用模拟数据")

	// 返回模拟数据用于测试
	return &ModelAPIResult{
		ImageURL:  "",
		ImageData: imageData, // 暂时直接返回原图
		Metadata: map[string]string{
			"provider": "google",
			"prompt":   prompt,
		},
	}, nil
}

// uploadToOSS 上传图像到 OSS
func (rp *RewritePipeline) uploadToOSS(ctx *StepContext, data []byte, filename string) (string, error) {
	if rp.context.OSSClient == nil {
		return "", fmt.Errorf("OSS 客户端未初始化")
	}

	// 上传图像
	url, err := rp.context.OSSClient.Upload(ctx.Ctx, data, filename)
	if err != nil {
		return "", fmt.Errorf("上传图像到 OSS 失败: %w", err)
	}

	return url, nil
}

// deleteFromOSS 从 OSS 删除图像
func (rp *RewritePipeline) deleteFromOSS(ctx *StepContext, url string) error {
	if rp.context.OSSClient == nil {
		return fmt.Errorf("OSS 客户端未初始化")
	}

	return rp.context.OSSClient.Delete(ctx.Ctx, url)
}

// GetPipeline 获取工作流流水线实例
func (rp *RewritePipeline) GetPipeline() *Pipeline {
	return rp.pipeline
}

// ExecuteRewrite 执行 rewrite 工作流的便捷函数
// req: rewrite 请求参数
// ossClient: OSS 客户端
// database: 数据库连接
// ctx: 上下文
// 返回: rewrite 响应结果
func ExecuteRewrite(req *RewriteRequest, ossClient oss.OSSClient, database *db.DB, ctx context.Context) (*RewriteResponse, error) {
	pipeline := NewRewritePipeline(req, ossClient, database)
	return pipeline.Execute(ctx)
}

// Step 步骤接口别名，用于外部扩展
type Step = StepInterface

// PipelineInterface 流水线接口别名，用于外部扩展
type PipelineInterface = *Pipeline