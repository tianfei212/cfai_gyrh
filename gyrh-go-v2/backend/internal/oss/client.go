package oss

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OSSClient 定义统一的存储客户端接口。
type OSSClient interface {
	Upload(ctx context.Context, data []byte, filename string) (string, error)
	Download(ctx context.Context, fileID string) ([]byte, error)
	Delete(ctx context.Context, fileID string) error
	GetSignedURL(ctx context.Context, fileID string, expire int) (string, error)
}

// UploadPolicy 表示百炼临时 OSS 的上传凭证。
type UploadPolicy struct {
	Policy    string `json:"policy"`
	Signature string `json:"signature"`
	AccessKey string `json:"access_key_id"`
	Bucket    string `json:"bucket"`
	Endpoint  string `json:"endpoint"`
	ObjectKey string `json:"object_key"`
}

// UploadFileResult 表示上传结果。
type UploadFileResult struct {
	URL       string `json:"url"`
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
}

type alOSSClient struct {
	baseURL   string
	apiKey    string
	client    *http.Client
}

// NewAlOSSClient 创建 aliOSS HTTP 客户端。
func NewAlOSSClient(baseURL, apiKey string) OSSClient {
	return &alOSSClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *alOSSClient) Upload(ctx context.Context, data []byte, filename string) (string, error) {
	if len(data) == 0 {
		return "", errors.New("上传数据不能为空")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("purpose", "assistants"); err != nil {
		return "", err
	}
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := fileWriter.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/files", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 aliOSS 上传失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("aliOSS 上传失败: %d %s", resp.StatusCode, string(raw))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析 aliOSS 上传响应失败: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("aliOSS 未返回文件 id")
	}

	return result.ID, nil
}

func (c *alOSSClient) Download(ctx context.Context, fileID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/files/"+url.PathEscape(fileID)+"/content", nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 aliOSS 文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("下载 aliOSS 文件失败: %d %s", resp.StatusCode, string(raw))
	}

	return io.ReadAll(resp.Body)
}

func (c *alOSSClient) Delete(ctx context.Context, fileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/files/"+url.PathEscape(fileID), nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("删除 aliOSS 文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除 aliOSS 文件失败: %d %s", resp.StatusCode, string(raw))
	}

	return nil
}

func (c *alOSSClient) GetSignedURL(ctx context.Context, fileID string, expire int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/view/"+url.PathEscape(fileID), nil)
	if err != nil {
		return "", err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取 aliOSS 签名地址失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("获取 aliOSS 签名地址失败: %d %s", resp.StatusCode, string(raw))
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析 aliOSS 签名地址失败: %w", err)
	}
	if result.URL == "" {
		return "", fmt.Errorf("aliOSS 返回的签名地址为空")
	}

	return result.URL, nil
}

// DashScopeClient 用于本地模式下上传到阿里云百炼临时 OSS。
type DashScopeClient struct {
	apiKey         string
	uploadEndpoint string
	client         *http.Client
}

// NewDashScopeClient 创建百炼临时 OSS 客户端。
func NewDashScopeClient(apiKey string) *DashScopeClient {
	return &DashScopeClient{
		apiKey:         apiKey,
		uploadEndpoint: "https://dashscope.aliyuncs.com/api/v1/storage/upload",
		client:         &http.Client{Timeout: 60 * time.Second},
	}
}

// Upload 上传文件到百炼临时 OSS。
func (c *DashScopeClient) Upload(ctx context.Context, data []byte, filename string) (string, error) {
	policy, err := c.GetUploadPolicy(ctx)
	if err != nil {
		return "", err
	}
	result, err := c.UploadFile(ctx, data, filename, policy)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// Download 百炼临时 OSS 不支持反向下载。
func (c *DashScopeClient) Download(ctx context.Context, fileID string) ([]byte, error) {
	return nil, errors.New("DashScope 临时 OSS 不支持下载")
}

// Delete 百炼临时 OSS 不支持主动删除。
func (c *DashScopeClient) Delete(ctx context.Context, fileID string) error {
	return nil
}

// GetSignedURL 百炼临时 OSS 返回原始对象地址。
func (c *DashScopeClient) GetSignedURL(ctx context.Context, fileID string, expire int) (string, error) {
	return fileID, nil
}

// GetUploadPolicy 获取上传凭证。
func (c *DashScopeClient) GetUploadPolicy(ctx context.Context) (*UploadPolicy, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.uploadEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取 DashScope 上传凭证失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取 DashScope 上传凭证失败: %d %s", resp.StatusCode, string(raw))
	}

	var wrapper struct {
		Code int          `json:"code"`
		Data UploadPolicy `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("解析 DashScope 上传凭证失败: %w", err)
	}
	return &wrapper.Data, nil
}

// UploadFile 使用凭证上传文件。
func (c *DashScopeClient) UploadFile(ctx context.Context, data []byte, filename string, policy *UploadPolicy) (*UploadFileResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"key":                   policy.ObjectKey,
		"policy":                policy.Policy,
		"OSSAccessKeyId":        policy.AccessKey,
		"signature":             policy.Signature,
		"success_action_status": "200",
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := fileWriter.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, policy.Endpoint, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("上传到 DashScope 临时 OSS 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("上传到 DashScope 临时 OSS 失败: %d %s", resp.StatusCode, string(raw))
	}

	return &UploadFileResult{
		URL:       fmt.Sprintf("oss://%s/%s", policy.Bucket, policy.ObjectKey),
		Bucket:    policy.Bucket,
		ObjectKey: policy.ObjectKey,
	}, nil
}

// ReadLocalFile 读取本地文件。
func ReadLocalFile(path string) ([]byte, error) {
	return os.ReadFile(filepath.Clean(path))
}
