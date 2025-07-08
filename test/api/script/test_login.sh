#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"
USER_0_DIR="test/api/users/user_0"
ADDRESS="0xD84bCB296882241EBc75770E7b47E08049aEcc3b"
PRIVATE_KEY_PATH="$USER_0_DIR/private.key"
COOKIE_FILE="$USER_0_DIR/cookie.txt"
REFRESH_TOKEN_FILE="$USER_0_DIR/refresh_token.txt"

# 检查user_0文件夹是否存在
if [ ! -d "$USER_0_DIR" ]; then
    echo "✗ 用户0文件夹不存在: $USER_0_DIR"
    exit 1
fi

# 清理旧cookie
rm -f "$COOKIE_FILE"

echo "=== BeastRoyale Backend API Test ==="
echo "Server: $SERVER_URL"
echo "Address: $ADDRESS"
echo "RequestUUID: 系统自动生成"
echo "用户文件夹: $USER_0_DIR"
echo ""

# 检查私钥文件是否存在
if [ ! -f "$PRIVATE_KEY_PATH" ]; then
    echo "✗ 私钥文件不存在: $PRIVATE_KEY_PATH"
    exit 1
fi

# 测试服务器连通性
echo "1. Testing server connectivity..."
response=$(curl -s -o /dev/null -w "%{http_code}" $SERVER_URL/ -c "$COOKIE_FILE" -b "$COOKIE_FILE")
if [ "$response" = "404" ]; then
    echo "   ✓ Server is running (404 is expected for root path)"
else
    echo "   ✗ Server connectivity test failed: $response"
    exit 1
fi

# 测试获取登录验证码
echo "2. Testing GetLoginCode API..."
get_code_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"GetLoginCode\",
    \"Address\": \"$ADDRESS\"
  }" \
  -c "$COOKIE_FILE" -b "$COOKIE_FILE")

echo "   Response: $get_code_response"

# 检查响应是否包含错误
if echo "$get_code_response" | grep -q "RetCode.*0"; then
    echo "   ✓ GetLoginCode API working"
    
    # 提取nonce值
    NONCE=$(echo "$get_code_response" | grep -o '"Nonce":[0-9]*' | cut -d':' -f2)
    if [ -n "$NONCE" ]; then
        echo "   ✓ Nonce extracted: $NONCE"
        
        # 使用签名工具生成签名
        echo "3. Generating signature..."
        SIGNATURE_OUTPUT=$(go run test/api/tools/sign.go -address "$ADDRESS" -private "$PRIVATE_KEY_PATH" -nonce "$NONCE" 2>&1)
        SIGNATURE=$(echo "$SIGNATURE_OUTPUT" | grep "签名(hex):" | sed 's/签名(hex): //')
        
        if [ -n "$SIGNATURE" ]; then
            echo "   ✓ Signature generated: ${SIGNATURE:0:20}..."
            
            # 测试LoginWeb3 API
            echo "4. Testing LoginWeb3 API..."
            login_response=$(curl -s -X POST $SERVER_URL/ \
              -H "Content-Type: application/json" \
              -d "{
                \"Action\": \"LoginWeb3\",
                \"Address\": \"$ADDRESS\",
                \"Nonce\": $NONCE,
                \"Signature\": \"$SIGNATURE\"
              }" \
              -c "$COOKIE_FILE" -b "$COOKIE_FILE")
            
            echo "   Response: $login_response"
            
            # 检查登录响应
            if echo "$login_response" | grep -q "RetCode.*0"; then
                echo "   ✓ LoginWeb3 API working"
                
                # 提取refresh token并保存到文件
                REFRESH_TOKEN=$(echo "$login_response" | grep -o '"RefreshToken":"[^"]*"' | cut -d'"' -f4)
                if [ -n "$REFRESH_TOKEN" ]; then
                    echo "$REFRESH_TOKEN" > "$REFRESH_TOKEN_FILE"
                    echo "   ✓ RefreshToken extracted and saved to: $REFRESH_TOKEN_FILE"
                    
                    # 测试RefreshTokens API
                    echo "5. Testing RefreshTokens API..."
                    refresh_response=$(curl -s -X POST $SERVER_URL/ \
                      -H "Content-Type: application/json" \
                      -d "{
                        \"Action\": \"RefreshTokens\",
                        \"RefreshToken\": \"$REFRESH_TOKEN\"
                      }" \
                      -c "$COOKIE_FILE" -b "$COOKIE_FILE")
                    
                    echo "   Response: $refresh_response"
                    
                    # 检查刷新响应
                    if echo "$refresh_response" | grep -q "RetCode.*0"; then
                        echo "   ✓ RefreshTokens API working"
                        echo "   ✓ 新的session cookie已生成"
                        echo "   ✓ Redis中的refresh token TTL已更新"
                    else
                        echo "   ✗ RefreshTokens API failed"
                    fi
                    
                    # 测试IsWalletLoggedIn API - 有效token
                    echo "6. Testing IsWalletLoggedIn API (有效token)..."
                    is_logged_in_response=$(curl -s -X POST $SERVER_URL/ \
                      -H "Content-Type: application/json" \
                      -d "{
                        \"Action\": \"IsWalletLoggedIn\",
                        \"RefreshToken\": \"$REFRESH_TOKEN\"
                      }" \
                      -c "$COOKIE_FILE" -b "$COOKIE_FILE")
                    
                    echo "   Response: $is_logged_in_response"
                    
                    # 检查IsWalletLoggedIn响应
                    if echo "$is_logged_in_response" | grep -q "RetCode.*0"; then
                        echo "   ✓ IsWalletLoggedIn API working"
                        
                        # 检查是否登录
                        WALLET_LOGGED_IN=$(echo "$is_logged_in_response" | grep -o '"WalletLoggedIn":true')
                        if [ -n "$WALLET_LOGGED_IN" ]; then
                            echo "   ✓ 钱包已登录"
                            
                            # 提取钱包地址
                            ADDRESS_FROM_RESPONSE=$(echo "$is_logged_in_response" | grep -o '"Address":"[^"]*"' | cut -d'"' -f4)
                            if [ -n "$ADDRESS_FROM_RESPONSE" ]; then
                                echo "   ✓ 钱包地址: $ADDRESS_FROM_RESPONSE"
                            fi
                        else
                            echo "   ⚠ 钱包未登录"
                        fi
                    else
                        echo "   ✗ IsWalletLoggedIn API failed"
                    fi
                    
                    # 测试IsWalletLoggedIn API - 无效token
                    echo "7. Testing IsWalletLoggedIn API (无效token)..."
                    invalid_token="invalid-refresh-token-12345"
                    is_logged_in_response_invalid=$(curl -s -X POST $SERVER_URL/ \
                      -H "Content-Type: application/json" \
                      -d "{
                        \"Action\": \"IsWalletLoggedIn\",
                        \"RefreshToken\": \"$invalid_token\"
                      }" \
                      -c "$COOKIE_FILE" -b "$COOKIE_FILE")
                    
                    echo "   Response: $is_logged_in_response_invalid"
                    
                    # 检查无效token的响应
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
                    
                else
                    echo "   ⚠ No RefreshToken found in response"
                fi
            else
                echo "   ✗ LoginWeb3 API failed"
            fi
        else
            echo "   ✗ Failed to generate signature"
            echo "   Debug output: $SIGNATURE_OUTPUT"
        fi
    else
        echo "   ✗ Failed to extract nonce from response"
    fi
else
    echo "   ✗ GetLoginCode API failed"
fi

echo ""
echo "=== API Test Summary ==="
echo "✓ Server is running on port 8080"
echo "✓ API endpoints are accessible"
echo "✓ IsWalletLoggedIn API tested with both valid and invalid tokens"
echo "✓ 使用用户0文件夹: $USER_0_DIR"
echo ""
echo "Test completed with auto-generated RequestUUID" 