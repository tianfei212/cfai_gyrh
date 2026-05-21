package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultCategoryParent = "default"
	DefaultCategoryChild  = "default"
)

// BackgroundCategory 表示背景图提示词的两级分类。
type BackgroundCategory struct {
	ID         int64     `json:"id"`
	ParentName string    `json:"parent_name"`
	ChildName  string    `json:"child_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BackgroundCategoryRepo 提供背景分类及其绑定关系的操作。
type BackgroundCategoryRepo struct {
	db *DB
}

// NewBackgroundCategoryRepo 创建背景分类仓库实例。
func NewBackgroundCategoryRepo(db *DB) *BackgroundCategoryRepo {
	return &BackgroundCategoryRepo{db: db}
}

// EnsureDefault 确保 default/default 分类存在。
func (r *BackgroundCategoryRepo) EnsureDefault() (*BackgroundCategory, error) {
	existing, err := r.GetByNames(DefaultCategoryParent, DefaultCategoryChild)
	if err == nil {
		return existing, nil
	}

	id, err := r.Create(DefaultCategoryParent, DefaultCategoryChild)
	if err != nil {
		existing, getErr := r.GetByNames(DefaultCategoryParent, DefaultCategoryChild)
		if getErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("创建默认背景分类失败: %w", err)
	}

	return r.GetByID(id)
}

// EnsureDefaultBindings 将未绑定分类的背景提示词补齐到 default/default。
func (r *BackgroundCategoryRepo) EnsureDefaultBindings() error {
	defaultCategory, err := r.EnsureDefault()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT OR IGNORE INTO background_category_bindings (category_id, background_prompt_id, created_at)
		SELECT ?, bp.id, ?
		FROM background_prompts bp
		WHERE NOT EXISTS (
			SELECT 1
			FROM background_category_bindings b
			WHERE b.background_prompt_id = bp.id
		)
	`, defaultCategory.ID, time.Now())
	if err != nil {
		return fmt.Errorf("补齐默认背景分类绑定失败: %w", err)
	}

	return nil
}

// Create 创建背景分类。
func (r *BackgroundCategoryRepo) Create(parentName, childName string) (int64, error) {
	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO background_categories (parent_name, child_name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, parentName, childName, now, now)
	if err != nil {
		return 0, fmt.Errorf("插入背景分类失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}
	return id, nil
}

// Update 更新背景分类，默认分类不允许重命名。
func (r *BackgroundCategoryRepo) Update(id int64, parentName, childName string) error {
	if err := r.ensureNotDefault(id); err != nil {
		return err
	}

	result, err := r.db.Exec(`
		UPDATE background_categories
		SET parent_name = ?, child_name = ?, updated_at = ?
		WHERE id = ?
	`, parentName, childName, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新背景分类失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("背景分类不存在: id=%d", id)
	}

	return nil
}

// Delete 删除背景分类，并将失去所有分类的背景提示词重新绑定到 default/default。
func (r *BackgroundCategoryRepo) Delete(id int64) error {
	if err := r.ensureNotDefault(id); err != nil {
		return err
	}

	return r.db.Transaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(`
			SELECT background_prompt_id
			FROM background_category_bindings
			WHERE category_id = ?
		`, id)
		if err != nil {
			return fmt.Errorf("查询待删除分类绑定失败: %w", err)
		}
		defer rows.Close()

		var backgroundIDs []int64
		for rows.Next() {
			var backgroundID int64
			if err := rows.Scan(&backgroundID); err != nil {
				return fmt.Errorf("扫描待删除分类绑定失败: %w", err)
			}
			backgroundIDs = append(backgroundIDs, backgroundID)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("遍历待删除分类绑定失败: %w", err)
		}

		if _, err := tx.Exec("DELETE FROM background_category_bindings WHERE category_id = ?", id); err != nil {
			return fmt.Errorf("删除背景分类绑定失败: %w", err)
		}

		result, err := tx.Exec("DELETE FROM background_categories WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("删除背景分类失败: %w", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("获取受影响行数失败: %w", err)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("背景分类不存在: id=%d", id)
		}

		defaultID, err := r.ensureDefaultInTx(tx)
		if err != nil {
			return err
		}
		for _, backgroundID := range backgroundIDs {
			if err := r.ensureDefaultBindingForBackgroundInTx(tx, backgroundID, defaultID); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetByID 根据 ID 获取背景分类。
func (r *BackgroundCategoryRepo) GetByID(id int64) (*BackgroundCategory, error) {
	row := r.db.QueryRow(`
		SELECT id, parent_name, child_name, created_at, updated_at
		FROM background_categories
		WHERE id = ?
	`, id)

	return scanBackgroundCategory(row)
}

// GetByNames 根据父子分类名获取背景分类。
func (r *BackgroundCategoryRepo) GetByNames(parentName, childName string) (*BackgroundCategory, error) {
	row := r.db.QueryRow(`
		SELECT id, parent_name, child_name, created_at, updated_at
		FROM background_categories
		WHERE parent_name = ? AND child_name = ?
	`, parentName, childName)

	return scanBackgroundCategory(row)
}

// List 获取所有背景分类。
func (r *BackgroundCategoryRepo) List() ([]*BackgroundCategory, error) {
	rows, err := r.db.Query(`
		SELECT id, parent_name, child_name, created_at, updated_at
		FROM background_categories
		ORDER BY parent_name ASC, child_name ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询背景分类列表失败: %w", err)
	}
	defer rows.Close()

	return scanBackgroundCategories(rows)
}

// ListByBackgroundID 获取指定背景提示词绑定的分类。
func (r *BackgroundCategoryRepo) ListByBackgroundID(backgroundID int64) ([]*BackgroundCategory, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.parent_name, c.child_name, c.created_at, c.updated_at
		FROM background_categories c
		INNER JOIN background_category_bindings b ON b.category_id = c.id
		WHERE b.background_prompt_id = ?
		ORDER BY c.parent_name ASC, c.child_name ASC, c.id ASC
	`, backgroundID)
	if err != nil {
		return nil, fmt.Errorf("查询背景提示词分类失败: %w", err)
	}
	defer rows.Close()

	return scanBackgroundCategories(rows)
}

// ListByBackgroundIDs 批量获取背景提示词绑定的分类。
func (r *BackgroundCategoryRepo) ListByBackgroundIDs(backgroundIDs []int64) (map[int64][]*BackgroundCategory, error) {
	result := make(map[int64][]*BackgroundCategory, len(backgroundIDs))
	if len(backgroundIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(backgroundIDs))
	args := make([]any, len(backgroundIDs))
	for i, id := range backgroundIDs {
		placeholders[i] = "?"
		args[i] = id
		result[id] = []*BackgroundCategory{}
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT b.background_prompt_id, c.id, c.parent_name, c.child_name, c.created_at, c.updated_at
		FROM background_category_bindings b
		INNER JOIN background_categories c ON c.id = b.category_id
		WHERE b.background_prompt_id IN (%s)
		ORDER BY b.background_prompt_id ASC, c.parent_name ASC, c.child_name ASC, c.id ASC
	`, strings.Join(placeholders, ", ")), args...)
	if err != nil {
		return nil, fmt.Errorf("批量查询背景提示词分类失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var backgroundID int64
		var category BackgroundCategory
		if err := rows.Scan(
			&backgroundID,
			&category.ID,
			&category.ParentName,
			&category.ChildName,
			&category.CreatedAt,
			&category.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描背景提示词分类失败: %w", err)
		}
		result[backgroundID] = append(result[backgroundID], &category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历背景提示词分类失败: %w", err)
	}

	return result, nil
}

// ListBackgroundIDs 获取指定分类绑定的背景提示词 ID。
func (r *BackgroundCategoryRepo) ListBackgroundIDs(categoryID int64) ([]int64, error) {
	rows, err := r.db.Query(`
		SELECT background_prompt_id
		FROM background_category_bindings
		WHERE category_id = ?
		ORDER BY background_prompt_id ASC
	`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("查询分类背景提示词失败: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描分类背景提示词失败: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分类背景提示词失败: %w", err)
	}

	return ids, nil
}

// ReplaceBackgroundBindings 替换背景提示词的分类绑定。
func (r *BackgroundCategoryRepo) ReplaceBackgroundBindings(backgroundID int64, categoryIDs []int64) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM background_category_bindings WHERE background_prompt_id = ?", backgroundID); err != nil {
			return fmt.Errorf("删除背景提示词分类绑定失败: %w", err)
		}

		now := time.Now()
		for _, categoryID := range categoryIDs {
			if _, err := tx.Exec(`
				INSERT OR IGNORE INTO background_category_bindings (category_id, background_prompt_id, created_at)
				VALUES (?, ?, ?)
			`, categoryID, backgroundID, now); err != nil {
				return fmt.Errorf("插入背景提示词分类绑定失败: %w", err)
			}
		}

		if len(categoryIDs) == 0 {
			defaultID, err := r.ensureDefaultInTx(tx)
			if err != nil {
				return err
			}
			return r.ensureDefaultBindingForBackgroundInTx(tx, backgroundID, defaultID)
		}

		return nil
	})
}

// EnsureDefaultBindingForBackground 确保指定背景提示词至少绑定到 default/default。
func (r *BackgroundCategoryRepo) EnsureDefaultBindingForBackground(backgroundID int64) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		defaultID, err := r.ensureDefaultInTx(tx)
		if err != nil {
			return err
		}
		return r.ensureDefaultBindingForBackgroundInTx(tx, backgroundID, defaultID)
	})
}

func (r *BackgroundCategoryRepo) ensureNotDefault(id int64) error {
	category, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if category.ParentName == DefaultCategoryParent && category.ChildName == DefaultCategoryChild {
		return fmt.Errorf("默认背景分类不能修改或删除")
	}
	return nil
}

func (r *BackgroundCategoryRepo) ensureDefaultInTx(tx *sql.Tx) (int64, error) {
	var id int64
	err := tx.QueryRow(`
		SELECT id
		FROM background_categories
		WHERE parent_name = ? AND child_name = ?
	`, DefaultCategoryParent, DefaultCategoryChild).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("查询默认背景分类失败: %w", err)
	}

	now := time.Now()
	result, err := tx.Exec(`
		INSERT INTO background_categories (parent_name, child_name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, DefaultCategoryParent, DefaultCategoryChild, now, now)
	if err != nil {
		return 0, fmt.Errorf("创建默认背景分类失败: %w", err)
	}
	id, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %w", err)
	}

	return id, nil
}

func (r *BackgroundCategoryRepo) ensureDefaultBindingForBackgroundInTx(tx *sql.Tx, backgroundID, defaultCategoryID int64) error {
	var count int
	if err := tx.QueryRow(`
		SELECT COUNT(1)
		FROM background_category_bindings
		WHERE background_prompt_id = ?
	`, backgroundID).Scan(&count); err != nil {
		return fmt.Errorf("统计背景提示词分类绑定失败: %w", err)
	}
	if count > 0 {
		return nil
	}

	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO background_category_bindings (category_id, background_prompt_id, created_at)
		VALUES (?, ?, ?)
	`, defaultCategoryID, backgroundID, time.Now()); err != nil {
		return fmt.Errorf("插入默认背景分类绑定失败: %w", err)
	}

	return nil
}

func scanBackgroundCategory(row *sql.Row) (*BackgroundCategory, error) {
	var item BackgroundCategory
	if err := row.Scan(&item.ID, &item.ParentName, &item.ChildName, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("背景分类不存在")
		}
		return nil, fmt.Errorf("查询背景分类失败: %w", err)
	}
	return &item, nil
}

func scanBackgroundCategories(rows *sql.Rows) ([]*BackgroundCategory, error) {
	var items []*BackgroundCategory
	for rows.Next() {
		var item BackgroundCategory
		if err := rows.Scan(&item.ID, &item.ParentName, &item.ChildName, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描背景分类失败: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历背景分类失败: %w", err)
	}
	return items, nil
}
