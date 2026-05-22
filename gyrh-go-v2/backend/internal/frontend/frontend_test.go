package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesSPAFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin_viewer", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<div id="root"></div>`) {
		t.Fatalf("expected embedded index.html, got body %q", rec.Body.String())
	}
}

func TestHandlerServesEmbeddedMediaPipeAssets(t *testing.T) {
	paths := []string{
		"/models/selfie_segmentation/selfie_segmentation.binarypb",
		"/models/selfie_segmentation/selfie_segmentation.js",
		"/models/selfie_segmentation/selfie_segmentation_solution_simd_wasm_bin.js",
		"/models/selfie_segmentation/selfie_segmentation_solution_wasm_bin.js",
		"/models/selfie_segmentation/selfie_segmentation_landscape.tflite",
		"/models/selfie_segmentation/selfie_segmentation.tflite",
		"/models/selfie_segmentation/selfie_segmentation_solution_simd_wasm_bin.wasm",
		"/models/selfie_segmentation/selfie_segmentation_solution_wasm_bin.wasm",
		"/models/selfie_segmentation/selfie_segmentation_solution_simd_wasm_bin.data",
	}

	for _, assetPath := range paths {
		t.Run(assetPath, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, assetPath, nil)
			rec := httptest.NewRecorder()

			Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}
			if rec.Header().Get("Cache-Control") != immutableCacheControl {
				t.Fatalf("expected immutable cache header, got %q", rec.Header().Get("Cache-Control"))
			}
			if assetPath != "/models/selfie_segmentation/selfie_segmentation_solution_simd_wasm_bin.data" && rec.Body.Len() == 0 {
				t.Fatal("expected embedded MediaPipe asset body")
			}
		})
	}
}

func TestHandlerDoesNotFallbackStaticAssetRequestsToIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/models/selfie_segmentation/missing.wasm", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing static asset status 404, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `<div id="root"></div>`) {
		t.Fatal("missing static asset must not return SPA index.html")
	}
}
