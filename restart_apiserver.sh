#!/bin/bash

echo "=== BeastRoyale API Server Restart Script ==="

# 1. 停止API服务
echo "1. Stopping API server..."
./stop.sh apiserver
if [ $? -eq 0 ]; then
    echo "   ✓ API server stopped successfully"
else
    echo "   ⚠ API server was not running or already stopped"
fi

# 清理API服务器日志
rm -rf ./bin/logs/*

# 2. 编译应用
echo "2. Building application..."
make apiserver
if [ $? -eq 0 ]; then
    echo "   ✓ Build completed successfully"
else
    echo "   ✗ Build failed!"
    exit 1
fi

# 3. 启动API服务
echo "3. Starting API server..."
./start.sh apiserver
if [ $? -eq 0 ]; then
    echo "   ✓ API server started successfully"
    echo ""
    echo "=== API Server restart completed successfully ==="
    echo "API server is running on port 8080"
    echo "Check logs with: tail -f logs/beast-royale.log"
else
    echo "   ✗ Failed to start API server!"
    exit 1
fi 