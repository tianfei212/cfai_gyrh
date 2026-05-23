package frontendauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gyrh-go-v2/backend/internal/config"
)

const CookieName = "gyrh_frontend_token"

type Claims struct {
	Username  string `json:"username"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp"`
}

type Session struct {
	Username  string `json:"username"`
	Role      string `json:"role"`
	Token     string `json:"token,omitempty"`
	ExpiresAt int64  `json:"expires_at"`
}

type Service struct {
	cfg         config.FrontendAuthConfig
	users       map[string]config.FrontendUserConfig
	projectRoot string
	mu          sync.RWMutex
}

type snapshot struct {
	cfg   config.FrontendAuthConfig
	users map[string]config.FrontendUserConfig
}

func NewService(cfg config.FrontendAuthConfig, projectRoot string) *Service {
	users := make(map[string]config.FrontendUserConfig, len(cfg.Users))
	for _, user := range cfg.Users {
		users[strings.TrimSpace(user.Username)] = user
	}
	return &Service{cfg: cfg, users: users, projectRoot: projectRoot}
}

func (s *Service) Login(username, password string) (Session, error) {
	snap, err := s.snapshot()
	if err != nil {
		return Session{}, err
	}
	user, ok := snap.users[strings.TrimSpace(username)]
	if !ok || !hmac.Equal([]byte(user.Password), []byte(password)) {
		return Session{}, fmt.Errorf("用户名或密码错误")
	}
	expiresAt := time.Now().Add(time.Duration(snap.cfg.TokenTTLMinutes) * time.Minute).Unix()
	token, err := snap.sign(Claims{
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return Session{}, err
	}
	return Session{
		Username:  user.Username,
		Role:      user.Role,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) SessionFromRequest(r *http.Request) (Session, error) {
	snap, err := s.snapshot()
	if err != nil {
		return Session{}, err
	}
	claims, err := snap.claimsFromRequest(r)
	if err != nil {
		return Session{}, err
	}
	return Session{
		Username:  claims.Username,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

func (s *Service) HasValidSession(r *http.Request) bool {
	_, err := s.SessionFromRequest(r)
	return err == nil
}

func RealIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for _, part := range parts {
			if ip := strings.TrimSpace(part); ip != "" {
				return ip
			}
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func (s *Service) snapshot() (*snapshot, error) {
	if err := s.reload(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make(map[string]config.FrontendUserConfig, len(s.users))
	for username, user := range s.users {
		users[username] = user
	}
	return &snapshot{cfg: s.cfg, users: users}, nil
}

func (s *Service) reload() error {
	if s.projectRoot == "" {
		return nil
	}
	values, err := config.LoadDotEnvValues(filepath.Join(s.projectRoot, ".env.local"))
	if err != nil {
		return err
	}
	cfg, err := config.FrontendAuthFromValues(values)
	if err != nil {
		return err
	}
	next := NewService(cfg, s.projectRoot)
	s.mu.Lock()
	s.cfg = next.cfg
	s.users = next.users
	s.mu.Unlock()
	return nil
}

func (s *snapshot) claimsFromRequest(r *http.Request) (*Claims, error) {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(value, "Bearer ") {
		return s.verify(strings.TrimSpace(strings.TrimPrefix(value, "Bearer ")))
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, fmt.Errorf("missing bearer token")
	}
	return s.verify(strings.TrimSpace(cookie.Value))
}

func (s *snapshot) sign(claims Claims) (string, error) {
	headerJSON, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload
	return unsigned + "." + s.signature(unsigned), nil
}

func (s *snapshot) verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token")
	}
	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(s.signature(unsigned))) {
		return nil, fmt.Errorf("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if claims.ExpiresAt <= time.Now().Unix() {
		return nil, fmt.Errorf("expired token")
	}
	if _, ok := s.users[claims.Username]; !ok {
		return nil, fmt.Errorf("unknown user")
	}
	return &claims, nil
}

func (s *snapshot) signature(unsigned string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
