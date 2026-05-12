package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gyrh-go-v2/backend/internal/config"
)

// prepareAliOSSRuntimeFiles 根据当前配置生成 aliOSS 子进程需要读取的运行时配置文件。
// 返回背景素材服务和生成图服务各自的配置路径，供启动 manager 时使用。
func prepareAliOSSRuntimeFiles(cfg *config.Config) (string, string, error) {
	rootDir, err := config.FindProjectRoot()
	if err != nil {
		return "", "", err
	}

	backgroundConfigPath := filepath.Join(rootDir, "configs", "alioss-agent.yaml")
	generatedConfigPath := filepath.Join(rootDir, "configs", "alioss-agent-generated.yaml")
	if err := os.MkdirAll(filepath.Dir(backgroundConfigPath), 0755); err != nil {
		return "", "", err
	}

	content := fmt.Sprintf(`oss:
  endpoint: %q
  bucket_name: %q
  bucket_prefix: %q
  generated_bucket_prefix: %q

server:
  port: %d
  link_expire_seconds: 3600
  openai_api_key: %q
`, cfg.AliOSS.Endpoint, cfg.AliOSS.BucketName, cfg.AliOSS.BackgroundBucketPrefix, cfg.AliOSS.GeneratedBucketPrefix, cfg.AliOSS.Port, cfg.AliOSS.OpenAIAPIKey)

	generatedContent := fmt.Sprintf(`oss:
  endpoint: %q
  bucket_name: %q
  bucket_prefix: %q
  generated_bucket_prefix: %q

server:
  port: %d
  link_expire_seconds: 3600
  openai_api_key: %q
`, cfg.AliOSS.Endpoint, cfg.AliOSS.BucketName, cfg.AliOSS.GeneratedBucketPrefix, cfg.AliOSS.GeneratedBucketPrefix, cfg.AliOSS.GeneratedPort, cfg.AliOSS.OpenAIAPIKey)

	if err := os.WriteFile(backgroundConfigPath, []byte(content), 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(generatedConfigPath, []byte(generatedContent), 0644); err != nil {
		return "", "", err
	}
	return backgroundConfigPath, generatedConfigPath, nil
}
