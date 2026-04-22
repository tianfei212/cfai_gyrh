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
	// ProviderQwen 通义千问视觉理解
	ProviderQwen StorageProvider = "qwen"
)

// SaveKind 表示存储分区类型。
type SaveKind string

const (
	// SaveKindAsset 背景图、参考图、前景图等素材类资源。
	SaveKindAsset SaveKind = "asset"
	// SaveKindGenerated 生成结果类资源。
	SaveKindGenerated SaveKind = "generated"
)

// StorageService 存储服务接口
type StorageService interface {
	// Save 保存图像数据，返回 assetID
	Save(ctx context.Context, data []byte, filename string) (string, error)

	// SaveWithKind 按资源类型保存图像数据，便于 OSS 路径分流。
	SaveWithKind(ctx context.Context, data []byte, filename string, kind SaveKind) (string, error)

	// Read 读取已存储的图像原始字节。
	Read(ctx context.Context, assetID string) ([]byte, error)

	// GetImageURL 获取用于前端查看的 URL
	GetImageURL(ctx context.Context, assetID string) (string, error)

	// GetThumbnailURL 获取用于前端查看的缩略图 URL
	GetThumbnailURL(ctx context.Context, assetID string, w, h int) (string, error)

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
	generated ossclient.OSSClient
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
		svc.generated = ossclient.NewAlOSSClient(
			fmt.Sprintf("http://127.0.0.1:%d", cfg.AliOSS.GeneratedPort),
			cfg.AliOSS.OpenAIAPIKey,
		)
	}

	return svc, nil
}

func (s *service) Save(ctx context.Context, data []byte, filename string) (string, error) {
	return s.SaveWithKind(ctx, data, filename, SaveKindAsset)
}

func (s *service) SaveWithKind(ctx context.Context, data []byte, filename string, kind SaveKind) (string, error) {
	if s.mode == "oss" {
		client, err := s.clientForKind(kind)
		if err != nil {
			return "", err
		}
		rawID, err := client.Upload(ctx, data, filename)
		if err != nil {
			return "", err
		}
		return encodeAssetID(kind, rawID), nil
	}
	return s.local.Save(ctx, data, filename)
}

func (s *service) Read(ctx context.Context, assetID string) ([]byte, error) {
	if s.mode == "oss" {
		kind, rawID := decodeAssetID(assetID)
		client, err := s.clientForKind(kind)
		if err != nil {
			return nil, err
		}
		return client.Download(ctx, rawID)
	}
	return s.local.LoadFile(assetID)
}

func (s *service) GetImageURL(ctx context.Context, assetID string) (string, error) {
	if s.mode == "oss" {
		kind, rawID := decodeAssetID(assetID)
		client, err := s.clientForKind(kind)
		if err != nil {
			return "", err
		}
		return client.GetSignedURL(ctx, rawID, 3600)
	}
	return s.local.GetImageURL(ctx, assetID)
}

func (s *service) GetThumbnailURL(ctx context.Context, assetID string, w, h int) (string, error) {
	if s.mode == "oss" {
		kind, rawID := decodeAssetID(assetID)
		client, err := s.clientForKind(kind)
		if err != nil {
			return "", err
		}
		// 如果 OSS 客户端支持 GetThumbnailURL，调用它
		if thClient, ok := client.(interface {
			GetThumbnailURL(ctx context.Context, fileID string, expire int, w, h int) (string, error)
		}); ok {
			return thClient.GetThumbnailURL(ctx, rawID, 3600, w, h)
		}
		// 回退到原图
		return client.GetSignedURL(ctx, rawID, 3600)
	}
	return s.local.GetImageURL(ctx, assetID)
}

func (s *service) GetForModelUpload(ctx context.Context, assetID string, provider StorageProvider) (string, error) {
	if s.mode == "oss" {
		kind, rawID := decodeAssetID(assetID)
		client, err := s.clientForKind(kind)
		if err != nil {
			return "", fmt.Errorf("aliOSS 客户端未初始化")
		}
		return client.GetSignedURL(ctx, rawID, 1800)
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
	case ProviderQwen:
		return s.dashscope.Upload(ctx, data, filepath.Base(assetID))
	default:
		return "", fmt.Errorf("不支持的模型提供者: %s", provider)
	}
}

func (s *service) Delete(ctx context.Context, assetID string) error {
	if s.mode == "oss" {
		kind, rawID := decodeAssetID(assetID)
		client, err := s.clientForKind(kind)
		if err != nil {
			return err
		}
		return client.Delete(ctx, rawID)
	}
	return s.local.Delete(ctx, assetID)
}

func (s *service) clientForKind(kind SaveKind) (ossclient.OSSClient, error) {
	switch kind {
	case SaveKindGenerated:
		if s.generated == nil {
			return nil, fmt.Errorf("生成图 aliOSS 客户端未初始化")
		}
		return s.generated, nil
	default:
		if s.alioss == nil {
			return nil, fmt.Errorf("素材 aliOSS 客户端未初始化")
		}
		return s.alioss, nil
	}
}

func encodeAssetID(kind SaveKind, rawID string) string {
	if rawID == "" {
		return rawID
	}
	return string(kind) + ":" + rawID
}

func decodeAssetID(assetID string) (SaveKind, string) {
	switch {
	case len(assetID) > len("generated:") && assetID[:len("generated:")] == "generated:":
		return SaveKindGenerated, assetID[len("generated:"):]
	case len(assetID) > len("asset:") && assetID[:len("asset:")] == "asset:":
		return SaveKindAsset, assetID[len("asset:"):]
	default:
		return SaveKindAsset, assetID
	}
}

// GetImagePath 获取本地文件路径。
func GetImagePath(basePath, assetID string) string {
	return filepath.Join(basePath, assetID)
}
