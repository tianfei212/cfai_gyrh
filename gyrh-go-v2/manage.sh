#!/bin/bash

# ==============================================================================
# 管理脚本：前后端自动化启动、停止、重启、状态查询
# 描述：如果程序存在或端口被占用就杀掉旧的程序释放端口，然后重启。
# ==============================================================================

# 配置信息
FRONTEND_PORT=9912
BACKEND_PORT=9913
ALIOSS_BACKGROUND_PORT=18080
ALIOSS_GENERATED_PORT=18081
BASE_DIR=$(pwd)
BACKEND_DIR="backend"
FRONTEND_DIR="frontend"
BACKEND_CMD="go run cmd/server/main.go"
FRONTEND_CMD="npm run dev"

# 颜色配置
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # 无颜色

# 打印带颜色的信息
function print_info() {
    echo -e "${GREEN}[INFO] $1${NC}"
}

function print_warn() {
    echo -e "${YELLOW}[WARN] $1${NC}"
}

function print_err() {
    echo -e "${RED}[ERROR] $1${NC}"
}

# 加载本地环境变量，供后端直连 302 GPT 等服务使用。
function load_env() {
    local env_file="$BASE_DIR/.env.local"
    if [ -f "$env_file" ]; then
        set -a
        source "$env_file"
        set +a
        print_info "已加载 .env.local"
    fi
}

# 检查端口占用并返回 PID 列表
function get_pids_by_port() {
    local port=$1
    lsof -t -i:$port
}

# 根据端口杀掉占用进程
function kill_by_port() {
    local port=$1
    local service_name=$2
    local pids=$(get_pids_by_port $port)
    
    if [ ! -z "$pids" ]; then
        print_warn "发现 $service_name 端口 ($port) 被占用，正在杀掉旧进程..."
        for pid in $pids; do
            print_warn "正在结束进程 PID: $pid"
            kill -9 $pid 2>/dev/null
        done
        print_info "$service_name 端口 ($port) 已释放"
    else
        print_info "$service_name 端口 ($port) 空闲"
    fi
}

# 启动服务
function start() {
    load_env

    # 启动前检查端口并清理
    kill_by_port $FRONTEND_PORT "前端"
    kill_by_port $BACKEND_PORT "后端"
    kill_by_port $ALIOSS_BACKGROUND_PORT "aliOSS 背景素材"
    kill_by_port $ALIOSS_GENERATED_PORT "aliOSS 生成素材"
    
    # 稍微等待端口完全释放
    sleep 1

    # 启动后端
    print_info "正在启动后端服务 (端口: $BACKEND_PORT)..."
    cd "$BASE_DIR/$BACKEND_DIR" || exit 1
    nohup $BACKEND_CMD > ../backend.log 2>&1 &
    local backend_pid=$!
    print_info "后端启动命令已发出，PID: $backend_pid。日志已重定向到 backend.log"

    # 启动前端
    print_info "正在启动前端服务 (端口: $FRONTEND_PORT)..."
    cd "$BASE_DIR/$FRONTEND_DIR" || exit 1
    nohup $FRONTEND_CMD > ../frontend.log 2>&1 &
    local frontend_pid=$!
    print_info "前端启动命令已发出，PID: $frontend_pid。日志已重定向到 frontend.log"

    cd "$BASE_DIR" || exit 1
    
    # 给服务一点时间启动再检查状态
    sleep 2
    print_info "服务启动命令已完成！"
    check_status
}

# 停止服务
function stop() {
    print_info "正在停止所有服务..."
    kill_by_port $FRONTEND_PORT "前端"
    kill_by_port $BACKEND_PORT "后端"
    kill_by_port $ALIOSS_BACKGROUND_PORT "aliOSS 背景素材"
    kill_by_port $ALIOSS_GENERATED_PORT "aliOSS 生成素材"
    print_info "服务已全部停止。"
}

# 查看服务状态
function check_status() {
    echo -e "\n==================== 服务状态 ===================="
    local backend_pids=$(get_pids_by_port $BACKEND_PORT)
    local frontend_pids=$(get_pids_by_port $FRONTEND_PORT)
    local alioss_background_pids=$(get_pids_by_port $ALIOSS_BACKGROUND_PORT)
    local alioss_generated_pids=$(get_pids_by_port $ALIOSS_GENERATED_PORT)
    
    if [ ! -z "$backend_pids" ]; then
        local pid_list=$(echo $backend_pids | tr '\n' ' ')
        print_info "后端 (端口 $BACKEND_PORT): 运行中 (PID: $pid_list)"
    else
        print_err "后端 (端口 $BACKEND_PORT): 已停止"
    fi

    if [ ! -z "$alioss_background_pids" ]; then
        local pid_list=$(echo $alioss_background_pids | tr '\n' ' ')
        print_info "aliOSS 背景素材 (端口 $ALIOSS_BACKGROUND_PORT): 运行中 (PID: $pid_list)"
    else
        print_err "aliOSS 背景素材 (端口 $ALIOSS_BACKGROUND_PORT): 已停止"
    fi

    if [ ! -z "$alioss_generated_pids" ]; then
        local pid_list=$(echo $alioss_generated_pids | tr '\n' ' ')
        print_info "aliOSS 生成素材 (端口 $ALIOSS_GENERATED_PORT): 运行中 (PID: $pid_list)"
    else
        print_err "aliOSS 生成素材 (端口 $ALIOSS_GENERATED_PORT): 已停止"
    fi
    
    if [ ! -z "$frontend_pids" ]; then
        local pid_list=$(echo $frontend_pids | tr '\n' ' ')
        print_info "前端 (端口 $FRONTEND_PORT): 运行中 (PID: $pid_list)"
    else
        print_err "前端 (端口 $FRONTEND_PORT): 已停止"
    fi
    echo -e "=================================================\n"
}

# 解析命令行参数
case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        print_info "正在重启服务..."
        stop
        sleep 2
        start
        ;;
    status)
        check_status
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status}"
        echo "  start   - 自动检测并杀掉冲突进程，然后启动前后端服务"
        echo "  stop    - 停止所有前后端服务"
        echo "  restart - 停止然后重新启动服务"
        echo "  status  - 查看服务运行状态"
        exit 1
        ;;
esac

exit 0