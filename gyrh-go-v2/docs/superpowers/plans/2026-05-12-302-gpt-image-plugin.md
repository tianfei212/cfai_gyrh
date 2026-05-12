# 302 GPT Image Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `302-gpt-image` as a third Go v2 image generation provider backed by the external `302Helpper` binary plugin, with frontend model switching and default database Skill support.

**Architecture:** Go v2 treats `302Helpper` as an external HTTP plugin process, never as a source dependency. Backend config and lifecycle code start or call the plugin, `llm/router` routes `provider=302-gpt-image` into a focused HTTP client, and frontend model state cycles through Wan, Gemini, and GPT. Database support comes from a default `302-gpt-image` Skill seeded through the existing `skill_files` repository.

**Tech Stack:** Go 1.25 backend, Gorilla mux, SQLite, YAML config, React/Vite frontend, Node test runner.

---

## File Structure

- Create `gyrh-go-v2/configs/skills/302-gpt-image.md`: default GPT Image fusion prompt used by `SkillRepo.GetActive("302-gpt-image")`.
- Modify `gyrh-go-v2/backend/internal/bootstrap/bootstrap.go`: seed and reset provider list includes `302-gpt-image`.
- Modify `gyrh-go-v2/backend/internal/config/config.go`: add `Helpper302Config`, env overrides, path resolution, and validation.
- Modify `gyrh-go-v2/configs/config.yaml`: add `helpper302` defaults.
- Create `gyrh-go-v2/backend/internal/helpper302/manager.go`: lifecycle manager for the two binary files, health wait, and graceful stop.
- Create `gyrh-go-v2/backend/internal/helpper302/manager_test.go`: tests for binary selection and no-autostart behavior.
- Create `gyrh-go-v2/backend/internal/core/llm/helpper302/client.go`: HTTP client for `POST /v1/tasks`, polling `GET /v1/tasks/download`, result download, and auth header construction.
- Create `gyrh-go-v2/backend/internal/core/llm/helpper302/client_test.go`: httptest coverage for multipart fields, async polling, failures, and timeout.
- Modify `gyrh-go-v2/backend/internal/core/llm/router.go`: instantiate the client, normalize `302-gpt-image`, route compose calls, and use active `302-gpt-image` Skill.
- Modify `gyrh-go-v2/backend/internal/core/llm/router_test.go`: provider normalization and prompt resolution tests.
- Modify `gyrh-go-v2/backend/cmd/server/main.go`: start and stop `helpper302.Manager` alongside existing external managers.
- Create `gyrh-go-v2/frontend/src/utils/modelProvider.js`: single source of truth for model labels, cycling, and provider mapping.
- Create `gyrh-go-v2/frontend/src/utils/modelProvider.test.js`: frontend mapping tests.
- Modify `gyrh-go-v2/frontend/src/App.jsx`: change model cycle from two-state to three-state.
- Modify `gyrh-go-v2/frontend/src/components/Layout.jsx`: display `GPT` label when selected.
- Modify `gyrh-go-v2/frontend/src/screens/DashboardScreen.jsx`: use model helper for label/prompt preview.
- Modify `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`: send `302-gpt-image` via helper mapping.
- Modify `gyrh-go-v2/frontend/src/screens/PreviewScreen.jsx`: send `302-gpt-image` via helper mapping.
- Modify `gyrh-go-v2/frontend/src/screens/SkillManagerScreen.jsx`: add `302-gpt-image` to provider creation and filter options.

### Task 1: Seed Default 302 GPT Image Skill

**Files:**
- Create: `gyrh-go-v2/configs/skills/302-gpt-image.md`
- Modify: `gyrh-go-v2/backend/internal/bootstrap/bootstrap.go`

- [ ] **Step 1: Write the default Skill file**

Create `gyrh-go-v2/configs/skills/302-gpt-image.md`:

```markdown
你是 GPT Image 图像融合专家，负责将透明背景人物图自然融合到用户选择的背景图中。

输入包含：
- 一张已经抠像的人物 PNG，人物主体带透明通道。
- 一张背景图，代表最终画面的场景、空间、光线和氛围。

生成要求：
- 保持人物身份、五官、服装和主体姿态一致。
- 将人物自然放入背景空间，匹配透视、光线方向、色温、阴影和景深。
- 输出电影级真实感图像，不要保留透明背景，不要生成拼贴感或贴纸感。
- 画面比例优先保持 16:9 横版，适合展厅大屏展示。
- 如果人物与背景尺度不一致，优先调整人物大小和位置，让画面构图自然。
- 不要添加多余人物，不要改变背景主体结构，不要生成文字、水印或边框。
```

- [ ] **Step 2: Update bootstrap provider list**

Modify `gyrh-go-v2/backend/internal/bootstrap/bootstrap.go` by introducing a shared provider list:

```go
var defaultSkillProviders = []string{"google", "wan", "302-gpt-image"}

func (s *Service) seedSkills() error {
	for _, provider := range defaultSkillProviders {
		path, err := resolveSkillFilePath(s.cfg.Skill.LocalPath, provider)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取默认 Skill 失败: provider=%s: %w", provider, err)
		}
		if err := s.skillRepo.UpsertByName(provider+".md", string(content), provider, true); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ResetSkills() error {
	for _, provider := range defaultSkillProviders {
		if err := s.skillRepo.DeleteByProvider(provider); err != nil {
			return err
		}
	}
	return s.seedSkills()
}
```

- [ ] **Step 3: Run backend tests for bootstrap fallout**

Run from `gyrh-go-v2/backend`:

```bash
go test ./internal/bootstrap ./internal/db
```

Expected: packages either pass or report no test files; no compile errors.

### Task 2: Add 302Helpper Configuration

**Files:**
- Modify: `gyrh-go-v2/backend/internal/config/config.go`
- Modify: `gyrh-go-v2/configs/config.yaml`

- [ ] **Step 1: Add config struct and default values**

In `Config`, add:

```go
Helpper302 Helpper302Config `yaml:"helpper302"`
```

Add the struct:

```go
type Helpper302Config struct {
	Enabled             bool   `yaml:"enabled"`
	AutoStart           bool   `yaml:"auto_start"`
	BinaryDir           string `yaml:"binary_dir"`
	ConfigPath          string `yaml:"config_path"`
	BaseURL             string `yaml:"base_url"`
	AuthToken           string `yaml:"auth_token"`
	Provider            string `yaml:"provider"`
	CategoryID          string `yaml:"category_id"`
	ModelName           string `yaml:"model_name"`
	Function            string `yaml:"function"`
	Mode                string `yaml:"mode"`
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	MaxWaitSeconds      int    `yaml:"max_wait_seconds"`
	HealthWaitSeconds   int    `yaml:"health_wait_seconds"`
}
```

In `DefaultConfig()`, add:

```go
Helpper302: Helpper302Config{
	Enabled:             true,
	AutoStart:           true,
	BinaryDir:           "./backend/bin/302helpper",
	ConfigPath:          "./backend/bin/302helpper/configs/302Helpper_config.yaml",
	BaseURL:             "http://127.0.0.1:8080",
	Provider:            "302-gpt-image",
	CategoryID:          "image",
	ModelName:           "gpt-image-2",
	Function:            "image-edit",
	Mode:                "async",
	PollIntervalSeconds: 2,
	MaxWaitSeconds:      300,
	HealthWaitSeconds:   15,
},
```

- [ ] **Step 2: Add env overrides**

In `applyEnvOverrides`, add:

```go
if v := os.Getenv("GYRH_302_HELPER_ENABLED"); v != "" {
	cfg.Helpper302.Enabled = isTrue(v)
}
if v := os.Getenv("GYRH_302_HELPER_AUTO_START"); v != "" {
	cfg.Helpper302.AutoStart = isTrue(v)
}
if v := os.Getenv("GYRH_302_HELPER_BINARY_DIR"); v != "" {
	cfg.Helpper302.BinaryDir = v
}
if v := os.Getenv("GYRH_302_HELPER_CONFIG_PATH"); v != "" {
	cfg.Helpper302.ConfigPath = v
}
if v := os.Getenv("GYRH_302_HELPER_BASE_URL"); v != "" {
	cfg.Helpper302.BaseURL = v
}
if v := os.Getenv("GYRH_302_HELPER_AUTH_TOKEN"); v != "" {
	cfg.Helpper302.AuthToken = v
}
if v := os.Getenv("GYRH_302_HELPER_PROVIDER"); v != "" {
	cfg.Helpper302.Provider = v
}
if v := os.Getenv("GYRH_302_HELPER_CATEGORY_ID"); v != "" {
	cfg.Helpper302.CategoryID = v
}
if v := os.Getenv("GYRH_302_HELPER_MODEL_NAME"); v != "" {
	cfg.Helpper302.ModelName = v
}
if v := os.Getenv("GYRH_302_HELPER_FUNCTION"); v != "" {
	cfg.Helpper302.Function = v
}
if v := os.Getenv("GYRH_302_HELPER_MODE"); v != "" {
	cfg.Helpper302.Mode = v
}
if v := os.Getenv("GYRH_302_HELPER_POLL_INTERVAL_SECONDS"); v != "" {
	value, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("GYRH_302_HELPER_POLL_INTERVAL_SECONDS 无效: %s", v)
	}
	cfg.Helpper302.PollIntervalSeconds = value
}
if v := os.Getenv("GYRH_302_HELPER_MAX_WAIT_SECONDS"); v != "" {
	value, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("GYRH_302_HELPER_MAX_WAIT_SECONDS 无效: %s", v)
	}
	cfg.Helpper302.MaxWaitSeconds = value
}
if v := os.Getenv("GYRH_302_HELPER_HEALTH_WAIT_SECONDS"); v != "" {
	value, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("GYRH_302_HELPER_HEALTH_WAIT_SECONDS 无效: %s", v)
	}
	cfg.Helpper302.HealthWaitSeconds = value
}
```

- [ ] **Step 3: Resolve paths and validate**

In `resolveConfigPaths`, add:

```go
cfg.Helpper302.BinaryDir = resolvePath(rootDir, cfg.Helpper302.BinaryDir)
cfg.Helpper302.ConfigPath = resolvePath(rootDir, cfg.Helpper302.ConfigPath)
```

In `validateConfig`, add:

```go
if cfg.Helpper302.Enabled {
	if strings.TrimSpace(cfg.Helpper302.BaseURL) == "" {
		return fmt.Errorf("helpper302.base_url 不能为空")
	}
	if strings.TrimSpace(cfg.Helpper302.Provider) == "" {
		return fmt.Errorf("helpper302.provider 不能为空")
	}
	if strings.TrimSpace(cfg.Helpper302.CategoryID) == "" {
		return fmt.Errorf("helpper302.category_id 不能为空")
	}
	if strings.TrimSpace(cfg.Helpper302.ModelName) == "" {
		return fmt.Errorf("helpper302.model_name 不能为空")
	}
	if strings.TrimSpace(cfg.Helpper302.Function) == "" {
		return fmt.Errorf("helpper302.function 不能为空")
	}
	if cfg.Helpper302.Mode != "async" && cfg.Helpper302.Mode != "sync" {
		return fmt.Errorf("helpper302.mode 仅支持 async 或 sync")
	}
	if cfg.Helpper302.PollIntervalSeconds <= 0 {
		cfg.Helpper302.PollIntervalSeconds = 2
	}
	if cfg.Helpper302.MaxWaitSeconds <= 0 {
		cfg.Helpper302.MaxWaitSeconds = 300
	}
	if cfg.Helpper302.HealthWaitSeconds <= 0 {
		cfg.Helpper302.HealthWaitSeconds = 15
	}
}
```

- [ ] **Step 4: Add YAML config**

Append to `gyrh-go-v2/configs/config.yaml`:

```yaml
helpper302:
  enabled: true
  auto_start: true
  binary_dir: ./backend/bin/302helpper
  config_path: ./backend/bin/302helpper/configs/302Helpper_config.yaml
  base_url: http://127.0.0.1:8080
  auth_token: ""
  provider: 302-gpt-image
  category_id: image
  model_name: gpt-image-2
  function: image-edit
  mode: async
  poll_interval_seconds: 2
  max_wait_seconds: 300
  health_wait_seconds: 15
```

- [ ] **Step 5: Run config compile tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/config
```

Expected: pass or no test files; no compile errors.

### Task 3: Add Plugin Lifecycle Manager

**Files:**
- Create: `gyrh-go-v2/backend/internal/helpper302/manager.go`
- Create: `gyrh-go-v2/backend/internal/helpper302/manager_test.go`
- Modify: `gyrh-go-v2/backend/cmd/server/main.go`

- [ ] **Step 1: Write manager tests**

Create `manager_test.go` with tests for binary names:

```go
package helpper302

import "testing"

func TestPlatformBinaryName(t *testing.T) {
	tests := []struct {
		goos string
		arch string
		want string
	}{
		{"linux", "amd64", "server-linux-amd64"},
		{"linux", "arm64", "server-linux-arm64"},
		{"darwin", "arm64", ""},
	}
	for _, tt := range tests {
		if got := platformBinaryName(tt.goos, tt.arch); got != tt.want {
			t.Fatalf("platformBinaryName(%q, %q) = %q, want %q", tt.goos, tt.arch, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Implement manager**

Create `manager.go`:

```go
package helpper302

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"gyrh-go-v2/backend/internal/config"
)

type Manager struct {
	cfg *config.Helpper302Config
	cmd *exec.Cmd
}

func NewManager(cfg *config.Helpper302Config) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) Start(ctx context.Context) error {
	if m == nil || m.cfg == nil || !m.cfg.Enabled || !m.cfg.AutoStart {
		return nil
	}
	binary, err := resolveBinaryPath(m.cfg.BinaryDir)
	if err != nil {
		return err
	}
	cmd := exec.Command(binary)
	cmd.Dir = m.cfg.BinaryDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PROVIDER_302_API_KEY="+os.Getenv("PROVIDER_302_API_KEY"))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 302Helpper 插件失败: %w", err)
	}
	m.cmd = cmd
	deadline := time.Now().Add(time.Duration(m.cfg.HealthWaitSeconds) * time.Second)
	for time.Now().Before(deadline) {
		if err := m.ping(ctx); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("等待 302Helpper 插件就绪超时")
}

func (m *Manager) Stop(ctx context.Context) error {
	if m == nil || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	_ = m.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- m.cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" {
			return err
		}
		return nil
	case <-ctx.Done():
		_ = m.cmd.Process.Kill()
		return ctx.Err()
	}
}

func (m *Manager) ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.BaseURL+"/v1/models", nil)
	if err != nil {
		return err
	}
	if m.cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+m.cfg.AuthToken)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return nil
	}
	return fmt.Errorf("302Helpper 状态码异常: %d", resp.StatusCode)
}

func resolveBinaryPath(binaryDir string) (string, error) {
	name := platformBinaryName(runtime.GOOS, runtime.GOARCH)
	if name == "" {
		return "", fmt.Errorf("302Helpper 不支持当前平台: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	path := filepath.Join(binaryDir, name)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path, nil
	}
	return "", fmt.Errorf("未找到 302Helpper 二进制: %s", path)
}

func platformBinaryName(goos, arch string) string {
	if goos == "linux" && arch == "amd64" {
		return "server-linux-amd64"
	}
	if goos == "linux" && arch == "arm64" {
		return "server-linux-arm64"
	}
	return ""
}
```

- [ ] **Step 3: Wire manager into server main**

In `gyrh-go-v2/backend/cmd/server/main.go`, import:

```go
helpper302manager "gyrh-go-v2/backend/internal/helpper302"
```

After existing external manager startup, add:

```go
helpper302Manager := helpper302manager.NewManager(&cfg.Helpper302)
if err := helpper302Manager.Start(ctx); err != nil {
	logger.Fatal("启动 302Helpper 插件失败: %v", err)
}
```

Pass `helpper302Manager` to shutdown handling by changing its signature to accept a small interface or by stopping it separately before final log:

```go
if err := helpper302Manager.Stop(shutdownCtx); err != nil {
	logger.Error("关闭 302Helpper 插件失败: %v", err)
}
```

- [ ] **Step 4: Run manager tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/helpper302
```

Expected: pass.

### Task 4: Add 302 GPT Image HTTP Client

**Files:**
- Create: `gyrh-go-v2/backend/internal/core/llm/helpper302/client.go`
- Create: `gyrh-go-v2/backend/internal/core/llm/helpper302/client_test.go`

- [ ] **Step 1: Write httptest coverage**

Create tests that assert a compose request sends multipart fields and polls result:

```go
package helpper302

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gyrh-go-v2/backend/internal/config"
)

func TestComposeAsyncSuccess(t *testing.T) {
	var sawTask bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/tasks":
			sawTask = true
			if r.Header.Get("X-Category-Id") != "image" {
				t.Fatalf("missing category header")
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			if got := r.FormValue("function"); got != "image-edit" {
				t.Fatalf("function = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 200,
				"data": map[string]any{"task_id": "task-1"},
			})
		case "/v1/tasks/download":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 200,
				"data": []map[string]any{{"url": serverImageURL(t)}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(&config.Helpper302Config{
		Enabled: true, BaseURL: server.URL, CategoryID: "image", ModelName: "gpt-image-2",
		Function: "image-edit", Mode: "async", PollIntervalSeconds: 1, MaxWaitSeconds: 3,
	})
	result, err := client.Compose(context.Background(), ComposeRequest{
		Prompt: "test prompt",
		Image:  []byte("fake-png"),
	})
	if err != nil {
		t.Fatalf("Compose error: %v", err)
	}
	if !sawTask || len(result.Image) == 0 {
		t.Fatalf("expected task and image result")
	}
}
```

In the actual test, implement `serverImageURL` with a nested `httptest.Server` that returns `Content-Type: image/png` and bytes.

- [ ] **Step 2: Implement client types**

Create:

```go
package helpper302

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"gyrh-go-v2/backend/internal/config"
)

type Client struct {
	cfg        *config.Helpper302Config
	httpClient *http.Client
}

type ComposeRequest struct {
	Prompt string
	Image  []byte
}

type ComposeResult struct {
	Image []byte
	URL   string
}

func NewClient(cfg *config.Helpper302Config) *Client {
	return &Client{cfg: cfg, httpClient: &http.Client{Timeout: time.Duration(cfg.MaxWaitSeconds+30) * time.Second}}
}
```

- [ ] **Step 3: Implement multipart task creation**

Add:

```go
func (c *Client) createTask(ctx context.Context, req ComposeRequest) (string, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("category_id", c.cfg.CategoryID)
	_ = writer.WriteField("model_name", c.cfg.ModelName)
	_ = writer.WriteField("function", c.cfg.Function)
	requestJSON, _ := json.Marshal(map[string]any{
		"prompt":          req.Prompt,
		"model":           c.cfg.ModelName,
		"quality":         "high",
		"size":            "1536x1024",
		"n":               1,
		"background":      "auto",
		"output_format":   "png",
		"input_fidelity":  "high",
	})
	_ = writer.WriteField("request_json", string(requestJSON))
	part, err := writer.CreateFormFile("image", "character.png")
	if err != nil {
		return "", "", err
	}
	if _, err := part.Write(req.Image); err != nil {
		return "", "", err
	}
	if err := writer.Close(); err != nil {
		return "", "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/v1/tasks", &body)
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-Category-Id", c.cfg.CategoryID)
	httpReq.Header.Set("X-Model-Name", c.cfg.ModelName)
	c.setAuth(httpReq)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("302Helpper 创建任务失败: status=%d body=%s", resp.StatusCode, string(data))
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", "", err
	}
	if envelope.Data.TaskID == "" {
		return "", "", fmt.Errorf("302Helpper 未返回 task_id")
	}
	return envelope.Data.TaskID, string(data), nil
}
```

- [ ] **Step 4: Implement polling and download**

Add:

```go
func (c *Client) Compose(ctx context.Context, req ComposeRequest) (*ComposeResult, error) {
	if c.cfg == nil || !c.cfg.Enabled {
		return nil, fmt.Errorf("302Helpper provider 未启用")
	}
	taskID, _, err := c.createTask(ctx, req)
	if err != nil {
		return nil, err
	}
	url, err := c.pollDownloadURL(ctx, taskID)
	if err != nil {
		return nil, err
	}
	image, err := c.downloadImage(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ComposeResult{Image: image, URL: url}, nil
}

func (c *Client) pollDownloadURL(ctx context.Context, taskID string) (string, error) {
	deadline := time.Now().Add(time.Duration(c.cfg.MaxWaitSeconds) * time.Second)
	for time.Now().Before(deadline) {
		url, done, err := c.fetchDownloadURL(ctx, taskID)
		if err != nil {
			return "", err
		}
		if done {
			return url, nil
		}
		time.Sleep(time.Duration(c.cfg.PollIntervalSeconds) * time.Second)
	}
	return "", fmt.Errorf("302-gpt-image 任务超时")
}

func (c *Client) fetchDownloadURL(ctx context.Context, taskID string) (string, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/v1/tasks/download?task_id="+taskID, nil)
	if err != nil {
		return "", false, err
	}
	httpReq.Header.Set("X-Category-Id", c.cfg.CategoryID)
	httpReq.Header.Set("X-Model-Name", c.cfg.ModelName)
	c.setAuth(httpReq)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusAccepted {
		return "", false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("302Helpper 任务下载失败: status=%d body=%s", resp.StatusCode, string(data))
	}
	var envelope struct {
		Code int `json:"code"`
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", false, err
	}
	if envelope.Code == 202 {
		return "", false, nil
	}
	if len(envelope.Data) == 0 || envelope.Data[0].URL == "" {
		return "", false, fmt.Errorf("302Helpper 未返回结果 URL")
	}
	return envelope.Data[0].URL, true, nil
}

func (c *Client) downloadImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 302-gpt-image 结果失败: status=%d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("302-gpt-image 结果不是图片: %s", contentType)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) setAuth(req *http.Request) {
	if c.cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AuthToken)
	}
}
```

Remember to import `strings`.

- [ ] **Step 5: Run client tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/core/llm/helpper302
```

Expected: pass.

### Task 5: Route 302 GPT Image Through LLM Service

**Files:**
- Modify: `gyrh-go-v2/backend/internal/core/llm/router.go`
- Modify: `gyrh-go-v2/backend/internal/core/llm/router_test.go`

- [ ] **Step 1: Add failing normalization test**

In `router_test.go`, add:

```go
func TestNormalizeProvider302GPTImage(t *testing.T) {
	if got := normalizeProvider("302-gpt-image"); got != "302-gpt-image" {
		t.Fatalf("normalizeProvider returned %q", got)
	}
}
```

- [ ] **Step 2: Update service struct and constructor**

In `router.go`, import:

```go
helpper302llm "gyrh-go-v2/backend/internal/core/llm/helpper302"
```

Add field:

```go
helpper302Client *helpper302llm.Client
```

In `NewService`, initialize:

```go
helpper302Client: helpper302llm.NewClient(&cfg.Helpper302),
```

- [ ] **Step 3: Add provider switch case**

Before the default Gemini case:

```go
case "302-gpt-image":
	characterAsset := firstImageByType(params.Images, ImageTypeCharacter)
	if characterAsset == "" {
		return nil, fmt.Errorf("302-gpt-image 需要人物图")
	}
	imageBytes, err := s.storageService.Read(ctx, characterAsset)
	if err != nil {
		return nil, fmt.Errorf("读取 302-gpt-image 人物图失败: %w", err)
	}
	result, err := s.helpper302Client.Compose(ctx, helpper302llm.ComposeRequest{
		Prompt: resolved.Prompt,
		Image:  imageBytes,
	})
	if err != nil {
		logger.Error("302-gpt-image Compose Error: %v", err)
		return nil, err
	}
	return &ComposeResult{
		Base64: base64.StdEncoding.EncodeToString(result.Image),
		Status: "succeeded",
	}, nil
```

Add `encoding/base64` import if absent.

- [ ] **Step 4: Normalize provider**

Update `normalizeProvider`:

```go
case "302-gpt-image", "gpt-image", "gpt":
	return "302-gpt-image"
```

- [ ] **Step 5: Make prompt resolution explicit**

At the start of `buildPrompt`, after `resolved` creation:

```go
if provider == "302-gpt-image" {
	if s.skillRepo == nil {
		return nil, fmt.Errorf("Skill 仓库未初始化，无法获取 302-gpt-image 融合提示词")
	}
	skill, err := s.skillRepo.GetActive(provider)
	if err != nil {
		return nil, fmt.Errorf("未找到当前模型的激活 Skill: provider=%s: %w", provider, err)
	}
	resolved.Prompt = strings.TrimSpace(skill.Content)
	logger.Debug("背景提示词来源: active skill id=%d, provider=%s, name=%s", skill.ID, provider, skill.Name)
	return resolved, nil
}
```

- [ ] **Step 6: Run router tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/core/llm
```

Expected: pass.

### Task 6: Add Frontend Model Mapping

**Files:**
- Create: `gyrh-go-v2/frontend/src/utils/modelProvider.js`
- Create: `gyrh-go-v2/frontend/src/utils/modelProvider.test.js`
- Modify: `gyrh-go-v2/frontend/src/App.jsx`
- Modify: `gyrh-go-v2/frontend/src/components/Layout.jsx`

- [ ] **Step 1: Add mapping utility**

Create `modelProvider.js`:

```javascript
export const MODEL_SEQUENCE = ['W', 'G', 'GPT'];

export const MODEL_LABELS = {
  W: 'W',
  G: 'G',
  GPT: 'GPT'
};

export const MODEL_PROVIDERS = {
  W: 'wan',
  G: 'google',
  GPT: '302-gpt-image'
};

export function getNextModel(current) {
  const index = MODEL_SEQUENCE.indexOf(current);
  if (index === -1) return MODEL_SEQUENCE[0];
  return MODEL_SEQUENCE[(index + 1) % MODEL_SEQUENCE.length];
}

export function getModelLabel(model) {
  return MODEL_LABELS[model] || MODEL_LABELS.W;
}

export function getProviderForModel(model) {
  return MODEL_PROVIDERS[model] || MODEL_PROVIDERS.W;
}

export function isGPTModel(model) {
  return model === 'GPT';
}
```

- [ ] **Step 2: Add tests**

Create `modelProvider.test.js`:

```javascript
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { getNextModel, getProviderForModel, getModelLabel } from './modelProvider.js';

test('cycles Wan Gemini GPT', () => {
  assert.equal(getNextModel('W'), 'G');
  assert.equal(getNextModel('G'), 'GPT');
  assert.equal(getNextModel('GPT'), 'W');
});

test('maps models to backend providers', () => {
  assert.equal(getProviderForModel('W'), 'wan');
  assert.equal(getProviderForModel('G'), 'google');
  assert.equal(getProviderForModel('GPT'), '302-gpt-image');
});

test('returns display labels', () => {
  assert.equal(getModelLabel('GPT'), 'GPT');
});
```

- [ ] **Step 3: Update App toggle**

In `App.jsx`, import:

```javascript
import { getNextModel } from './utils/modelProvider.js';
```

Replace toggle logic with:

```javascript
const handleToggleModel = () => {
  setModel((current) => {
    const next = getNextModel(current);
    console.log(`[App] Toggle model: ${next}`);
    return next;
  });
};
```

- [ ] **Step 4: Update header label**

In `Layout.jsx`, import:

```javascript
import { getModelLabel } from '../utils/modelProvider.js';
```

Replace:

```jsx
<HeaderIcon label={model === 'W' ? 'W' : 'G'} onClick={onToggleModel} />
```

with:

```jsx
<HeaderIcon label={getModelLabel(model)} onClick={onToggleModel} />
```

- [ ] **Step 5: Run frontend utility tests**

Run:

```bash
cd gyrh-go-v2/frontend && npm test -- modelProvider
```

Expected: modelProvider tests pass. If the repo test script does not accept a filter, run the existing frontend test command and confirm all tests pass.

### Task 7: Send 302 Provider From Generation Screens

**Files:**
- Modify: `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`
- Modify: `gyrh-go-v2/frontend/src/screens/PreviewScreen.jsx`
- Modify: `gyrh-go-v2/frontend/src/screens/DashboardScreen.jsx`

- [ ] **Step 1: Update CaptureScreen provider**

Import:

```javascript
import { getProviderForModel } from '../utils/modelProvider.js';
```

Replace:

```javascript
provider: model === 'W' ? 'wan' : 'google'
```

with:

```javascript
provider: getProviderForModel(model)
```

- [ ] **Step 2: Update PreviewScreen provider**

Make the same import and replacement in `PreviewScreen.jsx`.

- [ ] **Step 3: Update Dashboard header and prompt preview**

Import:

```javascript
import { getModelLabel, isGPTModel } from '../utils/modelProvider.js';
```

Replace Header label logic with `getModelLabel(model)`.

For card prompt title, replace:

```javascript
title={model === 'W' ? card.wan_prompt : card.gemini_prompt}
```

with:

```javascript
title={isGPTModel(model) ? '302 GPT Image 通用融合 Skill' : model === 'W' ? card.wan_prompt : card.gemini_prompt}
```

- [ ] **Step 4: Run frontend tests**

Run:

```bash
cd gyrh-go-v2/frontend && npm test
```

Expected: existing frontend tests pass.

### Task 8: Add GPT Provider To Skill Manager

**Files:**
- Modify: `gyrh-go-v2/frontend/src/screens/SkillManagerScreen.jsx`

- [ ] **Step 1: Define provider options**

Near the top of the file, add:

```javascript
const SKILL_PROVIDER_OPTIONS = [
  { value: 'wan', label: 'Wan' },
  { value: 'gemini', label: 'Gemini' },
  { value: '302-gpt-image', label: '302 GPT Image' }
];
```

- [ ] **Step 2: Replace hardcoded form options**

Replace:

```jsx
<option value="wan">Wan</option>
<option value="gemini">Gemini</option>
```

with:

```jsx
{SKILL_PROVIDER_OPTIONS.map((option) => (
  <option key={option.value} value={option.value}>{option.label}</option>
))}
```

Do this for both the edit form provider select and the filter provider select.

- [ ] **Step 3: Update create default provider**

Replace:

```javascript
setEditingItem({ provider: filterProvider || 'wan' })
```

with:

```javascript
setEditingItem({ provider: filterProvider || 'wan' })
```

This remains unchanged intentionally; new Skills default to Wan unless the filter is set to `302-gpt-image`.

- [ ] **Step 4: Run frontend tests**

Run:

```bash
cd gyrh-go-v2/frontend && npm test
```

Expected: pass.

### Task 9: Full Verification

**Files:**
- All changed files

- [ ] **Step 1: Run backend tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./...
```

Expected: all packages pass. If tests try to start missing external binaries, set `GYRH_302_HELPER_ENABLED=false` for tests and document why.

- [ ] **Step 2: Run frontend tests**

Run:

```bash
cd gyrh-go-v2/frontend && npm test
```

Expected: all tests pass.

- [ ] **Step 3: Check lints in edited files**

Use IDE diagnostics for:

```text
gyrh-go-v2/backend/internal/config/config.go
gyrh-go-v2/backend/internal/helpper302/manager.go
gyrh-go-v2/backend/internal/core/llm/helpper302/client.go
gyrh-go-v2/backend/internal/core/llm/router.go
gyrh-go-v2/frontend/src/utils/modelProvider.js
gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx
gyrh-go-v2/frontend/src/screens/PreviewScreen.jsx
gyrh-go-v2/frontend/src/screens/DashboardScreen.jsx
gyrh-go-v2/frontend/src/screens/SkillManagerScreen.jsx
```

Expected: no new diagnostics.

- [ ] **Step 4: Manual smoke test with plugin files present**

Prepare:

```text
gyrh-go-v2/backend/bin/302helpper/server-linux-amd64
gyrh-go-v2/backend/bin/302helpper/server-linux-arm64
gyrh-go-v2/backend/bin/302helpper/configs/302Helpper_config.yaml
```

Set `PROVIDER_302_API_KEY` and the selected plugin Bearer variables in `gyrh-go-v2/.env.local`.

Start backend and frontend. In the UI, click the model button until it shows `GPT`, generate with a local or gallery background, and confirm:

```text
request provider = 302-gpt-image
generated_images.provider = 302-gpt-image
history shows the generated image
```

---

## Self-Review

- Spec coverage: plan covers binary-only plugin lifecycle, Go v2 config, `.env.local` key source, `302-gpt-image` LLM routing, async polling, default Skill database support, frontend model switching, Skill Manager provider options, and verification.
- Placeholder scan: no `TBD`, `TODO`, or vague "handle errors" steps remain; code steps include concrete snippets and commands.
- Type consistency: provider name is consistently `302-gpt-image`; config name is consistently `Helpper302Config`; frontend model key is consistently `GPT`; backend package paths are distinct for lifecycle manager and LLM client.
