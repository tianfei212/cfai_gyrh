package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gyrh-go-v2/backend/internal/core/llm/qwen"
	"gyrh-go-v2/backend/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

func newBackgroundPromptCategoryTestHandler(t *testing.T) (*BackgroundPromptHandler, *db.BackgroundPromptRepo, *db.BackgroundCategoryRepo) {
	t.Helper()

	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := rawDB.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := rawDB.Exec(`
		CREATE TABLE background_prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			gemini_prompt TEXT NOT NULL DEFAULT '',
			gemini_negative_prompt TEXT NOT NULL DEFAULT '',
			gemini_prompt_zh TEXT NOT NULL DEFAULT '',
			gemini_negative_prompt_zh TEXT NOT NULL DEFAULT '',
			wan_prompt TEXT NOT NULL DEFAULT '',
			wan_negative_prompt TEXT NOT NULL DEFAULT '',
			wan_prompt_zh TEXT NOT NULL DEFAULT '',
			wan_negative_prompt_zh TEXT NOT NULL DEFAULT '',
			gpt_prompt TEXT NOT NULL DEFAULT '',
			gpt_negative_prompt TEXT NOT NULL DEFAULT '',
			gpt_prompt_zh TEXT NOT NULL DEFAULT '',
			gpt_negative_prompt_zh TEXT NOT NULL DEFAULT '',
			image_asset_id TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			image_width INTEGER NOT NULL DEFAULT 0,
			image_height INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX idx_background_prompts_name ON background_prompts(name);
		CREATE TABLE background_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_name TEXT NOT NULL,
			child_name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX idx_background_categories_parent_child ON background_categories(parent_name, child_name);
		CREATE TABLE background_category_bindings (
			category_id INTEGER NOT NULL,
			background_prompt_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (category_id, background_prompt_id)
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	testDB := &db.DB{DB: rawDB}
	promptRepo := db.NewBackgroundPromptRepo(testDB)
	categoryRepo := db.NewBackgroundCategoryRepo(testDB)
	if _, err := categoryRepo.EnsureDefault(); err != nil {
		t.Fatalf("ensure default category: %v", err)
	}

	return NewBackgroundPromptHandler(promptRepo, nil, nil, "", categoryRepo), promptRepo, categoryRepo
}

func createBackgroundPromptForCategoryTest(t *testing.T, repo *db.BackgroundPromptRepo, name string) int64 {
	t.Helper()

	id, err := repo.Create(
		name,
		"gemini prompt",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		0,
		0,
	)
	if err != nil {
		t.Fatalf("create background prompt %q: %v", name, err)
	}
	return id
}

func countBackgroundPromptsForCategoryTest(t *testing.T, repo *db.BackgroundPromptRepo) int64 {
	t.Helper()

	count, err := repo.Count()
	if err != nil {
		t.Fatalf("count background prompts: %v", err)
	}
	return count
}

func newBackgroundPromptCreateBindingFailureHandler(t *testing.T) (*BackgroundPromptHandler, *db.BackgroundPromptRepo) {
	t.Helper()

	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := rawDB.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := rawDB.Exec(`
		CREATE TABLE background_prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			gemini_prompt TEXT NOT NULL DEFAULT '',
			gemini_negative_prompt TEXT NOT NULL DEFAULT '',
			gemini_prompt_zh TEXT NOT NULL DEFAULT '',
			gemini_negative_prompt_zh TEXT NOT NULL DEFAULT '',
			wan_prompt TEXT NOT NULL DEFAULT '',
			wan_negative_prompt TEXT NOT NULL DEFAULT '',
			wan_prompt_zh TEXT NOT NULL DEFAULT '',
			wan_negative_prompt_zh TEXT NOT NULL DEFAULT '',
			gpt_prompt TEXT NOT NULL DEFAULT '',
			gpt_negative_prompt TEXT NOT NULL DEFAULT '',
			gpt_prompt_zh TEXT NOT NULL DEFAULT '',
			gpt_negative_prompt_zh TEXT NOT NULL DEFAULT '',
			image_asset_id TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			image_width INTEGER NOT NULL DEFAULT 0,
			image_height INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX idx_background_prompts_name ON background_prompts(name);
		CREATE TABLE background_category_bindings (
			category_id INTEGER NOT NULL,
			background_prompt_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (category_id, background_prompt_id)
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	testDB := &db.DB{DB: rawDB}
	promptRepo := db.NewBackgroundPromptRepo(testDB)
	categoryRepo := db.NewBackgroundCategoryRepo(testDB)
	return NewBackgroundPromptHandler(promptRepo, nil, nil, "", categoryRepo), promptRepo
}

type fakeQwenAdvisor struct{}

func (fakeQwenAdvisor) SuggestFromAsset(ctx context.Context, backgroundAssetID string) (*qwen.SuggestionResult, error) {
	return &qwen.SuggestionResult{
		GeminiPromptEN: "gemini prompt",
		WanPromptEN:    "wan prompt",
		GPTPromptEN:    "gpt prompt",
	}, nil
}

func (fakeQwenAdvisor) SyncEnglish(ctx context.Context, input qwen.SyncEnglishInput) (*qwen.SyncEnglishResult, error) {
	return &qwen.SyncEnglishResult{}, nil
}

func encodedPNGForCategoryTest(t *testing.T) string {
	t.Helper()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestBackgroundPromptListIncludesCategories(t *testing.T) {
	h, promptRepo, categoryRepo := newBackgroundPromptCategoryTestHandler(t)
	backgroundID := createBackgroundPromptForCategoryTest(t, promptRepo, "bg-with-category")
	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{categoryID}); err != nil {
		t.Fatalf("bind category: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/background-prompts?limit=6&offset=0", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Items []struct {
				ID         int64                   `json:"id"`
				Categories []db.BackgroundCategory `json:"categories"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(resp.Data.Items))
	}
	if resp.Data.Items[0].ID != backgroundID {
		t.Fatalf("item id = %d, want %d", resp.Data.Items[0].ID, backgroundID)
	}
	if len(resp.Data.Items[0].Categories) != 1 {
		t.Fatalf("categories len = %d, want 1", len(resp.Data.Items[0].Categories))
	}
	if resp.Data.Items[0].Categories[0].ID != categoryID {
		t.Fatalf("category id = %d, want %d", resp.Data.Items[0].Categories[0].ID, categoryID)
	}
}

func TestBackgroundPromptListFiltersByCategory(t *testing.T) {
	h, promptRepo, categoryRepo := newBackgroundPromptCategoryTestHandler(t)
	matchingID := createBackgroundPromptForCategoryTest(t, promptRepo, "matching-bg")
	otherID := createBackgroundPromptForCategoryTest(t, promptRepo, "other-bg")
	matchingCategoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create matching category: %v", err)
	}
	otherCategoryID, err := categoryRepo.Create("商业片", "城市夜景")
	if err != nil {
		t.Fatalf("create other category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(matchingID, []int64{matchingCategoryID}); err != nil {
		t.Fatalf("bind matching category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(otherID, []int64{otherCategoryID}); err != nil {
		t.Fatalf("bind other category: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/background-prompts?category_id=2&limit=6&offset=0", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Items []struct {
				ID int64 `json:"id"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Total != 1 {
		t.Fatalf("total = %d, want 1", resp.Data.Total)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(resp.Data.Items))
	}
	if resp.Data.Items[0].ID != matchingID {
		t.Fatalf("item id = %d, want matching id %d", resp.Data.Items[0].ID, matchingID)
	}
}

func TestBackgroundPromptListRejectsNonPositiveCategoryID(t *testing.T) {
	tests := []struct {
		name       string
		categoryID string
	}{
		{name: "zero", categoryID: "0"},
		{name: "negative", categoryID: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, _ := newBackgroundPromptCategoryTestHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/background-prompts?category_id="+tt.categoryID, nil)
			rec := httptest.NewRecorder()
			h.List(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body=%s, want 400", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBackgroundPromptListMissingCategoryReturnsEmptyPage(t *testing.T) {
	h, promptRepo, _ := newBackgroundPromptCategoryTestHandler(t)
	_ = createBackgroundPromptForCategoryTest(t, promptRepo, "unmatched-bg")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/background-prompts?category_id=99999&limit=6&offset=0", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Items []struct {
				ID int64 `json:"id"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Total != 0 {
		t.Fatalf("total = %d, want 0", resp.Data.Total)
	}
	if len(resp.Data.Items) != 0 {
		t.Fatalf("items len = %d, want 0", len(resp.Data.Items))
	}
}

func TestBackgroundPromptCategoryBindingsEndpointFallsBackToDefault(t *testing.T) {
	h, promptRepo, categoryRepo := newBackgroundPromptCategoryTestHandler(t)
	backgroundID := createBackgroundPromptForCategoryTest(t, promptRepo, "fallback-bg")
	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{categoryID}); err != nil {
		t.Fatalf("bind category: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/background-prompts/1/categories", strings.NewReader(`{"category_ids":[]}`))
	req = setMuxIDForTest(req, backgroundID)
	rec := httptest.NewRecorder()
	h.UpdateCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	categories, err := categoryRepo.ListByBackgroundID(backgroundID)
	if err != nil {
		t.Fatalf("list background categories: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("categories len = %d, want 1", len(categories))
	}
	if categories[0].ParentName != db.DefaultCategoryParent || categories[0].ChildName != db.DefaultCategoryChild {
		t.Fatalf("category = %s/%s, want default/default", categories[0].ParentName, categories[0].ChildName)
	}
}

func TestBackgroundPromptUpdateCategoriesMissingBackgroundReturnsNotFound(t *testing.T) {
	h, _, categoryRepo := newBackgroundPromptCategoryTestHandler(t)
	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/background-prompts/99999/categories", strings.NewReader(`{"category_ids":[1]}`))
	req = setMuxIDForTest(req, 99999)
	rec := httptest.NewRecorder()
	h.UpdateCategories(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s, want 404 for category %d", rec.Code, rec.Body.String(), categoryID)
	}
}

func TestBackgroundPromptCreateCleansPromptWhenDefaultBindingFails(t *testing.T) {
	h, promptRepo := newBackgroundPromptCreateBindingFailureHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/background-prompts", strings.NewReader(`{"name":"cleanup-bg","gemini_prompt":"prompt"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s, want 500", rec.Code, rec.Body.String())
	}
	if got := countBackgroundPromptsForCategoryTest(t, promptRepo); got != 0 {
		t.Fatalf("background prompt count = %d, want 0 after binding failure", got)
	}
}

func TestBackgroundPromptImportCleansPromptAndAssetWhenDefaultBindingFails(t *testing.T) {
	h, promptRepo := newBackgroundPromptCreateBindingFailureHandler(t)
	store := &fakeStorageService{}
	h.storageService = store
	h.qwenAdvisor = fakeQwenAdvisor{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/background-prompts/import", strings.NewReader(`{"name":"cleanup-import","image":"`+encodedPNGForCategoryTest(t)+`"}`))
	rec := httptest.NewRecorder()
	h.Import(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s, want 500", rec.Code, rec.Body.String())
	}
	if got := countBackgroundPromptsForCategoryTest(t, promptRepo); got != 0 {
		t.Fatalf("background prompt count = %d, want 0 after binding failure", got)
	}
	if len(store.deleted) != 1 {
		t.Fatalf("deleted asset count = %d, want 1", len(store.deleted))
	}
}
