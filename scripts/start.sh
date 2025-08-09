#!/bin/bash

# Tron区块链监控系统启动脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    print_info "检查依赖..."
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        print_error "Go未安装，请先安装Go 1.21+"
        exit 1
    fi
    
    # 检查Redis
    if ! command -v redis-server &> /dev/null; then
        print_warn "Redis未安装，请先安装Redis"
        print_info "可以使用Docker运行Redis: docker run -d -p 6379:6379 redis:7-alpine"
    fi
    
    print_info "依赖检查完成"
}

# 构建应用程序
build_app() {
    print_info "构建应用程序..."
    
    if [ ! -f "build/tron-monitor" ]; then
        make build
    else
        print_info "应用程序已构建"
    fi
}

# 启动Redis
start_redis() {
    print_info "检查Redis服务..."
    
    if ! pgrep -x "redis-server" > /dev/null; then
        print_warn "Redis服务未运行，尝试启动..."
        
        if command -v redis-server &> /dev/null; then
            redis-server --daemonize yes
            sleep 2
            print_info "Redis服务已启动"
        else
            print_error "无法启动Redis服务，请手动启动"
            exit 1
        fi
    else
        print_info "Redis服务正在运行"
    fi
}

# 启动应用程序
start_app() {
    print_info "启动Tron区块链监控系统..."
    
    # 检查配置文件
    if [ ! -f "config.yaml" ]; then
        print_error "配置文件config.yaml不存在"
        exit 1
    fi
    
    # 启动应用程序
    ./build/tron-monitor config.yaml
}

# 主函数
main() {
    print_info "Tron区块链监控系统启动脚本"
    print_info "================================"
    
    # 检查依赖
    check_dependencies
    
    # 构建应用程序
    build_app
    
    # 启动Redis
    start_redis
    
    # 启动应用程序
    start_app
}

# 运行主函数
main "$@"
