package middleware

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// CORSConfig CORS中间件配置
type CORSConfig struct {
	// AllowOrigins 允许的来源列表，为空则允许所有来源
	AllowOrigins []string
	// AllowMethods 允许的HTTP方法列表
	AllowMethods []string
	// AllowHeaders 允许的请求头列表
	AllowHeaders []string
	// AllowCredentials 是否允许携带凭证
	AllowCredentials bool
	// MaxAge 预检请求缓存时间（秒）
	MaxAge int
}

// DefaultCORSConfig 返回默认CORS配置
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		// 允许所有来源
		AllowOrigins: []string{},
		// 允许的方法
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		// 允许的请求头
		AllowHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Real-IP",
			"X-Public-Key",
			"X-Timestamp",
			"X-Signature",
		},
		// 允许携带凭证
		AllowCredentials: true,
		// 预检请求缓存时间：5分钟
		MaxAge: 300,
	}
}

// CORS CORS中间件
// 允许所有来源访问，支持GET, POST, PUT, DELETE, OPTIONS方法
// 支持常见的请求头：Content-Type, Authorization, X-Real-IP, X-Public-Key, X-Timestamp, X-Signature
func CORS() func(http.Handler) http.Handler {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig 使用自定义配置创建CORS中间件
func CORSWithConfig(config *CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 设置CORS响应头
			setCORSHeaders(w, r, config)

			// 处理预检请求
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// 调用下一个处理器
			next.ServeHTTP(w, r)
		})
	}
}

// setCORSHeaders 设置CORS响应头
func setCORSHeaders(w http.ResponseWriter, r *http.Request, config *CORSConfig) {
	origin := r.Header.Get("Origin")

	// 检查是否允许该来源
	allowOrigin := ""
	if len(config.AllowOrigins) == 0 {
		// 没有配置允许列表，允许所有来源
		allowOrigin = "*"
	} else if slices.Contains(config.AllowOrigins, origin) {
		// 检查请求来源是否在允许列表中
		allowOrigin = origin
	}

	// 如果设置了允许来源且需要凭证，不能使用通配符*
	if allowOrigin != "" && (!config.AllowCredentials || allowOrigin != "*") {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
	}

	// 设置凭证支持
	if config.AllowCredentials && allowOrigin != "*" {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// 设置允许的方法
	if len(config.AllowMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
	}

	// 设置允许的请求头
	if len(config.AllowHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
	}

	// 设置预检请求缓存时间
	if config.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
	}
}

// joinStrings 将字符串切片连接成逗号分隔的字符串
// 注意：此函数已弃用，请使用 strings.Join 代替
func joinStrings(items []string, sep string) string {
	return strings.Join(items, sep)
}
