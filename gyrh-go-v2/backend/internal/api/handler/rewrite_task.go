package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"gyrh-go-v2/backend/internal/core/llm"
	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
)

type rewriteTaskStatus string

const (
	rewriteTaskRunning   rewriteTaskStatus = "running"
	rewriteTaskSucceeded rewriteTaskStatus = "succeeded"
	rewriteTaskFailed    rewriteTaskStatus = "failed"
)

type rewriteTask struct {
	ID        string            `json:"task_id"`
	Status    rewriteTaskStatus `json:"status"`
	Response  *RewriteResponse  `json:"response,omitempty"`
	Error     string            `json:"error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	done      chan struct{}
}

func rewriteTaskFromDB(task *db.RewriteTask) rewriteTask {
	status := rewriteTaskStatus(task.Status)
	item := rewriteTask{
		ID:        task.TaskID,
		Status:    status,
		Error:     task.Error,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}
	if task.ImageID > 0 || task.AssetID != "" || task.ImageURL != "" {
		item.Response = &RewriteResponse{
			Success:  status == rewriteTaskSucceeded,
			ID:       task.ImageID,
			AssetID:  task.AssetID,
			ImageURL: task.ImageURL,
			Style:    task.StyleName,
			Status:   task.Status,
			Message:  "图像改写成功",
		}
	}
	return item
}

func (t *rewriteTask) snapshot() rewriteTask {
	return rewriteTask{
		ID:        t.ID,
		Status:    t.Status,
		Response:  t.Response,
		Error:     t.Error,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

type rewriteTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*rewriteTask
}

func newRewriteTaskStore() *rewriteTaskStore {
	return &rewriteTaskStore{tasks: make(map[string]*rewriteTask)}
}

func (s *rewriteTaskStore) create() *rewriteTask {
	now := time.Now()
	task := &rewriteTask{
		ID:        fmt.Sprintf("rewrite_%d", now.UnixNano()),
		Status:    rewriteTaskRunning,
		CreatedAt: now,
		UpdatedAt: now,
		done:      make(chan struct{}),
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()
	return task
}

func (s *rewriteTaskStore) restore(task rewriteTask) *rewriteTask {
	taskCopy := task
	taskCopy.done = make(chan struct{})
	if taskCopy.Status != rewriteTaskRunning {
		close(taskCopy.done)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.tasks[taskCopy.ID]; ok {
		return existing
	}
	s.tasks[taskCopy.ID] = &taskCopy
	return &taskCopy
}

func (s *rewriteTaskStore) get(id string) (*rewriteTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

func (s *rewriteTaskStore) snapshot(id string) (rewriteTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return rewriteTask{}, false
	}
	return task.snapshot(), true
}

func (s *rewriteTaskStore) doneChan(id string) (<-chan struct{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return nil, false
	}
	return task.done, true
}

func writeSSEEvent(w http.ResponseWriter, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		payload = []byte(`{"error":"事件序列化失败"}`)
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}

func (s *rewriteTaskStore) complete(id string, resp RewriteResponse) {
	s.update(id, rewriteTaskSucceeded, &resp, "")
}

func (s *rewriteTaskStore) fail(id string, err error) {
	msg := "任务失败"
	if err != nil {
		msg = err.Error()
	}
	s.update(id, rewriteTaskFailed, nil, msg)
}

func (s *rewriteTaskStore) update(id string, status rewriteTaskStatus, resp *RewriteResponse, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return
	}
	task.Status = status
	task.Response = resp
	task.Error = errMsg
	task.UpdatedAt = time.Now()
	select {
	case <-task.done:
	default:
		close(task.done)
	}
}

type rewriteJob struct {
	Request            RewriteRequest
	StylePrompt        string
	StyleName          string
	Inputs             []llm.ImageInput
	BackgroundPromptID int64
	ExternalTaskID     string
}

func (h *ImageHandler) runAsyncRewrite(taskID string, job rewriteJob) {
	ctx := context.Background()
	if len(job.Inputs) == 0 && job.ExternalTaskID == "" {
		inputs, err := h.prepareLLMInputs(ctx, job.Request)
		if err != nil {
			logger.Error("准备异步图像改写输入失败: task_id=%s, err=%v", taskID, err)
			if h.rewriteTaskRepo != nil {
				_ = h.rewriteTaskRepo.Fail(taskID, err.Error())
			}
			h.rewriteTasks.fail(taskID, err)
			return
		}
		if persistErr := h.persistBackgroundImage(job.Request, inputs); persistErr != nil {
			logger.Error("异步背景图入库失败: task_id=%s, err=%v", taskID, persistErr)
			if h.rewriteTaskRepo != nil {
				_ = h.rewriteTaskRepo.Fail(taskID, persistErr.Error())
			}
			h.rewriteTasks.fail(taskID, persistErr)
			return
		}
		job.Inputs = inputs
	}
	if job.Request.Provider == "302-gpt-image" && job.ExternalTaskID == "" {
		started, err := h.llmService.StartCompose(ctx, llm.ComposeParams{
			Provider:           job.Request.Provider,
			StylePrompt:        job.StylePrompt,
			Images:             job.Inputs,
			BackgroundPromptID: job.BackgroundPromptID,
		})
		if err != nil {
			logger.Error("创建 302-gpt-image 异步任务失败: task_id=%s, err=%v", taskID, err)
			if h.rewriteTaskRepo != nil {
				_ = h.rewriteTaskRepo.Fail(taskID, err.Error())
			}
			h.rewriteTasks.fail(taskID, err)
			return
		}
		job.ExternalTaskID = started.ExternalTaskID
		if h.rewriteTaskRepo != nil {
			if err := h.rewriteTaskRepo.SetExternalTaskID(taskID, started.ExternalTaskID); err != nil {
				logger.Error("保存 302-gpt-image 外部任务ID失败: %v", err)
			}
		}
	}
	resp, err := h.waitAndPersistRewrite(ctx, job)
	if err != nil {
		logger.Error("异步图像改写失败: task_id=%s, err=%v", taskID, err)
		if h.rewriteTaskRepo != nil {
			_ = h.rewriteTaskRepo.Fail(taskID, err.Error())
		}
		h.rewriteTasks.fail(taskID, err)
		return
	}
	if h.rewriteTaskRepo != nil {
		_ = h.rewriteTaskRepo.Complete(taskID, db.RewriteTaskResult{
			ImageID:  resp.ID,
			AssetID:  resp.AssetID,
			ImageURL: resp.ImageURL,
		})
	}
	h.rewriteTasks.complete(taskID, resp)
	logger.Info("异步图像改写完成: task_id=%s, image_id=%d, asset_id=%s", taskID, resp.ID, resp.AssetID)
}
