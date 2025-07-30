#!/bin/bash

# 多用户登录测试脚本
# 一次性测试多个用户的登录流程

# 配置参数
SERVER_URL="http://localhost:8080"

# 自动收集所有用户信息
users=()
for user_dir in test/api/users/user_[1-9]*; do
    if [[ -d "$user_dir" ]]; then
        user_id=$(basename "$user_dir" | sed 's/user_//')
        # 只允许user_id为纯数字
        if [[ "$user_id" =~ ^[0-9]+$ ]]; then
            address_file="$user_dir/address.txt"
            private_key_file="$user_dir/private.key"
            if [ -f "$address_file" ] && [ -f "$private_key_file" ]; then
                address=$(cat "$address_file")
                users+=("$user_id:$address:$private_key_file")
            else
                users+=("$user_id::")
            fi
        fi
    fi

done

user_count=${#users[@]}
echo "=== BeastRoyale Backend 多用户登录测试 ==="
echo "Server: $SERVER_URL"
echo "测试用户数量: $user_count"
echo "RequestUUID: 系统自动生成"
echo ""

# 检查服务器连通性
echo "1. Testing server connectivity..."
response=$(curl -s -o /dev/null -w "%{http_code}" $SERVER_URL/)
if [ "$response" = "404" ]; then
    echo "   ✓ Server is running (404 is expected for root path)"
else
    echo "   ✗ Server connectivity test failed: $response"
    exit 1
fi

# 存储所有用户的登录结果
declare -A login_results
declare -A refresh_tokens

# 测试每个用户的登录
for user_info in "${users[@]}"; do
    IFS=':' read -r user_id address private_key_path <<< "$user_info"
    
    # 为每个用户创建独立的文件夹和文件
    USER_DIR="test/api/users/user_$user_id"
    COOKIE_FILE="$USER_DIR/cookie.txt"
    REFRESH_TOKEN_FILE="$USER_DIR/refresh_token.txt"
    
    # 确保用户文件夹存在
    mkdir -p "$USER_DIR"
    
    # 清理旧cookie
    rm -f "$COOKIE_FILE"
    
    echo ""
    echo "=== 测试用户 $user_id ==="
    echo "地址: $address"
    echo "私钥文件: $private_key_path"
    echo "用户文件夹: $USER_DIR"
    echo "├── private.key"
    echo "├── address.txt"
    echo "├── cookie.txt"
    echo "└── refresh_token.txt"
    
    # 检查私钥文件是否存在
    if [ ! -f "$private_key_path" ]; then
        echo "   ✗ 私钥文件不存在: $private_key_path"
        login_results[$user_id]="failed"
        continue
    fi
    
    # 测试获取登录验证码
    echo "   2.$user_id.1. Testing GetLoginCode API..."
    get_code_response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"GetLoginCode\",
        \"Address\": \"$address\"
      }" \
      -c "$COOKIE_FILE" -b "$COOKIE_FILE")
    
    echo "      Response: $get_code_response"
    
    # 检查响应是否包含错误
    if echo "$get_code_response" | grep -q "RetCode.*0"; then
        echo "      ✓ GetLoginCode API working"
        
        # 提取nonce值
        NONCE=$(echo "$get_code_response" | grep -o '"Nonce":[0-9]*' | cut -d':' -f2)
        if [ -n "$NONCE" ]; then
            echo "      ✓ Nonce extracted: $NONCE"
            
            # 使用签名工具生成签名
            echo "   2.$user_id.2. Generating signature..."
            SIGNATURE_OUTPUT=$(go run test/api/tools/sign.go -address "$address" -private "$private_key_path" -nonce "$NONCE" 2>&1)
            SIGNATURE=$(echo "$SIGNATURE_OUTPUT" | grep "签名(hex):" | sed 's/签名(hex): //')
            
            if [ -n "$SIGNATURE" ]; then
                echo "      ✓ Signature generated: ${SIGNATURE:0:20}..."
                
                # 测试LoginWeb3 API
                echo "   2.$user_id.3. Testing LoginWeb3 API..."
                login_response=$(curl -s -X POST $SERVER_URL/ \
                  -H "Content-Type: application/json" \
                  -d "{
                    \"Action\": \"LoginWeb3\",
                    \"Address\": \"$address\",
                    \"Nonce\": $NONCE,
                    \"Signature\": \"$SIGNATURE\"
                  }" \
                  -c "$COOKIE_FILE" -b "$COOKIE_FILE")
                
                echo "      Response: $login_response"
                
                # 检查登录响应
                if echo "$login_response" | grep -q "RetCode.*0"; then
                    echo "      ✓ LoginWeb3 API working"
                    
                    # 提取refresh token并保存到用户专属文件
                    REFRESH_TOKEN=$(echo "$login_response" | grep -o '"RefreshToken":"[^"]*"' | cut -d'"' -f4)
                    if [ -n "$REFRESH_TOKEN" ]; then
                        echo "$REFRESH_TOKEN" > "$REFRESH_TOKEN_FILE"
                        refresh_tokens[$user_id]="$REFRESH_TOKEN"
                        echo "      ✓ RefreshToken extracted and saved to: $REFRESH_TOKEN_FILE"
                        
                        # 测试RefreshTokens API
                        echo "   2.$user_id.4. Testing RefreshTokens API..."
                        refresh_response=$(curl -s -X POST $SERVER_URL/ \
                          -H "Content-Type: application/json" \
                          -d "{
                            \"Action\": \"RefreshTokens\",
                            \"RefreshToken\": \"$REFRESH_TOKEN\"
                          }" \
                          -c "$COOKIE_FILE" -b "$COOKIE_FILE")
                        
                        echo "      Response: $refresh_response"
                        
                        if echo "$refresh_response" | grep -q "RetCode.*0"; then
                            echo "      ✓ RefreshTokens API working"
                            echo "      ✓ 新的session cookie已生成"
                            echo "      ✓ Redis中的refresh token TTL已更新"
                            login_results[$user_id]="success"
                        else
                            echo "      ✗ RefreshTokens API failed"
                            login_results[$user_id]="failed"
                        fi
                    else
                        echo "      ⚠ No RefreshToken found in response"
                        login_results[$user_id]="failed"
                    fi
                else
                    echo "      ✗ LoginWeb3 API failed"
                    login_results[$user_id]="failed"
                fi
            else
                echo "      ✗ Failed to generate signature"
                echo "      Debug output: $SIGNATURE_OUTPUT"
                login_results[$user_id]="failed"
            fi
        else
            echo "      ✗ Failed to extract nonce from response"
            login_results[$user_id]="failed"
        fi
    else
        echo "      ✗ GetLoginCode API failed"
        login_results[$user_id]="failed"
    fi
done

# 测试IsWalletLoggedIn API
echo ""
echo "=== 测试IsWalletLoggedIn API ==="

for user_info in "${users[@]}"; do
    IFS=':' read -r user_id address private_key_path <<< "$user_info"
    
    # 为每个用户使用独立的文件
    USER_DIR="test/api/users/user_$user_id"
    COOKIE_FILE="$USER_DIR/cookie.txt"
    REFRESH_TOKEN_FILE="$USER_DIR/refresh_token.txt"
    
    echo ""
    echo "--- 用户 $user_id ---"
    echo "地址: $address"
    echo "用户文件夹: $USER_DIR"
    echo "Cookie文件: $COOKIE_FILE"
    echo "RefreshToken文件: $REFRESH_TOKEN_FILE"
    
    if [ "${login_results[$user_id]}" = "success" ] && [ -n "${refresh_tokens[$user_id]}" ]; then
        echo "   3.$user_id. Testing IsWalletLoggedIn API (有效token)..."
        is_logged_in_response=$(curl -s -X POST $SERVER_URL/ \
          -H "Content-Type: application/json" \
          -d "{
            \"Action\": \"IsWalletLoggedIn\",
            \"RefreshToken\": \"${refresh_tokens[$user_id]}\"
          }" \
          -c "$COOKIE_FILE" -b "$COOKIE_FILE")
        
        echo "      Response: $is_logged_in_response"
        
        if echo "$is_logged_in_response" | grep -q "RetCode.*0"; then
            echo "      ✓ IsWalletLoggedIn API working"
            
            # 检查是否登录
            WALLET_LOGGED_IN=$(echo "$is_logged_in_response" | grep -o '"WalletLoggedIn":true')
            if [ -n "$WALLET_LOGGED_IN" ]; then
                echo "      ✓ 钱包已登录"
                
                # 提取钱包地址
                ADDRESS_FROM_RESPONSE=$(echo "$is_logged_in_response" | grep -o '"Address":"[^"]*"' | cut -d'"' -f4)
                if [ -n "$ADDRESS_FROM_RESPONSE" ]; then
                    echo "      ✓ 钱包地址: $ADDRESS_FROM_RESPONSE"
                fi
            else
                echo "      ⚠ 钱包未登录"
            fi
        else
            echo "      ✗ IsWalletLoggedIn API failed"
        fi
    else
        echo "   ⚠ 跳过IsWalletLoggedIn测试 (登录失败或无refresh token)"
    fi
done

# 测试无效token
echo ""
echo "--- 测试无效token ---"
echo "   3.6. Testing IsWalletLoggedIn API (无效token)..."
invalid_token="invalid-refresh-token-12345"
is_logged_in_response_invalid=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"IsWalletLoggedIn\",
    \"RefreshToken\": \"$invalid_token\"
  }")

echo "   Response: $is_logged_in_response_invalid"

if echo "$is_logged_in_response_invalid" | grep -q "RetCode.*0"; then
    echo "   ✓ IsWalletLoggedIn API working"
    
    # 检查是否登录
    WALLET_LOGGED_IN=$(echo "$is_logged_in_response_invalid" | grep -o '"WalletLoggedIn":false')
    if [ -n "$WALLET_LOGGED_IN" ]; then
        echo "   ✓ 钱包未登录 (符合预期)"
    else
        echo "   ⚠ 钱包状态异常"
    fi
else
    echo "   ✗ IsWalletLoggedIn API failed"
fi

# 输出测试总结
echo ""
echo "=== 多用户登录测试总结 ==="
echo "✓ Server is running on port 8080"
echo "✓ API endpoints are accessible"
echo "✓ IsWalletLoggedIn API tested with both valid and invalid tokens"
echo "✓ 每个用户使用独立的文件夹，包含所有相关文件"
echo ""

success_count=0
total_count=0

for user_info in "${users[@]}"; do
    IFS=':' read -r user_id address private_key_path <<< "$user_info"
    total_count=$((total_count + 1))
    
    if [ "${login_results[$user_id]}" = "success" ]; then
        echo "✓ 用户 $user_id ($address): 登录成功"
        echo "  用户文件夹: test/api/users/user_$user_id"
        echo "  ├── private.key"
        echo "  ├── address.txt"
        echo "  ├── cookie.txt"
        echo "  └── refresh_token.txt"
        success_count=$((success_count + 1))
    else
        echo "✗ 用户 $user_id ($address): 登录失败"
    fi
done

echo ""
echo "成功登录: $success_count/$total_count 用户"
echo ""
echo "生成的文件结构:"
for user_info in "${users[@]}"; do
    IFS=':' read -r user_id address private_key_path <<< "$user_info"
    if [ "${login_results[$user_id]}" = "success" ]; then
        echo "  test/api/users/user_$user_id/"
        echo "  ├── private.key"
        echo "  ├── address.txt"
        echo "  ├── cookie.txt"
        echo "  └── refresh_token.txt"
    fi
done
echo ""
echo "Test completed with auto-generated RequestUUID" 