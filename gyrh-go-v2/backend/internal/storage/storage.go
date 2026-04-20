package storage

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"

	"gyrh-go-v2/backend/internal/config"
	ossclient "gyrh-go-v2/backend/internal/oss"
)

// StorageProvider 存储提供者类型
type StorageProvider string

const (
	// ProviderWan 阿里云百炼
	ProviderWan StorageProvider = "wan"
	// ProviderGoogle Google AI
	ProviderGoogle StorageProvider = "google"
)

// StorageService 存储服务接口
type StorageService interface {
	// Save 保存图像数据，返回 assetID
	Save(ctx context.Context, data []byte, filename string) (string, error)

	// Read 读取已存储的图像原始字节。
	Read(ctx context.Context, assetID string) ([]byte, error)

	// GetImageURL 获取用于前端查看的 URL
	GetImageURL(ctx context.Context, assetID string) (string, error)

	// GetForModelUpload 获取用于模型 API 调用的上传路径/URL
	// provider: 模型提供者 (wan/google)
	GetForModelUpload(ctx context.Context, assetID string, provider StorageProvider) (string, error)

	// Delete 删除图像
	Delete(ctx context.Context, assetID string) error
}

// service 是统一的存储编排实现。
type service struct {
	mode      string
	local     *LocalStorage
	alioss    ossclient.OSSClient
	dashscope *ossclient.DashScopeClient
}

// NewStorageService 根据配置创建存储服务实例。
func NewStorageService(cfg *config.Config) (StorageService, error) {
	local := NewLocalStorage(cfg.Storage.LocalPath)
	svc := &service{
		mode:      cfg.Storage.Mode,
		local:     local,
		dashscope: ossclient.NewDashScopeClient(cfg.Storage.DashScopeAPIKey),
	}

	if cfg.AliOSS.Enabled {
		svc.alioss = ossclient.NewAlOSSClient(
			fmt.Sprintf("http://127.0.0.1:%d", cfg.AliOSS.Port),
			cfg.AliOSS.OpenAIAPIKey,
		)
	}

	return svc, nil
}

func (s *service) Save(ctx context.Context, data []byte, filename string) (string, error) {
	if s.mode == "oss" {
		if s.alioss == nil {
			return "", fmt.Errorf("aliOSS 客户端未初始化")
		}
		return s.alioss.Upload(ctx, data, filename)
	}
	return s.local.Save(ctx, data, filename)
}

func (s *service) Read(ctx context.Context, assetID string) ([]byte, error) {
	if s.mode == "oss" {
		if s.alioss == nil {
			return nil, fmt.Errorf("aliOSS 客户端未初始化")
		}
		return s.alioss.Download(ctx, assetID)
	}
	return s.local.LoadFile(assetID)
}

func (s *service) GetImageURL(ctx context.Context, assetID string) (string, error) {
	if s.mode == "oss" {
		if s.alioss == nil {
			return "", fmt.Errorf("aliOSS 客户端未初始化")
		}
		return s.alioss.GetSignedURL(ctx, assetID, 3600)
	}
	return s.local.GetImageURL(ctx, assetID)
}

func (s *service) GetForModelUpload(ctx context.Context, assetID string, provider StorageProvider) (string, error) {
	if s.mode == "oss" {
		if s.alioss == nil {
			return "", fmt.Errorf("aliOSS 客户端未初始化")
		}
		return s.alioss.GetSignedURL(ctx, assetID, 1800)
	}

	data, err := s.local.LoadFile(assetID)
	if err != nil {
		return "", fmt.Errorf("读取本地图片失败: %w", err)
	}

	switch provider {
	case ProviderWan:
		return s.dashscope.Upload(ctx, data, filepath.Base(assetID))
	case ProviderGoogle:
		url, err := s.dashscope.Upload(ctx, data, filepath.Base(assetID))
		if err == nil {
			return url, nil
		}
		return base64.StdEncoding.EncodeToString(data), nil
	default:
		return "", fmt.Errorf("不支持的模型提供者: %s", provider)
	}
}

func (s *service) Delete(ctx context.Context, assetID string) error {
	if s.mode == "oss" {
		if s.alioss == nil {
			return fmt.Errorf("aliOSS 客户端未初始化")
		}
		return s.alioss.Delete(ctx, assetID)
	}
	return s.local.Delete(ctx, assetID)
}

// GetImagePath 获取本地文件路径。
func GetImagePath(basePath, assetID string) string {
	return filepath.Join(basePath, assetID)
}
