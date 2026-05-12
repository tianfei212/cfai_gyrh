# Go Version Decoupling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Go 后端重组为清晰分层架构，并把前端拆成 `/` 一体机演示入口与 `/admin_viewer` 管理入口，同时补齐中文注释、详细日志、2K 触控屏样式和 `CHANGELOG`。

**Architecture:** 后端采用 `api / application / domain / infrastructure / platform` 分层，先建立兼容适配层，再逐步迁移图片、背景、Skill、风格和模型调用。前端使用轻量路由入口拆分，先复用现有页面能力，再裁剪演示端导航并抽离主题样式。

**Tech Stack:** Go 1.25、Gorilla mux、SQLite、React 18、Vite、CSS Modules-style split by file（继续使用全局 CSS import，不引入 CSS-in-JS）。

---

## File Structure Map

- Create: `backend/internal/platform/app/app.go`，负责服务装配和生命周期。
- Create: `backend/internal/platform/app/alioss.go`，负责 aliOSS 运行配置生成。
- Create: `backend/internal/platform/logging/sanitize.go`，负责日志脱敏。
- Create: `backend/internal/domain/model/model.go`，定义模型 provider 接口与请求响应。
- Create: `backend/internal/application/image/service.go`，承接图片列表、访问、上传、改写的业务编排。
- Create: `backend/internal/infrastructure/model/router.go`，作为旧 `core/llm` 到新模型分层的兼容入口。
- Modify: `backend/cmd/server/main.go`，瘦身为启动入口。
- Modify: `backend/internal/api/handler/image.go`，逐步把业务编排委托给 application service。
- Modify: `backend/internal/api/router.go`，增加路由注册日志和保持现有 API。
- Modify: `backend/internal/logger/logger.go`，补充脱敏辅助或与 `platform/logging` 对接。
- Create: `frontend/src/app/AppShell.jsx`，承载共享页面状态和 screen 渲染。
- Create: `frontend/src/app/routes.jsx`，根据路径选择 kiosk/admin 入口。
- Create: `frontend/src/pages/kiosk/KioskViewer.jsx`，`/` 演示入口。
- Create: `frontend/src/pages/admin/AdminViewer.jsx`，`/admin_viewer` 管理入口。
- Modify: `frontend/src/App.jsx`，改为路由入口。
- Modify: `frontend/src/components/Layout.jsx`，让 `ControlRail` 接收菜单列表，支持演示端隐藏管理菜单。
- Modify: `frontend/src/constants/index.js`，拆分 `adminScreens` 和 `kioskScreens`。
- Create: `frontend/src/theme/tokens.css`，抽离颜色、字号、间距、圆角和阴影。
- Create: `frontend/src/theme/glass.css`，抽离玻璃拟态按钮和面板。
- Create: `frontend/src/theme/kiosk.css`，抽离 2K 触控屏演示端样式。
- Create: `frontend/src/theme/admin.css`，抽离管理端样式。
- Modify: `frontend/src/main.jsx`，导入新的主题样式文件。
- Modify: `frontend/src/styles.css`，保留基础布局并逐步移出主题样式。
- Modify/Create: `CHANGELOG.md`，记录本次分层和双入口重构。

## Task 1: Backend Platform App Skeleton

**Files:**
- Create: `backend/internal/platform/app/app.go`
- Create: `backend/internal/platform/app/alioss.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/platform/app/app_test.go`

- [ ] **Step 1: Write app constructor smoke test**

Create `backend/internal/platform/app/app_test.go`:

```go
package app

import "testing"

func TestParseLogLevelDefaultsToInfo(t *testing.T) {
	if got := parseLogLevel("unknown"); got.String() != "info" {
		t.Fatalf("expected info level, got %s", got.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/app -run TestParseLogLevelDefaultsToInfo -v`

Expected: fail because package or function does not exist.

- [ ] **Step 3: Create platform app package**

Move startup orchestration from `cmd/server/main.go` into `backend/internal/platform/app/app.go`. The public API should be:

```go
// Run 加载配置、初始化依赖并启动 HTTP 服务。
// 该函数是后端服务的组合根，负责把配置、数据库、存储、模型、Handler 和路由串联起来。
func Run(ctx context.Context) error
```

Keep `cmd/server/main.go` minimal:

```go
package main

import (
	"context"
	"log"

	"gyrh-go-v2/backend/internal/platform/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
```

- [ ] **Step 4: Move aliOSS runtime file generation**

Move `prepareAliOSSRuntimeFiles` into `backend/internal/platform/app/alioss.go` with Chinese comments:

```go
// prepareAliOSSRuntimeFiles 根据当前配置生成 aliOSS 子进程需要读取的运行时配置文件。
// 返回背景素材服务和生成图服务各自的配置路径，供启动 manager 时使用。
```

- [ ] **Step 5: Verify backend compiles**

Run: `go test ./...`

Expected: all existing Go tests pass or only fail on pre-existing integration/env assumptions. Compile errors introduced by the move must be fixed.

## Task 2: Logging Sanitizer

**Files:**
- Create: `backend/internal/platform/logging/sanitize.go`
- Create: `backend/internal/platform/logging/sanitize_test.go`
- Modify: `backend/internal/api/middleware/logger.go`
- Modify: `backend/internal/302Helpper/GPT/client.go`

- [ ] **Step 1: Write sanitizer tests**

Create `sanitize_test.go`:

```go
package logging

import (
	"net/http"
	"strings"
	"testing"
)

func TestSanitizeHeadersMasksSecrets(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer secret-token")
	headers.Set("X-Request-ID", "req-1")

	got := SanitizeHeaders(headers)

	if got.Get("Authorization") == "Bearer secret-token" {
		t.Fatal("authorization header was not masked")
	}
	if got.Get("X-Request-ID") != "req-1" {
		t.Fatalf("expected request id to remain, got %q", got.Get("X-Request-ID"))
	}
}

func TestSanitizePayloadMasksKnownKeys(t *testing.T) {
	got := SanitizePayload([]byte(`{"api_key":"abc","prompt":"hello"}`))
	if strings.Contains(string(got), "abc") {
		t.Fatalf("secret leaked in payload: %s", got)
	}
	if !strings.Contains(string(got), "hello") {
		t.Fatalf("non-secret payload was removed: %s", got)
	}
}
```

- [ ] **Step 2: Implement sanitizer**

`sanitize.go` should expose:

```go
// SanitizeHeaders 复制并脱敏 HTTP 头，避免 debug 日志泄露密钥。
func SanitizeHeaders(headers http.Header) http.Header

// SanitizePayload 脱敏常见密钥字段，并限制日志 payload 最大长度。
func SanitizePayload(payload []byte) []byte
```

Mask keys containing `authorization`, `cookie`, `api_key`, `apikey`, `token`, `secret`, `private_key`, `signature`.

- [ ] **Step 3: Wire debug logging into model client**

In `backend/internal/302Helpper/GPT/client.go`, log request/response details at debug level using sanitizer. Keep all comments in Chinese and do not print raw `PROVIDER_302_API_KEY`.

- [ ] **Step 4: Verify sanitizer**

Run: `go test ./internal/platform/logging ./internal/302Helpper/GPT -v`

Expected: tests pass and no secret value appears in debug output.

## Task 3: Domain and Model Interface Layer

**Files:**
- Create: `backend/internal/domain/model/model.go`
- Create: `backend/internal/domain/rewrite/task.go`
- Create: `backend/internal/domain/image/image.go`
- Create: `backend/internal/infrastructure/model/router.go`
- Test: `backend/internal/domain/rewrite/task_test.go`

- [ ] **Step 1: Write rewrite task transition tests**

Create `backend/internal/domain/rewrite/task_test.go`:

```go
package rewrite

import "testing"

func TestTaskCanCompleteFromRunning(t *testing.T) {
	task := Task{Status: StatusRunning}
	if err := task.MarkCompleted("asset-1"); err != nil {
		t.Fatalf("expected complete transition, got %v", err)
	}
	if task.Status != StatusCompleted || task.ResultAssetID != "asset-1" {
		t.Fatalf("unexpected task after completion: %+v", task)
	}
}

func TestTaskCannotCompleteFromFailed(t *testing.T) {
	task := Task{Status: StatusFailed}
	if err := task.MarkCompleted("asset-1"); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
```

- [ ] **Step 2: Implement domain types**

Add Chinese comments for all exported identifiers. `domain/model/model.go` should define `ComposeRequest`, `ComposeResult`, `Provider` interface and provider constants. `domain/rewrite/task.go` should define task statuses and transition methods.

- [ ] **Step 3: Add infrastructure model router compatibility**

Create `backend/internal/infrastructure/model/router.go` as a thin wrapper around current `core/llm.Service`. This keeps the API stable while later tasks migrate clients into provider directories.

- [ ] **Step 4: Verify domain tests**

Run: `go test ./internal/domain/... ./internal/infrastructure/model -v`

Expected: pass.

## Task 4: Image Application Service

**Files:**
- Create: `backend/internal/application/image/service.go`
- Create: `backend/internal/application/image/service_test.go`
- Modify: `backend/internal/api/handler/image.go`

- [ ] **Step 1: Write list service test**

Create a service test with fake repositories and storage:

```go
func TestServiceListImagesAddsImageURL(t *testing.T) {
	// fake repo returns one image with AssetID "asset-1"
	// fake storage returns "http://example.test/asset-1.png"
	// assert service response contains ImageURL
}
```

- [ ] **Step 2: Implement Image Service**

`service.go` should expose:

```go
// Service 编排图片列表、访问、上传和改写流程。
type Service struct { ... }

// ListImages 查询生成历史，并为每张图片补齐可访问 URL。
func (s *Service) ListImages(ctx context.Context, limit int, offset int) (*ListImagesResult, error)
```

Use existing DB repo types initially to reduce blast radius. Later tasks can replace them with domain interfaces.

- [ ] **Step 3: Update handler List**

Change `ImageHandler.List` to parse HTTP params and call `image.Service.ListImages` instead of directly querying repo.

- [ ] **Step 4: Verify handler behavior**

Run: `go test ./internal/api/handler ./internal/application/image -v`

Expected: pass.

## Task 5: Frontend Route Split

**Files:**
- Create: `frontend/src/app/AppShell.jsx`
- Create: `frontend/src/app/routes.jsx`
- Create: `frontend/src/pages/kiosk/KioskViewer.jsx`
- Create: `frontend/src/pages/admin/AdminViewer.jsx`
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/constants/index.js`
- Modify: `frontend/src/components/Layout.jsx`

- [ ] **Step 1: Split screen constants**

Change `constants/index.js` to export:

```js
export const adminScreens = [...]
export const kioskScreens = adminScreens.filter(
  (item) => !['backgrounds', 'skills', 'styles', 'logout'].includes(item.key),
);
export const screens = adminScreens;
```

- [ ] **Step 2: Make ControlRail configurable**

Update `ControlRail({ screen, onSelect, items = screens })` and render `items.filter(...)`.

- [ ] **Step 3: Extract AppShell**

Move current `App` state machine into `AppShell.jsx` and accept:

```jsx
export function AppShell({ mode = 'admin', navigationItems = adminScreens }) { ... }
```

Pass `navigationItems` into `ControlRail`.

- [ ] **Step 4: Create route entry components**

`AdminViewer` returns `<AppShell mode="admin" navigationItems={adminScreens} />`.

`KioskViewer` returns `<AppShell mode="kiosk" navigationItems={kioskScreens} />`.

- [ ] **Step 5: Implement minimal path router**

Keep dependencies stable and implement `App.jsx` using `window.location.pathname`:

```jsx
function App() {
  const path = window.location.pathname;
  if (path.startsWith('/admin_viewer')) {
    return <AdminViewer />;
  }
  return <KioskViewer />;
}
```

- [ ] **Step 6: Verify frontend build**

Run: `npm run build`

Expected: Vite build succeeds.

## Task 6: Frontend Theme Split and 2K Kiosk Styling

**Files:**
- Create: `frontend/src/theme/tokens.css`
- Create: `frontend/src/theme/glass.css`
- Create: `frontend/src/theme/kiosk.css`
- Create: `frontend/src/theme/admin.css`
- Modify: `frontend/src/main.jsx`
- Modify: `frontend/src/styles.css`

- [ ] **Step 1: Import theme files**

Update `main.jsx` to import theme files before `styles.css`:

```js
import './theme/tokens.css';
import './theme/glass.css';
import './theme/admin.css';
import './theme/kiosk.css';
import './styles.css';
```

- [ ] **Step 2: Add tokens**

Create CSS variables for background gradients, glass surfaces, shadows, radii, large touch targets and kiosk font scale.

- [ ] **Step 3: Add kiosk scope**

Ensure kiosk root gets `app-mode-kiosk` class from `AppShell`. In `kiosk.css`, hide or enlarge as needed under `.app-mode-kiosk`.

- [ ] **Step 4: Verify hidden menus**

Search built DOM source or run browser manually and confirm `/` navigation does not include `背景库`、`SKILL 管理`、`风格转换配置`、`退出`; `/admin_viewer` still includes them.

- [ ] **Step 5: Verify frontend build**

Run: `npm run build`

Expected: build succeeds.

## Task 7: Changelog and Final Verification

**Files:**
- Modify: `CHANGELOG.md`
- Review: `git status`

- [ ] **Step 1: Update changelog**

Add a new entry under current date:

```markdown
## 2026-05-12

- 重构 Go 后端启动装配，新增分层架构骨架，降低入口和 Handler 耦合。
- 新增日志脱敏工具，为模型请求/响应 debug 日志提供安全基础。
- 拆分前端 `/` 演示入口与 `/admin_viewer` 管理入口，并隐藏演示端管理菜单。
- 抽离前端主题样式，为 55 寸 2K 触控屏演示优化玻璃拟态视觉。
```

- [ ] **Step 2: Run verification**

Run:

```bash
cd gyrh-go-v2/backend && go test ./...
cd ../frontend && npm run build
```

Expected: Go tests pass and Vite build succeeds.

- [ ] **Step 3: Check git status**

Run: `git status --short`

Expected: only source, docs, and changelog files relevant to this work are staged for commit. Do not stage `.env.local`, database files, generated images, logs, or binaries.

- [ ] **Step 4: Commit**

Commit message:

```text
refactor: decouple go backend and split viewer routes
```

