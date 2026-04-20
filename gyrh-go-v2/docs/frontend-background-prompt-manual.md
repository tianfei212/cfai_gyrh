# 前端接入手册：背景图提示词模板与图像改写

## 1. 目标

本手册用于指导前端完成以下两块能力：

1. 背景图提示词模板管理
2. 图像改写时选择背景图提示词模板

当前后端已经支持：

- 背景图提示词模板的 CRUD
- 基于背景图调用 Qwen 生成 Gemini/Wan 的中英双语默认建议
- 当前端修改中文提示词后，调用后端接口自动同步英文部分
- 在 `images/rewrite` 时通过 `background_prompt_id` 读取数据库模板
- 背景融合场景不再接受前端自由提示词
- 仅保留 `style_prompt` 用于风格转换类控制
- 按不同模型自动选择对应模板字段
  - Gemini / Google 使用 `gemini_prompt` 与 `gemini_negative_prompt`
  - Wan 使用 `wan_prompt` 与 `wan_negative_prompt`

## 2. 鉴权说明

除健康检查与 `GET /api/v1/skills/active` 外，其余接口都在登录态下访问。

前端请求时需要同时满足两类鉴权：

- 携带当前登录会话 Cookie
- 携带签名请求头

### 2.1 Cookie

- `fetch`/`axios` 需要打开 `credentials`

示例：

```ts
fetch("/api/v1/background-prompts", {
  method: "GET",
  credentials: "include",
});
```

### 2.2 请求头签名

后端鉴权中间件要求以下请求头：

- `X-Real-IP`
- `X-Public-Key`
- `X-Timestamp`
- `X-Signature`

签名规则：

- 拼接内容：`clientIP + publicKey + timestamp`
- 算法：`HMAC-SHA256`
- 密钥：`privateKey`
- 输出：十六进制小写字符串

`.env.local` 中当前使用的是：

```bash
GYRH_AUTH_PUBLIC_KEY=gyrh_web
GYRH_AUTH_PRIVATE_KEY=...
```

前端侧对应关系：

- `GYRH_AUTH_PUBLIC_KEY` -> 请求头 `X-Public-Key`
- `GYRH_AUTH_PRIVATE_KEY` -> 参与生成 `X-Signature`

前端生成请求头示例：

```ts
async function signRequest(clientIP: string, publicKey: string, privateKey: string) {
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const content = clientIP + publicKey + timestamp;

  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(privateKey),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );

  const signatureBuffer = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(content)
  );

  const signature = Array.from(new Uint8Array(signatureBuffer))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  return {
    "X-Real-IP": clientIP,
    "X-Public-Key": publicKey,
    "X-Timestamp": timestamp,
    "X-Signature": signature,
  };
}
```

请求示例：

```ts
const authHeaders = await signRequest(
  clientIP,
  import.meta.env.VITE_GYRH_AUTH_PUBLIC_KEY,
  import.meta.env.VITE_GYRH_AUTH_PRIVATE_KEY
);

await fetch("/api/v1/background-prompts", {
  method: "GET",
  credentials: "include",
  headers: authHeaders,
});
```

安全建议：

- 如果这是桌面端或内网受控前端，可以直接注入这两个值
- 如果是公网 Web 前端，不建议把 `privateKey` 直接暴露到浏览器，应改成由网关或后端代理完成签名

## 3. 通用响应结构

后端统一返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

说明：

- `code = 0` 表示成功
- `code != 0` 表示业务失败
- HTTP 状态码仍然有效，前端需要同时判断 HTTP 状态与 `code`

失败示例：

```json
{
  "code": 1,
  "message": "背景图提示词模板不存在"
}
```

## 4. 数据模型

背景图提示词模板对象：

```ts
type BackgroundPrompt = {
  id: number;
  name: string;
  gemini_prompt: string;
  gemini_negative_prompt: string;
  wan_prompt: string;
  wan_negative_prompt: string;
  created_at: string;
  updated_at: string;
};
```

字段建议展示：

- `name`: 模板名称，给用户选择
- `gemini_prompt`: Gemini 正向提示词
- `gemini_negative_prompt`: Gemini 反向提示词
- `wan_prompt`: Wan 正向提示词
- `wan_negative_prompt`: Wan 反向提示词

## 5. 模板管理接口

接口前缀：`/api/v1/background-prompts`

### 5.1 查询列表

`GET /api/v1/background-prompts?limit=20&offset=0`

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "森林清晨",
        "gemini_prompt": "晨雾森林，柔和逆光，层次丰富",
        "gemini_negative_prompt": "低清晰度，过度曝光，杂乱建筑",
        "wan_prompt": "misty forest in morning light, layered depth, cinematic",
        "wan_negative_prompt": "blur, overexposure, crowded buildings",
        "created_at": "2026-04-20T10:00:00Z",
        "updated_at": "2026-04-20T10:00:00Z"
      }
    ],
    "total": 1,
    "limit": 20,
    "offset": 0
  }
}
```

前端建议：

- 页面初始化先拉列表
- 支持分页或滚动加载
- 模板较少时也建议保留分页参数，避免后续改接口

### 5.2 查询详情

`GET /api/v1/background-prompts/{id}`

适用场景：

- 打开编辑弹窗前读取最新详情
- 避免列表数据与详情数据不一致

### 5.3 新建模板

`POST /api/v1/background-prompts`

请求体：

```json
{
  "name": "森林清晨",
  "gemini_prompt": "晨雾森林，柔和逆光，层次丰富",
  "gemini_negative_prompt": "低清晰度，过度曝光，杂乱建筑",
  "wan_prompt": "misty forest in morning light, layered depth, cinematic",
  "wan_negative_prompt": "blur, overexposure, crowded buildings"
}
```

校验规则：

- `name` 不能为空
- `gemini_prompt` 与 `wan_prompt` 不能同时为空
- `name` 不能重复

前端校验建议：

- 提交前做必填校验
- 对重复名称错误做明确提示，不要只弹通用失败 Toast

### 5.4 更新模板

`PUT /api/v1/background-prompts/{id}`

说明：

- 支持局部更新
- 只传需要修改的字段即可

请求体示例：

```json
{
  "name": "森林清晨-新版",
  "gemini_negative_prompt": "低清晰度，过曝，过饱和，噪点"
}
```

前端建议：

- 编辑时直接提交整个表单也可以
- 如果做局部提交，注意未修改字段不要误传 `null`

### 5.5 删除模板

`DELETE /api/v1/background-prompts/{id}`

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "deleted": 1
  }
}
```

前端建议：

- 删除前二次确认
- 删除成功后从本地列表移除
- 如果当前改写表单正在使用该模板，删除后应清空选中状态

### 5.6 基于背景图生成默认建议

`POST /api/v1/background-prompts/suggest-defaults`

用途：

- 用户选择一张背景图后，点击“AI 生成默认提示词”
- 后端调用 Qwen3.6 视觉理解，对背景图做图像反推
- 一次性返回 Gemini 与 Wan 两套中英双语建议

请求体：

```json
{
  "background": "<base64>"
}
```

或者：

```json
{
  "background_asset_id": "xxx"
}
```

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "gemini_prompt_zh": "人物自然融入林间小路背景，保持晨雾氛围、柔和逆光、地面接触阴影真实，整体色温统一。",
    "gemini_prompt_en": "Blend the person naturally into the forest path scene, preserve the misty morning mood, soft backlighting, realistic ground contact shadows, and a consistent color temperature.",
    "gemini_negative_prompt_zh": "避免人物悬浮、边缘抠图痕迹、透视错误、光源方向冲突、背景层次混乱。",
    "gemini_negative_prompt_en": "Avoid floating subjects, visible cutout edges, incorrect perspective, conflicting light direction, and messy background depth.",
    "wan_prompt_zh": "让人物与森林晨雾背景自然融合，强调空间纵深、空气透视、柔和体积光和统一色调。",
    "wan_prompt_en": "Fuse the person naturally into the misty forest background with strong spatial depth, atmospheric perspective, soft volumetric light, and unified color tones.",
    "wan_negative_prompt_zh": "避免人物比例异常、景深不匹配、边缘发灰、阴影缺失、色彩割裂。",
    "wan_negative_prompt_en": "Avoid abnormal body proportions, mismatched depth of field, gray edges, missing shadows, and disconnected colors."
  }
}
```

前端建议：

- 用户每次更换背景图后，清空旧建议
- 用户点击按钮后再触发该接口，避免频繁消耗模型额度
- 返回结果作为表单默认值，不要直接自动保存到数据库

### 5.7 中文修改后同步英文

`POST /api/v1/background-prompts/sync-english`

用途：

- 当前端用户修改中文提示词后
- 调用此接口，让后端通过 Qwen 自动生成最新英文版本

请求体：

```json
{
  "gemini_prompt_zh": "人物自然融入夜景街道背景，霓虹反射真实，边缘光协调。",
  "gemini_negative_prompt_zh": "避免悬浮感、光源冲突、边缘穿帮。",
  "wan_prompt_zh": "让人物自然融入夜间街景，保留潮湿路面反光与霓虹氛围。",
  "wan_negative_prompt_zh": "避免透视错误、阴影缺失、人物发灰。"
}
```

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "gemini_prompt_en": "Blend the person naturally into the night street scene with realistic neon reflections and coordinated rim lighting.",
    "gemini_negative_prompt_en": "Avoid a floating look, conflicting light sources, and broken edges.",
    "wan_prompt_en": "Integrate the person naturally into the night street background while preserving wet pavement reflections and a neon atmosphere.",
    "wan_negative_prompt_en": "Avoid perspective errors, missing shadows, and dull gray skin tones."
  }
}
```

前端建议：

- 对中文输入做 `debounce`
- 推荐在用户停止输入 `500ms~800ms` 后调用
- 如果用户连续快速输入，取消前一个请求，只保留最后一次
- 英文区域建议设为只读，由后端自动同步

## 6. Qwen Prompt 模板管理

接口前缀：`/api/v1/llm-prompt-templates`

用途：

- 管理后端给 Qwen 使用的系统提示词模板
- 当前至少有两条关键模板：
  - `qwen_background_prompt_suggestion`
  - `qwen_background_prompt_sync_english`

说明：

- `advisor.go` 中原来的硬编码提示词现在已经抽到数据库中
- Qwen 运行时按 `template_key` 从数据库读取模板内容
- 前端或后台管理端可以通过 CRUD 维护这些模板

模板对象：

```ts
type LLMPromptTemplate = {
  id: number;
  name: string;
  template_key: string;
  content: string;
  description: string;
  created_at: string;
  updated_at: string;
};
```

接口列表：

- `GET /api/v1/llm-prompt-templates`
- `GET /api/v1/llm-prompt-templates/{id}`
- `POST /api/v1/llm-prompt-templates`
- `PUT /api/v1/llm-prompt-templates/{id}`
- `DELETE /api/v1/llm-prompt-templates/{id}`

新建示例：

```json
{
  "name": "Qwen 背景图默认建议",
  "template_key": "qwen_background_prompt_suggestion",
  "content": "这里填写完整 Prompt 模板内容",
  "description": "根据背景图生成 Gemini/Wan 的中英双语正反向提示词建议"
}
```

前端建议：

- 这类模板适合放到“系统配置/模型配置”页面，而不是普通业务页
- `template_key` 建议默认只读，避免误改后导致运行时找不到模板
- 编辑 `content` 时使用大文本编辑器，并支持版本备份

## 7. 图像改写接口变更

接口：`POST /api/v1/images/rewrite`

关键字段：

- `background_prompt_id`
- `style_prompt`

### 7.1 请求体

```ts
type RewriteRequest = {
  foreground?: string;
  background?: string;
  references?: Array<{
    type: "upper" | "lower" | "dress" | "accessory" | "headwear" | "footwear";
    data: string;
  }>;
  provider?: "google" | "wan";
  style_prompt?: string;
  background_prompt_id?: number;
};
```

### 7.2 使用规则

- 背景融合模式：
  - 传 `background`
  - 必须同时传 `background_prompt_id`
  - 不允许再传自由提示词
- 风格转换模式：
  - 不传 `background`
  - 不传 `background_prompt_id`
  - 如有需要，可以传 `style_prompt`
- `style_prompt` 只用于风格转换类控制，例如黑白线稿、水彩、像素风
- 旧字段 `prompt` 已不再推荐使用；后端仅把它当作 `style_prompt` 的兼容别名读取

当前后端处理逻辑：

- `provider = google` 时使用 `gemini_prompt` 与 `gemini_negative_prompt`
- `provider = wan` 时使用 `wan_prompt` 与 `wan_negative_prompt`
- 后端会把以下内容合成为最终 Prompt：
  - Skill
  - 背景模板正向提示词
  - 背景模板反向提示词
  - `style_prompt`（仅在风格转换模式下使用）

### 7.3 请求示例

背景融合示例：

```json
{
  "foreground": "<base64>",
  "background": "<base64>",
  "references": [
    {
      "type": "dress",
      "data": "<base64>"
    }
  ],
  "provider": "google",
  "background_prompt_id": 3
}
```

风格转换示例：

```json
{
  "foreground": "<base64>",
  "provider": "google",
  "style_prompt": "将图片转换为黑白线稿风格"
}
```

### 7.4 前端交互建议

推荐改写页布局：

1. 模型选择
2. 前景图上传
3. 背景图上传
4. 背景图模板选择
5. AI 生成默认提示词按钮
6. 风格转换提示词输入框
6. 参考图上传
7. 提交生成

推荐交互规则：

- 未上传背景图时，背景模板选择器可禁用
- 上传了背景图时，背景模板选择器必须必选
- 上传了背景图时，隐藏或禁用 `style_prompt`
- 未上传背景图时，可显示 `style_prompt`
- 切换 `provider` 时不要清空模板选择
- 但要提示用户：同一个模板在不同模型下使用的字段不同
- 预览模板时同时展示 Gemini 与 Wan 两套文案，避免用户误解

## 8. 推荐页面设计

### 7.1 背景模板管理页

建议包含：

- 模板列表
- 搜索框
- 新建按钮
- 编辑弹窗
- 删除按钮
- 模型文案双栏展示

推荐列表列：

- 模板名称
- Gemini 正向提示词摘要
- Wan 正向提示词摘要
- 更新时间
- 操作

### 7.2 图像改写页

建议包含：

- 背景模板下拉选择框
- AI 生成默认提示词按钮
- 模板内容预览面板
- 当前模型提示
- 风格转换提示词输入框
- 生成结果区

页面建议拆成两种状态：

- 背景融合状态：显示背景图上传与模板选择，不显示 `style_prompt`
- 风格转换状态：不显示背景模板选择，可显示 `style_prompt`

模板预览建议：

- 左侧显示 Gemini 文案
- 右侧显示 Wan 文案
- 当前选中模型对应区域高亮

## 9. 错误处理建议

重点处理以下错误消息：

- `背景图提示词模板不存在`
- `提供背景图时必须同时提供 background_prompt_id`
- `未提供背景图时不能传 background_prompt_id`
- `背景融合场景不支持 style_prompt`
- `不支持的模型提供者，仅支持 google 或 wan`
- `请求参数解析失败`
- `模型融合失败`

前端建议策略：

- 参数错误：表单内提示
- 资源不存在：Toast + 刷新模板列表
- 模型调用失败：Toast + 保留用户表单输入，支持重试

## 10. 调用顺序建议

推荐初始化流程：

1. 页面加载后拉取模板列表
2. 用户选择模型
3. 用户选择模式
4. 若为背景融合：上传背景图并选择背景图模板
5. 若为风格转换：输入 `style_prompt`
6. 调用 `POST /api/v1/images/rewrite`
7. 成功后展示生成图

## 11. TypeScript 建议定义

```ts
export type ApiResponse<T> = {
  code: number;
  message: string;
  data?: T;
};

export type BackgroundPrompt = {
  id: number;
  name: string;
  gemini_prompt: string;
  gemini_negative_prompt: string;
  wan_prompt: string;
  wan_negative_prompt: string;
  created_at: string;
  updated_at: string;
};

export type LLMPromptTemplate = {
  id: number;
  name: string;
  template_key: string;
  content: string;
  description: string;
  created_at: string;
  updated_at: string;
};

export type BackgroundPromptListData = {
  items: BackgroundPrompt[];
  total: number;
  limit: number;
  offset: number;
};

export type RewriteResponseData = {
  success: boolean;
  id: number;
  asset_id: string;
  image_url: string;
  message: string;
  error?: string;
};
```

## 12. 前端实现建议

推荐拆分为以下模块：

- `backgroundPromptService`
  - `listBackgroundPrompts`
  - `getBackgroundPrompt`
  - `createBackgroundPrompt`
  - `updateBackgroundPrompt`
  - `deleteBackgroundPrompt`
  - `suggestBackgroundPromptDefaults`
  - `syncBackgroundPromptEnglish`
- `llmPromptTemplateService`
  - `listLLMPromptTemplates`
  - `getLLMPromptTemplate`
  - `createLLMPromptTemplate`
  - `updateLLMPromptTemplate`
  - `deleteLLMPromptTemplate`
- `imageRewriteService`
  - `rewriteImage`
- `useBackgroundPromptOptions`
  - 管理模板列表与选择状态

## 13. 联调检查清单

联调时请确认：

- 能正常获取模板列表
- 新建、编辑、删除模板都生效
- 背景图传入 `suggest-defaults` 后，能返回 Gemini/Wan 双语默认建议
- 中文字段修改后，`sync-english` 能返回对应英文更新
- `llm-prompt-templates` 能正常 CRUD
- `google` 改写时能读取 Gemini 模板字段
- `wan` 改写时能读取 Wan 模板字段
- 背景融合时不传 `background_prompt_id` 会被拦截
- 传模板但不传背景图时能收到明确报错
- 背景融合时传 `style_prompt` 会被拦截
- 风格转换时可正常使用 `style_prompt`
- 模板不存在时前端能正确提示

## 14. 模型配置

后端配置文件 `configs/config.yaml` 新增：

```yaml
models:
  gemini: gemini-1.5-flash
  wan: wanx-plus
  qwen: qwen3.6-plus
```

说明：

- `models.gemini` 控制 Gemini 图像生成模型名称
- `models.wan` 控制 Wan 图像生成模型名称
- `models.qwen` 控制 Qwen 视觉理解/英文同步模型名称
- 也可以用环境变量覆盖：
  - `GYRH_MODEL_GEMINI`
  - `GYRH_MODEL_WAN`
  - `GYRH_MODEL_QWEN`
