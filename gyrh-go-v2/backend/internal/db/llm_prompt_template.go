package db

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	// TemplateKeyQwenBackgroundSuggest Qwen 根据背景图生成双语建议的模板键。
	TemplateKeyQwenBackgroundSuggest = "qwen_background_prompt_suggestion"
	// TemplateKeyQwenSyncEnglish Qwen 将中文提示词同步成英文的模板键。
	TemplateKeyQwenSyncEnglish = "qwen_background_prompt_sync_english"
)

// LLMPromptTemplate 表示大模型提示词模板。
type LLMPromptTemplate struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	TemplateKey string    `json:"template_key"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LLMPromptTemplatePatch 用于按需更新模板。
type LLMPromptTemplatePatch struct {
	Name        *string
	TemplateKey *string
	Content     *string
	Description *string
}

// LLMPromptTemplateRepo 提供 llm_prompt_templates 表的 CRUD 操作。
type LLMPromptTemplateRepo struct {
	db *DB
}

// NewLLMPromptTemplateRepo 创建模板仓库实例。
func NewLLMPromptTemplateRepo(db *DB) *LLMPromptTemplateRepo {
	return &LLMPromptTemplateRepo{db: db}
}

// Create 创建模板。
func (r *LLMPromptTemplateRepo) Create(name, templateKey, content, description string) (int64, error) {
	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO llm_prompt_templates (name, template_key, content, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, name, templateKey, content, description, now, now)
	if err != nil {
		return 0, fmt.Errorf("插入大模型提示词模板失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}
	return id, nil
}

// GetByID 根据 ID 获取模板。
func (r *LLMPromptTemplateRepo) GetByID(id int64) (*LLMPromptTemplate, error) {
	row := r.db.QueryRow(`
		SELECT id, name, template_key, content, description, created_at, updated_at
		FROM llm_prompt_templates
		WHERE id = ?
	`, id)

	var item LLMPromptTemplate
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.TemplateKey,
		&item.Content,
		&item.Description,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("大模型提示词模板不存在: id=%d", id)
		}
		return nil, fmt.Errorf("查询大模型提示词模板失败: %w", err)
	}
	return &item, nil
}

// GetByKey 根据模板键获取模板。
func (r *LLMPromptTemplateRepo) GetByKey(templateKey string) (*LLMPromptTemplate, error) {
	row := r.db.QueryRow(`
		SELECT id, name, template_key, content, description, created_at, updated_at
		FROM llm_prompt_templates
		WHERE template_key = ?
	`, templateKey)

	var item LLMPromptTemplate
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.TemplateKey,
		&item.Content,
		&item.Description,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("大模型提示词模板不存在: key=%s", templateKey)
		}
		return nil, fmt.Errorf("查询大模型提示词模板失败: %w", err)
	}
	return &item, nil
}

// List 获取模板列表。
func (r *LLMPromptTemplateRepo) List(limit, offset int) ([]*LLMPromptTemplate, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if limit > 0 {
		rows, err = r.db.Query(`
			SELECT id, name, template_key, content, description, created_at, updated_at
			FROM llm_prompt_templates
			ORDER BY updated_at DESC, id DESC
			LIMIT ? OFFSET ?
		`, limit, offset)
	} else {
		rows, err = r.db.Query(`
			SELECT id, name, template_key, content, description, created_at, updated_at
			FROM llm_prompt_templates
			ORDER BY updated_at DESC, id DESC
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("查询大模型提示词模板列表失败: %w", err)
	}
	defer rows.Close()

	var items []*LLMPromptTemplate
	for rows.Next() {
		var item LLMPromptTemplate
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.TemplateKey,
			&item.Content,
			&item.Description,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描大模型提示词模板失败: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历大模型提示词模板失败: %w", err)
	}
	return items, nil
}

// Update 更新模板。
func (r *LLMPromptTemplateRepo) Update(id int64, patch LLMPromptTemplatePatch) error {
	setClauses := []string{}
	args := []any{}

	if patch.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *patch.Name)
	}
	if patch.TemplateKey != nil {
		setClauses = append(setClauses, "template_key = ?")
		args = append(args, *patch.TemplateKey)
	}
	if patch.Content != nil {
		setClauses = append(setClauses, "content = ?")
		args = append(args, *patch.Content)
	}
	if patch.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *patch.Description)
	}
	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now(), id)

	query := fmt.Sprintf("UPDATE llm_prompt_templates SET %s WHERE id = ?", joinStrings(setClauses, ", "))
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新大模型提示词模板失败: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("大模型提示词模板不存在: id=%d", id)
	}
	return nil
}

// Delete 删除模板。
func (r *LLMPromptTemplateRepo) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM llm_prompt_templates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除大模型提示词模板失败: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("大模型提示词模板不存在: id=%d", id)
	}
	return nil
}

// Count 获取模板总数。
func (r *LLMPromptTemplateRepo) Count() (int64, error) {
	var count int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM llm_prompt_templates").Scan(&count); err != nil {
		return 0, fmt.Errorf("统计大模型提示词模板失败: %w", err)
	}
	return count, nil
}

// UpsertByKey 按模板键更新或创建模板。
func (r *LLMPromptTemplateRepo) UpsertByKey(name, templateKey, content, description string) error {
	existing, err := r.GetByKey(templateKey)
	if err != nil {
		_, createErr := r.Create(name, templateKey, content, description)
		return createErr
	}

	return r.Update(existing.ID, LLMPromptTemplatePatch{
		Name:        &name,
		TemplateKey: &templateKey,
		Content:     &content,
		Description: &description,
	})
}
