package matting

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"sync"

	"gyrh-go-v2/backend/internal/config"
	"gyrh-go-v2/backend/internal/logger"
)

// MattingService 抠图服务接口
// 定义人像抠图处理的核心接口
type MattingService interface {
	// Process 对输入图像进行抠图处理
	// ctx 上下文
	// imageBytes 输入的原始图像字节数据 (JPEG/PNG)
	// 返回带透明通道的 RGBA 图像字节数据 (PNG 格式)
	Process(ctx context.Context, imageBytes []byte) ([]byte, error)

	// IsEnabled 返回抠图功能是否启用
	IsEnabled() bool

	// Close 关闭服务，释放资源
	Close() error
}

// mattingService 抠图服务实现
type mattingService struct {
	enabled      bool          // 是否启用抠图
	model        string        // 模型类型: mediapipe, u2net 等
	modelPath    string        // 模型文件路径
	mediapipeURL string        // MediaPipe 服务地址 (HTTP)
	client       *MediaPipeClient // MediaPipe HTTP 客户端
	mu           sync.RWMutex // 读写锁，保护客户端状态
}

// MattingConfig 抠图配置
type MattingConfig struct {
	Enabled   bool   // 是否启用本地抠图
	Model     string // 模型选型: mediapipe
	ModelPath string // 模型文件路径
	ServerURL string // MediaPipe Python 服务地址
}

// NewMattingService 创建抠图服务实例
// 根据配置初始化相应的抠图服务
func NewMattingService(cfg *config.Config) (MattingService, error) {
	serverURL := strings.TrimSpace(cfg.Matting.ServerURL)
	if serverURL == "" {
		serverURL = "http://127.0.0.1:5000"
	}
	if cfg.Matting.ModelPath != "" {
		if _, err := os.Stat(cfg.Matting.ModelPath); err != nil {
			logger.Warn("MediaPipe 模型文件不存在: %s", cfg.Matting.ModelPath)
		}
	}

	logger.Info("初始化抠图服务: enabled=%v, model=%s, modelPath=%s, serverURL=%s",
		cfg.Matting.Enabled, cfg.Matting.Model, cfg.Matting.ModelPath, serverURL)

	svc := &mattingService{
		enabled:      cfg.Matting.Enabled,
		model:        cfg.Matting.Model,
		modelPath:    cfg.Matting.ModelPath,
		mediapipeURL: serverURL,
	}

	// 如果启用且使用 mediapipe，初始化 HTTP 客户端
	if svc.enabled && svc.model == "mediapipe" {
		svc.client = NewMediaPipeClient(svc.mediapipeURL, 30*1000000000) // 30秒超时
		logger.Info("MediaPipe 客户端已初始化")
	}

	return svc, nil
}

// NewMattingServiceWithConfig 使用自定义配置创建抠图服务
func NewMattingServiceWithConfig(cfg *MattingConfig) (MattingService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("抠图配置不能为空")
	}

	logger.Info("初始化抠图服务: enabled=%v, model=%s, modelPath=%s",
		cfg.Enabled, cfg.Model, cfg.ModelPath)

	svc := &mattingService{
		enabled:      cfg.Enabled,
		model:        cfg.Model,
		modelPath:    cfg.ModelPath,
		mediapipeURL: cfg.ServerURL,
	}

	// 如果启用且使用 mediapipe，初始化 HTTP 客户端
	if svc.enabled && svc.model == "mediapipe" {
		svc.client = NewMediaPipeClient(svc.mediapipeURL, 30*1000000000) // 30秒超时
		logger.Info("MediaPipe 客户端已初始化")
	}

	return svc, nil
}

// Process 对输入图像进行抠图处理
// 将原图与 MediaPipe 生成的 mask 合并，生成带透明通道的 RGBA 图像
func (s *mattingService) Process(ctx context.Context, imageBytes []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查是否启用
	if !s.enabled {
		return nil, fmt.Errorf("抠图功能未启用")
	}

	// 检查客户端
	if s.client == nil {
		return nil, fmt.Errorf("MediaPipe 客户端未初始化")
	}

	logger.Debug("开始抠图处理: 输入大小=%d bytes", len(imageBytes))

	// 1. 解码输入图像
	srcImage, format, err := decodeImage(imageBytes)
	if err != nil {
		return nil, fmt.Errorf("解码输入图像失败: %w", err)
	}
	logger.Debug("输入图像格式: %s, 尺寸: %v", format, srcImage.Bounds())

	// 2. 调用 MediaPipe 进行图像分割
	mattingResult, err := s.client.ProcessImage(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("MediaPipe 分割失败: %w", err)
	}

	// 3. 验证 mask 尺寸与原图是否匹配
	srcBounds := srcImage.Bounds()
	if mattingResult.Width != srcBounds.Dx() || mattingResult.Height != srcBounds.Dy() {
		logger.Warn("警告: mask 尺寸 (%dx%d) 与原图尺寸 (%dx%d) 不匹配",
			mattingResult.Width, mattingResult.Height, srcBounds.Dx(), srcBounds.Dy())
	}

	// 4. 将原图和 mask 合并为 RGBA 图像
	rgbaImage := ConvertToRGBA(srcImage, mattingResult.Mask)

	// 5. 编码为 PNG 格式返回
	outputBytes, err := EncodeToPNG(rgbaImage)
	if err != nil {
		return nil, fmt.Errorf("编码输出图像失败: %w", err)
	}

	logger.Debug("抠图处理完成: 输出大小=%d bytes", len(outputBytes))

	return outputBytes, nil
}

// IsEnabled 返回抠图功能是否启用
func (s *mattingService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// Close 关闭服务，释放资源
func (s *mattingService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		err := s.client.Close()
		s.client = nil
		return err
	}
	return nil
}

// decodeImage 解码图像字节数据
// 支持 JPEG 和 PNG 格式
// 返回解码后的图像、图像格式和解码错误
func decodeImage(data []byte) (image.Image, string, error) {
	// 使用 bytes.Reader 支持多次读取
	reader := bytes.NewReader(data)

	// 首先尝试检测图像格式
	format, err := detectImageFormat(data)
	if err != nil {
		return nil, "", fmt.Errorf("检测图像格式失败: %w", err)
	}

	// 根据格式解码
	switch format {
	case "jpeg", "jpg":
		img, err := jpeg.Decode(reader)
		if err != nil {
			return nil, format, fmt.Errorf("JPEG 解码失败: %w", err)
		}
		return img, format, nil
	case "png":
		img, err := png.Decode(reader)
		if err != nil {
			return nil, format, fmt.Errorf("PNG 解码失败: %w", err)
		}
		return img, format, nil
	default:
		return nil, format, fmt.Errorf("不支持的图像格式: %s", format)
	}
}

// detectImageFormat 检测图像格式
// 通过文件头部的魔术数字识别图像格式
func detectImageFormat(data []byte) (string, error) {
	if len(data) < 4 {
		return "", fmt.Errorf("数据长度不足，无法检测格式")
	}

	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "jpeg", nil
	}

	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "png", nil
	}

	return "", fmt.Errorf("未知的图像格式")
}

// MockMattingService Mock 抠图服务实现 (用于测试或禁用真实抠图时)
type MockMattingService struct {
	enabled bool
}

// NewMockMattingService 创建 Mock 抠图服务
func NewMockMattingService(enabled bool) MattingService {
	return &MockMattingService{enabled: enabled}
}

// Process Mock 处理 - 直接返回原始图像的 RGBA 版本
func (s *MockMattingService) Process(ctx context.Context, imageBytes []byte) ([]byte, error) {
	if !s.enabled {
		return nil, fmt.Errorf("抠图功能未启用")
	}

	// 解码图像
	srcImage, _, err := decodeImage(imageBytes)
	if err != nil {
		return nil, fmt.Errorf("解码图像失败: %w", err)
	}

	// 转换为 RGBA (添加完全不透明的 alpha 通道)
	rgba := image.NewRGBA(srcImage.Bounds())
	for y := srcImage.Bounds().Min.Y; y < srcImage.Bounds().Max.Y; y++ {
		for x := srcImage.Bounds().Min.X; x < srcImage.Bounds().Max.X; x++ {
			r, g, b, _ := srcImage.At(x, y).RGBA()
			rgba.SetRGBA(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: 255,
			})
		}
	}

	// 编码为 PNG
	return EncodeToPNG(rgba)
}

// IsEnabled 返回是否启用
func (s *MockMattingService) IsEnabled() bool {
	return s.enabled
}

// Close 关闭服务
func (s *MockMattingService) Close() error {
	return nil
}
