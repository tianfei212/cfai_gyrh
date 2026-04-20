package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gyrh-go-v2/backend/internal/logger"
)

// LocalStorage 本地存储服务
type LocalStorage struct {
	basePath string // 本地存储根目录
}

// NewLocalStorage 创建本地存储服务实例
func NewLocalStorage(basePath string) *LocalStorage {
	// 确保存储目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		logger.Error("创建本地存储目录失败: %v", err)
	}
	return &LocalStorage{
		basePath: basePath,
	}
}

// Save 保存图像到本地存储
// 返回生成的 assetID（使用原文件名作为 ID）
func (s *LocalStorage) Save(ctx context.Context, data []byte, filename string) (string, error) {
	// 生成 assetID（直接使用原始文件名，避免路径遍历攻击）
	assetID := sanitizeFilename(filename)

	// 构建完整文件路径
	filePath := s.getFilePath(assetID)

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("创建存储子目录失败: %v", err)
		return "", fmt.Errorf("创建存储目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		logger.Error("保存文件失败: %v", err)
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	logger.Debug("文件保存成功: assetID=%s, path=%s", assetID, filePath)
	return assetID, nil
}

// GetImageURL 获取用于前端查看的 URL（本地模式返回相对路径）
func (s *LocalStorage) GetImageURL(ctx context.Context, assetID string) (string, error) {
	// 本地模式返回相对路径，前端通过 /images/view?id=xxx 访问
	// 注意：这里只返回路径，完整的 URL 由前端服务拼接
	viewURL := fmt.Sprintf("/images/view?id=%s", assetID)
	logger.Debug("获取本地图片 URL: assetID=%s, url=%s", assetID, viewURL)
	return viewURL, nil
}

// GetForModelUpload 获取用于模型 API 调用的上传路径
// 该方法仅为兼容旧调用保留，实际模型上传应统一走 StorageService。
func (s *LocalStorage) GetForModelUpload(ctx context.Context, assetID string, provider StorageProvider) (string, error) {
	_ = ctx
	_ = provider

	filePath := s.getFilePath(assetID)
	if _, err := os.Stat(filePath); err != nil {
		return "", fmt.Errorf("本地图片不存在: %w", err)
	}
	return filePath, nil
}

// Delete 删除本地存储的文件
func (s *LocalStorage) Delete(ctx context.Context, assetID string) error {
	filePath := s.getFilePath(assetID)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Warn("文件不存在，无法删除: assetID=%s", assetID)
		return fmt.Errorf("文件不存在: %s", assetID)
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		logger.Error("删除文件失败: %v", err)
		return fmt.Errorf("删除文件失败: %w", err)
	}

	logger.Debug("文件删除成功: assetID=%s", assetID)
	return nil
}

// getFilePath 获取文件的完整路径
func (s *LocalStorage) getFilePath(assetID string) string {
	return filepath.Join(s.basePath, assetID)
}

// LoadFile 加载文件内容（供内部使用，如生成 Base64）
func (s *LocalStorage) LoadFile(assetID string) ([]byte, error) {
	filePath := s.getFilePath(assetID)
	return os.ReadFile(filePath)
}

// sanitizeFilename 清理文件名，防止路径遍历攻击
func sanitizeFilename(filename string) string {
	// 移除所有路径分隔符
	filename = filepath.Base(filename)

	// 移除潜在的路径遍历序列
	filename = filepath.Clean(filename)

	// 如果清理后为空，使用默认名称
	if filename == "" || filename == "." {
		filename = "image"
	}

	return filename
}

// Exists 检查文件是否存在
func (s *LocalStorage) Exists(assetID string) bool {
	_, err := os.Stat(s.getFilePath(assetID))
	return err == nil
}

// GetFilePath 获取文件的绝对路径（供内部路由使用）
func (s *LocalStorage) GetFilePath(assetID string) string {
	return s.getFilePath(assetID)
}
