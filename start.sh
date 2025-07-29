#!/bin/bash

BIN_DIR="bin"
API_SERVER="ele-apiserver"
ROOM_SERVER="ele-roomserver"
SCANNER="ele-scanner"
API_CONFIG="config.yaml"
ROOM_CONFIG="config.yaml"
SCANNER_CONFIG="scanner_config.yaml"

# 显示使用说明
show_usage() {
    echo "Usage: $0 <service>"
    echo "  service: room-server | api-server | scanner | all"
    echo ""
    echo "Examples:"
    echo "  $0 all                # Start all services"
    echo "  $0 room-server        # Start only room server"
    echo "  $0 api-server         # Start only API server"
    echo "  $0 scanner            # Start only scanner"
}

# 启动 room server
start_room_server() {
    echo "Starting $ROOM_SERVER with config $ROOM_CONFIG..."
    nohup ./$BIN_DIR/$ROOM_SERVER run --config $ROOM_CONFIG > /dev/null 2>&1 &
    echo "✓ $ROOM_SERVER started with PID: $!"
}

# 启动 API server
start_api_server() {
    echo "Starting $API_SERVER with config $API_CONFIG..."
    nohup ./$BIN_DIR/$API_SERVER run --config $API_CONFIG > /dev/null 2>&1 &
    echo "✓ $API_SERVER started with PID: $!"
}

# 启动 scanner
start_scanner() {
    echo "Starting $SCANNER with config $SCANNER_CONFIG..."
    nohup ./$BIN_DIR/$SCANNER run --config $SCANNER_CONFIG > /dev/null 2>&1 &
    echo "✓ $SCANNER started with PID: $!"
}

# 启动所有服务
start_all() {
    echo "Starting all services..."
    start_room_server
    sleep 2
    start_api_server
    sleep 2
    start_scanner
    echo "✓ All services started successfully"
}

# 检查参数
if [ $# -eq 0 ]; then
    echo "Error: No service specified"
    show_usage
    exit 1
fi

# 主逻辑
case "$1" in
    "roomserver")
        start_room_server
        ;;
    "apiserver")
        start_api_server
        ;;
    "scanner")
        start_scanner
        ;;
    "all")
        start_all
        ;;
    *)
        echo "Error: Unknown service '$1'"
        show_usage
        exit 1
        ;;
esac