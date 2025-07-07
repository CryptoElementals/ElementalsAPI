#!/bin/bash

echo "=== BeastRoyale Backend Restart Script ==="

# 1. 停止服务
echo "1. Stopping server..."
./stop.sh
if [ $? -eq 0 ]; then
    echo "   ✓ Server stopped successfully"
else
    echo "   ⚠ Server was not running or already stopped"
fi

# 2. 编译应用
echo "2. Building application..."
make build
if [ $? -eq 0 ]; then
    echo "   ✓ Build completed successfully"
else
    echo "   ✗ Build failed!"
    exit 1
fi

# 3. 启动服务
echo "3. Starting server..."
./start.sh
if [ $? -eq 0 ]; then
    echo "   ✓ Server started successfully"
    echo ""
    echo "=== Restart completed successfully ==="
    echo "Server is running on port 8080"
    echo "Check logs with: tail -f stdout.log"
else
    echo "   ✗ Failed to start server!"
    exit 1
fi 