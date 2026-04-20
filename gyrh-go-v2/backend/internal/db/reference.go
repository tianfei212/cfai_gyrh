package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ReferenceImage 参考图像数据结构
type ReferenceImage struct {
	ID        int64     `json:"id"`         // 图像ID
	Name      string    `json:"name"`       // 图像名称
	Path      string    `json:"path"`       // 图像存储路径
	ImageType string    `json:"image_type"` // 图像类型：general, style, pose, face 等
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// ReferenceRepo 参考图像仓库，提供 reference_images 表的 CRUD 操作
type ReferenceRepo struct {
	db *DB
}

// NewReferenceRepo 创建参考图像仓库实例
func NewReferenceRepo(db *DB) *ReferenceRepo {
	return &ReferenceRepo{db: db}
}

// Create 创建新的参考图像记录
// name 图像名称
// path 图像存储路径
// imageType 图像类型
// 返回创建的图像ID和错误信息
func (r *ReferenceRepo) Create(name, path, imageType string) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO reference_images (name, path, image_type, created_at)
		VALUES (?, ?, ?, ?)
	`, name, path, imageType, time.Now())
	if err != nil {
		return 0, fmt.Errorf("插入参考图像记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}

	return id, nil
}

// GetByID 根据ID获取参考图像记录
// id 图像ID
// 返回图像结构和错误信息
func (r *ReferenceRepo) GetByID(id int64) (*ReferenceImage, error) {
	row := r.db.QueryRow(`
		SELECT id, name, path, image_type, created_at
		FROM reference_images
		WHERE id = ?
	`, id)

	var img ReferenceImage
	err := row.Scan(&img.ID, &img.Name, &img.Path, &img.ImageType, &img.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("参考图像记录不存在: id=%d", id)
		}
		return nil, fmt.Errorf("查询参考图像记录失败: %w", err)
	}

	return &img, nil
}

// GetByPath 根据路径获取参考图像记录
// path 图像存储路径
// 返回图像结构和错误信息
func (r *ReferenceRepo) GetByPath(path string) (*ReferenceImage, error) {
	row := r.db.QueryRow(`
		SELECT id, name, path, image_type, created_at
		FROM reference_images
		WHERE path = ?
	`, path)

	var img ReferenceImage
	err := row.Scan(&img.ID, &img.Name, &img.Path, &img.ImageType, &img.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("参考图像记录不存在: path=%s", path)
		}
		return nil, fmt.Errorf("查询参考图像记录失败: %w", err)
	}

	return &img, nil
}

// List 获取所有参考图像记录
// imageType 图像类型过滤，空字符串表示不过滤
// limit 限制返回数量，0表示不限制
// offset 偏移量，用于分页
// 返回图像列表和错误信息
func (r *ReferenceRepo) List(imageType string, limit, offset int) ([]*ReferenceImage, error) {
	var query string
	var args []interface{}

	if imageType != "" {
		if limit > 0 {
			query = `
				SELECT id, name, path, image_type, created_at
				FROM reference_images
				WHERE image_type = ?
				ORDER BY created_at DESC
				LIMIT ? OFFSET ?
			`
			args = []interface{}{imageType, limit, offset}
		} else {
			query = `
				SELECT id, name, path, image_type, created_at
				FROM reference_images
				WHERE image_type = ?
				ORDER BY created_at DESC
			`
			args = []interface{}{imageType}
		}
	} else {
		if limit > 0 {
			query = `
				SELECT id, name, path, image_type, created_at
				FROM reference_images
				ORDER BY created_at DESC
				LIMIT ? OFFSET ?
			`
			args = []interface{}{limit, offset}
		} else {
			query = `
				SELECT id, name, path, image_type, created_at
				FROM reference_images
				ORDER BY created_at DESC
			`
		}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询参考图像列表失败: %w", err)
	}
	defer rows.Close()

	var images []*ReferenceImage
	for rows.Next() {
		var img ReferenceImage
		if err := rows.Scan(&img.ID, &img.Name, &img.Path, &img.ImageType, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描参考图像记录失败: %w", err)
		}
		images = append(images, &img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历参考图像列表失败: %w", err)
	}

	return images, nil
}

// ListByType 根据图像类型获取参考图像列表
// imageType 图像类型
// limit 限制返回数量，0表示不限制
// 返回图像列表和错误信息
func (r *ReferenceRepo) ListByType(imageType string, limit int) ([]*ReferenceImage, error) {
	return r.List(imageType, limit, 0)
}

// Update 更新参考图像记录
// id 图像ID
// name 新的图像名称（空字符串表示不更新）
// path 新的图像路径（空字符串表示不更新）
// imageType 新的图像类型（空字符串表示不更新）
// 返回错误信息
func (r *ReferenceRepo) Update(id int64, name, path, imageType string) error {
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
	if imageType != "" {
		setClauses = append(setClauses, "image_type = ?")
		args = append(args, imageType)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE reference_images SET %s WHERE id = ?", joinStrings(setClauses, ", "))
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新参考图像记录失败: %w", err)
	}

	return nil
}

// Delete 删除参考图像记录
// id 图像ID
// 返回错误信息
func (r *ReferenceRepo) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM reference_images WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除参考图像记录失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("参考图像记录不存在: id=%d", id)
	}

	return nil
}

// Count 获取参考图像记录总数
// imageType 图像类型过滤，空字符串表示不过滤
// 返回总数和错误信息
func (r *ReferenceRepo) Count(imageType string) (int64, error) {
	var query string
	var args []interface{}

	if imageType != "" {
		query = "SELECT COUNT(*) FROM reference_images WHERE image_type = ?"
		args = []interface{}{imageType}
	} else {
		query = "SELECT COUNT(*) FROM reference_images"
	}

	var count int64
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计参考图像记录失败: %w", err)
	}
	return count, nil
}
