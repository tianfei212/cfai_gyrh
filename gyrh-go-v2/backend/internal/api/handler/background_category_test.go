package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"gyrh-go-v2/backend/internal/db"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func newCategoryHandlerTestRepo(t *testing.T) *db.BackgroundCategoryRepo {
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

	testDB := &db.DB{DB: rawDB}
	if _, err := rawDB.Exec(`
		CREATE TABLE background_prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			gemini_prompt TEXT NOT NULL DEFAULT '',
			gemini_negative_prompt TEXT NOT NULL DEFAULT '',
			wan_prompt TEXT NOT NULL DEFAULT '',
			wan_negative_prompt TEXT NOT NULL DEFAULT '',
			gpt_prompt TEXT NOT NULL DEFAULT '',
			gpt_negative_prompt TEXT NOT NULL DEFAULT '',
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
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	repo := db.NewBackgroundCategoryRepo(testDB)
	if _, err := repo.EnsureDefault(); err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	return repo
}

func setMuxIDForTest(req *http.Request, id int64) *http.Request {
	return mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(id, 10)})
}

func TestBackgroundCategoryHandlerCreateAndList(t *testing.T) {
	repo := newCategoryHandlerTestRepo(t)
	h := NewBackgroundCategoryHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/background-categories", strings.NewReader(`{"parent_name":"文旅片","child_name":"四川大佛"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/background-categories", nil)
	rec = httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []db.BackgroundCategory `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("category len = %d, want default plus created", len(resp.Data))
	}
}

func TestBackgroundCategoryHandlerRejectsDefaultDelete(t *testing.T) {
	repo := newCategoryHandlerTestRepo(t)
	h := NewBackgroundCategoryHandler(repo)
	defaultCategory, err := repo.EnsureDefault()
	if err != nil {
		t.Fatalf("ensure default: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/background-categories/1", nil)
	req = setMuxIDForTest(req, defaultCategory.ID)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("delete default status = %d, want 400 body=%s", rec.Code, rec.Body.String())
	}
}
