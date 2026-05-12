package middleware

import (
	"net/http"
	"time"

	"gyrh-go-v2/backend/internal/logger"
)

// responseWriter 包装 http.ResponseWriter 以拦截状态码
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Logger 记录 HTTP 请求和响应信息的中间件
func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 包装 ResponseWriter 以获取状态码
			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK, // 默认状态码
			}

			// 调用下一个处理器
			next.ServeHTTP(rw, r)

			// 计算耗时
			duration := time.Since(start)

			// 记录日志
			logger.Info(
				"[%s] %s %s | status: %d | duration: %v | size: %d | ip: %s",
				r.Method,
				r.Host,
				r.URL.Path,
				rw.status,
				duration,
				rw.size,
				r.RemoteAddr,
			)
		})
	}
}
