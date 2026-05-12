package logging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

const maxPayloadLogBytes = 16 * 1024

var secretKeyParts = []string{
	"authorization",
	"cookie",
	"api_key",
	"apikey",
	"token",
	"secret",
	"private_key",
	"signature",
}

// SanitizeHeaders 复制并脱敏 HTTP 头，避免 debug 日志泄露密钥。
func SanitizeHeaders(headers http.Header) http.Header {
	sanitized := make(http.Header, len(headers))
	for key, values := range headers {
		nextValues := make([]string, len(values))
		copy(nextValues, values)
		if isSecretKey(key) {
			for index := range nextValues {
				nextValues[index] = "[已脱敏]"
			}
		}
		sanitized[key] = nextValues
	}
	return sanitized
}

// SanitizePayload 脱敏常见密钥字段，并限制日志 payload 最大长度。
func SanitizePayload(payload []byte) []byte {
	limited := payload
	if len(limited) > maxPayloadLogBytes {
		limited = limited[:maxPayloadLogBytes]
	}

	var decoded any
	if err := json.Unmarshal(limited, &decoded); err != nil {
		return maskPlainPayload(limited)
	}
	cleaned := sanitizeJSONValue(decoded)
	data, err := json.Marshal(cleaned)
	if err != nil {
		return maskPlainPayload(limited)
	}
	if len(payload) > maxPayloadLogBytes {
		data = append(data, []byte("...[已截断]")...)
	}
	return data
}

func sanitizeJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSecretKey(key) {
				result[key] = "[已脱敏]"
				continue
			}
			result[key] = sanitizeJSONValue(item)
		}
		return result
	case []any:
		for index, item := range typed {
			typed[index] = sanitizeJSONValue(item)
		}
		return typed
	default:
		return typed
	}
}

func maskPlainPayload(payload []byte) []byte {
	result := make([]byte, len(payload))
	copy(result, payload)
	for _, key := range secretKeyParts {
		result = bytes.ReplaceAll(result, []byte(key), []byte("[敏感字段]"))
	}
	return result
}

func isSecretKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, part := range secretKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
