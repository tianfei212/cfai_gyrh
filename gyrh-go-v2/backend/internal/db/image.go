package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// GeneratedImage 生成图像数据结构
type GeneratedImage struct {
	ID                 int64     `json:"id"`                   // 图像ID
	Name               string    `json:"name"`                 // 图像名称
	Path               string    `json:"path"`                 // 兼容旧字段，等同存储路径
	AssetID            string    `json:"asset_id"`             // 存储资源ID
	IsUpscale          bool      `json:"is_upscale"`           // 是否为放大后的图像
	StyleTransform     string    `json:"style_transform"`      // 兼容旧字段
	Provider           string    `json:"provider"`             // 模型提供者
	Status             string    `json:"status"`               // 生成状态
	BackgroundPromptID int64     `json:"background_prompt_id"` // 关联的背景模板ID
	ImageWidth         int       `json:"image_width"`          // 图片宽度
	ImageHeight        int       `json:"image_height"`         // 图片高度
	ImageURL           string    `json:"image_url,omitempty"`  // 动态生成的访问链接
	CreatedAt          time.Time `json:"created_at"`           // 创建时间
}

// GeneratedImageCreateInput 创建生成图记录时使用的元数据。
type GeneratedImageCreateInput struct {
	Name               string
	Path               string
	AssetID            string
	IsUpscale          bool
	StyleTransform     string
	Provider           string
	Status             string
	BackgroundPromptID int64
	ImageWidth         int
	ImageHeight        int
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
func (r *ImageRepo) Create(input GeneratedImageCreateInput) (int64, error) {
	if input.AssetID == "" {
		input.AssetID = input.Path
	}
	if input.Path == "" {
		input.Path = input.AssetID
	}
	if input.Provider == "" {
		input.Provider = input.StyleTransform
	}
	if input.Status == "" {
		input.Status = "succeeded"
	}

	result, err := r.db.Exec(`
		INSERT INTO generated_images (
			name, path, asset_id, is_upscale, style_transform,
			provider, status, background_prompt_id, image_width, image_height, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.Name, input.Path, input.AssetID, input.IsUpscale, input.StyleTransform,
		input.Provider, input.Status, input.BackgroundPromptID, input.ImageWidth, input.ImageHeight, time.Now())
	if err != nil {
		return 0, fmt.Errorf("插入图像记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}

	return id, nil
}

const generatedImageSelectColumns = `
	id, name, path, asset_id, is_upscale, style_transform,
	provider, status, background_prompt_id, image_width, image_height, created_at
`

// GetByID 根据ID获取图像记录
// id 图像ID
// 返回图像结构和错误信息
func (r *ImageRepo) GetByID(id int64) (*GeneratedImage, error) {
	row := r.db.QueryRow(`
		SELECT `+generatedImageSelectColumns+`
		FROM generated_images
		WHERE id = ?
	`, id)

	img, err := scanGeneratedImage(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("图像记录不存在: id=%d", id)
		}
		return nil, fmt.Errorf("查询图像记录失败: %w", err)
	}
	return img, nil
}

// GetByPath 根据路径获取图像记录
// path 图像存储路径
// 返回图像结构和错误信息
func (r *ImageRepo) GetByPath(path string) (*GeneratedImage, error) {
	row := r.db.QueryRow(`
		SELECT `+generatedImageSelectColumns+`
		FROM generated_images
		WHERE path = ?
	`, path)

	img, err := scanGeneratedImage(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("图像记录不存在: path=%s", path)
		}
		return nil, fmt.Errorf("查询图像记录失败: %w", err)
	}
	return img, nil
}

// GetByName 根据名称获取图像记录。
func (r *ImageRepo) GetByName(name string) (*GeneratedImage, error) {
	row := r.db.QueryRow(`
		SELECT `+generatedImageSelectColumns+`
		FROM generated_images
		WHERE name = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, name)

	img, err := scanGeneratedImage(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("图像记录不存在: name=%s", name)
		}
		return nil, fmt.Errorf("查询图像记录失败: %w", err)
	}
	return img, nil
}

// List 获取所有图像记录
// limit 限制返回数量，0表示不限制
// offset 偏移量，用于分页
// 返回图像列表和错误信息
func (r *ImageRepo) List(limit, offset int) ([]*GeneratedImage, error) {
	var query string
	var args []any

	if limit > 0 {
		query = `
			SELECT ` + generatedImageSelectColumns + `
			FROM generated_images
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []any{limit, offset}
	} else {
		query = `
			SELECT ` + generatedImageSelectColumns + `
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
		img, err := scanGeneratedImage(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描图像记录失败: %w", err)
		}
		images = append(images, img)
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
	var args []any

	if limit > 0 {
		query = `
			SELECT ` + generatedImageSelectColumns + `
			FROM generated_images
			WHERE style_transform = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []any{styleTransform, limit}
	} else {
		query = `
			SELECT ` + generatedImageSelectColumns + `
			FROM generated_images
			WHERE style_transform = ?
			ORDER BY created_at DESC
		`
		args = []any{styleTransform}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询图像列表失败: %w", err)
	}
	defer rows.Close()

	var images []*GeneratedImage
	for rows.Next() {
		img, err := scanGeneratedImage(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描图像记录失败: %w", err)
		}
		images = append(images, img)
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
	args := []any{}

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

type generatedImageScanner interface {
	Scan(dest ...any) error
}

func scanGeneratedImage(scanner generatedImageScanner) (*GeneratedImage, error) {
	var img GeneratedImage
	var isUpscaleInt int
	err := scanner.Scan(
		&img.ID,
		&img.Name,
		&img.Path,
		&img.AssetID,
		&isUpscaleInt,
		&img.StyleTransform,
		&img.Provider,
		&img.Status,
		&img.BackgroundPromptID,
		&img.ImageWidth,
		&img.ImageHeight,
		&img.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	img.IsUpscale = isUpscaleInt == 1
	if img.AssetID == "" {
		img.AssetID = img.Path
	}
	if img.Provider == "" {
		img.Provider = img.StyleTransform
	}
	if img.Status == "" {
		img.Status = "succeeded"
	}
	return &img, nil
}

// joinStrings 辅助函数，拼接字符串切片
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	return strings.Join(strs, sep)
}
