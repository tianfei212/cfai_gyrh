package db

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	RewriteTaskStatusRunning   = "running"
	RewriteTaskStatusSucceeded = "succeeded"
	RewriteTaskStatusFailed    = "failed"
)

type RewriteTask struct {
	TaskID             string
	ExternalTaskID     string
	Provider           string
	Status             string
	BackgroundPromptID int64
	ImageID            int64
	AssetID            string
	ImageURL           string
	Error              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type RewriteTaskCreateInput struct {
	TaskID             string
	Provider           string
	BackgroundPromptID int64
}

type RewriteTaskResult struct {
	ImageID  int64
	AssetID  string
	ImageURL string
}

type RewriteTaskRepo struct {
	db *DB
}

func NewRewriteTaskRepo(db *DB) *RewriteTaskRepo {
	return &RewriteTaskRepo{db: db}
}

func (r *RewriteTaskRepo) Create(input RewriteTaskCreateInput) error {
	now := time.Now()
	_, err := r.db.Exec(`
		INSERT INTO image_rewrite_tasks (
			task_id, provider, status, background_prompt_id, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`, input.TaskID, input.Provider, RewriteTaskStatusRunning, input.BackgroundPromptID, now, now)
	if err != nil {
		return fmt.Errorf("创建图像改写任务失败: %w", err)
	}
	return nil
}

func (r *RewriteTaskRepo) SetExternalTaskID(taskID, externalTaskID string) error {
	return r.update(taskID, `
		external_task_id = ?,
		updated_at = ?
	`, externalTaskID, time.Now())
}

func (r *RewriteTaskRepo) Complete(taskID string, result RewriteTaskResult) error {
	return r.update(taskID, `
		status = ?,
		image_id = ?,
		asset_id = ?,
		image_url = ?,
		error = '',
		updated_at = ?
	`, RewriteTaskStatusSucceeded, result.ImageID, result.AssetID, result.ImageURL, time.Now())
}

func (r *RewriteTaskRepo) Fail(taskID string, errMsg string) error {
	return r.update(taskID, `
		status = ?,
		error = ?,
		updated_at = ?
	`, RewriteTaskStatusFailed, errMsg, time.Now())
}

func (r *RewriteTaskRepo) Get(taskID string) (*RewriteTask, error) {
	row := r.db.QueryRow(`
		SELECT task_id, external_task_id, provider, status, background_prompt_id,
			image_id, asset_id, image_url, error, created_at, updated_at
		FROM image_rewrite_tasks
		WHERE task_id = ?
	`, taskID)
	task, err := scanRewriteTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("图像改写任务不存在: %s", taskID)
		}
		return nil, fmt.Errorf("查询图像改写任务失败: %w", err)
	}
	return task, nil
}

func (r *RewriteTaskRepo) ListRunningWithExternalTaskIDs() ([]*RewriteTask, error) {
	rows, err := r.db.Query(`
		SELECT task_id, external_task_id, provider, status, background_prompt_id,
			image_id, asset_id, image_url, error, created_at, updated_at
		FROM image_rewrite_tasks
		WHERE status = ? AND external_task_id != ''
		ORDER BY created_at ASC
	`, RewriteTaskStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("查询运行中的图像改写任务失败: %w", err)
	}
	defer rows.Close()

	tasks := make([]*RewriteTask, 0)
	for rows.Next() {
		task, err := scanRewriteTask(rows)
		if err != nil {
			return nil, fmt.Errorf("解析运行中的图像改写任务失败: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历运行中的图像改写任务失败: %w", err)
	}
	return tasks, nil
}

func (r *RewriteTaskRepo) update(taskID string, setClause string, args ...any) error {
	args = append(args, taskID)
	result, err := r.db.Exec("UPDATE image_rewrite_tasks SET "+setClause+" WHERE task_id = ?", args...)
	if err != nil {
		return fmt.Errorf("更新图像改写任务失败: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取任务更新行数失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("图像改写任务不存在: %s", taskID)
	}
	return nil
}

func scanRewriteTask(row interface {
	Scan(dest ...any) error
}) (*RewriteTask, error) {
	var task RewriteTask
	if err := row.Scan(
		&task.TaskID,
		&task.ExternalTaskID,
		&task.Provider,
		&task.Status,
		&task.BackgroundPromptID,
		&task.ImageID,
		&task.AssetID,
		&task.ImageURL,
		&task.Error,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &task, nil
}
