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
