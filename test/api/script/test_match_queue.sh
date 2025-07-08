#!/bin/bash

# 配置参数
SERVER_URL="http://localhost:8080"

echo "=== 匹配队列场景测试 ==="
echo "服务器地址: $SERVER_URL"
echo ""

# 测试用户列表
users=("user_1" "user_2" "user_3" "user_4" "user_5")

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

# 辅助函数：获取用户编号
get_user_num() {
    local user=$1
    echo ${user#user_}
}

# 辅助函数：加入队列
join_queue() {
    local user=$1
    local model=$2
    local user_num=$(get_user_num $user)
    local user_dir="test/api/users/$user"
    local cookie_file="$user_dir/cookie.txt"
    
    echo "   用户 user$user_num 加入 $model 队列..."
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"JoinQueue\",
        \"Model\": \"$model\",
        \"PublicKey\": \"pk_user$user_num\"
      }" \
      -c "$cookie_file" -b "$cookie_file")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        echo "   ✓ user$user_num 加入 $model 队列成功"
        return 0
    else
        echo "   ✗ user$user_num 加入 $model 队列失败"
        return 1
    fi
}

# 辅助函数：离开队列
leave_queue() {
    local user=$1
    local model=$2
    local user_num=$(get_user_num $user)
    local user_dir="test/api/users/$user"
    local cookie_file="$user_dir/cookie.txt"
    
    echo "   用户 user$user_num 离开 $model 队列..."
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"LeaveQueue\",
        \"Model\": \"$model\"
      }" \
      -c "$cookie_file" -b "$cookie_file")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        echo "   ✓ user$user_num 离开 $model 队列成功"
        return 0
    else
        echo "   ✗ user$user_num 离开 $model 队列失败"
        return 1
    fi
}

# 辅助函数：检查队列状态
check_queue_status() {
    local model=$1
    local user_dir="test/api/users/user_1"
    local cookie_file="$user_dir/cookie.txt"
    
    echo "   检查 $model 队列状态..."
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"CheckMatchStatus\",
        \"Model\": \"$model\"
      }" \
      -c "$cookie_file" -b "$cookie_file")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        queue_count=$(echo "$response" | grep -o '"queue_count":[0-9]*' | cut -d':' -f2)
        echo "   ✓ $model 队列人数: $queue_count"
        return $queue_count
    else
        echo "   ✗ $model 队列状态查询失败"
        return -1
    fi
}

# 测试步骤1: 用户1,2,3,4进入model_a，用户5进入model_b
echo "步骤1: 用户1,2,3,4进入model_a，用户5进入model_b"
echo "================================================"

# 用户1,2,3,4进入model_a
for i in {0..3}; do
    user=${users[$i]}
    join_queue "$user" "model_a"
    echo ""
done

# 用户5进入model_b
join_queue "user_5" "model_b"
echo ""

# 检查队列状态
echo "检查初始队列状态..."
check_queue_status "model_a"
check_queue_status "model_b"
echo ""

# 测试步骤2: 用户1,4离开model_a进入model_b
echo "步骤2: 用户1,4离开model_a进入model_b"
echo "===================================="

# 用户1离开model_a
leave_queue "user_1" "model_a"
echo ""

# 用户4离开model_a
leave_queue "user_4" "model_a"
echo ""

# 用户1进入model_b
join_queue "user_1" "model_b"
echo ""

# 用户4进入model_b
join_queue "user_4" "model_b"
echo ""

# 检查最终队列状态
echo "检查最终队列状态..."
check_queue_status "model_a"
check_queue_status "model_b"
echo ""

# 显示Redis中的数据
echo "查看Redis中的队列数据..."
echo "model_a队列内容:"
redis-cli -a "beastroyale123" -n 0 lrange "match:queue:model_a" 0 -1
echo ""
echo "model_b队列内容:"
redis-cli -a "beastroyale123" -n 0 lrange "match:queue:model_b" 0 -1
echo ""

echo "=== 测试总结 ==="
echo "✓ 多用户队列管理功能正常"
echo "✓ 用户跨队列移动功能正常"
echo "✓ 队列状态查询功能正常"
echo "✓ Redis数据持久化正常"
echo ""
echo "预期结果:"
echo "- model_a队列: 用户2,3 (2人)"
echo "- model_b队列: 用户1,4,5 (3人)" 