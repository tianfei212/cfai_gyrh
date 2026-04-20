package matting

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"time"

	"gyrh-go-v2/backend/internal/logger"
)

// MediaPipeClient MediaPipe HTTP 客户端
type MediaPipeClient struct {
	serverURL  string        // MediaPipe Python 服务地址
	httpClient *http.Client  // HTTP 客户端
	timeout    time.Duration // 请求超时时间
}

// NewMediaPipeClient 创建 MediaPipe 客户端
// serverURL MediaPipe Python 服务的 HTTP 地址，例如 http://localhost:5000
func NewMediaPipeClient(serverURL string, timeout time.Duration) *MediaPipeClient {
	return &MediaPipeClient{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// MattingResult 抠图结果
type MattingResult struct {
	Mask   []byte // 分割 mask 图像 (灰度)
	Width  int    // 原图宽度
	Height int    // 原图高度
}

// ProcessImage 调用 MediaPipe 服务进行图像分割
// ctx 上下文
// imageBytes 输入的原始图像字节数据 (JPEG/PNG)
// 返回分割后的 mask 图像数据
func (c *MediaPipeClient) ProcessImage(ctx context.Context, imageBytes []byte) (*MattingResult, error) {
	// 创建带超时的请求上下文
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 构建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+"/predict", bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/octet-stream")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 MediaPipe 服务失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MediaPipe 服务返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应中的 mask 图像
	maskImg, width, height, err := decodeMaskResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("解析 mask 响应失败: %w", err)
	}

	logger.Debug("MediaPipe 分割完成: width=%d, height=%d", width, height)

	return &MattingResult{
		Mask:   maskImg,
		Width:  width,
		Height: height,
	}, nil
}

// decodeMaskResponse 从响应体解码 mask 图像
func decodeMaskResponse(resp *http.Response) ([]byte, int, int, error) {
	// 读取响应数据
	img, err := png.Decode(resp.Body)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("PNG 解码失败: %w", err)
	}

	// 获取图像尺寸
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 转换为灰度字节数组
	mask := make([]byte, width*height)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.At(x, y)
			r, g, b, _ := gray.RGBA()
			// 将 RGB 转换为灰度值 (使用 luminance 公式)
			luminance := (r*299 + g*587 + b*114) / 1000
			mask[(y-bounds.Min.Y)*width+(x-bounds.Min.X)] = byte(luminance >> 8)
		}
	}

	return mask, width, height, nil
}

// Close 关闭客户端，释放资源
func (c *MediaPipeClient) Close() error {
	// HTTP 客户端不需要显式关闭
	return nil
}

// HealthCheck 检查 MediaPipe 服务健康状态
func (c *MediaPipeClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("创建健康检查请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("健康检查请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("健康检查返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// ConvertToRGBA 将原图和 mask 合并为 RGBA 图像
// srcImage 源图像
// mask 分割 mask (与原图尺寸相同)
// 返回合并后的 RGBA 图像
func ConvertToRGBA(srcImage image.Image, mask []byte) *image.RGBA {
	bounds := srcImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 创建 RGBA 图像
	rgba := image.NewRGBA(bounds)

	// 遍历每个像素
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 获取原图像素
			srcColor := srcImage.At(x+bounds.Min.X, y+bounds.Min.Y)
			r, g, b, _ := srcColor.RGBA()

			// 获取 mask 值
			maskIndex := y*width + x
			maskValue := mask[maskIndex]

			// mask 值作为 alpha 通道 (0-255)
			// maskValue 越大表示越可能是前景，alpha 越高
			a := maskValue

			// 设置 RGBA 像素
			rgba.SetRGBA(x+bounds.Min.X, y+bounds.Min.Y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: a,
			})
		}
	}

	return rgba
}

// EncodeToPNG 将 RGBA 图像编码为 PNG 字节数据
func EncodeToPNG(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	encoder := png.Encoder{}
	err := encoder.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("PNG 编码失败: %w", err)
	}
	return buf.Bytes(), nil
}
