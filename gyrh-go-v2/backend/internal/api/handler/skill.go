package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"gyrh-go-v2/backend/internal/db"
	"gyrh-go-v2/backend/internal/logger"
	"gyrh-go-v2/backend/pkg/httpx"
)

// SkillHandler 提供 Skill 文件 CRUD 能力。
type SkillHandler struct {
	repo          *db.SkillRepo
	llmPromptRepo *db.LLMPromptTemplateRepo
}

// NewSkillHandler 创建 Skill 处理器。
func NewSkillHandler(repo *db.SkillRepo, llmPromptRepo *db.LLMPromptTemplateRepo) *SkillHandler {
	return &SkillHandler{
		repo:          repo,
		llmPromptRepo: llmPromptRepo,
	}
}

// List 返回 Skill 列表。
func (h *SkillHandler) List(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	limit, offset := parsePage(r)

	var (
		items []*db.SkillFile
		err   error
		total int64
	)
	if provider != "" {
		items, err = h.repo.ListByProvider(provider, false, limit)
		total, _ = h.repo.Count(provider, false)
	} else {
		items, err = h.repo.List(limit, offset)
		total, _ = h.repo.Count("", false)
	}
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "查询 Skill 列表失败"))
		return
	}

	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, h.toSkillItem(item, false))
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{
		"items":  result,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}))
}

// Get 返回 Skill 详情。
func (h *SkillHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的 Skill ID"))
		return
	}

	skill, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "Skill 文件不存在"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toSkillItem(skill, true)))
}

// GetActive 返回当前激活的 Skill。
func (h *SkillHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "provider 不能为空"))
		return
	}

	skill, err := h.repo.GetActive(provider)
	if err != nil {
		httpx.WriteJSON(w, http.StatusNotFound, httpx.Error(1, "未找到激活的 Skill"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toSkillItem(skill, true)))
}

// Create 创建 Skill。
func (h *SkillHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Content  string `json:"content"`
		Provider string `json:"provider"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}
	if req.Name == "" || req.Content == "" || req.Provider == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "name、content、provider 不能为空"))
		return
	}
	if h.isQwenSkill(req.Provider) {
		if err := h.upsertQwenPromptTemplate(req.Content); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "保存 Qwen Prompt 模板失败"))
			return
		}
	}

	id, err := h.repo.Create(req.Name, req.Content, req.Provider)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "创建 Skill 失败"))
		return
	}
	if req.IsActive {
		_ = h.repo.UpdateActive(id, true)
	}

	skill, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取新建 Skill 失败"))
		return
	}

	logger.Info("Skill 文件创建成功: %d", id)
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toSkillItem(skill, true)))
}

// Update 更新 Skill。
func (h *SkillHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的 Skill ID"))
		return
	}

	var req struct {
		Name     string `json:"name"`
		Content  string `json:"content"`
		Provider string `json:"provider"`
		IsActive *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "请求体格式错误"))
		return
	}

	if err := h.repo.Update(id, req.Name, req.Content, req.Provider); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "更新 Skill 失败"))
		return
	}
	if h.isQwenSkill(req.Provider) {
		if err := h.upsertQwenPromptTemplate(req.Content); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "保存 Qwen Prompt 模板失败"))
			return
		}
	}
	if req.IsActive != nil {
		if err := h.repo.UpdateActive(id, *req.IsActive); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "更新激活状态失败"))
			return
		}
	}

	skill, err := h.repo.GetByID(id)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "获取更新后的 Skill 失败"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.toSkillItem(skill, true)))
}

// Delete 删除 Skill。
func (h *SkillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromVars(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.Error(1, "无效的 Skill ID"))
		return
	}
	if err := h.repo.Delete(id); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, httpx.Error(1, "删除 Skill 失败"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]interface{}{"deleted": id}))
}

func (h *SkillHandler) toSkillItem(skill *db.SkillFile, withContent bool) map[string]interface{} {
	item := map[string]interface{}{
		"id":         skill.ID,
		"name":       skill.Name,
		"provider":   skill.Provider,
		"is_active":  skill.IsActive,
		"created_at": skill.CreatedAt.Format(time.RFC3339),
		"updated_at": skill.UpdatedAt.Format(time.RFC3339),
	}
	if withContent {
		item["content"] = h.skillContent(skill)
	}
	return item
}

func (h *SkillHandler) skillContent(skill *db.SkillFile) string {
	if !h.isQwenSkill(skill.Provider) || h.llmPromptRepo == nil {
		return skill.Content
	}
	template, err := h.llmPromptRepo.GetByKey(db.TemplateKeyQwenBackgroundSuggest)
	if err != nil {
		return skill.Content
	}
	return template.Content
}

func (h *SkillHandler) isQwenSkill(provider string) bool {
	return provider == "qwen" || provider == "Qwen"
}

func (h *SkillHandler) upsertQwenPromptTemplate(content string) error {
	if h.llmPromptRepo == nil {
		return nil
	}
	return h.llmPromptRepo.UpsertByKey(
		"Qwen 背景图默认建议",
		db.TemplateKeyQwenBackgroundSuggest,
		content,
		"根据背景图生成 Gemini/Wan/GPT Image 的中英双语正向与反向提示词建议。",
	)
}
