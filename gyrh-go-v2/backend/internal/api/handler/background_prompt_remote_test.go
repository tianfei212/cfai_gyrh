package handler

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
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = rawDB.Exec(`CREATE UNIQUE INDEX idx_background_prompts_name ON background_prompts(name)`)
	if err != nil {
		t.Fatalf("create index: %v", err)
	}

	repo := db.NewBackgroundPromptRepo(&db.DB{DB: rawDB})
	categoryRepo := db.NewBackgroundCategoryRepo(&db.DB{DB: rawDB})
	if _, err := categoryRepo.EnsureDefault(); err != nil {
		t.Fatalf("ensure default category: %v", err)
	}
	store := &fakeStorageService{}
	h := NewBackgroundPromptHandler(repo, store, nil, categoryRepo)

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
	if item.WanPromptZH != "远端中文提示词" || item.GeminiPromptZH != "远端中文提示词" || item.GPTPromptZH != "远端中文提示词" {
		t.Fatalf("prompt mapping failed: wan=%q gemini=%q gpt=%q", item.WanPromptZH, item.GeminiPromptZH, item.GPTPromptZH)
	}
	if item.WanPrompt != "" || item.GeminiPrompt != "" || item.GPTPrompt != "" {
		t.Fatalf("english prompts should stay empty: wan=%q gemini=%q gpt=%q", item.WanPrompt, item.GeminiPrompt, item.GPTPrompt)
	}
	if item.ImageWidth != 2 || item.ImageHeight != 3 {
		t.Fatalf("image size got %dx%d, want 2x3", item.ImageWidth, item.ImageHeight)
	}
	categories, err := categoryRepo.ListByBackgroundID(item.ID)
	if err != nil {
		t.Fatalf("list imported categories: %v", err)
	}
	if len(categories) != 1 || categories[0].ParentName != db.DefaultCategoryParent || categories[0].ChildName != db.DefaultCategoryChild {
		t.Fatalf("remote import category binding = %+v, want default/default", categories)
	}
}

func TestSyncRemoteSkipsExistingRemoteRecord(t *testing.T) {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer rawDB.Close()

	_, err = rawDB.Exec(`
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
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = rawDB.Exec(`CREATE UNIQUE INDEX idx_background_prompts_name ON background_prompts(name)`)
	if err != nil {
		t.Fatalf("create index: %v", err)
	}

	repo := db.NewBackgroundPromptRepo(&db.DB{DB: rawDB})
	categoryRepo := db.NewBackgroundCategoryRepo(&db.DB{DB: rawDB})
	if _, err := categoryRepo.EnsureDefault(); err != nil {
		t.Fatalf("ensure default category: %v", err)
	}
	_, err = repo.Create(
		"remote:remote-1",
		"edited english",
		"",
		"本地已编辑 Gemini",
		"",
		"edited wan",
		"",
		"本地已编辑 Wan",
		"",
		"edited gpt",
		"",
		"本地已编辑 GPT",
		"",
		"asset:existing",
		"",
		10,
		20,
	)
	if err != nil {
		t.Fatalf("seed existing record: %v", err)
	}

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "ok",
			"data": map[string]any{
				"items": []map[string]any{
					{
						"id":         "remote-1",
						"url":        "/media_images/remote-1.png",
						"dimensions": "2048x1152",
						"prompt":     "远端新提示词",
					},
				},
			},
		})
	}))
	defer remote.Close()

	store := &fakeStorageService{}
	h := NewBackgroundPromptHandler(repo, store, nil, categoryRepo)

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
	if resp.Data.Imported != 0 || resp.Data.Skipped != 1 || resp.Data.Failed != 0 {
		t.Fatalf("unexpected stats: %+v", resp.Data)
	}
	if len(store.saved) != 0 {
		t.Fatalf("expected no uploads for skipped record, got %d", len(store.saved))
	}

	item, err := repo.GetByName("remote:remote-1")
	if err != nil {
		t.Fatalf("get existing item: %v", err)
	}
	if item.WanPromptZH != "本地已编辑 Wan" || item.GeminiPromptZH != "本地已编辑 Gemini" || item.GPTPromptZH != "本地已编辑 GPT" {
		t.Fatalf("existing prompts were overwritten: wan=%q gemini=%q gpt=%q", item.WanPromptZH, item.GeminiPromptZH, item.GPTPromptZH)
	}
}
