package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"gyrh-go-v2/backend/internal/db"
)

func TestSkillGetQwenReturnsBackgroundSuggestionTemplateContent(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	skillRepo := db.NewSkillRepo(database)
	templateRepo := db.NewLLMPromptTemplateRepo(database)
	const templateContent = "Qwen LLM background suggestion template"
	if err := templateRepo.UpsertByKey("Qwen 背景图默认建议", db.TemplateKeyQwenBackgroundSuggest, templateContent, "description"); err != nil {
		t.Fatalf("create prompt template: %v", err)
	}
	skillID, err := skillRepo.Create("智能人物构图定位", "stale skill content", "qwen")
	if err != nil {
		t.Fatalf("create qwen skill: %v", err)
	}

	h := NewSkillHandler(skillRepo, templateRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": jsonNumber(skillID)})
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Content != templateContent {
		t.Fatalf("qwen skill content should come from llm prompt template, got %q", resp.Data.Content)
	}
}

func TestSkillUpdateQwenWritesBackgroundSuggestionTemplateContent(t *testing.T) {
	database, err := db.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	skillRepo := db.NewSkillRepo(database)
	templateRepo := db.NewLLMPromptTemplateRepo(database)
	if err := templateRepo.UpsertByKey("Qwen 背景图默认建议", db.TemplateKeyQwenBackgroundSuggest, "old template", "description"); err != nil {
		t.Fatalf("create prompt template: %v", err)
	}
	skillID, err := skillRepo.Create("智能人物构图定位", "stale skill content", "qwen")
	if err != nil {
		t.Fatalf("create qwen skill: %v", err)
	}

	h := NewSkillHandler(skillRepo, templateRepo)
	body := strings.NewReader(`{"name":"智能人物构图定位","provider":"qwen","content":"updated qwen template","is_active":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/1", body)
	req = mux.SetURLVars(req, map[string]string{"id": jsonNumber(skillID)})
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	template, err := templateRepo.GetByKey(db.TemplateKeyQwenBackgroundSuggest)
	if err != nil {
		t.Fatalf("get prompt template: %v", err)
	}
	if template.Content != "updated qwen template" {
		t.Fatalf("template content = %q", template.Content)
	}
}
