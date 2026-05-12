package rewrite

import "fmt"

const (
	// StatusPending 表示改写任务已创建但尚未开始执行。
	StatusPending = "pending"
	// StatusRunning 表示改写任务正在执行或等待上游模型完成。
	StatusRunning = "running"
	// StatusCompleted 表示改写任务已经生成结果图并完成入库。
	StatusCompleted = "completed"
	// StatusFailed 表示改写任务执行失败。
	StatusFailed = "failed"
)

// Task 表示一次图片改写任务的领域状态。
type Task struct {
	ID            string
	Status        string
	Provider      string
	ResultAssetID string
	ErrorMessage  string
}

// MarkRunning 将任务切换到运行中状态。
func (t *Task) MarkRunning() error {
	if t.Status != StatusPending && t.Status != "" {
		return fmt.Errorf("任务状态不能从 %s 切换到运行中", t.Status)
	}
	t.Status = StatusRunning
	t.ErrorMessage = ""
	return nil
}

// MarkCompleted 将运行中的任务标记为完成，并记录生成结果资源 ID。
func (t *Task) MarkCompleted(resultAssetID string) error {
	if t.Status != StatusRunning {
		return fmt.Errorf("任务状态不能从 %s 切换到完成", t.Status)
	}
	if resultAssetID == "" {
		return fmt.Errorf("完成任务必须提供结果资源 ID")
	}
	t.Status = StatusCompleted
	t.ResultAssetID = resultAssetID
	t.ErrorMessage = ""
	return nil
}

// MarkFailed 将任务标记为失败，并记录可展示的错误信息。
func (t *Task) MarkFailed(message string) {
	t.Status = StatusFailed
	t.ErrorMessage = message
}
