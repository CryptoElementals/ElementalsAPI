#!/bin/bash

# 显示使用说明
show_usage() {
    echo "Usage: $0 <service>"
    echo "  service: room-server | api-server | scanner | all"
    echo ""
    echo "Examples:"
    echo "  $0 all                # Stop all services"
    echo "  $0 room-server        # Stop only room server"
    echo "  $0 api-server         # Stop only API server"
    echo "  $0 scanner            # Stop only scanner"
    echo "  $0 stat               # Stop only stat"
}

# 停止指定进程
stop_process() {
    local process_name="$1"
    local display_name="$2"
    
    echo "Stopping $display_name..."
    PIDS=$(ps aux | grep "$process_name" | grep -v grep | grep -v .log | grep -v stop.sh | awk '{print $2}')
    
    if [ -z "$PIDS" ]; then
        echo "No $display_name process found."
        return 0
    fi
    
    echo "Killing $display_name processes: $PIDS"
    for PID in $PIDS; do
        kill $PID
        echo "Killed $display_name process $PID"
    done
    echo "✓ All $display_name processes stopped."
}

# 停止所有服务
stop_all() {
    echo "Stopping all services..."
    stop_process "ele-roomserver" "Room Server"
    stop_process "ele-apiserver" "API Server"
    stop_process "ele-scanner" "Scanner"
    stop_process "ele-stat" "Stat"
    echo "✓ All services stopped."
}

# 检查参数
if [ $# -eq 0 ]; then
    echo "Error: No service specified"
    show_usage
    exit 1
fi

# 主逻辑
case "$1" in
    "room-server")
        stop_process "ele-roomserver" "Room Server"
        ;;
    "api-server")
        stop_process "ele-apiserver" "API Server"
        ;;
    "scanner")
        stop_process "ele-scanner" "Scanner"
        ;;
    "stat")
        stop_process "ele-stat" "Stat"
        ;;
    "all")
        stop_all
        ;;
    *)
        echo "Error: Unknown service '$1'"
        show_usage
        exit 1
        ;;
esac 
