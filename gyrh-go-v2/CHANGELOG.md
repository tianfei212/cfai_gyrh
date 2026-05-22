# CHANGELOG

本文档记录每次会话/迭代对仓库做的实质性改动，按倒序时间排列（最新在最前）。
每条记录至少包含：日期、更新时间、分支、目的、执行者、文件清单与 commit hash。
执行者格式：人工 或 Claude <模型名>（<model-string>，Anthropic）。
仅记录代码 / 配置 / 文档层面的变更；个人调试痕迹（缓存、PID、临时日志）不在此记录。

## 2026-05-23 00:08

- 分支：`main`
- 目的：发布 `3.0.1`，同步背景分类管理、全终端响应式适配、工作台背景图全图放大预览和单二进制 release 产物说明。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- tag：`3.0.1`
- commit hash：本条记录随本次提交生成。

### 发布内容

- 合并并保留 `feature/fullscreen-responsive-adaptation` 的前端响应式与缓存逻辑，叠加背景图类型管理能力。
- 背景图类型支持“大类/小类”二级分类、`default/default` 兜底分类、类型 CRUD、背景图多类型绑定和工作台按类型筛选。
- 工作台背景图库新增放大镜 HUD 图标，点击后通过后端 OSS 全图 WebP 地址打开大图预览。
- 预览弹窗改为 React Portal 渲染到 `document.body`，使用百分比布局与 `object-fit: contain` 约束在当前视口内，避免移动端或低高度终端出现长滚动。
- 放大入口使用放大镜图标，关闭入口使用 `X` 图标。
- release 包继续保持“前端内嵌到 `gyrh-server` 单文件后端”的部署形态，线上只更新对应平台的 `bin/gyrh-server` 即可获得本次前端功能更新。

### 数据库与兼容性

- 新增 `background_categories` 与 `background_category_bindings`，背景图和类型为多对多关系。
- 数据库升级脚本会创建 `default/default` 类型，并将历史未绑定背景图回填到该类型。
- 删除非默认类型时，相关背景图会回归 `default/default`；`default/default` 不允许删除。
- 线上已有正确业务数据时，不应覆盖线上 `backend/data/gyrh.db`；只需执行/确认升级脚本和替换二进制。

### Release 产物

- 当前项目目录下的 release 产物：
  - `release/gyrh-go-v2-202605222328-44a412f-darwin-arm64.tar.gz`
  - `release/gyrh-go-v2-202605222328-44a412f-ubuntu-amd64.tar.gz`
- macOS ARM64 包内 `bin/gyrh-server` 为 Mach-O arm64 单文件。
- Ubuntu amd64 包内 `bin/gyrh-server` 为 Linux x86-64 ELF 单文件。
- 两个平台二进制均已确认内嵌最新前端入口：
  - `/assets/index-tdw_bnPy.js`
  - `/assets/index-vaheVFXX.css`

### 验证

- 前端单元测试通过：`npm test`，45/45 通过。
- 前端生产构建通过：`npm run build -- --outDir ../backend/internal/frontend/dist --emptyOutDir`。
- macOS release 单文件已从当前项目 `release/` 目录启动测试，`/`、`/admin_viewer`、MediaPipe 模型资源返回 `200`。
- 带签名背景列表 API 验证通过：`code=0`、`total=31`。
- Ubuntu amd64 二进制已静态验证为 Linux x86-64 ELF，并确认内嵌最新前端入口与 MediaPipe 资源；当前 macOS 环境无法直接运行 Linux ELF。
- 当前测试服务使用当前项目目录的 `release/gyrh-go-v2-202605222328-44a412f-darwin-arm64`，开发前端 `9912` 已关闭。

### 部署注意事项

- 如果目标机器已有正确配置、数据库、生成图目录和 OSS 二进制，本次前端与后端代码更新只需替换对应平台的 `bin/gyrh-server`。
- 如果目标机器缺少 `bin/oss-cli`，需要补齐对应平台 aliOSS 二进制；否则后端自动启动 OSS 时会失败。
- `release/`、运行态数据库、WAL/SHM、缓存和本地 OSS 软链接不纳入 git 提交。

## 2026-05-22 23:26

- 分支：`main`
- 目的：为工作台背景图库增加全图放大预览，并修正弹窗在全终端适配下产生长滚动的问题。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：`fe96815`

### 说明

- 在工作台背景图库卡片右上角新增放大镜 HUD 图标，点击后通过后端 OSS 全图 WebP 地址打开大图预览。
- 预览弹窗改为 React Portal 渲染到 `document.body`，避免被工作台外层滚动容器影响。
- 弹窗、图片舞台和图片本身使用百分比约束与 `object-fit: contain`，禁止内部长滚动，保持移动端和桌面端视口内可见。
- 关闭入口改为 `X` 图标按钮，放大入口改为放大镜图标按钮。

### 修改文件

- `frontend/src/screens/DashboardScreen.jsx`：新增背景图全图预览状态、Portal 弹窗、放大镜入口和 `X` 关闭按钮。
- `frontend/src/styles.css`：新增图库放大镜按钮与预览弹窗响应式布局样式。
- `frontend/src/utils/imageThumbs.js`：新增全图预览 URL 构建工具，优先使用 OSS 原图地址。
- `frontend/src/utils/imageThumbs.test.js`：补充全图预览 URL 构建测试。

### 验证

- 前端单元测试通过：`npm test`，45/45 通过。
- 前端生产构建通过：`npm run build -- --outDir ../backend/internal/frontend/dist --emptyOutDir`。
- 已重启前端、后端、背景 OSS 与生成图 OSS，本地端口 `9912`、`9913`、`18080`、`18081` 均处于监听状态。
- 嵌入式前端资源已更新到 `/assets/index-tdw_bnPy.js` 与 `/assets/index-vaheVFXX.css`。

## 2026-05-22 22:46

- 分支：`main`
- 目的：将已完成验证的 `feature/category-responsive-integration` 提升为本地主分支，作为背景分类管理与响应式适配集成后的主线版本。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：本条记录随本次提交生成。

### 说明

- 当前集成分支已包含 `origin/main` 与本地 `master` 历史，可快进为主线。
- 保留此前集成提交：
  - `acc3d95`：背景分类管理与响应式 shell 合并。
  - `08cee35`：记录集成 changelog。
  - `97046bc`：修复运行态前端分类与背景管理显示问题。
  - `055694f`：记录运行态修复 changelog。
- 本次仅记录分支提升动作，不新增业务代码。

### 验证

- 分支关系检查确认 `origin/main` 是当前集成分支祖先。
- 本地 `main` 分支不存在，避免覆盖已有本地 `main` 工作区。
- 未提交运行态数据库、WAL/SHM、日志、`.env.local`、aliOSS 私有配置或本地软链接。

## 2026-05-22 22:43

- 分支：`feature/category-responsive-integration`
- 目的：修复背景分类集成后的运行态前端错误和背景管理表格显示问题，并重新校准本地运行验证的数据库、鉴权和 OSS 配置。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：`97046bc`

### 本次问题

- 工作台打开时报 `ReferenceError: fetchApi is not defined`，导致背景分类列表加载失败。
- 背景管理页在当前缩放/响应式布局下，把 `data-label` 插入到了单元格内容前，显示成“图片名称xxx”和“类型default/default”。
- 本地验证过程中曾错误地从 integration worktree 启动后端，先后出现过测试库数据、鉴权 env 不一致、OSS 未启动、OSS bucket 前缀错误等运行态问题。

### 解决方案

- `DashboardScreen.jsx` 补充 `fetchApi` import，恢复工作台分类接口请求。
- `BackgroundManagerScreen.jsx` 移除背景管理表格行中会拼接到内容前的 `data-label`，保留表头展示，避免名称和类型内容被加前缀。
- 本地重启时使用主工作区真实数据库路径、主工作区 `.env.local` / `frontend/.env.local` 和真实 `configs/alioss-agent.yaml`，确保背景数据、签名鉴权、OSS 背景 bucket 前缀一致。

### 修改文件

- `frontend/src/screens/DashboardScreen.jsx`：补充 `fetchApi` import。
- `frontend/src/screens/BackgroundManagerScreen.jsx`：移除背景管理表格内容列的 `data-label`，避免响应式标签拼入文本。
- `CHANGELOG.md`：新增本次排障和修复记录。

### 验证

- 前端单元测试通过：`npm test`，43/43 通过。
- 前端生产构建通过：`npm run build`，生成新入口 `/assets/index-C1D8LZwJ.js`。
- 后端已同步新前端构建到 embed 目录，并重新启动。
- 当前运行端口：
  - 后端：`9913`。
  - 背景 OSS：`18080`。
  - 生成图 OSS：`18081`。
- 带签名背景列表 API 验证返回 `code=0`、`total=31`。
- 抽查背景管理首页前 10 个 `image_asset_id`，OSS `/view/...` 均返回 `200`。

### 未提交内容说明

- 本次提交不包含运行态数据库文件、WAL/SHM 文件、`frontend/dist/`、`backend/internal/frontend/dist/`、日志目录、`.env.local`、`configs/alioss-agent.yaml` 或本地 `bin/oss-cli` 软链接。

## 2026-05-22 22:20

- 分支：`feature/category-responsive-integration`
- 目的：完成 `feature/background-category-management` 与 `feature/fullscreen-responsive-adaptation` 的集成合并；保留前端响应式与背景缓存逻辑，并叠加背景图二级分类管理、绑定、筛选和线上数据库升级脚本。
- 执行者：Claude GPT-5.5（GPT-5.5，OpenAI）
- commit hash：`acc3d95`

### 合并说明

- 从 `feature/fullscreen-responsive-adaptation` 新建集成分支 `feature/category-responsive-integration`，手动合并 `feature/background-category-management`。
- 前端以响应式分支为基底，保留 Dashboard 背景缓存、响应式布局和管理表格的 `data-label` 适配，再接入分类选择、分类管理弹窗和背景-分类多选绑定。
- 后端同时保留远端图库同步的 `gallery.external_url` 默认接口逻辑，以及背景分类仓库、分类 CRUD、背景分类绑定和分类筛选接口。
- 新增 `scripts/upgrade_background_categories.sh`，用于线上 SQLite 数据库创建分类表、绑定表、默认 `default/default` 分类，并回填未绑定背景。

### 修改文件

- `backend/internal/db/background_category.go`、`backend/internal/db/background_category_test.go`：新增背景分类仓库、默认分类、绑定替换、删除回归默认分类等逻辑和测试。
- `backend/internal/db/background_prompt.go`、`backend/internal/db/db.go`：新增分类筛选查询、绑定清理和数据库建表迁移。
- `backend/internal/api/handler/background_category.go`、`backend/internal/api/handler/background_prompt.go`、`backend/internal/api/router.go`：新增分类 API、背景绑定 API，并让背景列表返回分类信息。
- `backend/internal/platform/app/app.go`：初始化分类仓库并在启动时确保默认绑定。
- `frontend/src/screens/DashboardScreen.jsx`、`frontend/src/screens/BackgroundManagerScreen.jsx`：接入工作台分类筛选、管理页分类维护和背景绑定 UI。
- `frontend/src/app/AppShell.jsx`、`frontend/src/utils/backgroundCache.js`、`frontend/src/utils/backgroundPagination.js`：让背景缓存和分页 URL 支持 `category_id`，避免不同分类缓存串用。
- `frontend/src/utils/backgroundCache.test.js`、`frontend/src/utils/backgroundPagination.test.js`：覆盖分类缓存隔离和分类 URL 参数。
- `scripts/upgrade_background_categories.sh`：新增线上数据库升级脚本。
- `CHANGELOG.md`：新增本次集成记录。

### 验证

- 前端依赖安装：`npm ci --prefer-offline --no-audit --no-fund`。
- 前端单元测试通过：`npm test`，43/43 通过。
- 前端生产构建通过：`npm run build`。
- 后端全量测试通过：`go test ./...`。
- 运行 smoke：
  - 前端 `http://127.0.0.1:9912/` 返回 200。
  - 后端 `http://127.0.0.1:9913/` 返回 200。
  - 受保护分类接口 `GET /api/v1/background-categories` 未登录返回 401。
  - 本地运行库分类校验结果为分类 1 条、绑定 9 条、未绑定背景 0 条。
- 前后端已从集成 worktree 重启：前端监听 `9912`，后端监听 `9913`。

### 未提交内容说明

- 本次提交不包含运行态文件：`backend/data/gyrh.db-shm`、`backend/data/gyrh.db-wal`、`frontend/node_modules/`、`frontend/dist/`、`backend/internal/frontend/dist/`、日志目录和本地 `bin/oss-cli` 软链接。
- 集成 worktree 为运行验证临时使用本地 aliOSS 配置与环境变量覆盖，相关私有配置没有纳入提交。

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
