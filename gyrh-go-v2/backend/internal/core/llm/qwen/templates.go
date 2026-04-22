package qwen

import (
	"strings"

	"gyrh-go-v2/backend/internal/db"
)

type defaultTemplate struct {
	Name        string
	TemplateKey string
	Content     string
	Description string
}

// DefaultTemplates 返回 Qwen 能力依赖的默认模板。
func DefaultTemplates() []defaultTemplate {
	return []defaultTemplate{
		{
			Name:        "Qwen 背景图默认建议",
			TemplateKey: db.TemplateKeyQwenBackgroundSuggest,
			Description: "根据背景图生成 Gemini/Wan 的中英双语正向与反向提示词建议。",
			Content: strings.TrimSpace(`
你是专业的图像融图提示词工程师。请分析这张背景图，并为“将单个人物自然融合进该背景”生成高质量建议。

请同时给出 Gemini 和 Wan 两套建议，每套都包含：
1. 正向提示词中文
2. 正向提示词英文
3. 反向提示词中文
4. 反向提示词英文

要求：
- 必须仔细分析背景图的构图、透视和可用空间，找出人物最应该出现、最自然的放置位置（例如“站在画面左侧的草地上”、“坐在画面中央的长椅上”、“靠在右侧的墙边”等），并在正向提示词中明确描述该位置。
- 聚焦于人物与背景的融合，不要描述人物服饰细节。
- 强调光照方向、透视关系、空间层次、景深、色温、氛围、边缘融合、阴影一致性。
- 反向提示词要避免人物悬浮、透视错误、边缘锯齿、光影不一致、色调冲突、背景穿帮等问题。
- 中文适合给中国运营和前端直接使用，英文适合直接送给图像模型。
- 返回 JSON，禁止 markdown、禁止代码块、禁止额外解释。

返回格式：
{
  "gemini_prompt_zh": "",
  "gemini_prompt_en": "",
  "gemini_negative_prompt_zh": "",
  "gemini_negative_prompt_en": "",
  "wan_prompt_zh": "",
  "wan_prompt_en": "",
  "wan_negative_prompt_zh": "",
  "wan_negative_prompt_en": ""
}`),
		},
		{
			Name:        "Qwen 中文同步英文",
			TemplateKey: db.TemplateKeyQwenSyncEnglish,
			Description: "根据当前中文提示词同步生成 Gemini/Wan 的英文提示词。",
			Content: strings.TrimSpace(`
你是专业的图像生成提示词翻译助手。请把输入 JSON 中的中文提示词同步改写为自然、专业、适合图像模型使用的英文提示词。

要求：
- 保持原意，不扩写业务之外的信息。
- 语气紧凑，适合 Gemini/Wan 图像模型。
- 对应中文为空时，英文也返回空字符串。
- 只返回 JSON，禁止 markdown、禁止额外解释。

返回格式：
{
  "gemini_prompt_en": "",
  "gemini_negative_prompt_en": "",
  "wan_prompt_en": "",
  "wan_negative_prompt_en": ""
}`),
		},
	}
}

// EnsureDefaultTemplates 确保默认模板已写入数据库。
func EnsureDefaultTemplates(repo *db.LLMPromptTemplateRepo) error {
	if repo == nil {
		return nil
	}
	for _, item := range DefaultTemplates() {
		if err := repo.UpsertByKey(item.Name, item.TemplateKey, item.Content, item.Description); err != nil {
			return err
		}
	}
	return nil
}
