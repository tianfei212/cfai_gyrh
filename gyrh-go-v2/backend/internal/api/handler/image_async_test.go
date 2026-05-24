package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	done    chan struct{}
}

func (s *blockingLLMService) Compose(ctx context.Context, params llm.ComposeParams) (*llm.ComposeResult, error) {
	defer closeIfPresent(s.done)
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
	defer closeIfPresent(s.done)
	if s.waited != nil {
		s.waited <- externalTaskID
	}
	<-s.release
	return &llm.ComposeResult{
		Base64: base64.StdEncoding.EncodeToString(tinyPNGBytes()),
		Status: "succeeded",
	}, nil
}

func closeIfPresent(ch chan struct{}) {
	if ch != nil {
		close(ch)
	}
}

func releaseAndWaitForBlockingRewrite(t *testing.T, svc *blockingLLMService) {
	t.Helper()
	close(svc.release)
	select {
	case <-svc.done:
	case <-time.After(time.Second):
		t.Fatal("expected async rewrite task to finish")
	}
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
		done:    make(chan struct{}),
	}

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
	releaseAndWaitForBlockingRewrite(t, llmSvc)
}

func TestRewriteGoogleReturnsTaskImmediately(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	storageSvc := &fakeStorageService{}
	llmSvc := &blockingLLMService{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}

	handler := NewImageHandler(db.NewImageRepo(database), nil, nil, storageSvc, llmSvc)
	body := `{"provider":"google","foreground":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `","background":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `"}`
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
		t.Fatalf("expected task id for google rewrite, got %+v", resp.Data)
	}
	select {
	case <-llmSvc.started:
	case <-time.After(time.Second):
		t.Fatalf("expected async google compose to start")
	}
	releaseAndWaitForBlockingRewrite(t, llmSvc)
}

func TestRewriteReturnsTaskBeforeUploadingForeground(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	storageSvc := &fakeStorageService{
		saveStarted: make(chan struct{}),
		saveRelease: make(chan struct{}),
	}
	llmSvc := &capturingLLMService{}
	handler := NewImageHandler(db.NewImageRepo(database), nil, nil, storageSvc, llmSvc)
	body := `{"provider":"google","foreground":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `","background":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/rewrite", strings.NewReader(body))
	w := httptest.NewRecorder()

	if err := handler.Rewrite(context.Background(), w, req); err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	taskID := decodeRewriteTaskID(t, w.Body.Bytes())
	select {
	case <-storageSvc.saveStarted:
	case <-time.After(time.Second):
		t.Fatalf("expected background task to start uploading foreground")
	}
	if len(storageSvc.saved) != 0 {
		t.Fatalf("rewrite response waited for storage save, saved len=%d", len(storageSvc.saved))
	}
	close(storageSvc.saveRelease)
	select {
	case <-mustRewriteDoneChan(t, handler, taskID):
	case <-time.After(time.Second):
		t.Fatal("expected async rewrite task to finish")
	}
}

func TestRewriteWanReturnsTaskImmediately(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	storageSvc := &fakeStorageService{}
	llmSvc := &blockingLLMService{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}

	handler := NewImageHandler(db.NewImageRepo(database), nil, nil, storageSvc, llmSvc)
	body := `{"provider":"wan","foreground":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `","background":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `"}`
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
		t.Fatalf("expected task id for wan rewrite, got %+v", resp.Data)
	}
	select {
	case <-llmSvc.started:
	case <-time.After(time.Second):
		t.Fatalf("expected async wan compose to start")
	}
	releaseAndWaitForBlockingRewrite(t, llmSvc)
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

func TestRewriteUsesBackgroundPromptAssetWithoutBackgroundBase64(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	backgroundRepo := db.NewBackgroundPromptRepo(database)
	backgroundID, err := backgroundRepo.Create(
		"cached background",
		"gemini prompt",
		"",
		"",
		"",
		"wan prompt",
		"",
		"",
		"",
		"gpt prompt",
		"",
		"",
		"",
		"asset:bg-existing",
		"https://oss.example.com/bg.png",
		1280,
		720,
	)
	if err != nil {
		t.Fatalf("create background prompt: %v", err)
	}

	llmSvc := &capturingLLMService{}
	handler := NewImageHandler(db.NewImageRepo(database), backgroundRepo, nil, &fakeStorageService{}, llmSvc)
	body := `{"provider":"google","foreground":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `","background_prompt_id":` + jsonNumber(backgroundID) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/rewrite", strings.NewReader(body))
	w := httptest.NewRecorder()

	if err := handler.Rewrite(context.Background(), w, req); err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}

	taskID := decodeRewriteTaskID(t, w.Body.Bytes())
	if _, ok := handler.rewriteTasks.doneChan(taskID); !ok {
		t.Fatalf("expected in-memory rewrite task %q", taskID)
	}
	select {
	case <-mustRewriteDoneChan(t, handler, taskID):
	case <-time.After(time.Second):
		t.Fatal("expected async rewrite task to finish")
	}

	if !containsImageInput(llmSvc.params.Images, llm.ImageTypeBackground, "asset:bg-existing") {
		t.Fatalf("expected existing background asset in compose inputs, got %+v", llmSvc.params.Images)
	}
}

func TestRewritePersistsStyleNameOnGeneratedImage(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	styleRepo := db.NewStylePromptRepo(database)
	styleID, err := styleRepo.Create("漫画转绘", "comic style prompt", "", true)
	if err != nil {
		t.Fatalf("create style prompt: %v", err)
	}

	imageRepo := db.NewImageRepo(database)
	handler := NewImageHandler(imageRepo, nil, styleRepo, &fakeStorageService{}, &capturingLLMService{})
	body := `{"provider":"google","foreground":"` + base64.StdEncoding.EncodeToString(tinyPNGBytes()) + `","style_prompt_id":` + jsonNumber(styleID) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/rewrite", strings.NewReader(body))
	w := httptest.NewRecorder()

	if err := handler.Rewrite(context.Background(), w, req); err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	taskID := decodeRewriteTaskID(t, w.Body.Bytes())
	select {
	case <-mustRewriteDoneChan(t, handler, taskID):
	case <-time.After(time.Second):
		t.Fatal("expected async rewrite task to finish")
	}

	task, ok := handler.rewriteTasks.snapshot(taskID)
	if !ok || task.Response == nil {
		t.Fatalf("expected completed task response, got task=%+v ok=%t", task, ok)
	}
	if task.Response.Style != "漫画转绘" {
		t.Fatalf("response style = %q, want 漫画转绘", task.Response.Style)
	}

	img, err := imageRepo.GetByID(task.Response.ID)
	if err != nil {
		t.Fatalf("get generated image: %v", err)
	}
	if img.StyleTransform != "漫画转绘" {
		t.Fatalf("style_transform = %q, want 漫画转绘", img.StyleTransform)
	}
	if img.Provider != "google" {
		t.Fatalf("provider = %q, want google", img.Provider)
	}
}

func TestThumbnailSetsThreeMinuteCacheHeader(t *testing.T) {
	handler := NewImageHandler(nil, nil, nil, &fakeStorageService{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/thumbnail?asset_id=asset:bg&w=400&h=225", nil)
	w := httptest.NewRecorder()

	if err := handler.Thumbnail(context.Background(), w, req); err != nil {
		t.Fatalf("Thumbnail returned error: %v", err)
	}

	if got := w.Header().Get("Cache-Control"); got != "public, max-age=180" {
		t.Fatalf("Cache-Control = %q", got)
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
		done:    make(chan struct{}),
	}
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
	releaseAndWaitForBlockingRewrite(t, llmSvc)
}

func decodeRewriteTaskID(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		Data RewriteResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode rewrite response: %v", err)
	}
	if resp.Data.TaskID == "" {
		t.Fatalf("expected task id in response: %s", string(body))
	}
	return resp.Data.TaskID
}

func mustRewriteDoneChan(t *testing.T, handler *ImageHandler, taskID string) <-chan struct{} {
	t.Helper()
	done, ok := handler.rewriteTasks.doneChan(taskID)
	if !ok {
		t.Fatalf("missing task %q", taskID)
	}
	return done
}

type capturingLLMService struct {
	params llm.ComposeParams
}

func (s *capturingLLMService) Compose(ctx context.Context, params llm.ComposeParams) (*llm.ComposeResult, error) {
	s.params = params
	return &llm.ComposeResult{
		Base64: base64.StdEncoding.EncodeToString(tinyPNGBytes()),
		Status: "succeeded",
	}, nil
}

func (s *capturingLLMService) StartCompose(ctx context.Context, params llm.ComposeParams) (*llm.StartedComposeTask, error) {
	s.params = params
	return &llm.StartedComposeTask{ExternalTaskID: "external-task-1"}, nil
}

func (s *capturingLLMService) WaitComposeResult(ctx context.Context, provider, externalTaskID string) (*llm.ComposeResult, error) {
	return &llm.ComposeResult{
		Base64: base64.StdEncoding.EncodeToString(tinyPNGBytes()),
		Status: "succeeded",
	}, nil
}

func containsImageInput(inputs []llm.ImageInput, imageType llm.ImageType, assetID string) bool {
	for _, input := range inputs {
		if input.Type == imageType && input.AssetID == assetID {
			return true
		}
	}
	return false
}

func jsonNumber(value int64) string {
	return strconv.FormatInt(value, 10)
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
