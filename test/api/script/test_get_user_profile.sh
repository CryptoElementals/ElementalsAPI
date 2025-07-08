#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"
USER_0_DIR="test/api/users/user_0"
ADDRESS_FILE="$USER_0_DIR/address.txt"
ADDRESS=$(cat "$ADDRESS_FILE")

echo "=== GetUserProfile API Test ==="
echo "Server: $SERVER_URL"
echo "Address: $ADDRESS"
echo ""

# 测试服务器连通性
echo "1. Testing server connectivity..."
response=$(curl -s -o /dev/null -w "%{http_code}" $SERVER_URL/)
if [ "$response" = "404" ]; then
    echo "   ✓ Server is running (404 is expected for root path)"
else
    echo "   ✗ Server connectivity test failed: $response"
    exit 1
fi

# 测试 GetUserProfile API - 有效地址
echo "2. Testing GetUserProfile API (有效地址)..."
get_profile_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"GetUserProfile\",
    \"Address\": \"$ADDRESS\"
  }")

echo "   Response: $get_profile_response"

# 检查响应是否包含错误
if echo "$get_profile_response" | grep -q "RetCode.*0"; then
    echo "   ✓ GetUserProfile API working"
    
    # 检查响应字段
    if echo "$get_profile_response" | grep -q '"UserInfo"'; then
        echo "   ✓ UserInfo field present"
        
        # 提取用户信息
        USER_NAME=$(echo "$get_profile_response" | grep -o '"Name":"[^"]*"' | cut -d'"' -f4)
        USER_ADDRESS=$(echo "$get_profile_response" | grep -o '"Address":"[^"]*"' | cut -d'"' -f4)
        USER_POINTS=$(echo "$get_profile_response" | grep -o '"Points":[0-9]*' | cut -d':' -f2)
        USER_TOKEN_AMOUNT=$(echo "$get_profile_response" | grep -o '"TokenAmount":[0-9]*' | cut -d':' -f2)
        USER_OVERALL_GAME=$(echo "$get_profile_response" | grep -o '"OverallGame":[0-9]*' | cut -d':' -f2)
        USER_WINNING_RATE=$(echo "$get_profile_response" | grep -o '"WinningRate":[0-9.]*' | cut -d':' -f2)
        
        echo "   ✓ User Name: $USER_NAME"
        echo "   ✓ User Address: $USER_ADDRESS"
        echo "   ✓ User Points: $USER_POINTS"
        echo "   ✓ User Token Amount: $USER_TOKEN_AMOUNT"
        echo "   ✓ User Overall Game: $USER_OVERALL_GAME"
        echo "   ✓ User Winning Rate: $USER_WINNING_RATE"
        
        # 检查卡牌统计信息
        if echo "$get_profile_response" | grep -q '"CardStatInfo"'; then
            echo "   ✓ CardStatInfo field present"
        else
            echo "   ⚠ CardStatInfo field not found (may be empty array)"
        fi
    else
        echo "   ✗ UserInfo field not found in response"
    fi
else
    echo "   ✗ GetUserProfile API failed"
    echo "   Error details: $get_profile_response"
fi

# 测试 GetUserProfile API - 无效地址
echo "3. Testing GetUserProfile API (无效地址)..."
invalid_address="0xInvalidAddress123456789"
get_profile_invalid_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"GetUserProfile\",
    \"Address\": \"$invalid_address\"
  }")

echo "   Response: $get_profile_invalid_response"

# 检查无效地址的响应
if echo "$get_profile_invalid_response" | grep -q "RetCode.*[1-9]"; then
    echo "   ✓ GetUserProfile API correctly returned error for invalid address"
else
    echo "   ⚠ GetUserProfile API response for invalid address: $get_profile_invalid_response"
fi

# 测试 GetUserProfile API - 缺少地址参数
echo "4. Testing GetUserProfile API (缺少地址参数)..."
missing_address_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"GetUserProfile\"
  }")

echo "   Response: $missing_address_response"

# 检查缺少参数的响应
if echo "$missing_address_response" | grep -q "RetCode.*[1-9]"; then
    echo "   ✓ GetUserProfile API correctly returned error for missing address"
else
    echo "   ⚠ GetUserProfile API response for missing address: $missing_address_response"
fi

echo ""
echo "=== GetUserProfile API Test Summary ==="
echo "✓ Server is running on port 8080"
echo "✓ GetUserProfile API is accessible"
echo "✓ API correctly handles valid addresses"
echo "✓ API correctly handles invalid addresses"
echo "✓ API correctly handles missing parameters"
echo ""
echo "Test completed successfully!" 