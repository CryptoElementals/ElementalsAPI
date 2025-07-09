#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"
BASE_USER_DIR="test/api/users"

echo "=== 批量RefreshTokens API 测试 ==="
echo "Server: $SERVER_URL"
echo "用户范围: user_1 到 user_5"
echo ""

# 检查服务器是否运行
echo "检查服务器状态..."
if ! curl -s "$SERVER_URL" > /dev/null; then
    echo "✗ 服务器未运行或无法访问: $SERVER_URL"
    exit 1
fi
echo "✓ 服务器运行正常"
echo ""

# 批量处理用户
for i in {1..5}; do
    USER_DIR="$BASE_USER_DIR/user_$i"
    REFRESH_TOKEN_FILE="$USER_DIR/refresh_token.txt"
    COOKIE_FILE="$USER_DIR/cookie.txt"
    
    echo "--- 处理 user_$i ---"
    
    # 检查refresh token文件是否存在
    if [ ! -f "$REFRESH_TOKEN_FILE" ]; then
        echo "✗ user_$i: Refresh token文件不存在: $REFRESH_TOKEN_FILE"
        echo "   跳过此用户"
        echo ""
        continue
    fi
    
    # 读取refresh token
    REFRESH_TOKEN=$(cat "$REFRESH_TOKEN_FILE")
    if [ -z "$REFRESH_TOKEN" ]; then
        echo "✗ user_$i: Refresh token文件为空"
        echo "   跳过此用户"
        echo ""
        continue
    fi
    
    echo "RefreshToken: ${REFRESH_TOKEN:0:20}..."
    
    # 测试RefreshTokens API
    echo "正在刷新 user_$i 的token..."
    refresh_response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"RefreshTokens\",
        \"RefreshToken\": \"$REFRESH_TOKEN\"
      }" \
      -c "$COOKIE_FILE" -b "$COOKIE_FILE")
    
    # 检查响应
    if echo "$refresh_response" | grep -q "RetCode.*0"; then
        echo "✓ user_$i: RefreshTokens API 成功"
        
        # 提取过期时间
        EXPIRATION_TIME=$(echo "$refresh_response" | grep -o '"RefreshTokenExpirationTime":[0-9]*' | cut -d':' -f2)
        if [ -n "$EXPIRATION_TIME" ]; then
            EXPIRATION_DATE=$(date -d @$EXPIRATION_TIME 2>/dev/null || echo "无法解析时间")
            echo "  - 过期时间: $EXPIRATION_TIME ($EXPIRATION_DATE)"
        fi
        
        echo "  - 新的session cookie已生成"
        echo "  - Redis中的refresh token TTL已更新"
    else
        echo "✗ user_$i: RefreshTokens API 失败"
        echo "  响应: $refresh_response"
    fi
    
    echo ""
done

echo "=== 批量刷新完成 ==="
echo ""

# 显示统计信息
echo "统计信息:"
for i in {1..5}; do
    USER_DIR="$BASE_USER_DIR/user_$i"
    COOKIE_FILE="$USER_DIR/cookie.txt"
    
    if [ -f "$COOKIE_FILE" ]; then
        COOKIE_SIZE=$(wc -c < "$COOKIE_FILE")
        echo "user_$i: cookie文件大小 ${COOKIE_SIZE} 字节"
    else
        echo "user_$i: 无cookie文件"
    fi
done

echo ""
echo "所有用户的cookie已更新完成！" 