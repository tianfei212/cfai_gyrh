#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_ROOT="$ROOT_DIR/release"
APP_NAME="gyrh-go-v2"
VERSION="$(date +%Y%m%d%H%M)"
GIT_SHA="$(git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || echo local)"
PACKAGE_NAME="${APP_NAME}-${VERSION}-${GIT_SHA}-ubuntu-amd64"
STAGE_DIR="$RELEASE_ROOT/$PACKAGE_NAME"
ARCHIVE_PATH="$RELEASE_ROOT/${PACKAGE_NAME}.tar.gz"

info() { printf '\033[0;32m[INFO]\033[0m %s\n' "$*"; }
warn() { printf '\033[0;33m[WARN]\033[0m %s\n' "$*"; }
fail() { printf '\033[0;31m[ERROR]\033[0m %s\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令: $1"
}

write_env_example() {
  local target="$1"
  cat > "$target" <<'ENVEOF'
# ==============================================================================
# GYRH 部署环境变量模板
# 说明：真实密钥必须在新机器上手动填写，构建脚本不会打包本地 .env.local。
# ==============================================================================

GYRH_AUTH_PUBLIC_KEY=
GYRH_AUTH_PRIVATE_KEY=

GEMINI_API_KEY=
WAN_API_KEY=
DASHSCOPE_API_KEY=
PROVIDER_302_API_KEY=

# 如需按 public key 分配不同私钥，可使用：
# GYRH_AUTH_PRIVATE_KEY_YOUR_PUBLIC_KEY=
ENVEOF
}

write_manage_script() {
  local target="$1"
  cat > "$target" <<'MANAGEEOF'
#!/usr/bin/env bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_PORT="${BACKEND_PORT:-9913}"
PID_FILE="$APP_DIR/runtime/gyrh.pid"
LOG_FILE="$APP_DIR/logs/gyrh.log"
BIN="$APP_DIR/bin/gyrh-server"

info() { printf '\033[0;32m[INFO]\033[0m %s\n' "$*"; }
warn() { printf '\033[0;33m[WARN]\033[0m %s\n' "$*"; }
err() { printf '\033[0;31m[ERROR]\033[0m %s\n' "$*" >&2; }

load_env() {
  if [ -f "$APP_DIR/.env.local" ]; then
    set -a
    # shellcheck disable=SC1091
    source "$APP_DIR/.env.local"
    set +a
  else
    warn "未找到 .env.local，请根据 .env.local.example 手动填写真实密钥"
  fi
}

is_running() {
  [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" >/dev/null 2>&1
}

ensure_binary_executable() {
  if [ ! -f "$BIN" ]; then
    err "未找到服务二进制: $BIN"
    exit 1
  fi
  if [ ! -x "$BIN" ]; then
    warn "服务二进制缺少执行权限，自动执行 chmod +x $BIN"
    chmod +x "$BIN" || {
      err "无法给服务二进制增加执行权限: $BIN"
      exit 1
    }
  fi
}

start() {
  mkdir -p "$APP_DIR/runtime" "$APP_DIR/logs"
  if is_running; then
    info "GYRH 已运行，PID=$(cat "$PID_FILE")"
    return 0
  fi
  ensure_binary_executable
  load_env
  cd "$APP_DIR"
  nohup "$BIN" >> "$LOG_FILE" 2>&1 &
  echo $! > "$PID_FILE"
  sleep 1
  if is_running; then
    info "GYRH 启动成功，PID=$(cat "$PID_FILE")，日志=$LOG_FILE"
  else
    err "GYRH 启动失败，请查看 $LOG_FILE"
    exit 1
  fi
}

stop() {
  if ! is_running; then
    warn "GYRH 未运行"
    rm -f "$PID_FILE"
    return 0
  fi
  local pid
  pid="$(cat "$PID_FILE")"
  kill "$pid" >/dev/null 2>&1 || true
  for _ in $(seq 1 20); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      rm -f "$PID_FILE"
      info "GYRH 已停止"
      return 0
    fi
    sleep 0.5
  done
  warn "进程未正常退出，强制停止 PID=$pid"
  kill -9 "$pid" >/dev/null 2>&1 || true
  rm -f "$PID_FILE"
}

status() {
  if is_running; then
    info "GYRH 运行中，PID=$(cat "$PID_FILE")，后端端口=$BACKEND_PORT"
  else
    err "GYRH 未运行"
    exit 1
  fi
}

logs() {
  mkdir -p "$APP_DIR/logs"
  touch "$LOG_FILE"
  tail -f "$LOG_FILE"
}

case "${1:-}" in
  start) start ;;
  stop) stop ;;
  restart) stop; start ;;
  status) status ;;
  logs) logs ;;
  *) echo "用法: $0 {start|stop|restart|status|logs}"; exit 1 ;;
esac
MANAGEEOF
  chmod +x "$target"
}

write_service_script() {
  local target="$1"
  cat > "$target" <<'SERVICEEOF'
#!/usr/bin/env bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVICE_NAME="${SERVICE_NAME:-gyrh}"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
RUN_USER="${RUN_USER:-$(id -un)}"

if [ "$(id -u)" -ne 0 ]; then
  echo "[ERROR] 请使用 sudo 运行: sudo $0" >&2
  exit 1
fi

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=GYRH Exhibition Backend Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$RUN_USER
WorkingDirectory=$APP_DIR
EnvironmentFile=-$APP_DIR/.env.local
ExecStart=$APP_DIR/bin/gyrh-server
Restart=always
RestartSec=3
StandardOutput=append:$APP_DIR/logs/gyrh-systemd.log
StandardError=append:$APP_DIR/logs/gyrh-systemd.log

[Install]
WantedBy=multi-user.target
EOF

mkdir -p "$APP_DIR/logs"
chown -R "$RUN_USER":"$RUN_USER" "$APP_DIR"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"
systemctl status "$SERVICE_NAME" --no-pager
SERVICEEOF
  chmod +x "$target"
}

write_nginx_script() {
  local target="$1"
  cat > "$target" <<'NGINXEOF'
#!/usr/bin/env bash
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER_NAME="${SERVER_NAME:-_}"
BACKEND_PORT="${BACKEND_PORT:-9913}"
SSL_CERT="${SSL_CERT:-/etc/ssl/certs/gyrh-selfsigned.crt}"
SSL_KEY="${SSL_KEY:-/etc/ssl/private/gyrh-selfsigned.key}"
SITE_FILE="/etc/nginx/sites-available/gyrh"

if [ "$(id -u)" -ne 0 ]; then
  echo "[ERROR] 请使用 sudo 运行: sudo $0" >&2
  exit 1
fi

if [ -f /etc/os-release ]; then
  # shellcheck disable=SC1091
  source /etc/os-release
  if [ "${ID:-}" != "ubuntu" ]; then
    echo "[ERROR] 当前脚本只支持 Ubuntu，检测到: ${PRETTY_NAME:-unknown}" >&2
    exit 1
  fi
else
  echo "[ERROR] 无法识别操作系统" >&2
  exit 1
fi

apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y nginx openssl

if [ ! -f "$SSL_CERT" ] || [ ! -f "$SSL_KEY" ]; then
  mkdir -p "$(dirname "$SSL_CERT")" "$(dirname "$SSL_KEY")"
  openssl req -x509 -nodes -newkey rsa:2048 -days 3650 \
    -keyout "$SSL_KEY" \
    -out "$SSL_CERT" \
    -subj "/CN=gyrh-local"
  chmod 600 "$SSL_KEY"
fi

cat > "$SITE_FILE" <<EOF
server {
    listen 80;
    server_name $SERVER_NAME;
    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $SERVER_NAME;

    ssl_certificate $SSL_CERT;
    ssl_certificate_key $SSL_KEY;

    client_max_body_size 80m;

    location / {
        proxy_pass http://127.0.0.1:$BACKEND_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }
}
EOF

ln -sf "$SITE_FILE" /etc/nginx/sites-enabled/gyrh
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl enable nginx
systemctl reload nginx || systemctl restart nginx
echo "[INFO] Nginx 已配置完成，对外服务端口: 443，所有请求反向代理到 127.0.0.1:$BACKEND_PORT"
NGINXEOF
  chmod +x "$target"
}

write_install_script() {
  local target="$1"
  cat > "$target" <<'INSTALLEOF'
#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${INSTALL_DIR:-/opt/gyrh}"

info() { printf '\033[0;32m[INFO]\033[0m %s\n' "$*"; }
fail() { printf '\033[0;31m[ERROR]\033[0m %s\n' "$*" >&2; exit 1; }

if [ -f /etc/os-release ]; then
  # shellcheck disable=SC1091
  source /etc/os-release
  [ "${ID:-}" = "ubuntu" ] || fail "当前安装脚本只支持 Ubuntu，检测到: ${PRETTY_NAME:-unknown}"
else
  fail "无法识别操作系统"
fi

mkdir -p "$INSTALL_DIR"
cp -a "$SOURCE_DIR/." "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/manage.sh" "$INSTALL_DIR/scripts/install_service.sh" "$INSTALL_DIR/scripts/install_nginx.sh"

if [ ! -f "$INSTALL_DIR/.env.local" ]; then
  cp "$INSTALL_DIR/.env.local.example" "$INSTALL_DIR/.env.local"
  info "已生成 $INSTALL_DIR/.env.local，请手动填写真实密钥后再启动服务"
fi

info "部署文件已释放到 $INSTALL_DIR"
info "下一步："
info "1. 编辑 $INSTALL_DIR/.env.local 填写真实密钥"
info "2. sudo $INSTALL_DIR/scripts/install_service.sh"
info "3. sudo $INSTALL_DIR/scripts/install_nginx.sh"
INSTALLEOF
  chmod +x "$target"
}

write_verify_script() {
  local target="$1"
  cat > "$target" <<'VERIFYEOF'
#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:9913}"
PUBLIC_URL="${PUBLIC_URL:-}"

check_url() {
  local base="$1"
  local path="$2"
  local url="${base%/}${path}"
  local code
  code="$(curl -k -sS -o /dev/null -w '%{http_code}' "$url")"
  if [ "$code" != "200" ]; then
    echo "[ERROR] $url => HTTP $code" >&2
    return 1
  fi
  echo "[OK] $url"
}

check_bundle() {
  local base="$1"
  local index
  index="$(curl -k -sS "${base%/}/")"
  local asset
  asset="$(printf '%s' "$index" | sed -n 's/.*src="\(\/assets\/index-[^"]*\.js\)".*/\1/p' | head -n 1)"
  if [ -z "$asset" ]; then
    echo "[ERROR] ${base%/}/ 未返回新前端 index.html 或缺少 JS 入口" >&2
    return 1
  fi
  check_url "$base" "$asset"
}

paths=(
  "/admin_viewer"
  "/models/selfie_segmentation/selfie_segmentation.binarypb"
  "/models/selfie_segmentation/selfie_segmentation.js"
  "/models/selfie_segmentation/selfie_segmentation_solution_simd_wasm_bin.js"
  "/models/selfie_segmentation/selfie_segmentation_solution_wasm_bin.js"
  "/models/selfie_segmentation/selfie_segmentation_solution_simd_wasm_bin.wasm"
  "/models/selfie_segmentation/selfie_segmentation_solution_wasm_bin.wasm"
  "/models/selfie_segmentation/selfie_segmentation_solution_simd_wasm_bin.data"
  "/models/selfie_segmentation/selfie_segmentation.tflite"
  "/models/selfie_segmentation/selfie_segmentation_landscape.tflite"
)

echo "[INFO] 检查后端单二进制: $BACKEND_URL"
check_bundle "$BACKEND_URL"
for path in "${paths[@]}"; do
  check_url "$BACKEND_URL" "$path"
done

if [ -n "$PUBLIC_URL" ]; then
  echo "[INFO] 检查公网/Nginx: $PUBLIC_URL"
  check_bundle "$PUBLIC_URL"
  for path in "${paths[@]}"; do
    check_url "$PUBLIC_URL" "$path"
  done
fi

echo "[INFO] 发布自检通过"
VERIFYEOF
  chmod +x "$target"
}

require_cmd npm
require_cmd go
require_cmd tar
require_cmd curl

build_backend_linux_amd64() {
  local output="$1"
  if [ "$(go env GOOS)" = "linux" ]; then
    info "使用 Go 构建 Ubuntu amd64 后端单文件二进制"
    (cd "$ROOT_DIR/backend" && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o "$output" ./cmd/server)
    return
  fi

  if command -v x86_64-linux-gnu-gcc >/dev/null 2>&1; then
    info "使用 Go 交叉编译 Ubuntu amd64 后端单文件二进制: CC=x86_64-linux-gnu-gcc"
    (cd "$ROOT_DIR/backend" && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-gnu-gcc go build -o "$output" ./cmd/server)
    return
  fi

  if command -v zig >/dev/null 2>&1; then
    info "使用 Go + zig cc 交叉编译 Ubuntu amd64 后端单文件二进制"
    (cd "$ROOT_DIR/backend" && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux-gnu" go build -o "$output" ./cmd/server)
    return
  fi

  fail "Go 支持跨平台编译，但当前项目使用 mattn/go-sqlite3(cgo)，还需要 Linux amd64 C 交叉编译器。请安装 x86_64-linux-gnu-gcc 或 zig 后重试。"
}

copy_config_files() {
  mkdir -p "$STAGE_DIR/configs"
  for item in "$ROOT_DIR"/configs/*; do
    [ -e "$item" ] || continue
    if [ "$(basename "$item")" = "302Helpper_config.yaml" ]; then
      continue
    fi
    cp -a "$item" "$STAGE_DIR/configs/"
  done
}

info "清理旧 release 目录"
rm -rf "$STAGE_DIR"
mkdir -p "$STAGE_DIR/bin" "$STAGE_DIR/backend/data" "$STAGE_DIR/configs" "$STAGE_DIR/scripts"

info "构建前端静态文件"
(cd "$ROOT_DIR/frontend" && npm run build)
rm -rf "$ROOT_DIR/backend/internal/frontend/dist"
mkdir -p "$ROOT_DIR/backend/internal/frontend"
cp -a "$ROOT_DIR/frontend/dist" "$ROOT_DIR/backend/internal/frontend/"

info "构建 Ubuntu amd64 后端二进制"
build_backend_linux_amd64 "$STAGE_DIR/bin/gyrh-server"

info "复制数据库和生成图数据"
if [ -f "$ROOT_DIR/backend/data/gyrh.db" ]; then
  cp "$ROOT_DIR/backend/data/gyrh.db" "$STAGE_DIR/backend/data/gyrh.db"
else
  warn "未找到 backend/data/gyrh.db，部署包将不包含数据库"
fi
if [ -d "$ROOT_DIR/backend/data/generated" ]; then
  cp -a "$ROOT_DIR/backend/data/generated" "$STAGE_DIR/backend/data/"
fi
if [ -f "$ROOT_DIR/bin/oss-cli-linux-amd64" ]; then
  cp "$ROOT_DIR/bin/oss-cli-linux-amd64" "$STAGE_DIR/bin/oss-cli"
  chmod +x "$STAGE_DIR/bin/oss-cli"
else
  warn "未找到 bin/oss-cli-linux-amd64，部署包将不包含 aliOSS Linux 二进制"
fi

info "复制配置文件，不打包真实 .env.local"
copy_config_files
write_env_example "$STAGE_DIR/.env.local.example"

info "生成部署管理脚本"
write_manage_script "$STAGE_DIR/manage.sh"
write_service_script "$STAGE_DIR/scripts/install_service.sh"
write_nginx_script "$STAGE_DIR/scripts/install_nginx.sh"
write_install_script "$STAGE_DIR/install_release.sh"
write_verify_script "$STAGE_DIR/scripts/verify_release.sh"

cat > "$STAGE_DIR/README_DEPLOY.md" <<'EOF'
# GYRH Ubuntu 部署包

后端以 Go 单二进制文件 `bin/gyrh-server` 运行；前端 `dist` 已在构建期内嵌到该二进制中。

## 安装

\`\`\`bash
tar -xzf <release>.tar.gz
cd <release>
sudo ./install_release.sh
sudo nano /opt/gyrh/.env.local
sudo /opt/gyrh/scripts/install_service.sh
sudo /opt/gyrh/scripts/install_nginx.sh
\`\`\`

## 访问

- 演示端：\`https://<机器IP>/\`
- 管理端：\`https://<机器IP>/admin_viewer\`

Nginx 对外监听 443，并将 `/` 全部反向代理到后端 `127.0.0.1:9913`。后端会同时处理 API 与前端静态页面。
EOF

info "生成压缩包"
mkdir -p "$RELEASE_ROOT"
tar -czf "$ARCHIVE_PATH" -C "$RELEASE_ROOT" "$PACKAGE_NAME"
info "部署包已生成: $ARCHIVE_PATH"
