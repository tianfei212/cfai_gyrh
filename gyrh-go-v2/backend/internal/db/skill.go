package db

import (
	"database/sql"
	"fmt"
	"time"
)

// SkillFile 技能文件数据结构。
type SkillFile struct {
	ID        int64     `json:"id"`         // 文件ID
	Name      string    `json:"name"`       // 文件名称
	Content   string    `json:"content"`    // 文件内容（支持 Markdown / JSON）
	Provider  string    `json:"provider"`   // 提供者/来源
	IsActive  bool      `json:"is_active"`  // 是否激活
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// SkillRepo 技能文件仓库，提供 skill_files 表的 CRUD 操作
type SkillRepo struct {
	db *DB
}

// NewSkillRepo 创建技能文件仓库实例
func NewSkillRepo(db *DB) *SkillRepo {
	return &SkillRepo{db: db}
}

// Create 创建新的技能文件记录。
func (r *SkillRepo) Create(name, content, provider string) (int64, error) {
	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO skill_files (name, content, provider, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, name, content, provider, true, now, now)
	if err != nil {
		return 0, fmt.Errorf("插入技能文件记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}

	return id, nil
}

// GetByID 根据ID获取技能文件记录
// id 文件ID
// 返回技能文件结构和错误信息
func (r *SkillRepo) GetByID(id int64) (*SkillFile, error) {
	row := r.db.QueryRow(`
		SELECT id, name, content, provider, is_active, created_at, updated_at
		FROM skill_files
		WHERE id = ?
	`, id)

	var skill SkillFile
	var isActiveInt int
	err := row.Scan(&skill.ID, &skill.Name, &skill.Content, &skill.Provider, &isActiveInt, &skill.CreatedAt, &skill.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("技能文件记录不存在: id=%d", id)
		}
		return nil, fmt.Errorf("查询技能文件记录失败: %w", err)
	}
	skill.IsActive = isActiveInt == 1

	return &skill, nil
}

// GetByName 根据名称获取技能文件记录
// name 文件名称
// 返回技能文件结构和错误信息
func (r *SkillRepo) GetByName(name string) (*SkillFile, error) {
	row := r.db.QueryRow(`
		SELECT id, name, content, provider, is_active, created_at, updated_at
		FROM skill_files
		WHERE name = ?
	`, name)

	var skill SkillFile
	var isActiveInt int
	err := row.Scan(&skill.ID, &skill.Name, &skill.Content, &skill.Provider, &isActiveInt, &skill.CreatedAt, &skill.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("技能文件记录不存在: name=%s", name)
		}
		return nil, fmt.Errorf("查询技能文件记录失败: %w", err)
	}
	skill.IsActive = isActiveInt == 1

	return &skill, nil
}

// GetActive 获取指定 provider 当前激活的 Skill。
func (r *SkillRepo) GetActive(provider string) (*SkillFile, error) {
	row := r.db.QueryRow(`
		SELECT id, name, content, provider, is_active, created_at, updated_at
		FROM skill_files
		WHERE provider = ? AND is_active = 1
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, provider)

	var skill SkillFile
	var isActiveInt int
	if err := row.Scan(&skill.ID, &skill.Name, &skill.Content, &skill.Provider, &isActiveInt, &skill.CreatedAt, &skill.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("未找到激活的技能文件: provider=%s", provider)
		}
		return nil, fmt.Errorf("查询激活的技能文件失败: %w", err)
	}
	skill.IsActive = isActiveInt == 1
	return &skill, nil
}

// List 获取所有技能文件记录
// limit 限制返回数量，0表示不限制
// offset 偏移量，用于分页
// 返回技能文件列表和错误信息
func (r *SkillRepo) List(limit, offset int) ([]*SkillFile, error) {
	var query string
	var args []interface{}

	if limit > 0 {
		query = `
			SELECT id, name, content, provider, is_active, created_at, updated_at
			FROM skill_files
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	} else {
		query = `
			SELECT id, name, content, provider, is_active, created_at, updated_at
			FROM skill_files
			ORDER BY created_at DESC
		`
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询技能文件列表失败: %w", err)
	}
	defer rows.Close()

	return r.scanSkills(rows)
}

// ListByProvider 根据提供者获取技能文件列表
// provider 提供者/来源
// limit 限制返回数量，0表示不限制
// activeOnly 是否只查询激活的记录
// 返回技能文件列表和错误信息
func (r *SkillRepo) ListByProvider(provider string, activeOnly bool, limit int) ([]*SkillFile, error) {
	var query string
	var args []interface{}

	if activeOnly {
		if limit > 0 {
			query = `
				SELECT id, name, content, provider, is_active, created_at, updated_at
				FROM skill_files
				WHERE provider = ? AND is_active = 1
				ORDER BY created_at DESC
				LIMIT ?
			`
			args = []interface{}{provider, limit}
		} else {
			query = `
				SELECT id, name, content, provider, is_active, created_at, updated_at
				FROM skill_files
				WHERE provider = ? AND is_active = 1
				ORDER BY created_at DESC
			`
			args = []interface{}{provider}
		}
	} else {
		if limit > 0 {
			query = `
				SELECT id, name, content, provider, is_active, created_at, updated_at
				FROM skill_files
				WHERE provider = ?
				ORDER BY created_at DESC
				LIMIT ?
			`
			args = []interface{}{provider, limit}
		} else {
			query = `
				SELECT id, name, content, provider, is_active, created_at, updated_at
				FROM skill_files
				WHERE provider = ?
				ORDER BY created_at DESC
			`
			args = []interface{}{provider}
		}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询技能文件列表失败: %w", err)
	}
	defer rows.Close()

	return r.scanSkills(rows)
}

// Update 更新技能文件记录。
func (r *SkillRepo) Update(id int64, name, content, provider string) error {
	setClauses := []string{}
	args := []interface{}{}

	if name != "" {
		setClauses = append(setClauses, "name = ?")
		args = append(args, name)
	}
	if content != "" {
		setClauses = append(setClauses, "content = ?")
		args = append(args, content)
	}
	if provider != "" {
		setClauses = append(setClauses, "provider = ?")
		args = append(args, provider)
	}

	// 更新时间戳
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())

	if len(setClauses) == 1 {
		return nil // 只有更新时间，没有其他字段
	}

	query := fmt.Sprintf("UPDATE skill_files SET %s WHERE id = ?", joinStrings(setClauses, ", "))
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新技能文件记录失败: %w", err)
	}

	return nil
}

// DeleteByProvider 删除指定 provider 的全部 Skill 记录。
func (r *SkillRepo) DeleteByProvider(provider string) error {
	if _, err := r.db.Exec("DELETE FROM skill_files WHERE provider = ?", provider); err != nil {
		return fmt.Errorf("删除 provider=%s 的技能文件失败: %w", provider, err)
	}
	return nil
}

// UpsertByName 按名称更新或创建 Skill 记录。
func (r *SkillRepo) UpsertByName(name, content, provider string, isActive bool) error {
	existing, err := r.GetByName(name)
	if err != nil {
		_, createErr := r.db.Exec(`
			INSERT INTO skill_files (name, content, provider, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, name, content, provider, isActive, time.Now(), time.Now())
		if createErr != nil {
			return fmt.Errorf("创建技能文件失败: %w", createErr)
		}
		return nil
	}

	_, err = r.db.Exec(`
		UPDATE skill_files
		SET content = ?, provider = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, content, provider, isActive, time.Now(), existing.ID)
	if err != nil {
		return fmt.Errorf("更新技能文件失败: %w", err)
	}
	return nil
}

// UpdateActive 更新技能文件的激活状态
// id 文件ID
// isActive 是否激活
// 返回错误信息
func (r *SkillRepo) UpdateActive(id int64, isActive bool) error {
	result, err := r.db.Exec(`
		UPDATE skill_files SET is_active = ?, updated_at = ? WHERE id = ?
	`, isActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新技能文件激活状态失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("技能文件记录不存在: id=%d", id)
	}

	return nil
}

// Delete 删除技能文件记录
// id 文件ID
// 返回错误信息
func (r *SkillRepo) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM skill_files WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除技能文件记录失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("技能文件记录不存在: id=%d", id)
	}

	return nil
}

// Count 获取技能文件记录总数
// provider 提供者过滤，空字符串表示不过滤
// activeOnly 是否只统计激活的记录
// 返回总数和错误信息
func (r *SkillRepo) Count(provider string, activeOnly bool) (int64, error) {
	var query string
	var args []interface{}

	if provider != "" {
		if activeOnly {
			query = "SELECT COUNT(*) FROM skill_files WHERE provider = ? AND is_active = 1"
			args = []interface{}{provider}
		} else {
			query = "SELECT COUNT(*) FROM skill_files WHERE provider = ?"
			args = []interface{}{provider}
		}
	} else {
		if activeOnly {
			query = "SELECT COUNT(*) FROM skill_files WHERE is_active = 1"
		} else {
			query = "SELECT COUNT(*) FROM skill_files"
		}
	}

	var count int64
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计技能文件记录失败: %w", err)
	}
	return count, nil
}

// scanSkills 辅助函数，扫描技能文件列表
func (r *SkillRepo) scanSkills(rows *sql.Rows) ([]*SkillFile, error) {
	var skills []*SkillFile
	for rows.Next() {
		var skill SkillFile
		var isActiveInt int
		if err := rows.Scan(&skill.ID, &skill.Name, &skill.Content, &skill.Provider, &isActiveInt, &skill.CreatedAt, &skill.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描技能文件记录失败: %w", err)
		}
		skill.IsActive = isActiveInt == 1
		skills = append(skills, &skill)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历技能文件列表失败: %w", err)
	}

	return skills, nil
}
