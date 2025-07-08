#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"

# 用户配置
USER_1_DIR="test/api/users/user_1"
USER_2_DIR="test/api/users/user_2"
USER_3_DIR="test/api/users/user_3"
USER_4_DIR="test/api/users/user_4"
USER_5_DIR="test/api/users/user_5"

# Cookie文件路径
COOKIE_1="$USER_1_DIR/cookie.txt"
COOKIE_2="$USER_2_DIR/cookie.txt"
COOKIE_3="$USER_3_DIR/cookie.txt"
COOKIE_4="$USER_4_DIR/cookie.txt"
COOKIE_5="$USER_5_DIR/cookie.txt"

echo "=== 检查用户Cookie有效性 ==="
echo "服务器地址: $SERVER_URL"
echo ""

# 检查用户cookie有效性
users=("user_1" "user_2" "user_3" "user_4" "user_5")
cookies=("$COOKIE_1" "$COOKIE_2" "$COOKIE_3" "$COOKIE_4" "$COOKIE_5")

for i in {0..4}; do
    user=${users[$i]}
    cookie=${cookies[$i]}
    
    echo "检查用户: $user"
    
    # 检查cookie文件是否存在
    if [ ! -f "$cookie" ]; then
        echo "   ✗ Cookie文件不存在: $cookie"
        echo ""
        continue
    fi
    
    echo "   ✓ Cookie文件存在: $cookie"
    
    # 使用SetUserProfile API检查cookie是否有效
    echo "   调用SetUserProfile API测试cookie..."
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"SetUserProfile\",
        \"Name\": \"test_name_$user\",
        \"AvatarURL\": \"https://example.com/avatar_$user.jpg\"
      }" \
      -b "$cookie")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        echo "   ✓ Cookie有效，用户已登录"
    elif echo "$response" | grep -q "Login cookie.*invalid"; then
        echo "   ✗ Cookie无效或已过期"
    elif echo "$response" | grep -q "未获取到玩家地址"; then
        echo "   ✗ 未获取到玩家地址，cookie可能有问题"
    else
        echo "   ✗ 其他错误"
    fi
    
    echo ""
done

echo "=== 检查完成 ===" 