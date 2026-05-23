package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist
var embedded embed.FS

const immutableCacheControl = "public, max-age=31536000, immutable"

type AuthGuard interface {
	HasValidSession(r *http.Request) bool
}

// Handler serves the embedded React app and falls back to index.html for SPA routes.
func Handler() http.Handler {
	return HandlerWithAuth(nil)
}

func HandlerWithAuth(auth AuthGuard) http.Handler {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return http.NotFoundHandler()
	}
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if requiresLogin(r.URL.Path) && auth != nil && !auth.HasValidSession(r) {
			target := "/login?next=" + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusFound)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			cleanPath = "index.html"
		}

		if isImmutableAsset(cleanPath) {
			w.Header().Set("Cache-Control", immutableCacheControl)
		}

		if info, err := fs.Stat(dist, cleanPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		if isStaticAssetRequest(cleanPath) {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(index)
	})
}

func requiresLogin(requestPath string) bool {
	if requestPath == "" || requestPath == "/" {
		return true
	}
	cleanPath := path.Clean("/" + requestPath)
	if cleanPath == "/login" || strings.HasPrefix(cleanPath, "/assets/") {
		return false
	}
	if isStaticAssetRequest(strings.TrimPrefix(cleanPath, "/")) {
		return false
	}
	return true
}

func isImmutableAsset(name string) bool {
	return strings.HasPrefix(name, "assets/") ||
		strings.HasPrefix(name, "models/selfie_segmentation/") ||
		strings.HasPrefix(name, "branding/")
}

func isStaticAssetRequest(name string) bool {
	return isImmutableAsset(name) || strings.Contains(path.Base(name), ".")
}
