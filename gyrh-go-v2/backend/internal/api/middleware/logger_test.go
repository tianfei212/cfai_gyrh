package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggerPreservesFlusher(t *testing.T) {
	handler := Logger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Fatalf("wrapped response writer should implement http.Flusher")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}
