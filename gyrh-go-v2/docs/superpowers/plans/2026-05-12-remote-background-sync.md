# Remote Background Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a background manager sync action that imports the current remote gallery page (default URL from `configs/config.yaml` → `gallery.external_url`, overridable via request `api_url`) into local OSS-backed `background_prompts` records without triggering Qwen prompt reverse generation.

**Architecture:** Add a dedicated backend sync endpoint separate from the existing manual import path. The frontend calls the sync endpoint with an optional `api_url`; when omitted the backend uses `gallery.external_url` from loaded config. The backend fetches remote JSON, downloads each unsynced image, uploads it through `StorageService`, creates `background_prompts` rows with remote Chinese prompt text, and reports imported/skipped/failed counts.

**Tech Stack:** Go `net/http`, Gorilla mux, SQLite repository layer, existing `StorageService`, React 18, Vite.

---

## File Structure

- Modify `backend/internal/api/handler/background_prompt.go`: add remote response structs, URL resolution helpers, sync result structs, and `SyncRemote`.
- Modify `backend/internal/api/router.go`: register `POST /api/v1/background-prompts/sync-remote`.
- Create `backend/internal/api/handler/background_prompt_remote_test.go`: unit tests for URL resolution, dimension parsing, and sync behavior using fake storage and `httptest`.
- Modify `frontend/src/screens/BackgroundManagerScreen.jsx`: wire the “同步” button to the new endpoint and show loading/success/failure feedback.

### Task 1: Backend Helpers And Tests

**Files:**
- Create: `backend/internal/api/handler/background_prompt_remote_test.go`
- Modify: `backend/internal/api/handler/background_prompt.go`

- [ ] **Step 1: Add failing tests for helper behavior**

Create `backend/internal/api/handler/background_prompt_remote_test.go` with:

```go
package handler

import "testing"

func TestResolveRemoteMediaURL(t *testing.T) {
	t.Run("relative path uses api url origin", func(t *testing.T) {
		got, err := resolveRemoteMediaURL("https://jjxo.chinafilmai.com/picGet/api/media?page=1", "/media_images/a.jpg")
		if err != nil {
			t.Fatalf("resolveRemoteMediaURL returned error: %v", err)
		}
		want := "https://jjxo.chinafilmai.com/media_images/a.jpg"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("absolute url is preserved", func(t *testing.T) {
		got, err := resolveRemoteMediaURL("https://jjxo.chinafilmai.com/picGet/api/media", "https://cdn.example.com/a.jpg")
		if err != nil {
			t.Fatalf("resolveRemoteMediaURL returned error: %v", err)
		}
		want := "https://cdn.example.com/a.jpg"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("empty media url errors", func(t *testing.T) {
		if _, err := resolveRemoteMediaURL("https://jjxo.chinafilmai.com/picGet/api/media", " "); err == nil {
			t.Fatal("expected error for empty media url")
		}
	})
}

func TestParseRemoteDimensions(t *testing.T) {
	width, height := parseRemoteDimensions("2048x1152")
	if width != 2048 || height != 1152 {
		t.Fatalf("got %dx%d, want 2048x1152", width, height)
	}

	width, height = parseRemoteDimensions("bad")
	if width != 0 || height != 0 {
		t.Fatalf("got %dx%d, want 0x0 for invalid dimensions", width, height)
	}
}
```

- [ ] **Step 2: Run helper tests and verify failure**

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/backend && go test ./internal/api/handler -run 'TestResolveRemoteMediaURL|TestParseRemoteDimensions' -v
```

Expected: FAIL because `resolveRemoteMediaURL` and `parseRemoteDimensions` are not defined.

- [ ] **Step 3: Implement helper functions**

In `backend/internal/api/handler/background_prompt.go`, extend imports:

```go
import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)
```

Add near the bottom of the file:

```go
func resolveRemoteMediaURL(apiURL string, mediaURL string) (string, error) {
	mediaURL = strings.TrimSpace(mediaURL)
	if mediaURL == "" {
		return "", fmt.Errorf("远端图片 URL 为空")
	}

	parsedMedia, err := url.Parse(mediaURL)
	if err != nil {
		return "", fmt.Errorf("远端图片 URL 无效")
	}
	if parsedMedia.IsAbs() {
		if parsedMedia.Scheme != "http" && parsedMedia.Scheme != "https" {
			return "", fmt.Errorf("远端图片 URL 只支持 http/https")
		}
		return parsedMedia.String(), nil
	}

	parsedAPI, err := url.Parse(strings.TrimSpace(apiURL))
	if err != nil || parsedAPI.Scheme == "" || parsedAPI.Host == "" {
		return "", fmt.Errorf("远端 API URL 无效")
	}
	if parsedAPI.Scheme != "http" && parsedAPI.Scheme != "https" {
		return "", fmt.Errorf("远端 API URL 只支持 http/https")
	}

	return parsedAPI.ResolveReference(parsedMedia).String(), nil
}

func parseRemoteDimensions(dimensions string) (int, int) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(dimensions)), "x")
	if len(parts) != 2 {
		return 0, 0
	}

	width, widthErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, heightErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0
	}
	return width, height
}
```

- [ ] **Step 4: Run helper tests and verify pass**

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/backend && go test ./internal/api/handler -run 'TestResolveRemoteMediaURL|TestParseRemoteDimensions' -v
```

Expected: PASS.

### Task 2: Backend Sync Endpoint

**Files:**
- Modify: `backend/internal/api/handler/background_prompt.go`
- Modify: `backend/internal/api/router.go`
- Test: `backend/internal/api/handler/background_prompt_remote_test.go`

- [ ] **Step 1: Add failing sync behavior test**

Append to `backend/internal/api/handler/background_prompt_remote_test.go`:

```go
import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/storage"

	_ "github.com/mattn/go-sqlite3"
)

type fakeStorageService struct {
	saved [][]byte
}

func (s *fakeStorageService) Save(ctx context.Context, data []byte, filename string) (string, error) {
	return s.SaveWithKind(ctx, data, filename, storage.SaveKindAsset)
}

func (s *fakeStorageService) SaveWithKind(ctx context.Context, data []byte, filename string, kind storage.SaveKind) (string, error) {
	s.saved = append(s.saved, data)
	return "asset:remote-test", nil
}

func (s *fakeStorageService) Read(ctx context.Context, assetID string) ([]byte, error) {
	return nil, nil
}

func (s *fakeStorageService) GetImageURL(ctx context.Context, assetID string) (string, error) {
	return "https://oss.example.com/" + assetID, nil
}

func (s *fakeStorageService) GetThumbnailURL(ctx context.Context, assetID string, w, h int) (string, error) {
	return "https://oss.example.com/thumb/" + assetID, nil
}

func (s *fakeStorageService) GetForModelUpload(ctx context.Context, assetID string, provider storage.StorageProvider) (string, error) {
	return "https://oss.example.com/model/" + assetID, nil
}

func (s *fakeStorageService) Delete(ctx context.Context, assetID string) error {
	return nil
}

func TestSyncRemoteImportsCurrentPageWithoutSuggest(t *testing.T) {
	var imageBytes bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&imageBytes, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/picGet/api/media":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "ok",
				"data": map[string]any{
					"items": []map[string]any{
						{
							"id":         "remote-1",
							"url":        "/media_images/remote-1.png",
							"dimensions": "2048x1152",
							"prompt":     "远端中文提示词",
						},
					},
					"page":       1,
					"pageSize":   20,
					"total":      1,
					"totalPages": 1,
				},
			})
		case "/media_images/remote-1.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(imageBytes.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer remote.Close()

	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer rawDB.Close()

	_, err = rawDB.Exec(`
		CREATE TABLE background_prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			gemini_prompt TEXT,
			gemini_negative_prompt TEXT,
			gemini_prompt_zh TEXT,
			gemini_negative_prompt_zh TEXT,
			wan_prompt TEXT,
			wan_negative_prompt TEXT,
			wan_prompt_zh TEXT,
			wan_negative_prompt_zh TEXT,
			image_asset_id TEXT,
			image_url TEXT,
			image_width INTEGER,
			image_height INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	repo := db.NewBackgroundPromptRepo(&db.DB{DB: rawDB})
	store := &fakeStorageService{}
	h := NewBackgroundPromptHandler(repo, store, nil, "")

	body := strings.NewReader(`{"api_url":"` + remote.URL + `/picGet/api/media?page=1&pageSize=20"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/background-prompts/sync-remote", body)
	rec := httptest.NewRecorder()

	h.SyncRemote(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status got %d, body %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Imported int `json:"imported"`
			Skipped  int `json:"skipped"`
			Failed   int `json:"failed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Imported != 1 || resp.Data.Skipped != 0 || resp.Data.Failed != 0 {
		t.Fatalf("unexpected stats: %+v", resp.Data)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved images got %d, want 1", len(store.saved))
	}

	item, err := repo.GetByName("remote:remote-1")
	if err != nil {
		t.Fatalf("get imported item: %v", err)
	}
	if item.WanPromptZH != "远端中文提示词" || item.GeminiPromptZH != "远端中文提示词" {
		t.Fatalf("prompt mapping failed: wan=%q gemini=%q", item.WanPromptZH, item.GeminiPromptZH)
	}
	if item.WanPrompt != "" || item.GeminiPrompt != "" {
		t.Fatalf("english prompts should stay empty: wan=%q gemini=%q", item.WanPrompt, item.GeminiPrompt)
	}
	if item.ImageWidth != 2 || item.ImageHeight != 3 {
		t.Fatalf("image size got %dx%d, want 2x3", item.ImageWidth, item.ImageHeight)
	}
}
```

If the file already has `import "testing"` from Task 1, replace it with the full import block shown above.

- [ ] **Step 2: Run sync test and verify failure**

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/backend && go test ./internal/api/handler -run TestSyncRemoteImportsCurrentPageWithoutSuggest -v
```

Expected: FAIL because `SyncRemote` is not defined or response types are missing.

- [ ] **Step 3: Implement remote sync structs and handler**

In `backend/internal/api/handler/background_prompt.go`, add these types after `NewBackgroundPromptHandler`:

```go
type remoteSyncRequest struct {
	APIURL string `json:"api_url"`
}

type remoteMediaResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []remoteMediaItem `json:"items"`
	} `json:"data"`
}

type remoteMediaItem struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Dimensions string `json:"dimensions"`
	Prompt     string `json:"prompt"`
}

type remoteSyncFailure struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type remoteSyncResult struct {
	Imported int                 `json:"imported"`
	Skipped  int                 `json:"skipped"`
	Failed   int                 `json:"failed"`
	Failures []remoteSyncFailure `json:"failures"`
}
```

Add this handler before `SyncEnglish`:

```go
// SyncRemote 同步远端图库当前页背景图，不触发提示词反推。
func (h *BackgroundPromptHandler) SyncRemote(w http.ResponseWriter, r *http.Request) {
	if h.storageService == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "存储服务未初始化"))
		return
	}

	var req remoteSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	apiURL := strings.TrimSpace(req.APIURL)
	if err := validateRemoteAPIURL(apiURL); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, err.Error()))
		return
	}

	remoteResp, err := fetchRemoteMedia(r.Context(), apiURL)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadGateway, httpx.Error(1, err.Error()))
		return
	}

	result := h.importRemoteMediaItems(r.Context(), apiURL, remoteResp.Data.Items)
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
```

Add helper functions near the bottom:

```go
func validateRemoteAPIURL(apiURL string) error {
	if strings.TrimSpace(apiURL) == "" {
		return fmt.Errorf("api_url 不能为空")
	}
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("api_url 不是合法 URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("api_url 只支持 http/https")
	}
	return nil
}

func fetchRemoteMedia(ctx context.Context, apiURL string) (*remoteMediaResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建远端图库请求失败")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求远端图库失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("远端图库状态码异常: %d", resp.StatusCode)
	}

	var remoteResp remoteMediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&remoteResp); err != nil {
		return nil, fmt.Errorf("解析远端图库响应失败")
	}
	return &remoteResp, nil
}

func (h *BackgroundPromptHandler) importRemoteMediaItems(ctx context.Context, apiURL string, items []remoteMediaItem) remoteSyncResult {
	result := remoteSyncResult{
		Failures: []remoteSyncFailure{},
	}

	for _, item := range items {
		remoteID := strings.TrimSpace(item.ID)
		if remoteID == "" {
			result.Failed++
			result.Failures = append(result.Failures, remoteSyncFailure{ID: "", Reason: "远端 id 为空"})
			continue
		}

		name := "remote:" + remoteID
		if _, err := h.repo.GetByName(name); err == nil {
			result.Skipped++
			continue
		}

		if err := h.importRemoteMediaItem(ctx, apiURL, name, item); err != nil {
			result.Failed++
			result.Failures = append(result.Failures, remoteSyncFailure{ID: remoteID, Reason: err.Error()})
			continue
		}
		result.Imported++
	}

	return result
}

func (h *BackgroundPromptHandler) importRemoteMediaItem(ctx context.Context, apiURL string, name string, item remoteMediaItem) error {
	imageURL, err := resolveRemoteMediaURL(apiURL, item.URL)
	if err != nil {
		return err
	}

	imageData, err := downloadRemoteImage(ctx, imageURL)
	if err != nil {
		return err
	}

	imageWidth, imageHeight, decodeErr := decodeImageSize(imageData)
	if decodeErr != nil {
		imageWidth, imageHeight = parseRemoteDimensions(item.Dimensions)
		if imageWidth == 0 || imageHeight == 0 {
			return fmt.Errorf("远端图片不是合法图片数据")
		}
	}

	assetID, err := h.storageService.SaveWithKind(ctx, imageData, fmt.Sprintf("remote_bg_%s.png", sanitizeRemoteID(item.ID)), storage.SaveKindAsset)
	if err != nil {
		return fmt.Errorf("保存背景图失败")
	}

	prompt := strings.TrimSpace(item.Prompt)
	if _, err := h.repo.Create(
		name,
		"",
		"",
		prompt,
		"",
		"",
		"",
		prompt,
		"",
		assetID,
		"",
		imageWidth,
		imageHeight,
	); err != nil {
		_ = h.storageService.Delete(context.Background(), assetID)
		return fmt.Errorf("保存背景图记录失败")
	}

	return nil
}

func downloadRemoteImage(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建远端图片请求失败")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载远端图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("远端图片状态码异常: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("读取远端图片失败")
	}
	return buf.Bytes(), nil
}

func sanitizeRemoteID(id string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "?", "_", "&", "_", "=", "_")
	cleaned := strings.TrimSpace(replacer.Replace(id))
	if cleaned == "" {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return cleaned
}
```

- [ ] **Step 4: Register the route**

In `backend/internal/api/router.go`, add the route with the other background routes:

```go
protected.HandleFunc("/background-prompts/sync-remote", backgroundPromptHandler.SyncRemote).Methods(http.MethodPost)
```

Place it near `/background-prompts/import`, before `/background-prompts/{id}`.

- [ ] **Step 5: Run sync tests and full backend tests**

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/backend && go test ./internal/api/handler -v
```

Expected: PASS.

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/backend && go test ./... 
```

Expected: PASS.

### Task 3: Frontend Sync Button

**Files:**
- Modify: `frontend/src/screens/BackgroundManagerScreen.jsx`

- [ ] **Step 1: Add syncing state**

Inside `BackgroundManagerScreen`, after `importing` state (if not already present):

```jsx
const [syncing, setSyncing] = useState(false);
```

- [ ] **Step 2: Add sync handler**

Inside `BackgroundManagerScreen`, after `handleImportClick`:

```jsx
const handleSyncRemote = async () => {
  setSyncing(true);
  try {
    const result = await fetchApi('/api/v1/background-prompts/sync-remote', {
      method: 'POST',
      body: JSON.stringify({})
    });
    await fetchBackgrounds();
    const failures = result.failures?.length ? `，失败 ${result.failed} 条` : '';
    alert(`同步完成：新增 ${result.imported} 条，跳过 ${result.skipped} 条${failures}`);
  } catch (err) {
    alert('同步失败: ' + err.message);
  } finally {
    setSyncing(false);
  }
};
```

- [ ] **Step 3: Wire the button**

Replace:

```jsx
<button className="tiny-chip" type="button">同步</button>
```

With:

```jsx
<button className="tiny-chip" type="button" onClick={handleSyncRemote} disabled={syncing || importing}>
  {syncing ? '同步中...' : '同步'}
</button>
```

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/frontend && npm run build
```

Expected: build succeeds.

### Task 4: Manual Verification

**Files:**
- No code changes.

- [ ] **Step 1: Verify route compiles with full backend tests**

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/backend && go test ./...
```

Expected: PASS.

- [ ] **Step 2: Verify frontend build**

Run:

```bash
cd /Users/derekt/devop/展厅在更新项目/gyrh/gyrh-go-v2/frontend && npm run build
```

Expected: PASS.

- [ ] **Step 3: Optional browser smoke test**

Start the app as the project normally does, open the background manager, click “同步”, and verify:

- Button changes to “同步中...” during request.
- Success alert includes imported/skipped counts.
- Local table refreshes with `remote:<id>` records.
- Editing a synced row shows `Wan 中文提示词` and `Gemini 中文提示词` populated from the remote `prompt`.
- English prompt fields remain empty until manually translated.

### Self-Review

- Spec coverage: plan covers dedicated backend endpoint, current-page-only sync, remote prompt mapping to Chinese fields, no Qwen reverse generation, skip-on-existing dedupe, relative URL resolution, failure stats, frontend button state, and validation.
- Placeholder scan: no TBD/TODO/fill-in placeholders remain.
- Type consistency: endpoint name, request field `api_url`, response fields, helper names, storage calls, and React handler names are consistent across tasks.
