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
