# 302 GPT 直连接入设计

## 背景

当前分支已有 `302-gpt-image` provider 的初步接入：Go 后端会启动本地 `302Helpper` 二进制插件，再通过插件暴露的 `/v1/tasks` 与 `/v1/tasks/download` 网关接口完成 GPT Image 图片编辑。新的方向是不再使用本地 CLI/二进制网关，而是把 `302Helpper` 项目中调用 302 上游的 GPT Image 逻辑内置到本项目。

参考仓库：`git@github.com:tianfei212/302Helpper.git`。其中网关接口 `/v1/tasks` 是 `302Helpper` 自己的统一任务抽象；直连 302 GPT 上游时，应调用 302.ai 的 GPT Image edits 接口。

## 目标

新增独立函数包目录 `backend/internal/302Helpper/GPT`，把 302 GPT Image 支持集中放在这个包下。现有业务层继续使用 provider 名称 `302-gpt-image`，但底层不再启动或调用本地 `302Helpper` CLI，而是直接用环境变量 `PROVIDER_302_API_KEY` 调用 302.ai。

直连接口为：

```text
POST https://api.302.ai/v1/images/edits?response_format=url&async=true
GET  https://api.302.ai/async_result?task_id=<provider_task_id>
```

鉴权方式固定为：

```text
Authorization: Bearer ${PROVIDER_302_API_KEY}
```

## 非目标

本次不改前端交互，不重命名业务 provider，不改变 `wan`、`google/gemini`、`qwen` 的行为。不把 `302Helpper` 作为 Go module 依赖，也不保留本地 `302Helpper` 二进制的启动链路。已有数据库和异步改写任务表结构保持不变，除非实现时发现必须补充字段。

## 推荐方案

推荐采用“内置 GPT 直连客户端 + 复用现有业务异步任务”的方案。

后端新增 `backend/internal/302Helpper/GPT` 包，职责只包含 302 GPT Image 上游调用：

- 构造 `/v1/images/edits` multipart 请求。
- 从 `PROVIDER_302_API_KEY` 读取 302 上游密钥。
- 提交异步任务并返回 302 上游 `task_id`。
- 轮询 `/async_result`，解析成功、处理中、失败状态。
- 下载最终图片 URL 并返回图片 bytes。

`llm/router` 继续暴露 `StartCompose`、`WaitComposeResult` 和 `Compose` 能力。这样 `api/handler/image.go` 当前的异步改写任务、SSE 事件和入库逻辑可以保持稳定，改动集中在底层 provider client 和启动配置。

## 备选方案

方案一：继续调用远程 `/v1/tasks` 网关。

优点是与现有初步实现最接近；缺点是仍然依赖 `302Helpper` 的网关抽象，不符合“不再用现有 CLI 的方式”和“直连 302 的 GPT”的目标。

方案二：完整复制 `302Helpper` 的 provider、adapter、task service。

优点是以后可以扩展更多 302 模型；缺点是引入大量非必要代码，维护成本高，也会和本项目现有 LLM router、rewrite task 体系重复。

方案三：只内置 GPT Image edits 客户端。

优点是边界清晰、改动小、完全满足当前需求；缺点是后续接入 302 其他模型时需要再抽象。当前推荐此方案。

## 组件设计

### `backend/internal/302Helpper/GPT`

建议对外类型：

```go
type Client struct { ... }

type ComposeRequest struct {
    Prompt          string
    ForegroundImage []byte
    BackgroundImage []byte
}

type ComposeResult struct {
    Image []byte
    URL   string
}
```

建议对外方法：

```go
func NewClient(cfg Config) *Client
func (c *Client) Compose(ctx context.Context, req ComposeRequest) (*ComposeResult, error)
func (c *Client) CreateTask(ctx context.Context, req ComposeRequest) (string, error)
func (c *Client) WaitResult(ctx context.Context, taskID string) (*ComposeResult, error)
```

`CreateTask` 直接提交到 `/v1/images/edits?response_format=url&async=true`。multipart 字段包含：

- `image`: 前景人物图。当前 302 edits 接口只定义一个 `image` 文件字段；背景融合需求应通过 prompt 描述目标背景，并保留人物主体。
- `prompt`: 最终 GPT prompt。
- `model`: 默认 `gpt-image-2`。
- `n`: `1`。
- `quality`: `high`。
- `background`: `auto`。
- `output_format`: `png`。
- `moderation`: `auto`。
- `input_fidelity`: `high`。
- `size`: 默认沿用当前实现的 `1536x1024`，如果 302 上游不接受该尺寸，实现时按 302 文档或实测调整为合法尺寸。

`BackgroundImage` 不直接作为第二个 `image` 文件上传，除非实现验证 302 上游支持同名多文件。为了不丢失现有背景融合语义，业务层仍从背景模板取 GPT prompt，并把背景描述写入 prompt。

### `llm/router`

`302-gpt-image` provider 仍由 `normalizeProvider` 识别。替换导入路径和 client 类型，从现有 `internal/core/llm/helpper302` 改为新的 `internal/302Helpper/GPT`。

`StartCompose` 和 `Compose` 继续读取人物图、背景图与背景 prompt。调用直连客户端时：

- 人物图作为 `ForegroundImage`。
- 背景图保留在 request 结构中，便于未来扩展 mask 或多图支持。
- prompt 使用 `background_prompts.gpt_prompt` 或激活的 `302-gpt-image` skill。

### 启动入口

`backend/cmd/server/main.go` 移除：

- `internal/helpper302` manager import。
- `NewManager(&cfg.Helpper302)`。
- `Start()`。
- shutdown 时的 `helpper302Manager`。

`backend/internal/helpper302/manager.go` 和相关测试可以删除，或先保留但不再引用。推荐删除，避免误以为仍需本地 CLI。

### 配置

`Helpper302Config` 需要从“插件配置”调整为“302 GPT provider 配置”。

保留字段：

- `enabled`
- `base_url`
- `provider`
- `model_name`
- `mode`
- `poll_interval_seconds`
- `max_wait_seconds`

废弃或不再使用字段：

- `auto_start`
- `binary_dir`
- `config_path`
- `auth_token`
- `category_id`
- `function`
- `health_wait_seconds`

默认值建议：

```yaml
helpper302:
  enabled: true
  base_url: https://api.302.ai
  provider: 302-gpt-image
  model_name: gpt-image-2
  mode: async
  poll_interval_seconds: 2
  max_wait_seconds: 300
```

`PROVIDER_302_API_KEY` 只从环境变量或 `.env.local` 读取，不写入 YAML。

## 数据流

1. 前端继续请求 `/api/v1/images/rewrite`，传 `provider=302-gpt-image`。
2. handler 保存前景图、背景图，创建本项目自己的 rewrite task。
3. `llm.StartCompose` 读取人物图和背景 prompt。
4. `GPT.Client.CreateTask` 调用 302.ai `/v1/images/edits?response_format=url&async=true`。
5. handler 保存 302 上游 task id 到本项目 rewrite task。
6. 后台 goroutine 调用 `WaitComposeResult`。
7. `GPT.Client.WaitResult` 轮询 `/async_result`。
8. 成功后下载图片 bytes，交回现有保存和入库逻辑。

## 错误处理

- 未设置 `PROVIDER_302_API_KEY`：返回明确错误，不发请求。
- 创建任务 HTTP 非 2xx：记录上游状态码和有限长度 body。
- 未返回 `task_id`：返回协议错误。
- `/async_result` 返回 `err`：标记任务失败并透出上游错误。
- 轮询超过 `max_wait_seconds`：返回超时。
- 成功响应没有图片 URL：返回协议错误。
- 下载图片非 200 或 content-type 非图片：返回下载错误。

日志中不能打印 `PROVIDER_302_API_KEY`。

## 测试计划

新增或调整 Go 测试：

- `GPT.Client.CreateTask` 使用 `httptest` 验证：
  - URL 路径是 `/v1/images/edits`。
  - query 包含 `response_format=url`、`async=true`。
  - Authorization 使用 `PROVIDER_302_API_KEY`。
  - multipart 字段包含 `image`、`prompt`、`model`、`quality`、`output_format`、`input_fidelity`。
- `WaitResult` 覆盖：
  - 处理中状态继续轮询。
  - 成功返回 URL 并下载图片。
  - 上游 `err` 失败。
  - 超时。
- `main.go` 或相关测试确认不再启动 `302Helpper` manager。
- 现有 `llm/router` 302 provider 测试更新为直连客户端语义。

## 迁移影响

不再需要提交或部署：

- `bin/302helpper-*`
- `configs/302Helpper_config.yaml`
- `302helpper.log`

需要保留：

- `.env.local` 或运行环境里的 `PROVIDER_302_API_KEY`
- `configs/skills/302-gpt-image.md`
- `background_prompts` 中 GPT prompt 字段

实现完成后，应清理配置中的本地 CLI 字段，避免运维误配。

