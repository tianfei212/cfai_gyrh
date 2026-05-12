# 修改记录（CHANGELOG）

本文档记录每次会话/迭代对仓库做的实质性改动，按倒序时间排列（最新在最前）。
每条记录至少包含：日期、更新时间、分支、目的、执行者、文件清单与 commit hash。
执行者格式：人工 或 Claude <模型名>（<model-string>，Anthropic）。
仅记录 代码 / 配置 / 文档 层面的变更；个人调试痕迹（缓存、PID、临时日志）不在此记录。

## 记录要求

- 每次提交代码前，必须先更新本文档。
- 新记录必须放在最上方，保持倒序时间排列。
- 文件清单仅列出本次会话/迭代中实质性修改的代码、配置或文档文件。
- commit hash 在提交完成后填写；提交前可临时标记为“未提交，提交后补充”。
- 个人调试痕迹、缓存、PID、临时日志、数据库运行时附属文件不应写入记录。

## 记录格式

```markdown
## YYYY-MM-DD HH:mm UTC+8

- 日期：YYYY-MM-DD
- 更新时间：HH:mm UTC+8
- 分支：branch-name
- 目的：本次会话/迭代的目的
- 执行者：人工 / Claude <模型名>（<model-string>，Anthropic）
- 文件清单：
  - `path/to/file`
- commit hash：未提交，提交后补充
```

## 2026-05-12 15:44 UTC+8

- 日期：2026-05-12
- 更新时间：15:44 UTC+8
- 分支：feature/302-gpt-direct-integration
- 目的：完成 302 GPT Image 直连集成与背景图库远端同步链路，并修复背景图管理页“查看原图”下载问题，改为当前页面内玻璃态原图预览。
- 执行者：GPT-5.5（GPT-5.5，OpenAI）
- 推送版本：feature/302-gpt-direct-integration@2026-05-12-1544（提交并推送后以远端分支 HEAD 为准）
- 详细修改内容：
  - 后端新增 302 GPT Image 客户端，支持创建异步图片编辑任务、轮询 async_result 并下载结果图。
  - LLM 路由新增 `302-gpt-image` provider，支持同步 Compose、外部异步任务启动与结果等待，并在背景提示词为空时回退到对应 provider 的激活 Skill。
  - 图像改写接口支持异步任务持久化、任务状态查询、SSE 事件订阅和服务重启后的运行中任务恢复。
  - 背景提示词数据结构扩展 GPT 正/反向提示词及中英文版本，导入、编辑、同步英文和列表返回均带上 GPT 字段。
  - 背景图库新增远端同步接口，支持从配置的远端图库 API 拉取图片、去重、下载入库并记录同步失败明细。
  - 前端模型切换从 Wan/Gemini 扩展为 Wan/Gemini/302 GPT Image，并统一通过 provider 工具函数映射请求参数和显示标签。
  - 拍摄页与预览页接入异步改写任务解析，302 GPT Image 可以先返回任务再等待最终图片。
  - 背景图管理页新增分页、远端同步、GPT 提示词编辑列，并将“查看原图”改为当前页面内的玻璃态弹层预览，避免远端图片响应头触发浏览器下载。
  - 技能管理页新增 302 GPT Image provider 选项，新增默认 302 GPT Image 技能配置。
  - 新增配置、数据库迁移、工具函数与单元测试，覆盖 provider 轮转、背景分页、302 配置、异步任务和远端同步等关键路径。
- 文件清单：
  - `CHANGELOG.md`
  - `gyrh-go-v2/backend/cmd/server/main.go`
  - `gyrh-go-v2/backend/internal/302Helpper/GPT/client.go`
  - `gyrh-go-v2/backend/internal/302Helpper/GPT/client_test.go`
  - `gyrh-go-v2/backend/internal/api/handler/background_prompt.go`
  - `gyrh-go-v2/backend/internal/api/handler/background_prompt_remote_test.go`
  - `gyrh-go-v2/backend/internal/api/handler/image.go`
  - `gyrh-go-v2/backend/internal/api/handler/image_async_test.go`
  - `gyrh-go-v2/backend/internal/api/handler/rewrite_task.go`
  - `gyrh-go-v2/backend/internal/api/middleware/logger.go`
  - `gyrh-go-v2/backend/internal/api/middleware/logger_test.go`
  - `gyrh-go-v2/backend/internal/api/router.go`
  - `gyrh-go-v2/backend/internal/bootstrap/bootstrap.go`
  - `gyrh-go-v2/backend/internal/bootstrap/bootstrap_test.go`
  - `gyrh-go-v2/backend/internal/config/config.go`
  - `gyrh-go-v2/backend/internal/config/config_test.go`
  - `gyrh-go-v2/backend/internal/core/llm/qwen/advisor.go`
  - `gyrh-go-v2/backend/internal/core/llm/router.go`
  - `gyrh-go-v2/backend/internal/core/llm/router_test.go`
  - `gyrh-go-v2/backend/internal/db/background_prompt.go`
  - `gyrh-go-v2/backend/internal/db/db.go`
  - `gyrh-go-v2/backend/internal/db/rewrite_task.go`
  - `gyrh-go-v2/backend/internal/db/rewrite_task_test.go`
  - `gyrh-go-v2/configs/config.yaml`
  - `gyrh-go-v2/configs/skills/302-gpt-image.md`
  - `gyrh-go-v2/configs/skills/qwen_background_prompt_suggestion.md`
  - `gyrh-go-v2/configs/skills/qwen_sync_english.md`
  - `gyrh-go-v2/docs/superpowers/plans/2026-05-11-local-background-skill-fusion.md`
  - `gyrh-go-v2/docs/superpowers/plans/2026-05-12-302-gpt-direct-integration.md`
  - `gyrh-go-v2/docs/superpowers/plans/2026-05-12-302-gpt-image-plugin.md`
  - `gyrh-go-v2/docs/superpowers/plans/2026-05-12-remote-background-sync.md`
  - `gyrh-go-v2/docs/superpowers/specs/2026-05-11-local-background-skill-fusion-design.md`
  - `gyrh-go-v2/docs/superpowers/specs/2026-05-12-302-gpt-direct-integration-design.md`
  - `gyrh-go-v2/docs/superpowers/specs/2026-05-12-302-gpt-image-plugin-design.md`
  - `gyrh-go-v2/docs/superpowers/specs/2026-05-12-remote-background-sync-design.md`
  - `gyrh-go-v2/frontend/src/App.jsx`
  - `gyrh-go-v2/frontend/src/components/Layout.jsx`
  - `gyrh-go-v2/frontend/src/screens/BackgroundManagerScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/DashboardScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/PreviewScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/SkillManagerScreen.jsx`
  - `gyrh-go-v2/frontend/src/services/api.js`
  - `gyrh-go-v2/frontend/src/services/rewriteTask.js`
  - `gyrh-go-v2/frontend/src/styles.css`
  - `gyrh-go-v2/frontend/src/utils/backgroundPagination.js`
  - `gyrh-go-v2/frontend/src/utils/backgroundPagination.test.js`
  - `gyrh-go-v2/frontend/src/utils/modelProvider.js`
  - `gyrh-go-v2/frontend/src/utils/modelProvider.test.js`
  - `gyrh-go-v2/manage.sh`
- 未纳入提交：
  - `gyrh-go-v2/backend/data/gyrh.db*`：本地 SQLite 运行时数据文件。
  - `gyrh-go-v2/bin/302helpper-*`：本地构建出的平台二进制产物。
  - `gyrh-go-v2/configs/302Helpper_config.yaml`：包含明文内部 bearer token，避免泄露敏感配置。
- commit hash：未提交，提交后补充

## 2026-05-11 23:36 UTC+8

- 日期：2026-05-11
- 更新时间：23:36 UTC+8
- 分支：fix/local-background-skill-fusion
- 目的：为独立历史记录页新增图片多选与批量删除能力，在每张历史图片上提供玻璃态选择框，并在刷新按钮旁新增带二次确认的删除按钮，删除后自动刷新并处理页码回退。
- 执行者：GPT-5.5（GPT-5.5，OpenAI）
- 文件清单：
  - `CHANGELOG.md`
  - `gyrh-go-v2/frontend/src/screens/HistoryScreen.jsx`
  - `gyrh-go-v2/frontend/src/styles.css`
  - `gyrh-go-v2/frontend/src/utils/historyRecords.js`
  - `gyrh-go-v2/frontend/src/utils/historyRecords.test.js`
- commit hash：`00fc93b`

## 2026-05-11 22:58 UTC+8

- 日期：2026-05-11
- 更新时间：22:58 UTC+8
- 分支：feature/frontend-addition
- 目的：实现工作台右侧历史记录侧栏接入真实数据库总数与倒序缩略图，支持点击历史缩略图进入单图全屏效果预览，并复用独立历史页的数据映射与单图预览模式。
- 执行者：GPT-5.5（GPT-5.5，OpenAI）
- 文件清单：
  - `CHANGELOG.md`
  - `gyrh-go-v2/frontend/src/App.jsx`
  - `gyrh-go-v2/frontend/src/components/Layout.jsx`
  - `gyrh-go-v2/frontend/src/screens/DashboardScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/HistoryScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/PreviewScreen.jsx`
  - `gyrh-go-v2/frontend/src/styles.css`
  - `gyrh-go-v2/frontend/src/utils/historyRecords.js`
  - `gyrh-go-v2/frontend/src/utils/historyRecords.test.js`
  - `gyrh-go-v2/frontend/src/utils/previewSelection.js`
  - `gyrh-go-v2/frontend/src/utils/previewSelection.test.js`
- commit hash：`eea7e73`

## 2026-05-11 22:12 UTC+8

- 日期：2026-05-11
- 更新时间：22:12 UTC+8
- 分支：feature/frontend-addition
- 目的：为摄像头拍摄页新增左侧人物大小数字变焦控制，支持从 -1.0x 缩小到 +1.5x 放大的偏移式调节，并完成摄像头左右镜像，确保实时预览、拍摄原图与 MediaPipe 抠像输入使用一致的中心缩放和镜像效果。
- 执行者：GPT-5.5（GPT-5.5，OpenAI）
- 文件清单：
  - `CHANGELOG.md`
  - `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`
  - `gyrh-go-v2/frontend/src/styles.css`
  - `gyrh-go-v2/frontend/src/utils/cameraZoom.js`
  - `gyrh-go-v2/frontend/src/utils/cameraZoom.test.js`
- commit hash：`60af51d`

## 2026-05-11 21:56 UTC+8

- 日期：2026-05-11
- 更新时间：21:56 UTC+8
- 分支：feature/frontend-addition
- 目的：修复 Gemini 输出规格与前端展示问题，约束 Gemini 生成 16:9 2K 横版图，统一历史/主页面缩略图为 16:9，并将效果图点击放大改为铺满屏幕的全屏预览。
- 执行者：GPT-5.5（GPT-5.5，OpenAI）
- 文件清单：
  - `CHANGELOG.md`
  - `gyrh-go-v2/backend/internal/core/llm/router.go`
  - `gyrh-go-v2/backend/internal/core/llm/router_test.go`
  - `gyrh-go-v2/frontend/src/screens/DashboardScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/HistoryScreen.jsx`
  - `gyrh-go-v2/frontend/src/screens/PreviewScreen.jsx`
  - `gyrh-go-v2/frontend/src/utils/imageThumbs.js`
  - `gyrh-go-v2/frontend/src/utils/imageThumbs.test.js`
- commit hash：`3462ad2`

## 2026-05-11 21:30 UTC+8

- 日期：2026-05-11
- 更新时间：21:30 UTC+8
- 分支：feature/frontend-addition
- 目的：建立项目修改记录规范，在根目录创建 `CHANGELOG.md`，约定后续每次提交代码前必须更新本文档。
- 执行者：GPT-5.5（GPT-5.5，OpenAI）
- 文件清单：
  - `CHANGELOG.md`
- commit hash：未提交，提交后补充；当前基线 `aac7402`
