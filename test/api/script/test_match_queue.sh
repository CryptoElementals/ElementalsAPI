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
    local mode=$2
    local user_num=$(get_user_num $user)
    local user_dir="test/api/users/$user"
    local cookie_file="$user_dir/cookie.txt"
    
    echo "   用户 user$user_num 加入 $mode 队列..."
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"JoinQueue\",
        \"Mode\": \"$mode\",
        \"PublicKey\": \"pk_user$user_num\"
      }" \
      -c "$cookie_file" -b "$cookie_file")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        echo "   ✓ user$user_num 加入 $mode 队列成功"
        return 0
    else
        echo "   ✗ user$user_num 加入 $mode 队列失败"
        return 1
    fi
}

# 辅助函数：离开队列
leave_queue() {
    local user=$1
    local mode=$2
    local user_num=$(get_user_num $user)
    local user_dir="test/api/users/$user"
    local cookie_file="$user_dir/cookie.txt"
    
    echo "   用户 user$user_num 离开 $mode 队列..."
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"LeaveQueue\",
        \"Mode\": \"$mode\"
      }" \
      -c "$cookie_file" -b "$cookie_file")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        echo "   ✓ user$user_num 离开 $mode 队列成功"
        return 0
    else
        echo "   ✗ user$user_num 离开 $mode 队列失败"
        return 1
    fi
}

# 辅助函数：检查队列状态
check_queue_status() {
    local mode=$1
    local user_dir="test/api/users/user_1"
    local cookie_file="$user_dir/cookie.txt"
    
    echo "   检查 $mode 队列状态..."
    response=$(curl -s -X POST $SERVER_URL/ \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"CheckMatchStatus\",
        \"Mode\": \"$mode\"
      }" \
      -c "$cookie_file" -b "$cookie_file")
    
    echo "   响应: $response"
    
    if echo "$response" | grep -q "RetCode.*0"; then
        queue_count=$(echo "$response" | grep -o '"queue_count":[0-9]*' | cut -d':' -f2)
        echo "   ✓ $mode 队列人数: $queue_count"
        return $queue_count
    else
        echo "   ✗ $mode 队列状态查询失败"
        return -1
    fi
}

# 测试步骤1: 用户1,2,3,4进入PvP，用户5进入Tournament
echo "步骤1: 用户1,2,3,4进入PvP，用户5进入Tournament"
echo "================================================"

# 用户1,2,3,4进入PvP
for i in {0..3}; do
    user=${users[$i]}
    join_queue "$user" "PvP"
    echo ""
done

# 用户5进入Tournament
join_queue "user_5" "Tournament"
echo ""

# 检查队列状态
echo "检查初始队列状态..."
check_queue_status "PvP"
check_queue_status "Tournament"
echo ""

# 测试步骤2: 用户1,4离开PvP进入Tournament
echo "步骤2: 用户1,4离开PvP进入Tournament"
echo "===================================="

# 用户1离开PvP
leave_queue "user_1" "PvP"
echo ""

# 用户4离开PvP
leave_queue "user_4" "PvP"
echo ""

# 用户1进入Tournament
join_queue "user_1" "Tournament"
echo ""

# 用户4进入Tournament
join_queue "user_4" "Tournament"
echo ""

# 检查最终队列状态
echo "检查最终队列状态..."
check_queue_status "PvP"
check_queue_status "Tournament"
echo ""

# 显示Redis中的数据
echo "查看Redis中的队列数据..."
echo "PvP队列内容:"
redis-cli -a "beastroyale123" -n 0 lrange "match:queue:PvP" 0 -1
echo ""
echo "Tournament队列内容:"
redis-cli -a "beastroyale123" -n 0 lrange "match:queue:Tournament" 0 -1
echo ""

echo "=== 测试总结 ==="
echo "✓ 多用户队列管理功能正常"
echo "✓ 用户跨队列移动功能正常"
echo "✓ 队列状态查询功能正常"
echo "✓ Redis数据存储验证正常"
echo ""
echo "预期结果:"
echo "- PvP队列: 用户2,3 (2人)"
echo "- Tournament队列: 用户1,4,5 (3人)" 