#!/usr/bin/env bash
set -euo pipefail

BACKEND_PORT="${BACKEND_PORT:-9913}"
OSS_BACKGROUND_PORT="${OSS_BACKGROUND_PORT:-18080}"
OSS_GENERATED_PORT="${OSS_GENERATED_PORT:-18081}"
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$BASE_DIR/bin"
LOG_DIR="$BASE_DIR/logs"
RUNTIME_DIR="$BASE_DIR/runtime"
BACKEND_BIN="$BIN_DIR/gyrh-server"
OSS_CLI_BIN="$BIN_DIR/oss-cli"
BACKEND_LOG="$LOG_DIR/backend.log"
OSS_BACKGROUND_LOG="$LOG_DIR/oss-background.log"
OSS_GENERATED_LOG="$LOG_DIR/oss-generated.log"
BACKEND_PID_FILE="$RUNTIME_DIR/backend.pid"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO] $1${NC}"; }
print_warn() { echo -e "${YELLOW}[WARN] $1${NC}"; }
print_err() { echo -e "${RED}[ERROR] $1${NC}"; }

ensure_dirs() {
  mkdir -p "$LOG_DIR" "$RUNTIME_DIR"
}

load_env() {
  local env_file="$BASE_DIR/.env.local"
  if [ -f "$env_file" ]; then
    set -a
    # shellcheck disable=SC1090
    source "$env_file"
    set +a
    print_info "已加载 .env.local"
  else
    print_warn "未找到 .env.local；将仅使用系统环境变量和 configs/config.yaml"
  fi
}

env_value() {
  local key="$1"
  local env_file="$BASE_DIR/.env.local"
  if [ ! -f "$env_file" ]; then
    return 0
  fi
  awk -F= -v key="$key" '$1 == key { sub(/^[^=]*=/, ""); gsub(/^"|"$/, ""); print; exit }' "$env_file"
}

get_pids_by_port() {
  local port="$1"
  lsof -t -i:"$port" 2>/dev/null || true
}

kill_by_port() {
  local port="$1"
  local service_name="$2"
  local pids
  pids="$(get_pids_by_port "$port")"
  if [ -n "$pids" ]; then
    print_warn "发现 $service_name 端口 ($port) 被占用，正在杀掉旧进程..."
    for pid in $pids; do
      print_warn "正在结束进程 PID: $pid"
      kill "$pid" 2>/dev/null || true
    done
    sleep 1
    print_info "$service_name 端口 ($port) 已释放"
  else
    print_info "$service_name 端口 ($port) 空闲"
  fi
}

write_pid() {
  local pid_file="$1"
  local pid="$2"
  echo "$pid" > "$pid_file"
}

start_backend() {
  if [ ! -x "$BACKEND_BIN" ]; then
    print_err "未找到可执行后端文件: $BACKEND_BIN"
    exit 1
  fi
  print_info "正在启动后端服务 (端口: $BACKEND_PORT)..."
  (
    cd "$BASE_DIR"
    nohup "$BACKEND_BIN" >> "$BACKEND_LOG" 2>&1 &
    write_pid "$BACKEND_PID_FILE" "$!"
  )
  print_info "后端启动命令已发出，PID: $(cat "$BACKEND_PID_FILE")，日志: $BACKEND_LOG"
}

start_oss_if_present() {
  if [ ! -x "$OSS_CLI_BIN" ]; then
    print_warn "未找到 OSS 可执行文件: ${OSS_CLI_BIN}，跳过 OSS 服务"
    return 0
  fi
  if [ -f "$BASE_DIR/configs/alioss-agent.yaml" ]; then
    print_info "正在启动 aliOSS 背景素材服务 (端口: $OSS_BACKGROUND_PORT)..."
    nohup "$OSS_CLI_BIN" server --config "$BASE_DIR/configs/alioss-agent.yaml" >> "$OSS_BACKGROUND_LOG" 2>&1 &
    print_info "aliOSS 背景素材启动命令已发出，PID: $!，日志: $OSS_BACKGROUND_LOG"
  fi
  if [ -f "$BASE_DIR/configs/alioss-agent-generated.yaml" ]; then
    print_info "正在启动 aliOSS 生成素材服务 (端口: $OSS_GENERATED_PORT)..."
    nohup "$OSS_CLI_BIN" server --config "$BASE_DIR/configs/alioss-agent-generated.yaml" >> "$OSS_GENERATED_LOG" 2>&1 &
    print_info "aliOSS 生成素材启动命令已发出，PID: $!，日志: $OSS_GENERATED_LOG"
  fi
}

start() {
  ensure_dirs
  load_env
  kill_by_port "$BACKEND_PORT" "后端"
  kill_by_port "$OSS_BACKGROUND_PORT" "aliOSS 背景素材"
  kill_by_port "$OSS_GENERATED_PORT" "aliOSS 生成素材"
  sleep 1
  start_backend
  start_oss_if_present
  sleep 2
  print_info "服务启动命令已完成；前端由后端 ${BACKEND_PORT} 端口提供"
  status
  print_access_info
}

stop() {
  print_info "正在停止所有服务..."
  kill_by_port "$BACKEND_PORT" "后端"
  kill_by_port "$OSS_BACKGROUND_PORT" "aliOSS 背景素材"
  kill_by_port "$OSS_GENERATED_PORT" "aliOSS 生成素材"
  rm -f "$BACKEND_PID_FILE"
  print_info "服务已全部停止。"
}

status() {
  echo -e "\n==================== 服务状态 ===================="
  local backend_pids
  local oss_background_pids
  local oss_generated_pids
  backend_pids="$(get_pids_by_port "$BACKEND_PORT")"
  oss_background_pids="$(get_pids_by_port "$OSS_BACKGROUND_PORT")"
  oss_generated_pids="$(get_pids_by_port "$OSS_GENERATED_PORT")"

  if [ -n "$backend_pids" ]; then
    print_info "后端 (端口 $BACKEND_PORT): 运行中 (PID: $(echo "$backend_pids" | tr '\n' ' '))"
    print_info "前端: 已内嵌在后端，通过 http://127.0.0.1:$BACKEND_PORT/ 访问"
  else
    print_err "后端 (端口 $BACKEND_PORT): 已停止"
  fi

  if [ -n "$oss_background_pids" ]; then
    print_info "aliOSS 背景素材 (端口 $OSS_BACKGROUND_PORT): 运行中 (PID: $(echo "$oss_background_pids" | tr '\n' ' '))"
  else
    print_err "aliOSS 背景素材 (端口 $OSS_BACKGROUND_PORT): 已停止"
  fi

  if [ -n "$oss_generated_pids" ]; then
    print_info "aliOSS 生成素材 (端口 $OSS_GENERATED_PORT): 运行中 (PID: $(echo "$oss_generated_pids" | tr '\n' ' '))"
  else
    print_err "aliOSS 生成素材 (端口 $OSS_GENERATED_PORT): 已停止"
  fi

  print_info "日志目录: $LOG_DIR"
  echo -e "=================================================\n"
}

print_access_info() {
  echo -e "\n==================== 访问信息 ===================="
  print_info "演示端: http://127.0.0.1:${BACKEND_PORT}/"
  print_info "管理端: http://127.0.0.1:${BACKEND_PORT}/admin_viewer"
  print_info "登录页: http://127.0.0.1:${BACKEND_PORT}/login"
  print_info "后端/API 端口: ${BACKEND_PORT}"
  print_info "OSS 背景素材端口: ${OSS_BACKGROUND_PORT}"
  print_info "OSS 生成素材端口: ${OSS_GENERATED_PORT}"

  local admin_user
  local admin_password
  local pshow_user
  local pshow_password
  admin_user="$(env_value GYRH_FRONTEND_AUTH_ADMIN_USERNAME)"
  admin_password="$(env_value GYRH_FRONTEND_AUTH_ADMIN_PASSWORD)"
  pshow_user="$(env_value GYRH_FRONTEND_AUTH_PSHOW_USERNAME)"
  pshow_password="$(env_value GYRH_FRONTEND_AUTH_PSHOW_PASSWORD)"

  if [ -n "$admin_user" ] || [ -n "$admin_password" ] || [ -n "$pshow_user" ] || [ -n "$pshow_password" ]; then
    print_info "admin 用户名: ${admin_user:-未配置}"
    print_info "admin 密码: ${admin_password:-未配置}"
    print_info "pshow 用户名: ${pshow_user:-未配置}"
    print_info "pshow 密码: ${pshow_password:-未配置}"
  else
    print_warn "未在 $BASE_DIR/.env.local 中找到前端登录用户名和密码"
  fi
  echo -e "=================================================\n"
}

show_login() {
  print_access_info
}

show_logs() {
  ensure_dirs
  touch "$BACKEND_LOG"
  print_info "查看后端启动日志: $BACKEND_LOG"
  tail -f "$BACKEND_LOG"
}

case "${1:-}" in
  start) start ;;
  stop) stop ;;
  restart)
    print_info "正在重启服务..."
    stop
    sleep 2
    start
    ;;
  status) status ;;
  login) show_login ;;
  logs) show_logs ;;
  *)
    echo "用法: $0 {start|stop|restart|status|logs|login}"
    echo "  start   - 自动检测并杀掉冲突进程，然后启动发布包服务"
    echo "  stop    - 停止发布包服务"
    echo "  restart - 停止然后重新启动服务"
    echo "  status  - 查看服务运行状态"
    echo "  logs    - 查看后端启动日志"
    echo "  login   - 打印访问地址和登录账号"
    exit 1
    ;;
esac
