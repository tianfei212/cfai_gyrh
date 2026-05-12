# 302 GPT Direct Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the local `302Helpper` CLI gateway with an in-process Go client that calls 302.ai GPT Image directly using `PROVIDER_302_API_KEY`.

**Architecture:** Add a focused `backend/internal/302Helpper/GPT` package for GPT Image edits, keep the existing `302-gpt-image` provider contract in `llm/router`, and remove server startup/shutdown wiring for the local `302Helpper` manager. Existing rewrite tasks, SSE events, storage, and image history persistence remain in place.

**Tech Stack:** Go 1.25, standard `net/http`, `mime/multipart`, `encoding/json`, `httptest`, existing YAML config loader.

---

## File Structure

- Create `gyrh-go-v2/backend/internal/302Helpper/GPT/client.go`: direct 302.ai GPT Image client, async submit, polling, image download, API key handling.
- Create `gyrh-go-v2/backend/internal/302Helpper/GPT/client_test.go`: unit tests for multipart submit, async polling success, upstream failure, missing API key.
- Modify `gyrh-go-v2/backend/internal/core/llm/router.go`: replace the old local gateway client import and calls with the new GPT direct client.
- Modify `gyrh-go-v2/backend/cmd/server/main.go`: remove `internal/helpper302` manager startup and shutdown wiring.
- Modify `gyrh-go-v2/backend/internal/config/config.go`: make `helpper302` config describe direct provider settings instead of CLI lifecycle settings; remove loading of `302Helpper_config.yaml`.
- Modify `gyrh-go-v2/configs/config.yaml`: remove local CLI fields and set `base_url: https://api.302.ai`.
- Delete `gyrh-go-v2/backend/internal/helpper302/manager.go` and `gyrh-go-v2/backend/internal/helpper302/manager_test.go`: no local CLI manager remains.
- Update or remove `gyrh-go-v2/backend/internal/core/llm/helpper302/client.go` and its test: old gateway client should no longer be imported. Prefer deleting after the new package is wired.

### Task 1: Configuration Cleanup

**Files:**
- Modify: `gyrh-go-v2/backend/internal/config/config.go`
- Modify: `gyrh-go-v2/configs/config.yaml`
- Test: `gyrh-go-v2/backend/internal/config/config_test.go`

- [ ] **Step 1: Update config tests first**

In `config_test.go`, change the 302 defaults test to expect direct settings:

```go
func TestDefaultConfigIncludesHelpper302(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Helpper302.Enabled {
		t.Fatalf("Helpper302 should be enabled by default")
	}
	if cfg.Helpper302.Provider != "302-gpt-image" {
		t.Fatalf("Provider = %q, want 302-gpt-image", cfg.Helpper302.Provider)
	}
	if cfg.Helpper302.BaseURL != "https://api.302.ai" {
		t.Fatalf("BaseURL = %q, want https://api.302.ai", cfg.Helpper302.BaseURL)
	}
	if cfg.Helpper302.ModelName != "gpt-image-2" {
		t.Fatalf("ModelName = %q, want gpt-image-2", cfg.Helpper302.ModelName)
	}
}
```

Change env override tests to keep only direct fields:

```go
t.Setenv("GYRH_302_HELPER_ENABLED", "false")
t.Setenv("GYRH_302_HELPER_BASE_URL", "https://example.test")
t.Setenv("GYRH_302_HELPER_MODEL_NAME", "gpt-image-1.5")
t.Setenv("GYRH_302_HELPER_MAX_WAIT_SECONDS", "12")
```

- [ ] **Step 2: Run config tests and verify they fail**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/config
```

Expected before implementation: failure because defaults still point at local `http://127.0.0.1:19080` and old fields are still present.

- [ ] **Step 3: Update `Helpper302Config`**

In `config.go`, reduce the struct to fields still used by direct provider:

```go
type Helpper302Config struct {
	Enabled             bool   `yaml:"enabled"`
	BaseURL             string `yaml:"base_url"`
	Provider            string `yaml:"provider"`
	ModelName           string `yaml:"model_name"`
	Mode                string `yaml:"mode"`
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	MaxWaitSeconds      int    `yaml:"max_wait_seconds"`
}
```

Update defaults:

```go
Helpper302: Helpper302Config{
	Enabled:             true,
	BaseURL:             "https://api.302.ai",
	Provider:            "302-gpt-image",
	ModelName:           "gpt-image-2",
	Mode:                "async",
	PollIntervalSeconds: 2,
	MaxWaitSeconds:      300,
},
```

Remove `loadHelpper302Config`, `helpper302File`, `ConfigPath` path resolution, and env override branches for old CLI fields.

- [ ] **Step 4: Update config validation**

Keep validation for `base_url`, `provider`, `model_name`, `mode`, `poll_interval_seconds`, and `max_wait_seconds`. Remove validation for `category_id`, `function`, and `health_wait_seconds`.

- [ ] **Step 5: Update YAML config**

Change `configs/config.yaml` helpper302 block to:

```yaml
helpper302:
  enabled: true
  base_url: https://api.302.ai
  provider: 302-gpt-image
  model_name: gpt-image-2
  mode: async
  poll_interval_seconds: 2
  max_wait_seconds: 300
```

- [ ] **Step 6: Run config tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/config
```

Expected: PASS.

### Task 2: Direct GPT Client

**Files:**
- Create: `gyrh-go-v2/backend/internal/302Helpper/GPT/client.go`
- Create: `gyrh-go-v2/backend/internal/302Helpper/GPT/client_test.go`

- [ ] **Step 1: Write client tests**

Create tests that use `httptest.Server` and `t.Setenv("PROVIDER_302_API_KEY", "test-key")`. The create-task test should assert:

```go
if r.URL.Path != "/v1/images/edits" { t.Fatalf(...) }
if r.URL.Query().Get("response_format") != "url" { t.Fatalf(...) }
if r.URL.Query().Get("async") != "true" { t.Fatalf(...) }
if r.Header.Get("Authorization") != "Bearer test-key" { t.Fatalf(...) }
if err := r.ParseMultipartForm(10 << 20); err != nil { t.Fatalf(...) }
if got := r.FormValue("prompt"); got != "test prompt" { t.Fatalf(...) }
if got := r.FormValue("model"); got != "gpt-image-2" { t.Fatalf(...) }
if len(r.MultipartForm.File["image"]) != 1 { t.Fatalf(...) }
```

The wait-result success test should return `{"status_code":200,"data":"<image-url>"}` from `/async_result`, serve an image at the image URL, and assert `result.Image` equals the bytes.

The upstream failure test should return `{"status_code":500,"err":"bad upstream"}` and expect an error containing `bad upstream`.

The missing key test should clear `PROVIDER_302_API_KEY` and expect an error containing `PROVIDER_302_API_KEY`.

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/302Helpper/GPT
```

Expected: fail because package does not exist.

- [ ] **Step 3: Implement `client.go`**

Implement:

```go
package GPT

type Config struct {
	Enabled             bool
	BaseURL             string
	ModelName           string
	PollIntervalSeconds int
	MaxWaitSeconds      int
}
```

`NewClient` trims `BaseURL`, defaults it to `https://api.302.ai`, creates `http.Client` timeout of `MaxWaitSeconds + 30` seconds, and stores defaults.

`CreateTask` validates enabled config, API key, prompt, and foreground image; builds multipart form to `/v1/images/edits?response_format=url&async=true`; writes fields `image`, `prompt`, `model`, `n`, `quality`, `background`, `output_format`, `moderation`, `input_fidelity`, `size`; parses `task_id`, `job_id`, or `id` from JSON.

`WaitResult` polls `/async_result?task_id=...` until success URL, upstream error, context cancellation, or timeout. Success parser accepts `data` string URL or `data.data[0].url`.

`downloadImage` GETs the result URL and requires HTTP 200 and image content-type when present.

- [ ] **Step 4: Run direct client tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/302Helpper/GPT
```

Expected: PASS.

### Task 3: Wire LLM Router to Direct Client

**Files:**
- Modify: `gyrh-go-v2/backend/internal/core/llm/router.go`
- Modify: `gyrh-go-v2/backend/internal/core/llm/router_test.go`
- Delete: `gyrh-go-v2/backend/internal/core/llm/helpper302/client.go`
- Delete: `gyrh-go-v2/backend/internal/core/llm/helpper302/client_test.go`

- [ ] **Step 1: Update imports and service field**

Replace:

```go
helpper302llm "gyrh-go-v2/backend/internal/core/llm/helpper302"
```

with:

```go
gpt302 "gyrh-go-v2/backend/internal/302Helpper/GPT"
```

Change service field:

```go
helpper302Client *gpt302.Client
```

Initialize:

```go
helpper302Client: gpt302.NewClient(gpt302.Config{
	Enabled:             cfg.Helpper302.Enabled,
	BaseURL:             cfg.Helpper302.BaseURL,
	ModelName:           cfg.Helpper302.ModelName,
	PollIntervalSeconds: cfg.Helpper302.PollIntervalSeconds,
	MaxWaitSeconds:      cfg.Helpper302.MaxWaitSeconds,
}),
```

- [ ] **Step 2: Update method calls**

Replace request type usage:

```go
gpt302.ComposeRequest{
	Prompt:          resolved.Prompt,
	ForegroundImage: foregroundBytes,
	BackgroundImage: backgroundBytes,
}
```

Keep `StartCompose`, `WaitComposeResult`, and sync `Compose` public behavior unchanged.

- [ ] **Step 3: Remove old gateway client files**

Delete old `internal/core/llm/helpper302` package after router no longer imports it.

- [ ] **Step 4: Run LLM tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/core/llm
```

Expected: PASS after tests are updated for direct client semantics.

### Task 4: Remove CLI Manager Startup

**Files:**
- Modify: `gyrh-go-v2/backend/cmd/server/main.go`
- Delete: `gyrh-go-v2/backend/internal/helpper302/manager.go`
- Delete: `gyrh-go-v2/backend/internal/helpper302/manager_test.go`

- [ ] **Step 1: Remove server import**

Delete:

```go
helpper302manager "gyrh-go-v2/backend/internal/helpper302"
```

- [ ] **Step 2: Remove manager construction and start**

Delete:

```go
helpper302Manager := helpper302manager.NewManager(&cfg.Helpper302)
if startErr := helpper302Manager.Start(context.Background()); startErr != nil {
	logger.Fatal("启动 302Helpper 插件失败: %v", startErr)
}
```

- [ ] **Step 3: Update shutdown call**

Change:

```go
waitForShutdown(ctx, cancel, server, helpper302Manager, aliOSSManager, generatedOSSManager)
```

to:

```go
waitForShutdown(ctx, cancel, server, aliOSSManager, generatedOSSManager)
```

- [ ] **Step 4: Delete manager package**

Delete `backend/internal/helpper302/manager.go` and `manager_test.go`.

- [ ] **Step 5: Run server package tests/build**

Run:

```bash
cd gyrh-go-v2/backend && go test ./cmd/server
```

Expected: PASS or `[no test files]` with successful compile.

### Task 5: Full Verification and Cleanup

**Files:**
- Check: all modified Go files
- Check: `gyrh-go-v2/configs/config.yaml`
- Check: `gyrh-go-v2/docs/superpowers/specs/2026-05-12-302-gpt-direct-integration-design.md`

- [ ] **Step 1: Run targeted tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./internal/config ./internal/302Helpper/GPT ./internal/core/llm ./cmd/server
```

Expected: PASS.

- [ ] **Step 2: Run broader backend tests**

Run:

```bash
cd gyrh-go-v2/backend && go test ./...
```

Expected: PASS. If unrelated tests fail due to existing dirty workspace state, record exact package and failure.

- [ ] **Step 3: Run lints in Cursor**

Use IDE diagnostics for:

```text
gyrh-go-v2/backend/internal/302Helpper/GPT/client.go
gyrh-go-v2/backend/internal/config/config.go
gyrh-go-v2/backend/internal/core/llm/router.go
gyrh-go-v2/backend/cmd/server/main.go
```

Expected: no new diagnostics.

- [ ] **Step 4: Report changed behavior**

Final response should mention:

- Branch remains `feature/302-gpt-direct-integration`.
- `302-gpt-image` now calls `api.302.ai` directly with `PROVIDER_302_API_KEY`.
- Local `302Helpper` CLI startup has been removed.
- Which tests were run and whether any failed.

