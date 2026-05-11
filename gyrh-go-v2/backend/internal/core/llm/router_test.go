package llm

import (
	"strings"
	"testing"
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
