package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestHeader HTTP 请求头部封装
type RequestHeader struct {
	header http.Header
}

// NewRequestHeader 创建请求头部
func NewRequestHeader() *RequestHeader {
	return &RequestHeader{
		header: make(http.Header),
	}
}

// Add 添加自定义头部
func (r *RequestHeader) Add(key, value string) *RequestHeader {
	r.header.Add(key, value)
	return r
}

// Apply 应用头部到请求
func (r *RequestHeader) Apply(req *http.Request) {
	for key, values := range r.header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

// Response 统一响应封装
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(data interface{}) *Response {
	return &Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// Error 错误响应
func Error(code int, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
	}
}

// WriteJSON JSON 响应写入
func WriteJSON(w http.ResponseWriter, statusCode int, resp *Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(resp)
}

// Client HTTP 客户端封装
type Client struct {
	client      *http.Client
	baseURL     string
	requestHeader *RequestHeader
}

// NewClient 创建客户端
func NewClient(baseURL string) *Client {
	return &Client{
		client:      &http.Client{},
		baseURL:     baseURL,
		requestHeader: NewRequestHeader(),
	}
}

// SetHeader 设置全局请求头
func (c *Client) SetHeader(key, value string) *Client {
	c.requestHeader.Add(key, value)
	return c
}

// Get 发送 GET 请求
func (c *Client) Get(path string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建GET请求失败: %w", err)
	}
	c.requestHeader.Apply(req)
	return c.client.Do(req)
}

// Post 发送 POST 请求
func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建POST请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.requestHeader.Apply(req)
	return c.client.Do(req)
}

// GetWithHeader 发送带自定义头部的 GET 请求
func (c *Client) GetWithHeader(path string, header *RequestHeader) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建GET请求失败: %w", err)
	}
	c.requestHeader.Apply(req)
	header.Apply(req)
	return c.client.Do(req)
}

// PostWithHeader 发送带自定义头部的 POST 请求
func (c *Client) PostWithHeader(path string, body interface{}, header *RequestHeader) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建POST请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.requestHeader.Apply(req)
	header.Apply(req)
	return c.client.Do(req)
}
