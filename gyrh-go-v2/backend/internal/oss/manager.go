package oss

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"gyrh-go-v2/backend/internal/config"
)

// Manager 负责管理 aliOSS 二进制服务的生命周期。
type Manager struct {
	cfg        *config.AliOSSConfig
	configPath string
	cmd        *exec.Cmd
}

// NewManager 创建 aliOSS 管理器。
func NewManager(cfg *config.AliOSSConfig, configPath string) *Manager {
	return &Manager{cfg: cfg, configPath: configPath}
}

// Endpoint 返回 aliOSS 本地服务地址。
func (m *Manager) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", m.cfg.Port)
}

// Start 启动 aliOSS 二进制服务。
func (m *Manager) Start(ctx context.Context) error {
	if m == nil || m.cfg == nil || !m.cfg.Enabled || !m.cfg.AutoStart {
		return nil
	}
	if m.cfg.BinaryPath == "" {
		return fmt.Errorf("alioss.binary_path 未配置")
	}

	binaryPath, err := resolveBinaryPath(m.cfg.BinaryPath)
	if err != nil {
		return err
	}

	projectRoot := filepath.Dir(filepath.Dir(binaryPath))
	args := []string{"server", "-p", strconv.Itoa(m.cfg.Port)}
	if configPath := m.resolveAgentConfigPath(projectRoot); configPath != "" {
		args = append(args, "--config", configPath)
	}

	// 注意这里去掉了 ctx，防止上层 ctx cancel 时直接把子进程干掉
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "OPENAI_API_KEY="+m.cfg.OpenAIAPIKey)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 aliOSS 二进制失败: %w", err)
	}
	m.cmd = cmd

	deadline := time.Now().Add(time.Duration(m.cfg.HealthWaitSeconds) * time.Second)
	for time.Now().Before(deadline) {
		if err := m.ping(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 等待超时，尝试读取二进制输出帮助排查问题
	return fmt.Errorf("等待 aliOSS 服务就绪超时，请检查 OSS_ACCESS_KEY_ID 是否正确配置，或者查看控制台错误输出")
}

func (m *Manager) resolveAgentConfigPath(projectRoot string) string {
	if m != nil && m.configPath != "" {
		if info, err := os.Stat(m.configPath); err == nil && !info.IsDir() {
			return m.configPath
		}
	}
	candidates := []string{
		filepath.Join(projectRoot, "configs", "alioss-agent.yaml"),
		filepath.Join(projectRoot, "config.yaml"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func resolveBinaryPath(configured string) (string, error) {
	baseDir := filepath.Dir(configured)
	platformBinary := platformBinaryName()
	candidates := []string{
		configured,
		filepath.Join(baseDir, platformBinary),
		filepath.Join(baseDir, "oss-cli-darwin-arm64"),
		filepath.Join(baseDir, "oss-cli-linux-amd64"),
		filepath.Join(filepath.Dir(configured), "alOSS_agent_go"),
		filepath.Join(filepath.Dir(configured), "oss-cli"),
		filepath.Join(filepath.Dir(configured), "alioss"),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("未找到 aliOSS 二进制，请先从 github.com/tianfei212/alOSS_agent_go 下载并配置到 %s", configured)
}

func platformBinaryName() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "oss-cli-darwin-arm64"
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			return "oss-cli-linux-amd64"
		}
	}
	return "oss-cli"
}

// Stop 优雅停止 aliOSS 服务。
func (m *Manager) Stop(ctx context.Context) error {
	if m == nil || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	_ = m.cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" {
			return err
		}
		return nil
	case <-ctx.Done():
		_ = m.cmd.Process.Kill()
		return ctx.Err()
	}
}

func (m *Manager) ping() error {
	req, err := http.NewRequest(http.MethodGet, m.Endpoint()+"/v1/files", nil)
	if err != nil {
		return err
	}
	if m.cfg.OpenAIAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.cfg.OpenAIAPIKey)
	}

	// 增加超时机制，防止假死阻塞健康检查
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return nil
	}

	return fmt.Errorf("alioss 状态码异常: %d", resp.StatusCode)
}
