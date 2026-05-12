package image

import (
	"context"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/internal/storage"
)

// ImageRepository 定义生成图片列表用例需要的数据库能力。
type ImageRepository interface {
	List(limit, offset int) ([]*db.GeneratedImage, error)
	Count() (int64, error)
}

// Service 编排图片列表、访问、上传和改写流程。
type Service struct {
	imageRepo      ImageRepository
	storageService storage.StorageService
}

// NewService 创建图片应用服务。
func NewService(imageRepo ImageRepository, storageService storage.StorageService) *Service {
	return &Service{
		imageRepo:      imageRepo,
		storageService: storageService,
	}
}

// ListImagesResult 表示图片列表用例的返回结果。
type ListImagesResult struct {
	Images []*db.GeneratedImage
	Total  int64
}

// ListImages 查询生成历史，并为每张图片补齐可访问 URL。
func (s *Service) ListImages(ctx context.Context, limit int, offset int) (*ListImagesResult, error) {
	logger.Info("应用服务查询图像列表: limit=%d, offset=%d", limit, offset)
	images, err := s.imageRepo.List(limit, offset)
	if err != nil {
		logger.Error("应用服务查询图像列表失败: %v", err)
		return nil, err
	}

	total, err := s.imageRepo.Count()
	if err != nil {
		logger.Warn("应用服务获取图像总数失败: %v", err)
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
		imageURL, urlErr := s.storageService.GetImageURL(ctx, img.AssetID)
		if urlErr != nil {
			logger.Warn("应用服务生成历史图像访问 URL 失败: id=%d, asset_id=%s, err=%v", img.ID, img.AssetID, urlErr)
			continue
		}
		img.ImageURL = imageURL
	}

	logger.Info("应用服务图像列表查询成功: count=%d, total=%d", len(images), total)
	return &ListImagesResult{
		Images: images,
		Total:  total,
	}, nil
}
