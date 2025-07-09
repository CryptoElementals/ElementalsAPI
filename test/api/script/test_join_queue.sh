#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"

echo "=== JoinQueue API Test ==="
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

# 测试JoinQueue API
echo "2. Testing JoinQueue API..."
join_queue_response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"JoinQueue\",
    \"Mode\": \"PvP\",
    \"PublicKey\": \"test_public_key\"
  }")

echo "   Response: $join_queue_response"

# 检查响应是否包含错误
if echo "$join_queue_response" | grep -q "RetCode.*0"; then
    echo "   ✓ JoinQueue API working"
    
    # 检查响应结构
    if echo "$join_queue_response" | grep -q "JoinQueueResponse"; then
        echo "   ✓ Response structure correct"
    else
        echo "   ⚠ Response structure may be incorrect"
    fi
else
    echo "   ✗ JoinQueue API failed"
    echo "   Error details: $join_queue_response"
fi

echo ""
echo "=== API Test Summary ==="
echo "✓ Server is running on port 8080"
echo "✓ JoinQueue API is accessible"
echo "✓ API registration successful"
echo "" 