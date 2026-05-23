package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"gyrh-go-v2/backend/internal/api/handler"
	"gyrh-go-v2/backend/internal/api/middleware"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/pkg/httpx"
)

// RegisterRoutes 注册全部 HTTP 路由。
func RegisterRoutes(
	router *mux.Router,
	imageHandler *handler.ImageHandler,
	referenceHandler *handler.ReferenceHandler,
	skillHandler *handler.SkillHandler,
	llmPromptTemplateHandler *handler.LLMPromptTemplateHandler,
	backgroundPromptHandler *handler.BackgroundPromptHandler,
	stylePromptHandler *handler.StylePromptHandler,
	frontendAuthHandler *handler.FrontendAuthHandler,
	authConfig *middleware.AuthConfig,
) {
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthCheck).Methods(http.MethodGet)
	api.HandleFunc("/frontend-auth/login", frontendAuthHandler.Login).Methods(http.MethodPost)
	api.HandleFunc("/frontend-auth/session", frontendAuthHandler.Session).Methods(http.MethodGet)
	api.HandleFunc("/frontend-auth/logout", frontendAuthHandler.Logout).Methods(http.MethodPost)

	sessionAPI := api.NewRoute().Subrouter()
	sessionAPI.Use(frontendAuthHandler.RequireSession)
	sessionAPI.HandleFunc("/skills/active", skillHandler.GetActive).Methods(http.MethodGet)
	sessionAPI.HandleFunc("/style-prompts", stylePromptHandler.List).Methods(http.MethodGet)

	// 图片资源常用于 <img> 标签，不能依赖自定义 Header，但必须有前端登录 Cookie。
	sessionAPI.Handle("/images/thumbnail", adaptErr(imageHandler.Thumbnail)).Methods(http.MethodGet)
	sessionAPI.Handle("/images/download", adaptErr(imageHandler.Download)).Methods(http.MethodGet)
	sessionAPI.Handle("/images/view", adaptErr(imageHandler.View)).Methods(http.MethodGet)

	protected := api.NewRoute().Subrouter()
	protected.Use(frontendAuthHandler.RequireSession)
	protected.Use(middleware.Auth(authConfig))

	protected.Handle("/images", adaptErr(imageHandler.List)).Methods(http.MethodGet)
	protected.Handle("/images/upload", adaptErr(imageHandler.Upload)).Methods(http.MethodPost)
	protected.Handle("/images/rewrite", adaptErr(imageHandler.Rewrite)).Methods(http.MethodPost)
	protected.Handle("/images/rewrite/tasks/{id}", adaptErr(imageHandler.RewriteTask)).Methods(http.MethodGet)
	protected.Handle("/images/rewrite/tasks/{id}/events", adaptErr(imageHandler.RewriteTaskEvents)).Methods(http.MethodGet)
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

	protected.HandleFunc("/llm-prompt-templates", llmPromptTemplateHandler.List).Methods(http.MethodGet)
	protected.HandleFunc("/llm-prompt-templates/{id}", llmPromptTemplateHandler.Get).Methods(http.MethodGet)
	protected.HandleFunc("/llm-prompt-templates", llmPromptTemplateHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/llm-prompt-templates/{id}", llmPromptTemplateHandler.Update).Methods(http.MethodPut)
	protected.HandleFunc("/llm-prompt-templates/{id}", llmPromptTemplateHandler.Delete).Methods(http.MethodDelete)

	protected.HandleFunc("/background-prompts", backgroundPromptHandler.List).Methods(http.MethodGet)
	protected.HandleFunc("/background-prompts/import", backgroundPromptHandler.Import).Methods(http.MethodPost)
	protected.HandleFunc("/background-prompts/suggest-defaults", backgroundPromptHandler.SuggestDefaults).Methods(http.MethodPost)
	protected.HandleFunc("/background-prompts/sync-remote", backgroundPromptHandler.SyncRemote).Methods(http.MethodPost)
	protected.HandleFunc("/background-prompts/sync-english", backgroundPromptHandler.SyncEnglish).Methods(http.MethodPost)
	protected.HandleFunc("/background-prompts/{id}", backgroundPromptHandler.Get).Methods(http.MethodGet)
	protected.HandleFunc("/background-prompts", backgroundPromptHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/background-prompts/{id}", backgroundPromptHandler.Update).Methods(http.MethodPut)
	protected.HandleFunc("/background-prompts/{id}", backgroundPromptHandler.Delete).Methods(http.MethodDelete)

	protected.HandleFunc("/style-prompts/{id}", stylePromptHandler.Get).Methods(http.MethodGet)
	protected.HandleFunc("/style-prompts", stylePromptHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/style-prompts/{id}", stylePromptHandler.Update).Methods(http.MethodPut)
	protected.HandleFunc("/style-prompts/{id}", stylePromptHandler.Delete).Methods(http.MethodDelete)
}

func adaptErr(fn func(context.Context, http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(r.Context(), w, r); err != nil {
			// 在服务端打印错误日志
			logger.Error("[%s] %s %s - Handler Error: %v", r.Method, r.Host, r.URL.Path, err)
			httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, err.Error()))
		}
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]string{"status": "ok"}))
}
