package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"gyrh-go-v2/backend/internal/api/handler"
	"gyrh-go-v2/backend/internal/api/middleware"
	"gyrh-go-v2/backend/pkg/httpx"
)

// RegisterRoutes 注册全部 HTTP 路由。
func RegisterRoutes(
	router *mux.Router,
	imageHandler *handler.ImageHandler,
	referenceHandler *handler.ReferenceHandler,
	skillHandler *handler.SkillHandler,
	authConfig *middleware.AuthConfig,
) {
	router.Use(middleware.CORS())

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthCheck).Methods(http.MethodGet)
	api.HandleFunc("/skills/active", skillHandler.GetActive).Methods(http.MethodGet)

	protected := api.NewRoute().Subrouter()
	protected.Use(middleware.Auth(authConfig))

	protected.Handle("/images", adaptErr(imageHandler.List)).Methods(http.MethodGet)
	protected.Handle("/images/download", adaptErr(imageHandler.Download)).Methods(http.MethodGet)
	protected.Handle("/images/view", adaptErr(imageHandler.View)).Methods(http.MethodGet)
	protected.Handle("/images/upload", adaptErr(imageHandler.Upload)).Methods(http.MethodPost)
	protected.Handle("/images/rewrite", adaptErr(imageHandler.Rewrite)).Methods(http.MethodPost)
	protected.Handle("/images", adaptErr(imageHandler.Delete)).Methods(http.MethodDelete)

	protected.HandleFunc("/references", referenceHandler.List).Methods(http.MethodGet)
	protected.HandleFunc("/references/view", referenceHandler.View).Methods(http.MethodGet)
	protected.HandleFunc("/references/upload", referenceHandler.Upload).Methods(http.MethodPost)
	protected.HandleFunc("/references/{id}", referenceHandler.Update).Methods(http.MethodPut)
	protected.HandleFunc("/references", referenceHandler.Delete).Methods(http.MethodDelete)

	protected.HandleFunc("/skills", skillHandler.List).Methods(http.MethodGet)
	protected.HandleFunc("/skills/{id}", skillHandler.Get).Methods(http.MethodGet)
	protected.HandleFunc("/skills", skillHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/skills/{id}", skillHandler.Update).Methods(http.MethodPut)
	protected.HandleFunc("/skills/{id}", skillHandler.Delete).Methods(http.MethodDelete)
}

func adaptErr(fn func(context.Context, http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(r.Context(), w, r); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, err.Error()))
		}
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]string{"status": "ok"}))
}
