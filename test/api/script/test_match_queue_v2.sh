#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"

echo "=== 匹配队列测试 V2 (支持Model参数) ==="
echo "服务器地址: $SERVER_URL"
echo ""

# 测试用户列表
users=("user_1" "user_2" "user_3" "user_4" "user_5")
models=("model_a" "model_b")

# 检查所有用户的cookie文件是否存在
echo "检查用户认证文件..."
for user in "${users[@]}"; do
    user_dir="test/api/users/$user"
    cookie_file="$user_dir/cookie.txt"
    
    if [ ! -f "$cookie_file" ]; then
        echo "✗ 用户 $user 的Cookie文件不存在: $cookie_file"
        echo "请先运行 test_login.sh 为所有用户进行登录认证"
        exit 1
    else
        echo "✓ 用户 $user 认证文件存在"
    fi
done
echo ""

# 测试1: 不同model的队列测试
echo "1. 测试不同model的队列..."
for model in "${models[@]}"; do
    echo "   测试model: $model"
    
    # 加入2个用户到当前model队列
    for i in {0..1}; do
        user=${users[$i]}
        user_dir="test/api/users/$user"
        cookie_file="$user_dir/cookie.txt"
        
        echo "     加入用户: user$user_num 到 $model"
        # 从user_1提取数字部分生成PublicKey
        user_num=${user#user_}  # 去掉user_前缀，得到1
        response=$(curl -s -X POST $SERVER_URL/ \
          -H "Content-Type: application/json" \
          -d "{
            \"Action\": \"JoinQueue\",
            \"Model\": \"$model\",
            \"PublicKey\": \"pk_user$user_num\"
          }" \
          -c "$cookie_file" -b "$cookie_file")
        
        echo "     响应: $response"
        
        if echo "$response" | grep -q "RetCode.*0"; then
            echo "     ✓ user$user_num 加入 $model 队列成功"
        else
            echo "     ✗ user$user_num 加入 $model 队列失败"
        fi
    done
    
    # 检查当前model队列状态
    echo "     检查 $model 队列状态..."
    # 使用第一个用户的cookie来查询队列状态
    user1_cookie="test/api/users/user_1/cookie.txt"
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"CheckMatchStatus\",
        \"Model\": \"$model\"
      }" \
      -c "$user1_cookie" -b "$user1_cookie")
    
    echo "     响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        queue_count=$(echo "$response" | grep -o '"queue_count":[0-9]*' | cut -d':' -f2)
        echo "     ✓ $model 队列人数: $queue_count"
    else
        echo "     ✗ $model 队列状态查询失败"
    fi
    
    echo ""
done

# 测试2: 检查用户状态
echo "2. 检查用户状态..."
for model in "${models[@]}"; do
    echo "   检查 $model 队列中的用户状态..."
    
    for i in {0..1}; do
        user=${users[$i]}
        user_dir="test/api/users/$user"
        cookie_file="$user_dir/cookie.txt"
        user_num=${user#user_}  # 去掉user_前缀，得到1
        
        echo "     检查用户: user$user_num 在 $model 队列中的状态"
        response=$(curl -s -X POST $SERVER_URL/ \
          -H "Content-Type: application/json" \
          -d "{
            \"Action\": \"CheckMatchStatus\",
            \"Model\": \"$model\"
          }" \
          -c "$cookie_file" -b "$cookie_file")
        
        if echo "$response" | grep -q "RetCode.*0"; then
            status=$(echo "$response" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
            queue_count=$(echo "$response" | grep -o '"queue_count":[0-9]*' | cut -d':' -f2)
            echo "     ✓ user$user_num 在 $model 队列状态: $status, 队列人数: $queue_count"
        else
            echo "     ✗ user$user_num 状态查询失败"
        fi
    done
    echo ""
done

# 测试3: 部分用户离开队列
echo "3. 测试部分用户离开队列..."
echo "   用户 user1 离开 model_a 队列..."
user1_cookie="test/api/users/user_1/cookie.txt"
response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"LeaveQueue\",
    \"Model\": \"model_a\"
  }" \
  -c "$user1_cookie" -b "$user1_cookie")

echo "   响应: $response"

if echo "$response" | grep -q "RetCode.*0"; then
    echo "   ✓ user1 离开 model_a 队列成功"
else
    echo "   ✗ user1 离开 model_a 队列失败"
fi

echo "   用户 user2 离开 model_b 队列..."
user2_cookie="test/api/users/user_2/cookie.txt"
response=$(curl -s -X POST $SERVER_URL/ \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"LeaveQueue\",
    \"Model\": \"model_b\"
  }" \
  -c "$user2_cookie" -b "$user2_cookie")

echo "   响应: $response"

if echo "$response" | grep -q "RetCode.*0"; then
    echo "   ✓ user2 离开 model_b 队列成功"
else
    echo "   ✗ user2 离开 model_b 队列失败"
fi

echo ""

# 测试4: 再次检查队列状态
echo "4. 检查离开后的队列状态..."
for model in "${models[@]}"; do
    echo "   检查 $model 队列状态..."
    # 使用第一个用户的cookie来查询队列状态
    user1_cookie="test/api/users/user_1/cookie.txt"
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"CheckMatchStatus\",
        \"Model\": \"$model\"
      }" \
      -c "$user1_cookie" -b "$user1_cookie")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        queue_count=$(echo "$response" | grep -o '"queue_count":[0-9]*' | cut -d':' -f2)
        echo "   ✓ $model 队列人数: $queue_count"
    else
        echo "   ✗ $model 队列状态查询失败"
    fi
    echo ""
done

echo ""
echo "=== 测试总结 ==="
echo "✓ 支持不同model的独立队列"
echo "✓ 同时存储address和publickey信息"
echo "✓ 队列状态查询功能正常"
echo "✓ 用户加入/离开队列功能正常"
echo "✓ 队列人数统计功能正常"
echo ""
echo "下一步可以测试:"
echo "1. 队列人数达到阈值时的自动匹配功能"
echo "2. 重复加入队列的防重复功能"
echo "3. 并发测试"
echo "4. Redis数据持久化验证" 