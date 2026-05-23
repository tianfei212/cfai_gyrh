package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB 数据库连接结构体
type DB struct {
	*sql.DB
}

// NewDB 创建数据库连接
// dataSourceName 数据库文件路径
func NewDB(dataSourceName string) (*DB, error) {
	// 确保数据库目录存在
	dbDir := filepath.Dir(dataSourceName)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 连接 SQLite 数据库
	db, err := sql.Open("sqlite3", dataSourceName+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=10000")
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生命周期

	// 验证数据库连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 启用 WAL 模式
	if err := enableWALMode(db); err != nil {
		return nil, fmt.Errorf("启用 WAL 模式失败: %w", err)
	}

	// 自动迁移表结构
	if err := migrateTables(db); err != nil {
		return nil, fmt.Errorf("迁移表结构失败: %w", err)
	}

	return &DB{DB: db}, nil
}

// enableWALMode 启用 WAL 模式
// WAL (Write-Ahead Logging) 模式可以提高并发读写性能
func enableWALMode(db *sql.DB) error {
	_, err := db.Exec("PRAGMA journal_mode=WAL")
	return err
}

// migrateTables 自动迁移表结构
// 如果表不存在则创建，如果表已存在则跳过
func migrateTables(db *sql.DB) error {
	// 创建 generated_images 表（生成的图像记录）
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS generated_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			asset_id TEXT NOT NULL DEFAULT '',
			is_upscale INTEGER NOT NULL DEFAULT 0,
			style_transform TEXT NOT NULL DEFAULT '',
			provider TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'succeeded',
			background_prompt_id INTEGER NOT NULL DEFAULT 0,
			image_width INTEGER NOT NULL DEFAULT 0,
			image_height INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 generated_images 表失败: %w", err)
	}

	_, _ = db.Exec("ALTER TABLE generated_images ADD COLUMN asset_id TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE generated_images ADD COLUMN provider TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE generated_images ADD COLUMN status TEXT NOT NULL DEFAULT 'succeeded'")
	_, _ = db.Exec("ALTER TABLE generated_images ADD COLUMN background_prompt_id INTEGER NOT NULL DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE generated_images ADD COLUMN image_width INTEGER NOT NULL DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE generated_images ADD COLUMN image_height INTEGER NOT NULL DEFAULT 0")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS image_rewrite_tasks (
			task_id TEXT PRIMARY KEY,
			external_task_id TEXT NOT NULL DEFAULT '',
			provider TEXT NOT NULL DEFAULT '',
			style_name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'running',
			background_prompt_id INTEGER NOT NULL DEFAULT 0,
			image_id INTEGER NOT NULL DEFAULT 0,
			asset_id TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 image_rewrite_tasks 表失败: %w", err)
	}
	_, _ = db.Exec("ALTER TABLE image_rewrite_tasks ADD COLUMN style_name TEXT NOT NULL DEFAULT ''")

	// 创建 reference_images 表（参考图像记录）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS reference_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			image_type TEXT NOT NULL DEFAULT 'general',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 reference_images 表失败: %w", err)
	}

	// 创建 skill_files 表（技能文件记录）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS skill_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			content TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 skill_files 表失败: %w", err)
	}

	// 创建 background_prompts 表（背景图提示词模板）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS background_prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			gemini_prompt TEXT NOT NULL DEFAULT '',
			gemini_negative_prompt TEXT NOT NULL DEFAULT '',
			wan_prompt TEXT NOT NULL DEFAULT '',
			wan_negative_prompt TEXT NOT NULL DEFAULT '',
			gpt_prompt TEXT NOT NULL DEFAULT '',
			gpt_negative_prompt TEXT NOT NULL DEFAULT '',
			image_asset_id TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			image_width INTEGER NOT NULL DEFAULT 0,
			image_height INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 background_prompts 表失败: %w", err)
	}

	// 自动升级字段
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN image_asset_id TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN image_url TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN image_width INTEGER NOT NULL DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN image_height INTEGER NOT NULL DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN gemini_prompt_zh TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN gemini_negative_prompt_zh TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN wan_prompt_zh TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN wan_negative_prompt_zh TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN gpt_prompt TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN gpt_negative_prompt TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN gpt_prompt_zh TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE background_prompts ADD COLUMN gpt_negative_prompt_zh TEXT NOT NULL DEFAULT ''")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS background_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_name TEXT NOT NULL,
			child_name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建背景分类表失败: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS background_category_bindings (
			category_id INTEGER NOT NULL,
			background_prompt_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (category_id, background_prompt_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("创建背景分类绑定表失败: %w", err)
	}

	// 创建 llm_prompt_templates 表（大模型提示词模板）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS llm_prompt_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			template_key TEXT NOT NULL,
			content TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 llm_prompt_templates 表失败: %w", err)
	}

	// 创建 style_prompts 表（风格转换提示词模板）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS style_prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			prompt TEXT NOT NULL,
			negative_prompt TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 style_prompts 表失败: %w", err)
	}

	// 创建索引以提高查询性能
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_generated_images_created_at ON generated_images(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_reference_images_image_type ON reference_images(image_type)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_files_provider ON skill_files(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_files_is_active ON skill_files(is_active)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_background_prompts_name ON background_prompts(name)`,
		`CREATE INDEX IF NOT EXISTS idx_background_prompts_updated_at ON background_prompts(updated_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_background_categories_parent_child ON background_categories(parent_name, child_name)`,
		`CREATE INDEX IF NOT EXISTS idx_background_category_bindings_background ON background_category_bindings(background_prompt_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_prompt_templates_key ON llm_prompt_templates(template_key)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_prompt_templates_updated_at ON llm_prompt_templates(updated_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_style_prompts_name ON style_prompts(name)`,
		`CREATE INDEX IF NOT EXISTS idx_style_prompts_is_active ON style_prompts(is_active)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.DB.Close()
}

// Transaction 执行事务
// fn 事务回调函数，如果返回错误则回滚事务
func (db *DB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("事务回滚失败: %w (原错误: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("事务提交失败: %w", err)
	}

	return nil
}
