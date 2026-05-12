# 302 GPT Image 插件接入设计

## 背景

Go v2 当前图像生成链路已支持 `wan` 与 `google/gemini` provider。现在需要新增一个 GPT Image provider，用 `302Helpper` 提供的 302.ai 网关能力接入 GPT 图像接口。接入方式限定为只使用两个 Linux 二进制文件与配置文件，不下载、不拷贝、不依赖 `302Helpper` 源码。

## 目标

新增 provider `302-gpt-image`，与现有 `wan`、`google/gemini` 并列。当前 `/api/v1/images/rewrite` 生成链路在收到 `provider=302-gpt-image` 时，调用本地或外部运行的 `302Helpper` 插件网关，使用 GPT Image `image-edit` 能力完成前景人物与背景图融合，并把最终结果继续保存到现有生成历史记录。Go v2 前端模型切换也要支持该 provider，数据库初始化数据需要包含该 provider 的默认 Skill。

插件文件放在：

```text
gyrh-go-v2/backend/bin/302helpper/
├── server-linux-amd64
├── server-linux-arm64
└── configs/
    └── 302Helpper_config.yaml
```

## 非目标

本次不引入 `302Helpper` 源码，不把它作为 Go module 依赖，不把其内部 adapter、provider 或 task 代码复制进 Go v2。根目录旧版 `siteConfig.json` 不在本次范围内。现有 `wan` 与 `google/gemini` provider 行为保持不变。

## 配置设计

在 `gyrh-go-v2/configs/config.yaml` 新增独立配置段：

```yaml
helpper302:
  enabled: true
  auto_start: true
  binary_dir: ./backend/bin/302helpper
  config_path: ./backend/bin/302helpper/configs/302Helpper_config.yaml
  base_url: http://127.0.0.1:8080
  auth_token: ""
  provider: 302-gpt-image
  category_id: image
  model_name: gpt-image-2
  function: image-edit
  mode: async
  poll_interval_seconds: 2
  max_wait_seconds: 300
  health_wait_seconds: 15
```

密钥只放在 `gyrh-go-v2/.env.local` 或运行环境变量中，不写入 YAML，不提交到 Git。建议新增环境变量：

```env
PROVIDER_302_API_KEY=
GYRH_302_HELPER_AUTH_TOKEN=
GYRH_302_HELPER_GETWAY_KEY=
GYRH_302_HELPER_GETWAY_PRIVATE_KEY=
```

`PROVIDER_302_API_KEY` 供 `302Helpper` 访问 302.ai 上游。`GYRH_302_HELPER_AUTH_TOKEN` 用于 Go v2 调用插件网关的静态 Bearer。若插件配置启用动态 Bearer，则使用 `GYRH_302_HELPER_GETWAY_KEY` 与 `GYRH_302_HELPER_GETWAY_PRIVATE_KEY` 派生访问令牌。

## 进程管理

`auto_start=true` 时，Go v2 后端启动阶段按运行平台选择插件二进制：

- `linux/amd64` 使用 `server-linux-amd64`
- `linux/arm64` 使用 `server-linux-arm64`

插件进程工作目录设为 `gyrh-go-v2/backend/bin/302helpper/`，确保它能从相对路径读取 `configs/302Helpper_config.yaml`。启动后按 `health_wait_seconds` 等待服务可用。

`auto_start=false` 时，Go v2 不启动子进程，只使用 `base_url` 调用已由运维或容器编排启动的插件服务。

## Provider 调用流程

1. 前端或调用方继续请求 `/api/v1/images/rewrite`，并传入 `provider=302-gpt-image`。
2. `llm/router` 将 `302-gpt-image` 路由到新的 `helpper302` handler。
3. handler 读取现有存储里的前景人物 PNG、背景图以及最终 prompt。
4. handler 组装 `POST /v1/tasks` 请求：
   - Header：`Authorization: Bearer <token>`、`X-Category-Id: image`、`X-Model-Name: gpt-image-2`
   - Form：`category_id=image`、`model_name=gpt-image-2`、`function=image-edit`
   - Form：`image=@人物透明PNG`
   - Form：`request_json={...}`
5. `request_json` 默认包含：
   - `prompt`：当前背景模板 prompt 或激活 Skill prompt
   - `model`: `gpt-image-2`
   - `quality`: `high`
   - `size`: `1536x1024` 或配置值
   - `n`: `1`
   - `background`: `auto`
   - `output_format`: `png`
   - `input_fidelity`: `high`
6. 默认 `mode=async`：创建任务后轮询 `GET /v1/tasks/download?task_id=...`，直到返回可下载图片 URL、任务失败或超时。
7. 下载结果图片后，复用现有保存逻辑写入 storage 与 `generated_images` 历史记录。
8. `mode=sync` 作为兼容配置保留，但首选验证异步路径。

## 前端模型切换

Go v2 前端当前模型状态是 `W/G` 二态，需要扩展为三态：

- `W`：显示为 `W` 或 `Wan`，请求 provider 为 `wan`
- `G`：显示为 `G` 或 `Gemini`，请求 provider 为 `google`
- `GPT`：显示为 `GPT`，请求 provider 为 `302-gpt-image`

模型切换按钮继续放在现有 Header 位置，不新增页面。点击时按 `W -> G -> GPT -> W` 循环，保持 Apple 风格的简洁玻璃态按钮。所有会发起生成的页面都应通过统一映射函数取得 provider，避免在 `CaptureScreen`、`PreviewScreen` 等页面继续写死 `model === 'W' ? 'wan' : 'google'`。

背景卡片的提示词预览也要适配第三种模型。首版不强制给背景模板表增加 GPT 专属字段，因此 `GPT` 模型下的卡片提示词预览可以展示 `302-gpt-image` 的通用 Skill 名称或简短说明；实际生成时以后端 `302-gpt-image` 激活 Skill 为准。

Skill 管理页也要支持 `302-gpt-image`：

- 新建或编辑 Skill 时 provider 下拉增加 `302-gpt-image`
- provider 筛选下拉增加 `302-gpt-image`
- 列表展示保持原 provider 字符串，不做截断混淆

## 数据库与初始化数据

数据库 `skill_files.provider` 已是自由文本字段，不需要为 provider 增加枚举列。需要新增默认 Skill 文件与初始化逻辑：

- 新增 `gyrh-go-v2/configs/skills/302-gpt-image.md`
- `bootstrap.seedSkills()` 从 `[]string{"google", "wan"}` 扩展为包含 `302-gpt-image`
- `reset` 或重新导入 Skill 时也包含 `302-gpt-image`
- `SkillRepo.GetActive("302-gpt-image")` 应能取到默认激活 Skill

如果背景模板需要 GPT Image 专属 prompt，后续可以给 `background_prompts` 增加 `gpt_prompt`、`gpt_negative_prompt` 等字段。本次首版不改 `background_prompts` 表，`302-gpt-image` 即使收到 `background_prompt_id > 0`，也优先使用 `302-gpt-image` 的激活 Skill 完成融合，不复用 Gemini 或 Wan 的背景模板 prompt。

## Prompt 策略

`302-gpt-image` 使用独立 prompt 解析策略，避免误用 Gemini 或 Wan 的模型专属 prompt：

1. `wan` 与 `google/gemini` 继续沿用当前背景模板 prompt 与 Skill 解析规则。
2. `302-gpt-image` 首版始终读取 provider 为 `302-gpt-image` 的激活 Skill。
3. 若 `302-gpt-image` 没有激活 Skill，应返回明确错误，不静默退回其它 provider。

需要为 `302-gpt-image` 准备独立 Skill 内容，避免直接复用 Gemini 或 Wan 的模型专属措辞。

## 错误处理

- 插件二进制不存在、架构不支持、启动超时：返回 `302Helpper 插件不可用`，日志记录具体原因。
- provider 未启用但请求 `302-gpt-image`：返回明确配置错误。
- 插件 HTTP 返回 401 或 400：返回 `302Helpper 鉴权或请求参数错误`，日志记录状态码，不打印完整 token。
- 任务轮询超过 `max_wait_seconds`：返回 `302-gpt-image 任务超时`，不写入生成历史。
- 任务返回失败或 404：返回 `302-gpt-image 任务失败或不存在`。
- 结果 URL 下载失败或内容不是图片：返回 `302-gpt-image 结果下载失败`，不写入坏记录。

## 测试与验证

建议覆盖：

- `normalizeProvider("302-gpt-image")` 能路由到新 handler。
- `wan` 与 `google/gemini` provider 仍保持原行为。
- 二进制选择逻辑能正确处理 `linux/amd64` 与 `linux/arm64`。
- `auto_start=false` 时不会启动子进程，只调用 `base_url`。
- multipart 请求字段包含 `category_id`、`model_name`、`function`、`request_json`、`image`。
- 异步轮询能处理 processing、成功 URL、404、失败与超时。
- 结果图片下载后能进入现有 storage 与历史记录。
- `.env.local` 中的 302 API key 与插件访问 token 不会进入日志和 Git。
- 前端模型切换能按 `W -> G -> GPT -> W` 循环，并在生成请求中发送 `302-gpt-image`。
- Skill 管理页能创建、筛选、编辑 `302-gpt-image` provider 的 Skill。
- 初始化数据库后，`skill_files` 中存在激活的 `302-gpt-image` 默认 Skill。

## 实施范围

主要涉及：

- `gyrh-go-v2/configs/config.yaml`
- `gyrh-go-v2/configs/skills/302-gpt-image.md`
- `gyrh-go-v2/.env.local`（本地新增密钥项，不提交）
- `gyrh-go-v2/backend/internal/config/config.go`
- `gyrh-go-v2/backend/internal/bootstrap/bootstrap.go`
- `gyrh-go-v2/backend/internal/helpper302/`
- `gyrh-go-v2/backend/internal/core/llm/helpper302/`
- `gyrh-go-v2/backend/internal/core/llm/router.go`
- `gyrh-go-v2/backend/cmd/server/main.go`
- `gyrh-go-v2/frontend/src/App.jsx`
- `gyrh-go-v2/frontend/src/components/Layout.jsx`
- `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`
- `gyrh-go-v2/frontend/src/screens/PreviewScreen.jsx`
- `gyrh-go-v2/frontend/src/screens/DashboardScreen.jsx`
- `gyrh-go-v2/frontend/src/screens/SkillManagerScreen.jsx`
- 前端模型/provider 映射工具与测试
- 后端相关单元测试

## 待用户准备

用户需要提供或放置以下文件：

- `gyrh-go-v2/backend/bin/302helpper/server-linux-amd64`
- `gyrh-go-v2/backend/bin/302helpper/server-linux-arm64`
- `gyrh-go-v2/backend/bin/302helpper/configs/302Helpper_config.yaml`

`302Helpper_config.yaml` 中应注册 `image/gpt-image-2/image-edit`，并与 Go v2 的 `category_id`、`model_name`、`function` 配置一致。
