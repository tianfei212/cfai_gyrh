# Branding Assets

把展厅定制资源放在这个目录，Vite 会以 `/branding/...` 路径公开访问。

示例：

- `logo.png`
- `background.jpg`
- `backgrounds/home.jpg`

然后在 `brand-config.json` 中配置：

```json
{
  "appName": "展厅光影系统",
  "productName": "AI Portrait",
  "logo": "logo.png",
  "background": "background.jpg",
  "previewWatermark": {
    "brand": "EXPO",
    "product": "AI PORTRAIT"
  }
}
```

`logo` 和 `background` 可以写相对路径，也可以写以 `/` 开头的站内绝对路径或 `https://` 远程地址。
