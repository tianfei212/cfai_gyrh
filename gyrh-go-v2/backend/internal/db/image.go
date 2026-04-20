package db

import (
	"database/sql"
	"fmt"
	"time"
)

// GeneratedImage 生成图像数据结构
type GeneratedImage struct {
	ID             int64     `json:"id"`              // 图像ID
	Name           string    `json:"name"`            // 图像名称
	Path           string    `json:"path"`            // 图像存储路径
	IsUpscale      bool      `json:"is_upscale"`      // 是否为放大后的图像
	StyleTransform string    `json:"style_transform"` // 风格转换类型
	CreatedAt      time.Time `json:"created_at"`       // 创建时间
}

// ImageRepo 图像仓库，提供 generated_images 表的 CRUD 操作
type ImageRepo struct {
	db *DB
}

// NewImageRepo 创建图像仓库实例
func NewImageRepo(db *DB) *ImageRepo {
	return &ImageRepo{db: db}
}

// Create 创建新的图像记录
// name 图像名称
// path 图像存储路径
// isUpscale 是否为放大后的图像
// styleTransform 风格转换类型
// 返回创建的图像ID和错误信息
func (r *ImageRepo) Create(name, path string, isUpscale bool, styleTransform string) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO generated_images (name, path, is_upscale, style_transform, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, name, path, isUpscale, styleTransform, time.Now())
	if err != nil {
		return 0, fmt.Errorf("插入图像记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}

	return id, nil
}

// GetByID 根据ID获取图像记录
// id 图像ID
// 返回图像结构和错误信息
func (r *ImageRepo) GetByID(id int64) (*GeneratedImage, error) {
	row := r.db.QueryRow(`
		SELECT id, name, path, is_upscale, style_transform, created_at
		FROM generated_images
		WHERE id = ?
	`, id)

	var img GeneratedImage
	var isUpscaleInt int
	err := row.Scan(&img.ID, &img.Name, &img.Path, &isUpscaleInt, &img.StyleTransform, &img.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("图像记录不存在: id=%d", id)
		}
		return nil, fmt.Errorf("查询图像记录失败: %w", err)
	}
	img.IsUpscale = isUpscaleInt == 1

	return &img, nil
}

// GetByPath 根据路径获取图像记录
// path 图像存储路径
// 返回图像结构和错误信息
func (r *ImageRepo) GetByPath(path string) (*GeneratedImage, error) {
	row := r.db.QueryRow(`
		SELECT id, name, path, is_upscale, style_transform, created_at
		FROM generated_images
		WHERE path = ?
	`, path)

	var img GeneratedImage
	var isUpscaleInt int
	err := row.Scan(&img.ID, &img.Name, &img.Path, &isUpscaleInt, &img.StyleTransform, &img.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("图像记录不存在: path=%s", path)
		}
		return nil, fmt.Errorf("查询图像记录失败: %w", err)
	}
	img.IsUpscale = isUpscaleInt == 1

	return &img, nil
}

// GetByName 根据名称获取图像记录。
func (r *ImageRepo) GetByName(name string) (*GeneratedImage, error) {
	row := r.db.QueryRow(`
		SELECT id, name, path, is_upscale, style_transform, created_at
		FROM generated_images
		WHERE name = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, name)

	var img GeneratedImage
	var isUpscaleInt int
	err := row.Scan(&img.ID, &img.Name, &img.Path, &isUpscaleInt, &img.StyleTransform, &img.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("图像记录不存在: name=%s", name)
		}
		return nil, fmt.Errorf("查询图像记录失败: %w", err)
	}
	img.IsUpscale = isUpscaleInt == 1
	return &img, nil
}

// List 获取所有图像记录
// limit 限制返回数量，0表示不限制
// offset 偏移量，用于分页
// 返回图像列表和错误信息
func (r *ImageRepo) List(limit, offset int) ([]*GeneratedImage, error) {
	var query string
	var args []interface{}

	if limit > 0 {
		query = `
			SELECT id, name, path, is_upscale, style_transform, created_at
			FROM generated_images
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	} else {
		query = `
			SELECT id, name, path, is_upscale, style_transform, created_at
			FROM generated_images
			ORDER BY created_at DESC
		`
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询图像列表失败: %w", err)
	}
	defer rows.Close()

	var images []*GeneratedImage
	for rows.Next() {
		var img GeneratedImage
		var isUpscaleInt int
		if err := rows.Scan(&img.ID, &img.Name, &img.Path, &isUpscaleInt, &img.StyleTransform, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描图像记录失败: %w", err)
		}
		img.IsUpscale = isUpscaleInt == 1
		images = append(images, &img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历图像列表失败: %w", err)
	}

	return images, nil
}

// ListByStyleTransform 根据风格转换类型获取图像列表
// styleTransform 风格转换类型
// limit 限制返回数量，0表示不限制
// 返回图像列表和错误信息
func (r *ImageRepo) ListByStyleTransform(styleTransform string, limit int) ([]*GeneratedImage, error) {
	var query string
	var args []interface{}

	if limit > 0 {
		query = `
			SELECT id, name, path, is_upscale, style_transform, created_at
			FROM generated_images
			WHERE style_transform = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{styleTransform, limit}
	} else {
		query = `
			SELECT id, name, path, is_upscale, style_transform, created_at
			FROM generated_images
			WHERE style_transform = ?
			ORDER BY created_at DESC
		`
		args = []interface{}{styleTransform}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询图像列表失败: %w", err)
	}
	defer rows.Close()

	var images []*GeneratedImage
	for rows.Next() {
		var img GeneratedImage
		var isUpscaleInt int
		if err := rows.Scan(&img.ID, &img.Name, &img.Path, &isUpscaleInt, &img.StyleTransform, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描图像记录失败: %w", err)
		}
		img.IsUpscale = isUpscaleInt == 1
		images = append(images, &img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历图像列表失败: %w", err)
	}

	return images, nil
}

// Update 更新图像记录
// id 图像ID
// name 新的图像名称（空字符串表示不更新）
// path 新的图像路径（空字符串表示不更新）
// 返回错误信息
func (r *ImageRepo) Update(id int64, name, path string) error {
	// 构建动态更新SQL
	setClauses := []string{}
	args := []interface{}{}

	if name != "" {
		setClauses = append(setClauses, "name = ?")
		args = append(args, name)
	}
	if path != "" {
		setClauses = append(setClauses, "path = ?")
		args = append(args, path)
	}

	if len(setClauses) == 0 {
		return nil // 没有需要更新的字段
	}

	query := fmt.Sprintf("UPDATE generated_images SET %s WHERE id = ?", joinStrings(setClauses, ", "))
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新图像记录失败: %w", err)
	}

	return nil
}

// Delete 删除图像记录
// id 图像ID
// 返回错误信息
func (r *ImageRepo) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM generated_images WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除图像记录失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("图像记录不存在: id=%d", id)
	}

	return nil
}

// Count 获取图像记录总数
// 返回总数和错误信息
func (r *ImageRepo) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM generated_images").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计图像记录失败: %w", err)
	}
	return count, nil
}

// joinStrings 辅助函数，拼接字符串切片
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
