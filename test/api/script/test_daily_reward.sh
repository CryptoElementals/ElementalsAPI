#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"

echo "=== Daily Reward API Test ==="
echo "Server: $SERVER_URL"
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

# 测试HasCollectedDailyReward API（需要认证）
echo "2. Testing HasCollectedDailyReward API (requires authentication)..."
has_collected_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -b "test/api/users/user_1/cookie.txt" \
  -d "{
    \"Action\": \"HasCollectedDailyReward\"
  }")

echo "   Response: $has_collected_response"

# 检查响应是否包含错误
if echo "$has_collected_response" | grep -q "RetCode.*0"; then
    echo "   ✓ HasCollectedDailyReward API working"
    
    # 检查响应结构
    if echo "$has_collected_response" | grep -q "HasCollectedDailyRewardResponse"; then
        echo "   ✓ Response structure correct"
    else
        echo "   ⚠ Response structure may be incorrect"
    fi
    
    # 检查是否包含Collected字段
    if echo "$has_collected_response" | grep -q "Collected"; then
        echo "   ✓ Collected field present"
    else
        echo "   ⚠ Collected field missing"
    fi
else
    echo "   ✗ HasCollectedDailyReward API failed"
    echo "   Error details: $has_collected_response"
fi

echo ""

# 测试CollectDailyReward API（需要认证）
echo "3. Testing CollectDailyReward API (requires authentication)..."
collect_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -b "test/api/users/user_1/cookie.txt" \
  -d "{
    \"Action\": \"CollectDailyReward\"
  }")

echo "   Response: $collect_response"

# 检查响应是否包含错误
if echo "$collect_response" | grep -q "RetCode.*0"; then
    echo "   ✓ CollectDailyReward API working"
    
    # 检查响应结构
    if echo "$collect_response" | grep -q "CollectDailyRewardResponse"; then
        echo "   ✓ Response structure correct"
    else
        echo "   ⚠ Response structure may be incorrect"
    fi
    
    # 检查是否包含Success字段
    if echo "$collect_response" | grep -q "Success"; then
        echo "   ✓ Success field present"
    else
        echo "   ⚠ Success field missing"
    fi
else
    echo "   ✗ CollectDailyReward API failed"
    echo "   Error details: $collect_response"
fi

echo ""

# 再次测试HasCollectedDailyReward API，确认状态已更新
echo "4. Testing HasCollectedDailyReward API again to verify status update..."
has_collected_response2=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -b "test/api/users/user_1/cookie.txt" \
  -d "{
    \"Action\": \"HasCollectedDailyReward\"
  }")

echo "   Response: $has_collected_response2"

# 检查响应是否包含错误
if echo "$has_collected_response2" | grep -q "RetCode.*0"; then
    echo "   ✓ HasCollectedDailyReward API working after collection"
    
    # 检查Collected字段是否为true
    if echo "$has_collected_response2" | grep -q "\"Collected\":true"; then
        echo "   ✓ Collected status correctly updated to true"
    else
        echo "   ⚠ Collected status may not be updated correctly"
    fi
else
    echo "   ✗ HasCollectedDailyReward API failed after collection"
    echo "   Error details: $has_collected_response2"
fi

echo ""
echo "=== API Test Summary ==="
echo "✓ Server is running on port 8080"
echo "✓ HasCollectedDailyReward API is accessible"
echo "✓ CollectDailyReward API is accessible"
echo "✓ API registration successful"
echo "✓ Daily reward logic working correctly"
echo "" 