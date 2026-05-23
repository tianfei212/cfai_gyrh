package handler

import (
	"net/http"
	"strings"
	"time"

	"gyrh-go-v2/backend/internal/application/frontendauth"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/pkg/httpx"
)

const (
	frontendAuthUsernameHeader = "HOME1"
	frontendAuthPasswordHeader = "HOME2"
	FrontendAuthCookieName     = frontendauth.CookieName
)

type FrontendAuthHandler struct {
	service *frontendauth.Service
}

func NewFrontendAuthHandler(service *frontendauth.Service) *FrontendAuthHandler {
	return &FrontendAuthHandler{service: service}
}

func (h *FrontendAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.Header.Get(frontendAuthUsernameHeader))
	password := r.Header.Get(frontendAuthPasswordHeader)
	realIP := frontendauth.RealIP(r)
	userAgent := r.UserAgent()
	session, err := h.service.Login(username, password)
	if err != nil {
		logger.Warn("前端登录失败: username=%q real_ip=%s remote_addr=%s user_agent=%q error=%v", username, realIP, r.RemoteAddr, userAgent, err)
		logger.LoginError(realIP, username, r.RemoteAddr, userAgent, err)
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.Error(1, "用户名或密码错误"))
		return
	}
	logger.Info("前端登录成功: username=%q role=%q real_ip=%s remote_addr=%s user_agent=%q expires_at=%d", session.Username, session.Role, realIP, r.RemoteAddr, userAgent, session.ExpiresAt)
	http.SetCookie(w, &http.Cookie{
		Name:     FrontendAuthCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  time.Unix(session.ExpiresAt, 0),
		MaxAge:   int(time.Until(time.Unix(session.ExpiresAt, 0)).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	_ = httpx.WriteJSON(w, http.StatusOK, httpx.Success(session))
}

func (h *FrontendAuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.SessionFromRequest(r)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.Error(1, "登录已失效"))
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, httpx.Success(session))
}

func (h *FrontendAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     FrontendAuthCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	_ = httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]bool{"ok": true}))
}

func (h *FrontendAuthHandler) HasValidSession(r *http.Request) bool {
	return h.service.HasValidSession(r)
}

func (h *FrontendAuthHandler) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.HasValidSession(r) {
			_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.Error(1, "请先登录"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
