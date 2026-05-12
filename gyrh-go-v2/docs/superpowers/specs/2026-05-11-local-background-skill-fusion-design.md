# 本地背景图默认 Skill 融合修复设计

## 背景

首页支持本地上传背景图，用户选择后进入拍摄页，拍摄前景人像并点击“使用”触发生成。当前本地背景图被当作临时 `blob:` 图像处理，但前端在生成请求里硬编码 `background_prompt_id = 1`。当前数据库没有 `id=1` 的背景模板，后端在保存背景图后尝试回写 `background_prompts` 失败，前端弹出“生成失败：背景图入库失败”。

## 目标

本地上传的背景图只参与本次生成，不保存到背景图库数据库表。前景拍照图和本地背景图以 base64 方式进入生成链路，由后端按当前模型 provider 读取激活的 Skill 内容作为默认融合提示词。生成结果仍保存到系统历史记录中，方便预览、下载和历史侧栏展示。

## 非目标

本次不改背景图库导入流程，不自动为本地临时背景创建 `background_prompts` 记录，不新增前端页面或重做 UI。已有背景图库卡片生成仍继续使用其绑定的 `background_prompt_id` 和背景模板提示词。

## 用户流程

1. 用户在首页点击或拖拽上传本地背景图。
2. 前端仅保存该图片的临时浏览器 URL 用于预览和进入拍摄页。
3. 用户拍照后点击“使用”。
4. 前端把抠像后的人物图和本地背景图转成 base64，调用 `/api/v1/images/rewrite`。
5. 请求包含 `foreground`、`background`、`provider`，不包含 `background_prompt_id`。
6. 后端保存输入图像为模型可用资产，读取当前 provider 激活的 Skill 内容作为 prompt。
7. 模型返回结果后，后端保存生成图和 `generated_images` 记录，`background_prompt_id` 保持 `0`。

## 架构与数据流

### 前端

`CaptureScreen` 根据 `selectedBg` 类型区分来源：

- 背景图库对象：继续读取 `selectedBg.image_url`，并传递 `selectedBg.id` 作为 `background_prompt_id`。
- 本地上传字符串：读取 `blob:` 图像并转 base64，只传 `background`，不传 `background_prompt_id`。

本地背景的错误处理保持在前端生成流程内：如果读取本地图片失败，应提示生成失败，不静默退化成无背景生成。

### 后端 Rewrite

`ImageHandler.Rewrite` 接受两种背景融合模式：

- 模板背景融合：`background` + `background_prompt_id > 0`，用于背景图库卡片。
- 临时背景融合：`background` + `background_prompt_id == 0`，用于本地上传背景图。

后端不再把“有背景图但没有背景模板 ID”视为非法请求。`persistBackgroundImage` 只在 `background_prompt_id > 0` 时更新背景图库记录；临时背景直接跳过入库。

### Prompt 解析

`llm.buildPrompt` 按优先级解析 prompt：

1. 如果提供了 `style_prompt` 或 `style_prompt_id`，用于风格转换场景。
2. 如果有背景图且 `background_prompt_id > 0`，读取对应背景模板的 provider 专属提示词。
3. 如果有背景图且 `background_prompt_id == 0`，读取当前 provider 激活的 Skill 内容。

Skill 查询使用 `SkillRepo.GetActive(provider)`。当没有激活 Skill 时，生成请求应返回明确错误，例如“未找到当前模型的激活 Skill”，而不是继续空 prompt 生成。

### 生成记录

无论背景来源是图库还是本地临时图，生成结果都继续写入 `generated_images`：

- `asset_id`、`path` 指向生成结果。
- `provider` 记录实际使用的模型。
- `status` 记录模型返回状态。
- `background_prompt_id` 对临时本地背景保持 `0`。
- 图片宽高继续从生成结果解析。

## 错误处理

本地背景图不会触发背景图库回写，因此不应再出现“背景图入库失败”。后端应保留以下可见错误：

- `foreground` 或 `background` 不是合法 base64：返回 400。
- 当前 provider 没有激活 Skill：返回 400 或 500，并说明缺少激活 Skill。
- 模型调用失败：返回“模型融合失败”。
- 生成结果保存或记录创建失败：返回对应保存/记录错误。

## 测试与验证

建议覆盖以下验证：

- 本地上传背景图后拍摄并生成，请求不包含 `background_prompt_id`，生成成功且生成图进入历史记录。
- 数据库 `background_prompts` 不新增、不更新本地背景图记录。
- 数据库 `generated_images` 新增生成记录，`background_prompt_id = 0`。
- 选择背景图库卡片后生成，仍使用该卡片的 `background_prompt_id` 和模板提示词。
- 当前 provider 缺少激活 Skill 时，返回明确错误。

## 实施范围

主要涉及：

- `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`
- `gyrh-go-v2/backend/internal/api/handler/image.go`
- `gyrh-go-v2/backend/internal/core/llm/router.go`

如需补充测试，可在 Go 后端增加 prompt 解析单元测试，并在前端补充请求 payload 逻辑测试。
