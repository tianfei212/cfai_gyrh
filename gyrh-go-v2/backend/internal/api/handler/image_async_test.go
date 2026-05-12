package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gyrh-go-v2/backend/internal/core/llm"
	"gyrh-go-v2/backend/internal/db"

	"github.com/gorilla/mux"
)

type blockingLLMService struct {
	started chan struct{}
	release chan struct{}
	waited  chan string
}

func (s *blockingLLMService) Compose(ctx context.Context, params llm.ComposeParams) (*llm.ComposeResult, error) {
	close(s.started)
	<-s.release
	return &llm.ComposeResult{
		Base64: base64.StdEncoding.EncodeToString(tinyPNGBytes()),
		Status: "succeeded",
	}, nil
}

func (s *blockingLLMService) StartCompose(ctx context.Context, params llm.ComposeParams) (*llm.StartedComposeTask, error) {
	close(s.started)
	return &llm.StartedComposeTask{ExternalTaskID: "external-task-1"}, nil
}

func (s *blockingLLMService) WaitComposeResult(ctx context.Context, provider, externalTaskID string) (*llm.ComposeResult, error) {
	if s.waited != nil {
		s.waited <- externalTaskID
	}
	<-s.release
	return &llm.ComposeResult{
		Base64: base64.StdEncoding.EncodeToString(tinyPNGBytes()),
		Status: "succeeded",
	}, nil
}

func TestRewrite302GPTImageReturnsTaskImmediately(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	storageSvc := &fakeStorageService{}
	llmSvc := &blockingLLMService{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	defer close(llmSvc.release)

	handler := NewImageHandler(db.NewImageRepo(database), nil, nil, storageSvc, llmSvc)
	body := `{"provider":"302-gpt-image","foreground":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `","background":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/rewrite", strings.NewReader(body))
	w := httptest.NewRecorder()

	if err := handler.Rewrite(context.Background(), w, req); err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int             `json:"code"`
		Data RewriteResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.TaskID == "" {
		t.Fatalf("expected async task id, got response: %+v", resp.Data)
	}
	if resp.Data.Status != "running" {
		t.Fatalf("status = %q, want running", resp.Data.Status)
	}
	select {
	case <-llmSvc.started:
	case <-time.After(time.Second):
		t.Fatalf("expected background goroutine to start compose")
	}
}

func TestRewriteTaskStatusReturnsCompletedResult(t *testing.T) {
	handler := NewImageHandler(nil, nil, nil, &fakeStorageService{}, nil)
	task := handler.rewriteTasks.create()
	handler.rewriteTasks.complete(task.ID, RewriteResponse{
		Success:  true,
		ImageURL: "https://oss.example.com/generated.png",
		Status:   "succeeded",
		Message:  "图像改写成功",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/rewrite/tasks/"+task.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": task.ID})
	w := httptest.NewRecorder()

	if err := handler.RewriteTask(context.Background(), w, req); err != nil {
		t.Fatalf("RewriteTask returned error: %v", err)
	}

	var resp struct {
		Code int         `json:"code"`
		Data rewriteTask `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Status != rewriteTaskSucceeded {
		t.Fatalf("status = %q, want succeeded", resp.Data.Status)
	}
	if resp.Data.Response == nil || resp.Data.Response.ImageURL == "" {
		t.Fatalf("expected completed response image url, got %+v", resp.Data.Response)
	}
}

func TestRewriteTaskStatusFallsBackToPersistedResult(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repo := db.NewRewriteTaskRepo(database)
	if err := repo.Create(db.RewriteTaskCreateInput{
		TaskID:   "rewrite_persisted",
		Provider: "302-gpt-image",
	}); err != nil {
		t.Fatalf("create persisted task: %v", err)
	}
	if err := repo.SetExternalTaskID("rewrite_persisted", "external-1"); err != nil {
		t.Fatalf("set external task id: %v", err)
	}
	if err := repo.Complete("rewrite_persisted", db.RewriteTaskResult{
		ImageID:  27,
		AssetID:  "generated:rewrite_1.png",
		ImageURL: "https://oss.example.com/rewrite_1.png",
	}); err != nil {
		t.Fatalf("complete persisted task: %v", err)
	}

	handler := NewImageHandler(nil, nil, nil, &fakeStorageService{}, nil, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/rewrite/tasks/rewrite_persisted", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rewrite_persisted"})
	w := httptest.NewRecorder()

	if err := handler.RewriteTask(context.Background(), w, req); err != nil {
		t.Fatalf("RewriteTask returned error: %v", err)
	}

	var resp struct {
		Code int         `json:"code"`
		Data rewriteTask `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Status != rewriteTaskSucceeded {
		t.Fatalf("status = %q, want succeeded", resp.Data.Status)
	}
	if resp.Data.Response == nil || resp.Data.Response.AssetID != "generated:rewrite_1.png" {
		t.Fatalf("expected persisted response, got %+v", resp.Data.Response)
	}
}

func TestRewriteTaskStatusRestartsRunningPersistedTask(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repo := db.NewRewriteTaskRepo(database)
	if err := repo.Create(db.RewriteTaskCreateInput{
		TaskID:   "rewrite_running",
		Provider: "302-gpt-image",
	}); err != nil {
		t.Fatalf("create persisted task: %v", err)
	}
	if err := repo.SetExternalTaskID("rewrite_running", "external-running"); err != nil {
		t.Fatalf("set external task id: %v", err)
	}

	llmSvc := &blockingLLMService{
		started: make(chan struct{}),
		release: make(chan struct{}),
		waited:  make(chan string, 1),
	}
	defer close(llmSvc.release)
	handler := NewImageHandler(db.NewImageRepo(database), nil, nil, &fakeStorageService{}, llmSvc, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/rewrite/tasks/rewrite_running", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rewrite_running"})
	w := httptest.NewRecorder()

	if err := handler.RewriteTask(context.Background(), w, req); err != nil {
		t.Fatalf("RewriteTask returned error: %v", err)
	}

	select {
	case got := <-llmSvc.waited:
		if got != "external-running" {
			t.Fatalf("external task id = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected persisted running task to restart background wait")
	}
}

func tinyPNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0xf8, 0xff, 0xff, 0x3f,
		0x00, 0x05, 0xfe, 0x02, 0xfe, 0xdc, 0xcc, 0x59,
		0xe7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
