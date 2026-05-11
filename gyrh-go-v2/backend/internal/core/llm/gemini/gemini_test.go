package gemini

import (
	"encoding/json"
	"testing"
)

func TestEncodeImagePartUsesGeminiInlineDataField(t *testing.T) {
	part := encodeImagePart([]byte("fake-image"), "image/png")

	payload, err := json.Marshal(part)
	if err != nil {
		t.Fatalf("marshal part: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal part: %v", err)
	}

	if _, ok := decoded["imageData"]; ok {
		t.Fatalf("Gemini request must not use imageData field: %s", payload)
	}

	inlineData, ok := decoded["inline_data"].(map[string]any)
	if !ok {
		t.Fatalf("Gemini request should use inline_data field, got: %s", payload)
	}
	if got := inlineData["mime_type"]; got != "image/png" {
		t.Fatalf("mime_type = %v, want image/png", got)
	}
	if got := inlineData["data"]; got != "ZmFrZS1pbWFnZQ==" {
		t.Fatalf("data = %v, want base64 encoded image", got)
	}
}

func TestParseResponseAcceptsCamelCaseInlineData(t *testing.T) {
	raw := []byte(`{
		"candidates": [
			{
				"content": {
					"parts": [
						{
							"inlineData": {
								"mimeType": "image/png",
								"data": "aW1hZ2U="
							}
						}
					]
				}
			}
		]
	}`)

	var resp GeminiResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	result, err := (&GeminiHandler{}).parseResponse(&resp)
	if err != nil {
		t.Fatalf("parseResponse returned error: %v", err)
	}
	if result.Base64 != "aW1hZ2U=" {
		t.Fatalf("Base64 = %q, want camelCase inlineData payload", result.Base64)
	}
}
