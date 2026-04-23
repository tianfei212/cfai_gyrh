package db

import (
	"database/sql"
	"fmt"
	"time"
)

// StylePrompt 风格转换提示词结构体
type StylePrompt struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Prompt         string    `json:"prompt"`
	NegativePrompt string    `json:"negative_prompt"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// StylePromptRepo 提供 style_prompts 表的 CRUD 操作
type StylePromptRepo struct {
	db *DB
}

// NewStylePromptRepo 创建风格提示词仓库实例
func NewStylePromptRepo(db *DB) *StylePromptRepo {
	return &StylePromptRepo{db: db}
}

// Create 创建新的风格提示词
func (r *StylePromptRepo) Create(name, prompt, negativePrompt string, isActive bool) (int64, error) {
	now := time.Now()
	activeInt := 0
	if isActive {
		activeInt = 1
	}

	result, err := r.db.Exec(`
		INSERT INTO style_prompts (name, prompt, negative_prompt, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, name, prompt, negativePrompt, activeInt, now, now)
	if err != nil {
		return 0, fmt.Errorf("插入风格提示词记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}

	return id, nil
}

// Update 更新风格提示词
func (r *StylePromptRepo) Update(id int64, name, prompt, negativePrompt string, isActive bool) error {
	activeInt := 0
	if isActive {
		activeInt = 1
	}

	result, err := r.db.Exec(`
		UPDATE style_prompts
		SET name = ?, prompt = ?, negative_prompt = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, name, prompt, negativePrompt, activeInt, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新风格提示词失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("记录不存在")
	}

	return nil
}

// Delete 删除风格提示词
func (r *StylePromptRepo) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM style_prompts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除风格提示词失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("记录不存在")
	}

	return nil
}

// GetByID 根据ID获取风格提示词
func (r *StylePromptRepo) GetByID(id int64) (*StylePrompt, error) {
	row := r.db.QueryRow(`
		SELECT id, name, prompt, negative_prompt, is_active, created_at, updated_at
		FROM style_prompts
		WHERE id = ?
	`, id)

	var sp StylePrompt
	var activeInt int
	err := row.Scan(&sp.ID, &sp.Name, &sp.Prompt, &sp.NegativePrompt, &activeInt, &sp.CreatedAt, &sp.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("记录不存在")
		}
		return nil, fmt.Errorf("查询风格提示词失败: %w", err)
	}

	sp.IsActive = activeInt == 1
	return &sp, nil
}

// List 获取所有风格提示词
func (r *StylePromptRepo) List() ([]*StylePrompt, error) {
	rows, err := r.db.Query(`
		SELECT id, name, prompt, negative_prompt, is_active, created_at, updated_at
		FROM style_prompts
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询风格提示词列表失败: %w", err)
	}
	defer rows.Close()

	var prompts []*StylePrompt
	for rows.Next() {
		var sp StylePrompt
		var activeInt int
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.Prompt, &sp.NegativePrompt, &activeInt, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描风格提示词记录失败: %w", err)
		}
		sp.IsActive = activeInt == 1
		prompts = append(prompts, &sp)
	}

	return prompts, nil
}

// ListActive 获取所有激活的风格提示词
func (r *StylePromptRepo) ListActive() ([]*StylePrompt, error) {
	rows, err := r.db.Query(`
		SELECT id, name, prompt, negative_prompt, is_active, created_at, updated_at
		FROM style_prompts
		WHERE is_active = 1
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询激活风格提示词列表失败: %w", err)
	}
	defer rows.Close()

	var prompts []*StylePrompt
	for rows.Next() {
		var sp StylePrompt
		var activeInt int
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.Prompt, &sp.NegativePrompt, &activeInt, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描风格提示词记录失败: %w", err)
		}
		sp.IsActive = activeInt == 1
		prompts = append(prompts, &sp)
	}

	return prompts, nil
}
