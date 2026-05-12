package db

import (
	"database/sql"
	"fmt"
	"time"
)

// BackgroundPrompt 背景图提示词模板。
type BackgroundPrompt struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	GeminiPrompt           string    `json:"gemini_prompt"`
	GeminiNegativePrompt   string    `json:"gemini_negative_prompt"`
	GeminiPromptZH         string    `json:"gemini_prompt_zh"`
	GeminiNegativePromptZH string    `json:"gemini_negative_prompt_zh"`
	WanPrompt              string    `json:"wan_prompt"`
	WanNegativePrompt      string    `json:"wan_negative_prompt"`
	WanPromptZH            string    `json:"wan_prompt_zh"`
	WanNegativePromptZH    string    `json:"wan_negative_prompt_zh"`
	GPTPrompt              string    `json:"gpt_prompt"`
	GPTNegativePrompt      string    `json:"gpt_negative_prompt"`
	GPTPromptZH            string    `json:"gpt_prompt_zh"`
	GPTNegativePromptZH    string    `json:"gpt_negative_prompt_zh"`
	ImageAssetID           string    `json:"image_asset_id"`
	ImageURL               string    `json:"image_url"`
	ImageWidth             int       `json:"image_width"`
	ImageHeight            int       `json:"image_height"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// BackgroundPromptPatch 用于按需更新背景图提示词模板。
type BackgroundPromptPatch struct {
	Name                   *string
	GeminiPrompt           *string
	GeminiNegativePrompt   *string
	GeminiPromptZH         *string
	GeminiNegativePromptZH *string
	WanPrompt              *string
	WanNegativePrompt      *string
	WanPromptZH            *string
	WanNegativePromptZH    *string
	GPTPrompt              *string
	GPTNegativePrompt      *string
	GPTPromptZH            *string
	GPTNegativePromptZH    *string
	ImageAssetID           *string
	ImageURL               *string
	ImageWidth             *int
	ImageHeight            *int
}

// BackgroundPromptRepo 提供 background_prompts 表的 CRUD 操作。
type BackgroundPromptRepo struct {
	db *DB
}

// NewBackgroundPromptRepo 创建背景图提示词仓库实例。
func NewBackgroundPromptRepo(db *DB) *BackgroundPromptRepo {
	return &BackgroundPromptRepo{db: db}
}

// Create 创建新的背景图提示词模板。
func (r *BackgroundPromptRepo) Create(name, geminiPrompt, geminiNegativePrompt, geminiPromptZH, geminiNegativePromptZH, wanPrompt, wanNegativePrompt, wanPromptZH, wanNegativePromptZH, gptPrompt, gptNegativePrompt, gptPromptZH, gptNegativePromptZH, imageAssetID, imageURL string, imageWidth, imageHeight int) (int64, error) {
	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO background_prompts (
			name, gemini_prompt, gemini_negative_prompt, gemini_prompt_zh, gemini_negative_prompt_zh, wan_prompt, wan_negative_prompt, wan_prompt_zh, wan_negative_prompt_zh, gpt_prompt, gpt_negative_prompt, gpt_prompt_zh, gpt_negative_prompt_zh, image_asset_id, image_url, image_width, image_height, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, name, geminiPrompt, geminiNegativePrompt, geminiPromptZH, geminiNegativePromptZH, wanPrompt, wanNegativePrompt, wanPromptZH, wanNegativePromptZH, gptPrompt, gptNegativePrompt, gptPromptZH, gptNegativePromptZH, imageAssetID, imageURL, imageWidth, imageHeight, now, now)
	if err != nil {
		return 0, fmt.Errorf("插入背景图提示词模板失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}
	return id, nil
}

// GetByID 根据 ID 获取背景图提示词模板。
func (r *BackgroundPromptRepo) GetByID(id int64) (*BackgroundPrompt, error) {
	row := r.db.QueryRow(`
		SELECT id, name, gemini_prompt, gemini_negative_prompt, gemini_prompt_zh, gemini_negative_prompt_zh, wan_prompt, wan_negative_prompt, wan_prompt_zh, wan_negative_prompt_zh, gpt_prompt, gpt_negative_prompt, gpt_prompt_zh, gpt_negative_prompt_zh, image_asset_id, image_url, image_width, image_height, created_at, updated_at
		FROM background_prompts
		WHERE id = ?
	`, id)

	var item BackgroundPrompt
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.GeminiPrompt,
		&item.GeminiNegativePrompt,
		&item.GeminiPromptZH,
		&item.GeminiNegativePromptZH,
		&item.WanPrompt,
		&item.WanNegativePrompt,
		&item.WanPromptZH,
		&item.WanNegativePromptZH,
		&item.GPTPrompt,
		&item.GPTNegativePrompt,
		&item.GPTPromptZH,
		&item.GPTNegativePromptZH,
		&item.ImageAssetID,
		&item.ImageURL,
		&item.ImageWidth,
		&item.ImageHeight,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("背景图提示词模板不存在: id=%d", id)
		}
		return nil, fmt.Errorf("查询背景图提示词模板失败: %w", err)
	}

	return &item, nil
}

// GetByName 根据名称获取背景图提示词模板。
func (r *BackgroundPromptRepo) GetByName(name string) (*BackgroundPrompt, error) {
	row := r.db.QueryRow(`
		SELECT id, name, gemini_prompt, gemini_negative_prompt, gemini_prompt_zh, gemini_negative_prompt_zh, wan_prompt, wan_negative_prompt, wan_prompt_zh, wan_negative_prompt_zh, gpt_prompt, gpt_negative_prompt, gpt_prompt_zh, gpt_negative_prompt_zh, image_asset_id, image_url, image_width, image_height, created_at, updated_at
		FROM background_prompts
		WHERE name = ?
	`, name)

	var item BackgroundPrompt
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.GeminiPrompt,
		&item.GeminiNegativePrompt,
		&item.GeminiPromptZH,
		&item.GeminiNegativePromptZH,
		&item.WanPrompt,
		&item.WanNegativePrompt,
		&item.WanPromptZH,
		&item.WanNegativePromptZH,
		&item.GPTPrompt,
		&item.GPTNegativePrompt,
		&item.GPTPromptZH,
		&item.GPTNegativePromptZH,
		&item.ImageAssetID,
		&item.ImageURL,
		&item.ImageWidth,
		&item.ImageHeight,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("背景图提示词模板不存在: name=%s", name)
		}
		return nil, fmt.Errorf("查询背景图提示词模板失败: %w", err)
	}

	return &item, nil
}

// List 获取背景图提示词模板列表。
func (r *BackgroundPromptRepo) List(limit, offset int) ([]*BackgroundPrompt, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if limit > 0 {
		rows, err = r.db.Query(`
			SELECT id, name, gemini_prompt, gemini_negative_prompt, gemini_prompt_zh, gemini_negative_prompt_zh, wan_prompt, wan_negative_prompt, wan_prompt_zh, wan_negative_prompt_zh, gpt_prompt, gpt_negative_prompt, gpt_prompt_zh, gpt_negative_prompt_zh, image_asset_id, image_url, image_width, image_height, created_at, updated_at
			FROM background_prompts
			ORDER BY updated_at DESC, id DESC
			LIMIT ? OFFSET ?
		`, limit, offset)
	} else {
		rows, err = r.db.Query(`
			SELECT id, name, gemini_prompt, gemini_negative_prompt, gemini_prompt_zh, gemini_negative_prompt_zh, wan_prompt, wan_negative_prompt, wan_prompt_zh, wan_negative_prompt_zh, gpt_prompt, gpt_negative_prompt, gpt_prompt_zh, gpt_negative_prompt_zh, image_asset_id, image_url, image_width, image_height, created_at, updated_at
			FROM background_prompts
			ORDER BY updated_at DESC, id DESC
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("查询背景图提示词模板列表失败: %w", err)
	}
	defer rows.Close()

	return r.scanBackgroundPrompts(rows)
}

// Update 更新背景图提示词模板。
func (r *BackgroundPromptRepo) Update(id int64, patch BackgroundPromptPatch) error {
	setClauses := []string{}
	args := []any{}

	if patch.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *patch.Name)
	}
	if patch.GeminiPrompt != nil {
		setClauses = append(setClauses, "gemini_prompt = ?")
		args = append(args, *patch.GeminiPrompt)
	}
	if patch.GeminiNegativePrompt != nil {
		setClauses = append(setClauses, "gemini_negative_prompt = ?")
		args = append(args, *patch.GeminiNegativePrompt)
	}
	if patch.GeminiPromptZH != nil {
		setClauses = append(setClauses, "gemini_prompt_zh = ?")
		args = append(args, *patch.GeminiPromptZH)
	}
	if patch.GeminiNegativePromptZH != nil {
		setClauses = append(setClauses, "gemini_negative_prompt_zh = ?")
		args = append(args, *patch.GeminiNegativePromptZH)
	}
	if patch.WanPrompt != nil {
		setClauses = append(setClauses, "wan_prompt = ?")
		args = append(args, *patch.WanPrompt)
	}
	if patch.WanNegativePrompt != nil {
		setClauses = append(setClauses, "wan_negative_prompt = ?")
		args = append(args, *patch.WanNegativePrompt)
	}
	if patch.WanPromptZH != nil {
		setClauses = append(setClauses, "wan_prompt_zh = ?")
		args = append(args, *patch.WanPromptZH)
	}
	if patch.WanNegativePromptZH != nil {
		setClauses = append(setClauses, "wan_negative_prompt_zh = ?")
		args = append(args, *patch.WanNegativePromptZH)
	}
	if patch.GPTPrompt != nil {
		setClauses = append(setClauses, "gpt_prompt = ?")
		args = append(args, *patch.GPTPrompt)
	}
	if patch.GPTNegativePrompt != nil {
		setClauses = append(setClauses, "gpt_negative_prompt = ?")
		args = append(args, *patch.GPTNegativePrompt)
	}
	if patch.GPTPromptZH != nil {
		setClauses = append(setClauses, "gpt_prompt_zh = ?")
		args = append(args, *patch.GPTPromptZH)
	}
	if patch.GPTNegativePromptZH != nil {
		setClauses = append(setClauses, "gpt_negative_prompt_zh = ?")
		args = append(args, *patch.GPTNegativePromptZH)
	}
	if patch.ImageAssetID != nil {
		setClauses = append(setClauses, "image_asset_id = ?")
		args = append(args, *patch.ImageAssetID)
	}
	if patch.ImageURL != nil {
		setClauses = append(setClauses, "image_url = ?")
		args = append(args, *patch.ImageURL)
	}
	if patch.ImageWidth != nil {
		setClauses = append(setClauses, "image_width = ?")
		args = append(args, *patch.ImageWidth)
	}
	if patch.ImageHeight != nil {
		setClauses = append(setClauses, "image_height = ?")
		args = append(args, *patch.ImageHeight)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now(), id)

	query := fmt.Sprintf("UPDATE background_prompts SET %s WHERE id = ?", joinStrings(setClauses, ", "))
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新背景图提示词模板失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("背景图提示词模板不存在: id=%d", id)
	}
	return nil
}

// Delete 删除背景图提示词模板。
func (r *BackgroundPromptRepo) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM background_prompts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除背景图提示词模板失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("背景图提示词模板不存在: id=%d", id)
	}

	return nil
}

// Count 获取背景图提示词模板总数。
func (r *BackgroundPromptRepo) Count() (int64, error) {
	var count int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM background_prompts").Scan(&count); err != nil {
		return 0, fmt.Errorf("统计背景图提示词模板失败: %w", err)
	}
	return count, nil
}

func (r *BackgroundPromptRepo) scanBackgroundPrompts(rows *sql.Rows) ([]*BackgroundPrompt, error) {
	var items []*BackgroundPrompt
	for rows.Next() {
		var item BackgroundPrompt
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.GeminiPrompt,
			&item.GeminiNegativePrompt,
			&item.GeminiPromptZH,
			&item.GeminiNegativePromptZH,
			&item.WanPrompt,
			&item.WanNegativePrompt,
			&item.WanPromptZH,
			&item.WanNegativePromptZH,
			&item.GPTPrompt,
			&item.GPTNegativePrompt,
			&item.GPTPromptZH,
			&item.GPTNegativePromptZH,
			&item.ImageAssetID,
			&item.ImageURL,
			&item.ImageWidth,
			&item.ImageHeight,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描背景图提示词模板失败: %w", err)
		}
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历背景图提示词模板失败: %w", err)
	}

	return items, nil
}
