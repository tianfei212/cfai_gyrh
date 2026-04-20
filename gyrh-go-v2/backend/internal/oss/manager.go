package oss

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"gyrh-go-v2/backend/internal/config"
)

// Manager 负责管理 aliOSS 二进制服务的生命周期。
type Manager struct {
	cfg *config.AliOSSConfig
	cmd *exec.Cmd
}

// NewManager 创建 aliOSS 管理器。
func NewManager(cfg *config.AliOSSConfig) *Manager {
	return &Manager{cfg: cfg}
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
	if configPath := resolveAgentConfigPath(projectRoot); configPath != "" {
		args = append(args, "--config", configPath)
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
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

	return fmt.Errorf("等待 aliOSS 服务就绪超时")
}

func resolveAgentConfigPath(projectRoot string) string {
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
	candidates := []string{
		configured,
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return nil
	}

	return fmt.Errorf("alioss 状态码异常: %d", resp.StatusCode)
}
