# 远端背景图库同步设计

## 背景

背景图管理页顶部有“同步”按钮，目前还没有绑定业务逻辑。`siteConfig.json` 中的 `gallery.apiUrl` 指向远端图库接口：

```text
https://jjxo.chinafilmai.com/picGet/api/media?page=1&pageSize=20&sort=createTime,desc
```

该接口返回当前页媒体列表，每条数据包含 `id`、`url`、`thumbUrl`、`dimensions`、`prompt` 和 `params`。现有 `/api/v1/background-prompts/import` 会在上传图片后调用 Qwen 反推提示词，不适合远端图库同步。本次同步功能需要直接把远端图片上传到本系统 OSS，并创建或补齐本地背景图记录，但不触发任何提示词反推。

## 目标

点击“同步”后，系统从配置文件中的远端图库地址拉取当前页数据，将未同步过的远端图片下载并上传到本系统 OSS，再写入 `background_prompts` 表。远端 `prompt` 作为已有中文文案写入本地提示词字段，英文提示词保持空值，后续可由管理员手工编辑或单独点击翻译。

## 非目标

本次不自动同步所有分页，不做后台定时任务，不重做背景管理 UI，不调用 Qwen 图片理解反推提示词，不自动翻译英文提示词，也不覆盖已经同步并在本地编辑过的记录。

## 用户流程

1. 管理员进入背景图管理页。
2. 点击右上角“同步”按钮。
3. 前端读取 `siteConfig.gallery.apiUrl`，调用后端同步接口。
4. 后端只同步该 URL 当前页数据。
5. 对已经同步过的远端 `id`，后端跳过，不覆盖本地记录。
6. 对未同步过的数据，后端下载远端原图，上传到 OSS，创建背景图记录。
7. 同步完成后，前端展示导入、跳过、失败统计，并刷新本地库列表。

## 推荐方案

新增专用接口：

```text
POST /api/v1/background-prompts/sync-remote
```

请求体：

```json
{
  "api_url": "https://jjxo.chinafilmai.com/picGet/api/media?page=1&pageSize=20&sort=createTime,desc"
}
```

响应体：

```json
{
  "imported": 18,
  "skipped": 2,
  "failed": 0,
  "failures": []
}
```

该接口独立于现有 `Import` 流程，避免复用时误触发 Qwen 反推，也让“本地手动导入”和“远端图库同步”的职责保持清晰。

## 前端设计

`BackgroundManagerScreen` 中的“同步”按钮从静态按钮改为可触发状态按钮：

- 未同步时显示“同步”。
- 请求中显示“同步中...”并禁用按钮。
- 成功后刷新本地库列表。
- 失败时提示错误原因。

前端不下载图片、不转 base64，也不逐条调用现有导入接口。前端只把 `siteConfig.gallery.apiUrl` 发送给后端，由后端完成远端读取、图片下载、OSS 上传和数据库写入。

## 后端数据流

1. 校验 `api_url` 非空，并限制为 `http` 或 `https` URL。
2. 请求远端接口并解析响应。
3. 遍历 `data.items`。
4. 用远端 `id` 生成稳定本地名称：`remote:<id>`。
5. 调用 `BackgroundPromptRepo.GetByName` 检查是否已经同步。
6. 已存在则计入 `skipped`，不更新数据库。
7. 未存在则补全图片 URL，下载原图。
8. 解析图片宽高，优先使用实际图片尺寸；若解析失败，再尝试使用 `dimensions`。
9. 调用 `storageService.SaveWithKind(..., storage.SaveKindAsset)` 上传到 OSS。
10. 调用 `BackgroundPromptRepo.Create` 创建背景图库记录。

如某条记录失败，继续处理后续记录，并在 `failures` 中返回失败项的 `id` 和原因。

## 字段映射

远端数据映射到 `background_prompts`：

- `name`: `remote:<远端id>`
- `gemini_prompt_zh`: 远端 `prompt`
- `wan_prompt_zh`: 远端 `prompt`
- `gemini_prompt`: 空字符串
- `wan_prompt`: 空字符串
- `gemini_negative_prompt`: 空字符串
- `wan_negative_prompt`: 空字符串
- `gemini_negative_prompt_zh`: 空字符串
- `wan_negative_prompt_zh`: 空字符串
- `image_asset_id`: 上传 OSS 后返回的 asset id
- `image_url`: 空字符串，保持现有动态签名 URL 读取方式
- `image_width` / `image_height`: 图片实际尺寸或远端 `dimensions`

远端 `params` 本次不入库。后续如果需要展示更友好的名称，可在独立设计中增加来源字段或标题字段。

## URL 处理

远端接口中的 `url` 可能是相对路径，例如：

```text
/media_images/fcc15018dbd4b48f_20260506110411.jpg
```

后端根据 `api_url` 的 scheme 和 host 补全为完整地址：

```text
https://jjxo.chinafilmai.com/media_images/fcc15018dbd4b48f_20260506110411.jpg
```

如果远端返回完整 URL，则直接使用。

## 去重策略

采用稳定名称 `remote:<id>` 作为去重键。重复点击同步时：

- 本地已有 `remote:<id>`：跳过。
- 本地没有 `remote:<id>`：下载并创建记录。

该策略避免覆盖管理员手动编辑过的本地提示词，也不需要本次修改数据库表结构。

## 错误处理

接口级错误直接返回失败：

- `api_url` 为空或不是合法 HTTP URL。
- 远端接口不可访问。
- 远端响应不是预期 JSON 结构。
- 存储服务未初始化。

单条记录错误计入 `failures`，同步继续：

- 远端图片 URL 为空。
- 图片下载失败。
- 图片不是合法图片数据。
- OSS 上传失败。
- 数据库创建失败。

如果某条记录已经上传 OSS 但数据库创建失败，应尝试删除刚上传的 asset，避免孤儿文件。

## 测试与验证

建议覆盖：

- 点击“同步”后调用新接口，并在完成后刷新本地库。
- 只同步 `siteConfig.gallery.apiUrl` 当前页，不自动翻页。
- 远端 `prompt` 同时写入 `wan_prompt_zh` 和 `gemini_prompt_zh`。
- 英文提示词字段保持空值。
- 不调用 `qwenAdvisor.SuggestFromAsset`。
- 重复同步时已存在 `remote:<id>` 的记录被跳过，不覆盖本地编辑。
- 相对图片 URL 能正确补全并下载。
- 部分记录失败时，其他记录继续同步，响应包含失败明细。

## 实施范围

主要涉及：

- `gyrh-go-v2/frontend/src/screens/BackgroundManagerScreen.jsx`
- `gyrh-go-v2/backend/internal/api/router.go`
- `gyrh-go-v2/backend/internal/api/handler/background_prompt.go`

可选补充：

- 为远端响应解析、URL 补全、去重和失败统计增加 Go 单元测试。
