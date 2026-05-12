package llm

import (
	"strings"
	"testing"

	"gyrh-go-v2/backend/internal/db"
)

func TestComposeGeminiPromptAdds16x9TwoKRequirement(t *testing.T) {
	prompt := composeGeminiPrompt("cinematic portrait", "")

	if !strings.Contains(prompt, "16:9") {
		t.Fatalf("prompt should require 16:9 output, got: %s", prompt)
	}
	if !strings.Contains(prompt, "2K") {
		t.Fatalf("prompt should require 2K output, got: %s", prompt)
	}
}

func TestComposeGeminiPromptKeepsNegativePromptAfterOutputRequirement(t *testing.T) {
	prompt := composeGeminiPrompt("cinematic portrait", "low quality")

	if !strings.Contains(prompt, "Negative prompt: low quality") {
		t.Fatalf("prompt should keep negative prompt, got: %s", prompt)
	}
	if strings.Index(prompt, "16:9") > strings.Index(prompt, "Negative prompt:") {
		t.Fatalf("output requirements should appear before negative prompt, got: %s", prompt)
	}
}

func TestBuildPromptUsesActiveSkillForTemporaryBackground(t *testing.T) {
	testDB, err := db.NewDB(t.TempDir() + "/gyrh.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	skillRepo := db.NewSkillRepo(testDB)
	const skillContent = "Use the temporary uploaded background as the scene and blend the transparent character naturally."
	if _, err := skillRepo.Create("google.md", skillContent, "google"); err != nil {
		t.Fatalf("create active skill: %v", err)
	}

	svc := &service{skillRepo: skillRepo}
	resolved, err := svc.buildPrompt(nil, "google", ComposeParams{
		Images: []ImageInput{
			{Type: ImageTypeCharacter, AssetID: "asset:foreground.png"},
			{Type: ImageTypeBackground, AssetID: "asset:background.png"},
		},
		BackgroundPromptID: 0,
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	if resolved.Prompt != skillContent {
		t.Fatalf("prompt should use active provider skill, got: %q", resolved.Prompt)
	}
}

func TestNormalizeProvider302GPTImage(t *testing.T) {
	if got := normalizeProvider("302-gpt-image"); got != "302-gpt-image" {
		t.Fatalf("normalizeProvider returned %q", got)
	}
	if got := normalizeProvider("gpt"); got != "302-gpt-image" {
		t.Fatalf("normalizeProvider alias returned %q", got)
	}
}

func TestBuildPromptUses302GPTImageActiveSkillEvenWithBackgroundPrompt(t *testing.T) {
	testDB, err := db.NewDB(t.TempDir() + "/gyrh.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	skillRepo := db.NewSkillRepo(testDB)
	const skillContent = "Use GPT Image to blend the transparent character into the background naturally."
	if _, err := skillRepo.Create("302-gpt-image.md", skillContent, "302-gpt-image"); err != nil {
		t.Fatalf("create active skill: %v", err)
	}
	backgroundRepo := db.NewBackgroundPromptRepo(testDB)
	backgroundID, err := backgroundRepo.Create(
		"crystal cave",
		"gemini prompt",
		"gemini negative",
		"Gemini 中文",
		"Gemini 中文反向",
		"wan prompt",
		"wan negative",
		"Wan 中文",
		"Wan 中文反向",
		"",
		"",
		"GPT 中文",
		"GPT 中文反向",
		"asset:bg.png",
		"",
		2048,
		1152,
	)
	if err != nil {
		t.Fatalf("create background prompt: %v", err)
	}

	svc := &service{skillRepo: skillRepo, backgroundPromptRepo: backgroundRepo}
	resolved, err := svc.buildPrompt(nil, "302-gpt-image", ComposeParams{
		Images: []ImageInput{
			{Type: ImageTypeCharacter, AssetID: "asset:foreground.png"},
			{Type: ImageTypeBackground, AssetID: "asset:background.png"},
		},
		BackgroundPromptID: backgroundID,
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	if resolved.Prompt != skillContent {
		t.Fatalf("302-gpt-image prompt should use active provider skill, got: %q", resolved.Prompt)
	}
	if resolved.NegativePrompt != "" {
		t.Fatalf("302-gpt-image should not reuse background negative prompt, got: %q", resolved.NegativePrompt)
	}
}

func TestBuildPromptUses302GPTImageBackgroundPromptWhenPresent(t *testing.T) {
	testDB, err := db.NewDB(t.TempDir() + "/gyrh.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	skillRepo := db.NewSkillRepo(testDB)
	if _, err := skillRepo.Create("302-gpt-image.md", "default skill prompt", "302-gpt-image"); err != nil {
		t.Fatalf("create active skill: %v", err)
	}
	backgroundRepo := db.NewBackgroundPromptRepo(testDB)
	backgroundID, err := backgroundRepo.Create(
		"crystal cave",
		"gemini prompt",
		"gemini negative",
		"Gemini 中文",
		"Gemini 中文反向",
		"wan prompt",
		"wan negative",
		"Wan 中文",
		"Wan 中文反向",
		"gpt background prompt",
		"gpt negative prompt",
		"GPT 中文",
		"GPT 中文反向",
		"asset:bg.png",
		"",
		2048,
		1152,
	)
	if err != nil {
		t.Fatalf("create background prompt: %v", err)
	}

	svc := &service{skillRepo: skillRepo, backgroundPromptRepo: backgroundRepo}
	resolved, err := svc.buildPrompt(nil, "302-gpt-image", ComposeParams{
		Images: []ImageInput{
			{Type: ImageTypeCharacter, AssetID: "asset:foreground.png"},
			{Type: ImageTypeBackground, AssetID: "asset:background.png"},
		},
		BackgroundPromptID: backgroundID,
	})
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	if resolved.Prompt != "gpt background prompt" {
		t.Fatalf("302-gpt-image prompt should use background gpt prompt, got: %q", resolved.Prompt)
	}
	if resolved.NegativePrompt != "gpt negative prompt" {
		t.Fatalf("302-gpt-image negative prompt should use background gpt negative prompt, got: %q", resolved.NegativePrompt)
	}
}
