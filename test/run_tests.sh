#!/bin/bash

echo "=== Running BeastRoyale Backend Tests ==="
echo ""

# 切换到测试目录
cd "$(dirname "$0")"

# 运行所有测试
echo "Running all tests..."
go test -v

echo ""
echo "=== Tests completed ===" 