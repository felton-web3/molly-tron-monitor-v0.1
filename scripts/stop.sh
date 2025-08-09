#!/bin/bash

# Tron区块链监控系统停止脚本

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

# 停止应用程序
stop_app() {
    print_info "停止Tron区块链监控系统..."
    
    # 查找并停止应用程序进程
    if pgrep -f "tron-monitor" > /dev/null; then
        print_info "找到运行中的应用程序进程，正在停止..."
        pkill -f "tron-monitor"
        sleep 2
        
        # 检查是否还有进程在运行
        if pgrep -f "tron-monitor" > /dev/null; then
            print_warn "应用程序仍在运行，强制停止..."
            pkill -9 -f "tron-monitor"
        fi
        
        print_info "应用程序已停止"
    else
        print_info "未找到运行中的应用程序进程"
    fi
}

# 停止Redis（可选）
stop_redis() {
    print_info "检查Redis服务..."
    
    if pgrep -x "redis-server" > /dev/null; then
        read -p "是否停止Redis服务？(y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_info "停止Redis服务..."
            redis-cli shutdown
            print_info "Redis服务已停止"
        else
            print_info "保持Redis服务运行"
        fi
    else
        print_info "Redis服务未运行"
    fi
}

# 清理临时文件
cleanup() {
    print_info "清理临时文件..."
    
    # 清理日志文件
    if [ -d "logs" ]; then
        rm -rf logs/*
        print_info "日志文件已清理"
    fi
    
    # 清理构建文件（可选）
    read -p "是否清理构建文件？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "清理构建文件..."
        make clean
        print_info "构建文件已清理"
    fi
}

# 主函数
main() {
    print_info "Tron区块链监控系统停止脚本"
    print_info "================================"
    
    # 停止应用程序
    stop_app
    
    # 停止Redis（可选）
    stop_redis
    
    # 清理临时文件
    cleanup
    
    print_info "停止脚本执行完成"
}

# 运行主函数
main "$@"
