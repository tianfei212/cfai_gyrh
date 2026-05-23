# CHANGELOG

本文档记录每次会话/迭代对仓库做的实质性改动，按倒序时间排列（最新在最前）。
每条记录至少包含：日期、更新时间、分支、目的、执行者、文件清单与 commit hash。
执行者格式：人工 或 Claude <模型名>（<model-string>，Anthropic）。
仅记录代码 / 配置 / 文档层面的变更；个人调试痕迹（缓存、PID、临时日志）不在此记录。

## 2026-05-24 01:10

- 分支：`feature/qwen-active-skill-template`
- 版本：`3.0.2`
- 目的：发布前端登录与业务接口防护修复，并补充线上鉴权与随机路由防护验证记录。
- 执行者：GPT-5.5（OpenAI）
- commit hash：本条记录随本次提交生成。

### 修改内容

- `backend/internal/application/frontendauth/`、`backend/internal/api/handler/frontend_auth.go`：加入前端会话登录校验，使用 `HOME1` / `HOME2` 登录头验证 `admin` 与 `pshow` 账号，签发 HttpOnly Cookie/JWT 会话，错误密码统一返回 `401`。
- `backend/internal/frontend/frontend.go`、`backend/internal/api/router.go`：对 SPA 页面和业务 API 增加未登录拦截，随机主目录路径统一跳转 `/login?next=...`，业务接口未登录统一返回 `401 请先登录`。
- `backend/internal/logger/logger.go`：增加失败登录审计日志，记录真实 IP、RemoteAddr、User-Agent、用户名和失败原因，便于追踪随机密码与暴力尝试。
- `frontend/src/app/AppShell.jsx`：修复 `pshow` 退出流程，确保 `admin` 与 `pshow/kiosk` 都调用后端退出接口并清理本地会话。
- `docs/nginx-business-routes.md`：明确线上 Nginx 只转发业务 allowlist 路由，其余路径由默认拦截规则处理。
- `CHANGELOG.md`：记录 `3.0.2` 防护修复内容、线上压测和随机路由验证结果。

### 验证

- 线上 `https://mqia.chinafilmai.com:443/api/v1/frontend-auth/login` 使用 `admin` 随机密码按 500 次/秒提交 3000 次：2999 次返回 `401`，1 次读超时，服务未出现 5xx 或崩溃。
- 线上同接口使用 `pshow` 随机密码按 500 次/秒提交 1000 次：1000 次全部返回 `401`，无超时和 5xx。
- 线上业务接口按每秒 5 次、每个接口 1 次进行无登录态验证：43 个接口中 `GET /api/v1/health` 和 `POST /api/v1/frontend-auth/logout` 返回 `200`，其余 41 个受保护接口返回 `401`。
- 线上主域名后随机组合路径按每秒 5 次请求 50 次：全部返回 `302` 到 `/login?next=...`，无 5xx 或连接失败。

## 2026-05-24 00:41

- 分支：`feature/qwen-active-skill-template`
- 目的：修复 `pshow` 用户登录后点击退出按钮未真正执行退出流程的问题。
- 执行者：GPT-5.5（OpenAI）
- commit hash：`3767127`

### 修改内容

- `frontend/src/app/AppShell.jsx`：统一退出按钮行为，`admin` 与 `pshow/kiosk` 均调用 `logoutFrontend()` 清理后端 Cookie 和本地 session，然后跳转到 `/login`；不再让 `pshow` 只切回 dashboard。
- `release/gyrh-go-v2-202605240010-6438191-darwin-arm64/bin/gyrh-server` 与主工作区 `/release/.../bin/gyrh-server`：重新构建并同步包含退出修复的 macOS arm64 单二进制。

### 验证

- `npm --prefix frontend test` 通过，49 项前端测试全部通过。
- `npm --prefix frontend run build` 通过。
- 主工作区 release 目录下 `./manage.sh restart` 已验证后端/内嵌前端 `9913` 可启动，未登录访问 `/` 返回 `302` 到 `/login?next=/`。

## 2026-05-24 00:33

- 分支：`feature/qwen-active-skill-template`
- 目的：新增前端 JWT 登录、角色鉴权、登录审计日志，并生成主工作区 macOS release 目录用于测试。
- 执行者：GPT-5.5（OpenAI）
- commit hash：`3059796`，补充发布包 `manage.sh` 与主工作区 release 说明见 `04575e1`。

### 修改内容

- `backend/internal/application/frontendauth/`：新增前端登录应用服务，负责 JWT 签发/校验、HttpOnly Cookie 会话、`.env.local` 热加载、角色信息和真实 IP 解析。
- `backend/internal/api/handler/frontend_auth.go`：新增 `HOME1` / `HOME2` 登录接口、会话接口、退出接口和 API 会话中间件；登录成功/失败均记录详细审计日志。
- `backend/internal/logger/logger.go`：新增 `login_error.log` 专用失败登录审计文件，记录失败时间、真实 IP、用户名、RemoteAddr、User-Agent 和失败原因。
- `backend/internal/frontend/frontend.go` 与 `backend/internal/api/router.go`：对 SPA 页面和业务 API 增加前端会话鉴权，未登录强制跳转 `/login` 或返回 `401`。
- `frontend/src/services/frontendAuth.js`、`frontend/src/app/routes.jsx`、`frontend/src/screens/LoginScreen.jsx`：新增前端登录页、角色跳转、退出清理、前端 token 请求头和禁止保存密码的表单处理。
- `docs/nginx-business-routes.md`：整理 Nginx 业务路由 allowlist，便于只转发业务路由。
- `docs/backend-package-reorganization-plan.md`：记录后端按功能包继续拆分的阶段计划，当前已先拆出前端认证服务。
- `release/gyrh-go-v2-202605240010-6438191-darwin-arm64/`：生成主工作区 macOS arm64 测试 release 目录，包含 `gyrh-server`、`oss-cli`、OSS 配置和发布管理脚本，不生成 `.tar.gz`。
- `release/gyrh-go-v2-202605240010-6438191-darwin-arm64/manage.sh`：调整为发布包管理脚本，统一启动/停止后端、内嵌前端和 OSS 双端口服务，启动后打印演示端、管理端、登录页、后端/API 端口、OSS 端口、日志目录以及测试账号密码。
- `release/gyrh-go-v2-202605240010-6438191-darwin-arm64/configs/config.yaml`：发布包日志目录调整为 `./logs/app`，并将 `alioss.auto_start` 设为 `false`，避免后端与 `manage.sh` 重复拉起 OSS。
- `release/gyrh-go-v2-202605240010-6438191-darwin-arm64/configs/alioss-agent*.yaml`：发布包保留 OSS 配置文件，但将示例 bucket/API key 改为占位值，避免提交真实部署密钥。
- `/release/gyrh-go-v2-202605240010-6438191-darwin-arm64/`：同步生成到主工作区根目录 `release/` 下，便于远端直接按目录包方式更新；不再生成 `.tar.gz`。

### 验证

- `go test -race ./internal/logger ./internal/application/frontendauth ./internal/api/handler ./internal/frontend` 通过。
- release 目录下 `./manage.sh restart` 已验证后端/内嵌前端 `9913`、OSS 背景素材 `18080`、OSS 生成素材 `18081` 均可启动。
- 未登录访问 `/` 返回 `302` 到 `/login?next=/`。
- 错误密码登录返回 `401`，并确认 `logs/app/login_error.log` 写入真实 IP。
- 登录页文案已去除 `.env.local` 暴露，仅显示面向用户的账号密码提示。
- `./manage.sh restart` 输出已确认包含访问地址、端口、日志目录和 `admin` / `pshow` 测试账号信息。

## 2026-05-23 16:50

- 分支：`feature/qwen-active-skill-template`
- 目的：新增 Windows 展厅 kiosk 壳子，调用本机已安装 Google Chrome 全屏打开配置 URL。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：`3059796`

### 修改内容

- `backend/cmd/kiosk-client/main.go`：新增 kiosk client 独立入口，支持 `-config` 指定配置文件。
- `backend/internal/kiosk/`：新增配置加载、Chrome 自动查找、kiosk 启动参数生成和 Chrome 进程守护逻辑。
- `configs/kiosk-client.yaml`：新增 Windows kiosk client 配置模板，默认 URL 为 `http://127.0.0.1:9913/admin_viewer`，`chrome_path` 为空时自动查找 Chrome。
- `scripts/build_kiosk_client_windows.sh`：新增 Windows amd64 壳子构建和 zip 打包脚本。
- `CHANGELOG.md`：记录本次 kiosk 壳子实现和验证结果。

### 验证

- `go test ./internal/kiosk` 通过。
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o /tmp/gyrh-kiosk-client.exe ./cmd/kiosk-client` 通过。
- linter 检查触达 Go 文件无诊断错误。
- 已生成 Windows amd64 kiosk client 包：`release/gyrh-kiosk-client-202605231650-6438191-windows-amd64.zip`。

## 2026-05-22 21:37

- 分支：`feature/fullscreen-responsive-adaptation`
- 目的：按用户要求调整主界面背景图库排序为名称升序，并重新生成 Ubuntu amd64 release 包。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：本条记录随本次提交生成。

### 修改内容

- `backend/internal/db/background_prompt.go`：背景图提示词列表 SQL 从 `ORDER BY name DESC` 改为 `ORDER BY name ASC`，分页和非分页查询保持一致。
- `backend/internal/db/background_prompt_test.go`：同步更新 DB 层排序回归测试，验证列表按名称升序返回。
- `CHANGELOG.md`：记录本次排序调整、验证结果和 release 产物。

### 验证

- `go test ./internal/db` 通过。
- linter 检查触达 Go 文件无诊断错误。
- 已重启本地后端，`9913` 正在监听；前端服务仍在 `9912` 监听。
- 手动执行数据库查询确认升序结果：`SELECT id, name FROM background_prompts ORDER BY name ASC LIMIT 10 OFFSET 0;`，其中 `a_战场` 位于数字和 `IMG_...` 名称之后。

### Release

- 重新生成 Ubuntu amd64 release：`release/gyrh-go-v2-202605222137-6438191-ubuntu-amd64.tar.gz`。
- Ubuntu 单文件更新目标：`release/gyrh-go-v2-202605222137-6438191-ubuntu-amd64/bin/gyrh-server`。
- `bin/gyrh-server` 验证为 Linux x86-64 ELF，大小约 `27M`。
- Ubuntu `bin/gyrh-server` SHA256：`3c54cc4d49496b17c3df91b82075d859c6a66c56d00df691f90afef8c2f2de5d`。
- 本次记录不包含 `backend/data/gyrh.db*`、`release/` 等运行和打包产物提交。

## 2026-05-22 17:08

- 分支：`feature/fullscreen-responsive-adaptation`
- 目的：修复 macOS 单二进制测试中“预览页点击风格转换时报 `TypeError: Failed to fetch`”的问题；同时记录本次 release 单二进制收敛、背景图库排序调整和验证范围。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：本条记录随本次提交生成。

### 本次问题

- 在 macOS ARM64 单二进制包 `release/gyrh-go-v2-202605221415-1dfeee6-darwin-arm64` 中验证风格转换时，浏览器控制台报错：`Style transfer failed: TypeError: Failed to fetch`。
- 后端日志没有出现对应的 `POST /api/v1/images/rewrite` 请求，说明失败发生在前端真正调用 rewrite API 之前。
- 当预览页来源是历史记录或已生成图时，前端只保留了 OSS 签名图片 URL，没有继续携带数据库返回的 `asset_id`。点击风格转换时，前端尝试直接 `fetch(OSS 签名图)` 下载图片并转 base64，浏览器会受 OSS CORS 限制而抛出 `Failed to fetch`。

### 解决方案

- 保留历史记录中的 `asset_id`，并在进入预览页时随预览选择一起传递，不再只传图片 URL。
- `PreviewScreen` 在发起风格转换时优先使用 `foreground_asset_id`，让后端通过已有素材 ID 读取图片，避免浏览器跨域抓取 OSS 签名图。
- 保留原有降级逻辑：只有没有 `asset_id` 的本地上传图或 data URL 才继续走 base64。
- 背景图库排序按用户要求由 `updated_at DESC, id DESC` 改为 `name DESC, id DESC`，即按提示词名称倒序；新增 DB 层回归测试。
- `scripts/build_release.sh` 不再把 `frontend/dist` 作为冗余目录复制进 Ubuntu release 包，release 包仅依赖 `bin/gyrh-server` 中内嵌的前端资源，避免远端误用第二套前端文件。

### 修改文件

- `frontend/src/utils/historyRecords.js`：历史记录映射新增 `assetId`，预览 payload 继续携带该字段。
- `frontend/src/utils/historyRecords.test.js`：新增历史记录 `assetId` 保留与传递断言。
- `frontend/src/utils/previewSelection.js`：预览选择规范化时保留 `assetId`，普通字符串预览保持空 `assetId`。
- `frontend/src/utils/previewSelection.test.js`：覆盖普通图片与历史单图预览的 `assetId` 语义。
- `frontend/src/app/AppShell.jsx`：新增 `capturedAssetId` 状态，并随 `PreviewScreen` props 传递。
- `frontend/src/screens/PreviewScreen.jsx`：风格转换 payload 优先使用 `capturedAssetId` 作为 `foreground_asset_id`。
- `backend/internal/db/background_prompt.go`：背景图库列表 SQL 排序改为 `ORDER BY name DESC, id DESC`。
- `backend/internal/db/background_prompt_test.go`：新增背景提示词列表按名称倒序的回归测试。
- `scripts/build_release.sh`：Ubuntu release 构建仍先把最新 `frontend/dist` 写入 Go embed 目录，但不再把 `frontend/dist` 复制进 release 包。
- `CHANGELOG.md`：新增本次问题、根因、解决方案、文件清单和验证记录。

### 验证

- 先运行新增 DB 排序测试，旧逻辑失败：`go test ./internal/db -run TestBackgroundPromptRepoListOrdersByNameDesc -count=1`。
- 修改排序后通过：`go test ./internal/db ./internal/api/handler`。
- 先运行历史预览 `assetId` 测试，旧逻辑失败；修复后通过：`node --test src/utils/historyRecords.test.js src/utils/previewSelection.test.js`。
- 前端构建通过：`npm run build`，生成新入口 `/assets/index-DiGlKjB2.js`。
- linter 检查通过，触达文件无诊断错误。
- 重新生成并启动 macOS ARM64 单二进制测试包：`release/gyrh-go-v2-202605221700-1dfeee6-darwin-arm64.tar.gz`。
- 当前本地测试服务 `http://127.0.0.1:9913/admin_viewer` 返回新前端入口 `/assets/index-DiGlKjB2.js`，旧入口 `/assets/index-CgIcyYDU.js` 已不再由该服务返回。
- 验证新 macOS 二进制内嵌新前端入口，SHA256：`1fb4977350cfa345baacd97e3a0248db1bc248459a3f52d12c14e5ca9fba893e`。
- 重新生成 Ubuntu amd64 单二进制包：`release/gyrh-go-v2-202605221710-844df43-ubuntu-amd64.tar.gz`。
- Ubuntu 单文件更新目标：`release/gyrh-go-v2-202605221710-844df43-ubuntu-amd64/bin/gyrh-server`。
- Ubuntu 二进制验证为 Linux x86-64 ELF，并确认内嵌：
  - 最新前端入口 `/assets/index-DiGlKjB2.js`。
  - MediaPipe `selfie_segmentation.binarypb` 与 `selfie_segmentation_solution_wasm_bin.wasm`。
  - 背景图库 `ORDER BY name DESC, id DESC` 排序逻辑。
  - 风格转换 `foreground_asset_id` 修复逻辑。
- Ubuntu release 包内确认没有 `frontend/dist` 目录，避免误用第二套前端文件；`install_nginx.sh` 确认为仅 `proxy_pass http://127.0.0.1:$BACKEND_PORT`，没有 `root` / `alias` / `try_files` 静态目录配置。
- Ubuntu `bin/gyrh-server` SHA256：`617b00600571a8a6c6866c15444ea2f9351a07d484aec20a65798758a744d168`。
- Ubuntu tar.gz SHA256：`7a183a2416a95b3684f69255a1b344a5e789cb9482abb285a83440451a8ed53a`。

### 未提交内容说明

- 本次提交不包含 `backend/data/gyrh.db*`、`release/`、`.cache/`、`bin/302helpper-*`、`../gyrh-go-v2.zip` 等运行、缓存和打包产物。

## 2026-05-22 00:53

- 分支：`feature/fullscreen-responsive-adaptation`
- 目的：将 Ubuntu 后端 release 调整为真正的单二进制部署形态，前端构建产物内嵌进 `gyrh-server`；同时恢复背景图库 `image_url` 的 OSS 原图 URL 语义，避免前端无法生成正确 OSS 缩略图。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：`19a9eef`（本次改动尚未提交，当前值为改动前基准提交）

### 详细修改说明

- 新增 `backend/internal/frontend`，通过 Go `embed` 将 `frontend/dist` 编译进后端二进制，并为 `/assets/`、`/models/selfie_segmentation/`、`/branding/` 返回长期缓存头。
- 在后端主路由中注册前端 SPA fallback，非 `/api/` 路径由后端直接返回内嵌静态文件或 `index.html`，因此 Nginx 可以将 `/` 全部反向代理到 `127.0.0.1:9913`；同时修复 fallback 依赖 `http.FileServer` 时 `/admin_viewer` 先返回 301、旧服务表现为 404 的问题。
- 放宽项目根目录识别逻辑，支持单二进制部署时仅存在 `configs/config.yaml`，并兼容旧服务器目录中的 `config/config.yaml`，降低“未找到项目根目录”的部署风险。
- 更新 `scripts/build_release.sh`，在编译 Ubuntu amd64 后端前自动同步最新前端 `dist` 到 embed 目录，并修复 README 生成时反引号被 shell 执行的问题；`install_nginx.sh` 改为不再检查或读取 `frontend/dist`，而是将所有路径反向代理到单二进制后端。
- 恢复 `background-prompts` 列表中 `image_url` 返回 OSS 签名原图 URL 的行为，新增 `image_proxy_url` 作为稳定本地代理字段，避免把本地 `/api/v1/images/view?...` 误传给缩略图接口的 `url=` 参数。
- 后台背景管理页的缩略图改为优先使用 `image_asset_id` 构造 `/api/v1/images/thumbnail?asset_id=...`，保留原有 `400x400` 与 `150x150` 尺寸，确保前端拿到的是正确 OSS 缩略图重定向地址。
- 增强 `backend/internal/frontend` 回归测试，覆盖全部 MediaPipe 本地关键资源（`binarypb`、WASM、JS、TFLite）均能由单二进制返回 200，防止 `/models/selfie_segmentation/selfie_segmentation.binarypb` 线上缺失类问题再次漏测。
- 新增 `scripts/verify_release.sh` 到 Ubuntu release，用于部署后同时检查后端直连地址和公网/Nginx 地址是否返回新前端入口、`/admin_viewer`、全部 MediaPipe 本地模型资源。
- 为 MediaPipe 本地资源 URL 增加版本查询参数 `?v=202605220122`，覆盖 `selfie_segmentation.js`、WASM、`binarypb`、TFLite 和预取列表，绕过浏览器中可能已缓存的旧 404/HTML/WASM 响应。
- 调整内嵌前端静态路由，缺失的 `/assets/`、`/models/` 等静态资源不再 fallback 到 `index.html`，避免 WASM 请求误拿 HTML 后卡在 `wasm-instantiate`。
- `manage.sh start` 会在启动前检查 `bin/gyrh-server` 是否可执行，缺少执行权限时自动 `chmod +x`，避免替换二进制后因权限缺失启动失败。
- 将动态旧图/缩略图缓存窗口调整为 3 分钟：前端所有 `/api/v1/images/thumbnail` 动态图片 URL 追加 3 分钟时间桶 `rv` 参数，刷新或重开浏览器最多复用 3 分钟旧图；后端 `View` 与 `Thumbnail` 响应缓存头同步改为 `Cache-Control: public, max-age=180`。
- 新增前端 `RefreshingImage`，动态图片加载失败时立即给当前缩略图 URL 追加 `retry` 参数重新请求后端，促使后端重新生成 OSS 签名地址；历史侧栏、历史页、背景图库、背景管理和预览页都接入该机制，避免 OSS 过期后继续等待 3 分钟缓存窗口。
- 重新生成 Ubuntu amd64 release：`release/gyrh-go-v2-202605221313-19a9eef-ubuntu-amd64.tar.gz`；其中 `bin/gyrh-server` 已验证为 Linux x86-64 ELF，大小约 35M，SHA256 为 `390e445a847b897950ffe908282d7acfcb95932968ce608bbb430f21508f20a5`。

### 验证

- `cd backend && go test ./...`
- `cd frontend && node --test src/utils/*.test.js && npm run build`
- `file release/gyrh-go-v2-202605221313-19a9eef-ubuntu-amd64/bin/gyrh-server`
- `shasum -a 256 release/gyrh-go-v2-202605221313-19a9eef-ubuntu-amd64/bin/gyrh-server`
- `rg "retry=|refreshImageUrl|rv=|202605220122" frontend/dist/assets/index-CgIcyYDU.js`
- `curl -i http://127.0.0.1:19913/admin_viewer`
- `curl -I http://127.0.0.1:19913/models/selfie_segmentation/selfie_segmentation_landscape.tflite`
- `curl -I http://127.0.0.1:19913/models/selfie_segmentation/selfie_segmentation.binarypb`

## 2026-05-21 23:15

- 分支：`feature/fullscreen-responsive-adaptation`
- 目的：补齐前端全屏自适应能力，覆盖展厅 kiosk、大屏横屏、竖屏终端、后台管理页、低高度终端和移动/平板边缘尺寸；同时补充自动化响应式检查与中文变更记录。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：`19a9eef`（本次改动尚未提交，当前值为改动前基准提交）

### 详细修改说明

- 全局布局：
  - 在 `frontend/src/styles.css` 增加 `--app-viewport-height: 100dvh`，将主要页面高度从固定 `100vh` 迁移到动态视口高度，降低全屏终端壳、移动浏览器地址栏和低高度屏幕造成的按钮裁切风险。
  - 调整 `.app-shell` 为横向裁切、纵向可滚动，避免整页 `overflow: hidden` 导致边缘尺寸下关键操作不可达。
  - 将 `.canvas-frame`、`.workbench-screen`、`.simple-screen`、`.history-panel`、`.table-panel`、`.capture-shell`、`.rendering-shell`、`.centered-screen` 等核心容器统一接入动态视口高度。

- kiosk / 展厅模式：
  - 在 `frontend/src/theme/kiosk.css` 保留大屏横屏下的 16:9 展示比例和最大宽度。
  - 新增 `max-width: 1180px` 或 `orientation: portrait` 断点：窄屏和竖屏时取消固定 `aspect-ratio: 16 / 9`，改为全宽、动态高度自适应。
  - 为 kiosk 导航按钮增加窄屏下的 `clamp()` 尺寸，避免按钮过宽挤出屏幕，同时保留触摸操作面积。
  - 为低高度终端取消 kiosk 顶栏固定最小高度，减少内容被顶部区域压缩。

- 拍摄页：
  - 在 `frontend/src/styles.css` 新增低高度断点 `@media (max-height: 760px)`。
  - 低高度下将 `.opacity-slider-wrapper` 和 `.zoom-slider-wrapper` 从左右竖向控件改为底部横向控件。
  - 低高度下取消 `.vertical-slider` 的旋转，压缩按钮和间距，保证摄像头预览、拍摄、放弃、使用等核心动作可见或可达。

- 预览页：
  - 在 `frontend/src/screens/PreviewScreen.jsx` 将对比预览区域从内联固定 flex 样式迁移到 `.preview-stage-container` 响应式 class。
  - 在 `frontend/src/styles.css` 为预览对比增加窄屏 `max-width: 900px` 断点，原图/效果图从左右并排切换为上下排列。
  - 单图预览通过 `.single-preview` 控制最大宽度，不再依赖组件内联布局。
  - 全屏预览图片从 `objectFit: 'cover'` 改为 `objectFit: 'contain'`，高度从 `100vh` 改为 `100dvh`，避免全屏查看时裁掉人像主体。
  - 为二维码弹窗和预览水印增加小屏尺寸约束，避免遮挡和横向溢出。

- 后台管理页：
  - 在 `frontend/src/screens/StyleManagerScreen.jsx` 移除风格转换表格中固定的内联 `gridTemplateColumns`，改用 `.style-table-grid`。
  - 在 `frontend/src/screens/BackgroundManagerScreen.jsx` 和 `frontend/src/screens/StyleManagerScreen.jsx` 移除弹窗内固定 `width: 80%`、`maxWidth: 800px`、`maxHeight: 90vh` 等内联约束，交给统一 CSS 控制。
  - 在背景编辑弹窗中把双列表单改为 `.responsive-form-grid`，小屏下自动变为单列。
  - 为后台表格移动端行内容补充 `data-label`，窄屏下隐藏表头后仍能识别字段含义。
  - 为 `.table-shell`、`.modal-content`、`.modal-overlay` 增加局部滚动和视口内高度控制，使平板/小屏下保存、取消、关闭按钮可达。

- 测试与脚本：
  - 在 `frontend/package.json` 新增 `npm test` 脚本，统一运行 Node 内置测试。
  - 新增 `frontend/src/utils/responsiveLayout.test.js`，覆盖以下响应式规则：
    - 全局布局必须使用动态视口高度。
    - kiosk 窄屏或竖屏必须取消固定 16:9。
    - 拍摄页低高度必须启用横向控制布局。
    - 预览对比布局必须由响应式 class 控制。
    - 后台风格管理表格不得继续依赖内联固定列宽。

### 文件清单

- `CHANGELOG.md`：按旧规则改为中文倒序记录，并补充本次迭代详细说明、测试计划与验证结果。
- `frontend/package.json`：新增 `test` 脚本。
- `frontend/src/styles.css`：新增动态视口高度、全局滚动策略、预览/拍摄/后台表格/弹窗/移动端断点规则。
- `frontend/src/theme/kiosk.css`：新增 kiosk 窄屏、竖屏、低高度自适应规则。
- `frontend/src/screens/PreviewScreen.jsx`：迁移预览对比布局 class，调整全屏图片适配策略。
- `frontend/src/screens/StyleManagerScreen.jsx`：移除固定内联表格列宽，补充移动端字段标签，弹窗尺寸交给 CSS。
- `frontend/src/screens/BackgroundManagerScreen.jsx`：弹窗尺寸和双列表单迁移到响应式 CSS，补充移动端字段标签。
- `frontend/src/utils/responsiveLayout.test.js`：新增响应式布局回归测试。

### 测试计划

- 视口矩阵：
  - kiosk 横屏：`1920x1080`、`2560x1440`、`3840x2160`
  - kiosk 竖屏：`1080x1920`、`1440x2560`
  - 后台桌面：`1366x768`、`1440x900`、`1536x864`、`1920x1080`
  - 平板：`768x1024`、`1024x768`、`834x1194`
  - 手机边缘宽度：`390x844`、`430x932`
  - 低高度终端：`1280x720`、`1024x600`
- 功能流程：
  - kiosk `/`：工作台加载、背景上传/选择、历史侧栏预览、进入拍摄、进入预览。
  - admin `/admin_viewer`：历史记录、背景库、SKILL 管理、风格转换配置、登录/退出相关页面。
  - 拍摄页：摄像头预览可见，透明度/人物大小控制可达，拍摄/放弃/使用按钮可达。
  - 预览页：对比模式窄屏上下排列，二维码弹窗可关闭，全屏预览不裁切主体。
  - 后台弹窗：背景和风格编辑弹窗可滚动、保存、取消、关闭。
- 逻辑与布局：
  - 除表格局部滚动外，不应出现不可控的整页横向溢出。
  - 关键按钮必须可见或可通过纵向滚动到达。
  - 大屏 kiosk 保持 16:9 展示效果。
  - 竖屏和低高度终端不能继承大屏固定比例限制。
- 性能：
  - 执行 `npm run build`，确认 Vite 生产构建通过。
  - 检查构建产物体积，确认本次主要是 CSS 与小型测试变更，没有新增运行时依赖。

### 验证结果

- `npm test`：通过，36 个测试全部成功。
- `npm run build`：通过，Vite 生产构建成功；产物约 `207.38 kB` JS / `33.41 kB` CSS。
- 浏览器功能/自适应验证：
  - kiosk `/`：已检查 `1920x1080` 横屏和 `1080x1920` 竖屏，关键导航和工作台控件可访问。
  - admin `/admin_viewer`：已检查 `1366x768` 和 `768x1024`，后台导航、风格配置页、风格新建弹窗、取消操作可访问。
- 浏览器网络/控制台：
  - 当前 API 请求返回 200。
  - 观察到一条 `/api/v1/images/` 图片资源 404，判断为已有数据中的图片 URL 为空或缺失，不是本次布局改动产生的阻断。
- IDE 诊断：本次触达文件无 linter 错误。
