# GYRH macOS 测试包

此包用于本机 macOS 测试，后端单文件位于 `bin/gyrh-server`，前端已嵌入二进制。

## 启动

```bash
cp .env.local.example .env.local
# 按需填写 .env.local 后启动
./manage.sh start
```

## 访问

- 演示端: http://127.0.0.1:9913/
- 管理端: http://127.0.0.1:9913/admin_viewer
