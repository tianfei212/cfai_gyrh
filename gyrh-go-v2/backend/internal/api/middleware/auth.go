package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"gyrh-go-v2/backend/pkg/httpx"
)

const (
	// HeaderXRealIP 请求头：客户端真实IP
	HeaderXRealIP = "X-Real-IP"
	// HeaderXPublicKey 请求头：公钥
	HeaderXPublicKey = "X-Public-Key"
	// HeaderXTimestamp 请求头：时间戳
	HeaderXTimestamp = "X-Timestamp"
	// HeaderXSignature 请求头：签名
	HeaderXSignature = "X-Signature"

	// ContextKeyClientIP context中存储客户端IP的键
	ContextKeyClientIP contextKey = "client_ip"
	// ContextKeyPublicKey context中存储公钥的键
	ContextKeyPublicKey contextKey = "public_key"

	// TimestampWindow 时间戳验证窗口（5分钟）
	TimestampWindow = 5 * time.Minute
)

// contextKey 用于context中存储值的键类型
type contextKey string

// AuthConfig 鉴权中间件配置
type AuthConfig struct {
	// PrivateKeyFetcher 私钥查询函数，通过公钥获取对应的私钥
	// 如果返回空字符串，表示该公钥无效
	PrivateKeyFetcher func(publicKey string) string
}

// Auth 鉴权中间件（IP + 私钥 + 公钥 三因素鉴权）
// 从请求头获取 X-Real-IP, X-Public-Key, X-Timestamp, X-Signature
// 验证流程：
//  1. 验证时间戳（5分钟窗口）
//  2. 通过公钥查询私钥
//  3. 验证签名：HMAC-SHA256(私钥, IP + 公钥 + 时间戳)
//  4. 鉴权成功后将 client_ip, public_key 存入 context
func Auth(config *AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. 获取请求头
			clientIP := r.Header.Get(HeaderXRealIP)
			publicKey := r.Header.Get(HeaderXPublicKey)
			timestampStr := r.Header.Get(HeaderXTimestamp)
			signature := r.Header.Get(HeaderXSignature)

			// 2. 验证必填参数
			if clientIP == "" {
				writeAuthError(w, http.StatusUnauthorized, "缺少请求头 X-Real-IP")
				return
			}
			if publicKey == "" {
				writeAuthError(w, http.StatusUnauthorized, "缺少请求头 X-Public-Key")
				return
			}
			if timestampStr == "" {
				writeAuthError(w, http.StatusUnauthorized, "缺少请求头 X-Timestamp")
				return
			}
			if signature == "" {
				writeAuthError(w, http.StatusUnauthorized, "缺少请求头 X-Signature")
				return
			}

			// 3. 验证时间戳
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "无效的时间戳格式")
				return
			}
			requestTime := time.Unix(timestamp, 0)
			now := time.Now()
			if now.Sub(requestTime) > TimestampWindow || requestTime.Sub(now) > TimestampWindow {
				writeAuthError(w, http.StatusUnauthorized, "请求已过期，请重新签名")
				return
			}

			// 4. 通过公钥查询私钥
			privateKey := config.PrivateKeyFetcher(publicKey)
			if privateKey == "" {
				writeAuthError(w, http.StatusUnauthorized, "无效的公钥")
				return
			}

			// 5. 验证签名
			// 签名内容：IP + 公钥 + 时间戳
			signContent := clientIP + publicKey + timestampStr
			expectedSignature := computeHMAC(signContent, privateKey)
			if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
				writeAuthError(w, http.StatusUnauthorized, "签名验证失败")
				return
			}

			// 6. 鉴权成功，将信息存入context
			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyClientIP, clientIP)
			ctx = context.WithValue(ctx, ContextKeyPublicKey, publicKey)

			// 继续处理请求
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// computeHMAC 计算HMAC-SHA256签名
// data 待签名的内容
// key 密钥
// 返回十六进制编码的签名字符串
func computeHMAC(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// writeAuthError 写入鉴权错误响应
func writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	resp := httpx.Error(statusCode, message)
	httpx.WriteJSON(w, statusCode, resp)
}

// GetClientIP 从context中获取客户端IP
func GetClientIP(ctx context.Context) string {
	if val := ctx.Value(ContextKeyClientIP); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetPublicKey 从context中获取公钥
func GetPublicKey(ctx context.Context) string {
	if val := ctx.Value(ContextKeyPublicKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// RequestBodyReader 请求体读取辅助函数
// 将请求体读出并重新放回请求中（用于多次读取body的场景）
func RequestBodyReader(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %w", err)
	}
	// 重新填充body，以便后续处理器可以继续读取
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}
