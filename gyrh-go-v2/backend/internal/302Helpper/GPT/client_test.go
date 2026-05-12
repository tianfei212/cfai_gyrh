package GPT

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateTaskSubmitsMultipartTo302Edits(t *testing.T) {
	t.Setenv("PROVIDER_302_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" {
			t.Fatalf("path = %s, want /v1/images/edits", r.URL.Path)
		}
		if r.URL.Query().Get("response_format") != "url" {
			t.Fatalf("response_format = %q, want url", r.URL.Query().Get("response_format"))
		}
		if r.URL.Query().Get("async") != "true" {
			t.Fatalf("async = %q, want true", r.URL.Query().Get("async"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("prompt"); got != "test prompt" {
			t.Fatalf("prompt = %q, want test prompt", got)
		}
		if got := r.FormValue("model"); got != "gpt-image-2" {
			t.Fatalf("model = %q, want gpt-image-2", got)
		}
		if got := r.FormValue("quality"); got != "high" {
			t.Fatalf("quality = %q, want high", got)
		}
		if got := r.FormValue("output_format"); got != "png" {
			t.Fatalf("output_format = %q, want png", got)
		}
		if got := r.FormValue("input_fidelity"); got != "high" {
			t.Fatalf("input_fidelity = %q, want high", got)
		}
		if len(r.MultipartForm.File["image"]) != 1 {
			t.Fatalf("image files = %d, want 1", len(r.MultipartForm.File["image"]))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-123"})
	}))
	defer server.Close()

	client := NewClient(Config{
		Enabled:             true,
		BaseURL:             server.URL,
		ModelName:           "gpt-image-2",
		PollIntervalSeconds: 1,
		MaxWaitSeconds:      3,
	})

	taskID, err := client.CreateTask(context.Background(), ComposeRequest{
		Prompt:          "test prompt",
		ForegroundImage: []byte("foreground"),
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if taskID != "task-123" {
		t.Fatalf("taskID = %q, want task-123", taskID)
	}
}

func TestWaitResultPollsAsyncResultAndDownloadsImage(t *testing.T) {
	t.Setenv("PROVIDER_302_API_KEY", "test-key")

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer imageServer.Close()

	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/async_result" {
			t.Fatalf("path = %s, want /async_result", r.URL.Path)
		}
		if r.URL.Query().Get("task_id") != "task-123" {
			t.Fatalf("task_id = %q, want task-123", r.URL.Query().Get("task_id"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", r.Header.Get("Authorization"))
		}
		pollCount++
		if pollCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 102})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status_code": 200,
			"data":        imageServer.URL + "/result.png",
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		Enabled:             true,
		BaseURL:             server.URL,
		ModelName:           "gpt-image-2",
		PollIntervalSeconds: 1,
		MaxWaitSeconds:      3,
	})
	client.pollInterval = time.Millisecond

	result, err := client.WaitResult(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("WaitResult returned error: %v", err)
	}
	if string(result.Image) != "png-bytes" {
		t.Fatalf("image = %q, want png-bytes", string(result.Image))
	}
	if result.URL != imageServer.URL+"/result.png" {
		t.Fatalf("URL = %q", result.URL)
	}
}

func TestWaitResultReturnsUpstreamError(t *testing.T) {
	t.Setenv("PROVIDER_302_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status_code": 500,
			"err":         "bad upstream",
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		Enabled:             true,
		BaseURL:             server.URL,
		ModelName:           "gpt-image-2",
		PollIntervalSeconds: 1,
		MaxWaitSeconds:      3,
	})
	client.pollInterval = time.Millisecond

	_, err := client.WaitResult(context.Background(), "task-123")
	if err == nil || !strings.Contains(err.Error(), "bad upstream") {
		t.Fatalf("error = %v, want bad upstream", err)
	}
}

func TestWaitResultTreatsResultPendingAsProcessing(t *testing.T) {
	t.Setenv("PROVIDER_302_API_KEY", "test-key")

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer imageServer.Close()

	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pollCount++
		if pollCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status_code": 500,
				"err":         "result pending",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status_code": 200,
			"data":        imageServer.URL + "/result.png",
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		Enabled:             true,
		BaseURL:             server.URL,
		ModelName:           "gpt-image-2",
		PollIntervalSeconds: 1,
		MaxWaitSeconds:      3,
	})
	client.pollInterval = time.Millisecond

	result, err := client.WaitResult(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("WaitResult returned error: %v", err)
	}
	if string(result.Image) != "png-bytes" {
		t.Fatalf("image = %q, want png-bytes", string(result.Image))
	}
}

func TestCreateTaskRequiresProvider302APIKey(t *testing.T) {
	t.Setenv("PROVIDER_302_API_KEY", "")

	client := NewClient(Config{
		Enabled:        true,
		BaseURL:        "https://api.302.ai",
		ModelName:      "gpt-image-2",
		MaxWaitSeconds: 3,
	})

	_, err := client.CreateTask(context.Background(), ComposeRequest{
		Prompt:          "test prompt",
		ForegroundImage: []byte("foreground"),
	})
	if err == nil || !strings.Contains(err.Error(), "PROVIDER_302_API_KEY") {
		t.Fatalf("error = %v, want missing PROVIDER_302_API_KEY", err)
	}
}
