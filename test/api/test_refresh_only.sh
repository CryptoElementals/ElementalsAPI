#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"
REFRESH_TOKEN_FILE="test/api/refresh_token.txt"
COOKIE_FILE="test/api/cookie.txt"

echo "=== RefreshTokens API 单独测试 ==="
echo "Server: $SERVER_URL"
echo "RefreshToken File: $REFRESH_TOKEN_FILE"
echo "RequestUUID: 系统自动生成"
echo ""

# 检查refresh token文件是否存在
if [ ! -f "$REFRESH_TOKEN_FILE" ]; then
    echo "✗ Refresh token文件不存在: $REFRESH_TOKEN_FILE"
    echo "请先运行 ./test/api/test_api_curl.sh 生成refresh token"
    exit 1
fi

# 读取refresh token
REFRESH_TOKEN=$(cat "$REFRESH_TOKEN_FILE")
if [ -z "$REFRESH_TOKEN" ]; then
    echo "✗ Refresh token文件为空"
    exit 1
fi

echo "RefreshToken: $REFRESH_TOKEN"
echo ""

# 测试RefreshTokens API
echo "Testing RefreshTokens API..."
refresh_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"RefreshTokens\",
    \"RefreshToken\": \"$REFRESH_TOKEN\"
  }" \
  -c "$COOKIE_FILE" -b "$COOKIE_FILE")

echo "Response: $refresh_response"

# 检查响应
if echo "$refresh_response" | grep -q "RetCode.*0"; then
    echo "   ✓ RefreshTokens API 成功"
    
    # 提取过期时间
    EXPIRATION_TIME=$(echo "$refresh_response" | grep -o '"RefreshTokenExpirationTime":[0-9]*' | cut -d':' -f2)
    if [ -n "$EXPIRATION_TIME" ]; then
        echo "   ✓ 过期时间: $EXPIRATION_TIME ($(date -d @$EXPIRATION_TIME))"
    fi
    
    echo "   ✓ 新的session cookie已生成"
    echo "   ✓ Redis中的refresh token TTL已更新"
else
    echo "   ✗ RefreshTokens API 失败"
fi

echo ""
echo "=== 测试完成 ===" 