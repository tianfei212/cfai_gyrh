package qwen

import (
	"fmt"
	"os"
	"path/filepath"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
)

type defaultTemplate struct {
	Name        string
	TemplateKey string
	Description string
	FileName    string
}

// DefaultTemplates 返回 Qwen 能力依赖的默认模板。
func DefaultTemplates() []defaultTemplate {
	return []defaultTemplate{
		{
			Name:        "Qwen 背景图默认建议",
			TemplateKey: db.TemplateKeyQwenBackgroundSuggest,
			Description: "根据背景图生成 Gemini/Wan 的中英双语正向与反向提示词建议。",
			FileName:    "qwen_background_prompt_suggestion.md",
		},
		{
			Name:        "Qwen 中文同步英文",
			TemplateKey: db.TemplateKeyQwenSyncEnglish,
			Description: "根据当前中文提示词同步生成 Gemini/Wan 的英文提示词。",
			FileName:    "qwen_sync_english.md",
		},
	}
}

// EnsureDefaultTemplates 确保默认模板已写入数据库。
func EnsureDefaultTemplates(repo *db.LLMPromptTemplateRepo, skillsDir string) error {
	if repo == nil {
		return nil
	}
	for _, item := range DefaultTemplates() {
		content, err := loadTemplateContent(skillsDir, item.FileName)
		if err != nil {
			logger.Warn("读取 Qwen 模板文件失败 (%s): %v", item.FileName, err)
			continue
		}
		if err := repo.UpsertByKey(item.Name, item.TemplateKey, content, item.Description); err != nil {
			return err
		}
	}
	return nil
}

func loadTemplateContent(skillsDir, fileName string) (string, error) {
	path := filepath.Join(skillsDir, fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取本地文件失败: %w", err)
	}
	return string(data), nil
}
