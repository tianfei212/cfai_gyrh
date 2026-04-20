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
	db.SetMaxOpenConns(25)                  // 最大打开连接数
	db.SetMaxIdleConns(5)                   // 最大空闲连接数
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
			is_upscale INTEGER NOT NULL DEFAULT 0,
			style_transform TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 generated_images 表失败: %w", err)
	}

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

	// 创建索引以提高查询性能
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_generated_images_created_at ON generated_images(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_reference_images_image_type ON reference_images(image_type)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_files_provider ON skill_files(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_files_is_active ON skill_files(is_active)`,
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
