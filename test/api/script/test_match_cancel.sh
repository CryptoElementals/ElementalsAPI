#!/bin/bash

# 测试匹配取消功能
# 测试场景：
# 1. 两个用户加入队列并匹配成功
# 2. 其中一个用户尝试离开队列（应该失败）
# 3. 用户取消匹配（应该成功）
# 4. 用户再次尝试离开队列（应该成功）

# 配置
BASE_URL="http://localhost:8080"
COOKIE_DIR="../users"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查cookie文件是否存在
check_cookie() {
    local user=$1
    local cookie_file="$COOKIE_DIR/${user}/cookie.txt"
    
    if [ ! -f "$cookie_file" ]; then
        log_error "Cookie文件不存在: $cookie_file"
        log_info "请先运行登录脚本获取cookie"
        exit 1
    fi
    echo "$cookie_file"
}

# 发送API请求
send_request() {
    local endpoint=$1
    local data=$2
    local cookie_file=$3
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -b "$cookie_file" \
        -d "$data" \
        "$BASE_URL$endpoint")
    
    echo "$response"
}

# 测试数据
USER1="user_1"
USER2="user_2"
MODE="PvP"

log_info "开始测试匹配取消功能..."

# 检查cookie文件
cookie1=$(check_cookie $USER1)
cookie2=$(check_cookie $USER2)

log_info "使用用户1: $USER1"
log_info "使用用户2: $USER2"
log_info "使用游戏模式: $MODE"

# 步骤1: 两个用户加入队列
log_info "步骤1: 两个用户加入匹配队列"

# 用户1加入队列
data1='{
    "Action": "JoinQueue",
    "RequestUUID": "test-join-1",
    "Mode": "'$MODE'",
    "PublicKey": "user1_public_key"
}'

response1=$(send_request "/" "$data1" "$cookie1")
log_info "用户1加入队列响应: $response1"

# 用户2加入队列
data2='{
    "Action": "JoinQueue",
    "RequestUUID": "test-join-2",
    "Mode": "'$MODE'",
    "PublicKey": "user2_public_key"
}'

response2=$(send_request "/" "$data2" "$cookie2")
log_info "用户2加入队列响应: $response2"

# 等待匹配完成
log_info "等待匹配完成..."
sleep 3

# 步骤2: 检查匹配状态
log_info "步骤2: 检查匹配状态"

# 检查用户1的匹配状态
status_data1='{
    "Action": "CheckMatchStatus",
    "RequestUUID": "test-status-1",
    "Mode": "'$MODE'"
}'

status_response1=$(send_request "/" "$status_data1" "$cookie1")
log_info "用户1匹配状态: $status_response1"

# 检查用户2的匹配状态
status_data2='{
    "Action": "CheckMatchStatus",
    "RequestUUID": "test-status-2",
    "Mode": "'$MODE'"
}'

status_response2=$(send_request "/" "$status_data2" "$cookie2")
log_info "用户2匹配状态: $status_response2"

# 提取MatchID（如果匹配成功）
match_id=$(echo "$status_response1" | grep -o '"MatchId":"[^"]*"' | cut -d'"' -f4)

if [ -n "$match_id" ]; then
    log_info "匹配成功，MatchID: $match_id"
    
    # 步骤3: 用户1尝试离开队列（应该失败）
    log_info "步骤3: 用户1尝试离开队列（应该失败）"
    
    leave_data1='{
        "Action": "LeaveQueue",
        "RequestUUID": "test-leave-1",
        "Mode": "'$MODE'"
    }'
    
    leave_response1=$(send_request "/" "$leave_data1" "$cookie1")
    log_info "用户1离开队列响应: $leave_response1"
    
    # 步骤4: 用户1取消匹配
    log_info "步骤4: 用户1取消匹配"
    
    cancel_data1='{
        "Action": "CancelMatch",
        "RequestUUID": "test-cancel-1",
        "MatchId": "'$match_id'"
    }'
    
    cancel_response1=$(send_request "/" "$cancel_data1" "$cookie1")
    log_info "用户1取消匹配响应: $cancel_response1"
    
    # 步骤5: 用户1再次尝试离开队列（应该成功）
    log_info "步骤5: 用户1再次尝试离开队列（应该成功）"
    
    leave_data1_retry='{
        "Action": "LeaveQueue",
        "RequestUUID": "test-leave-1-retry",
        "Mode": "'$MODE'"
    }'
    
    leave_response1_retry=$(send_request "/" "$leave_data1_retry" "$cookie1")
    log_info "用户1再次离开队列响应: $leave_response1_retry"
    
else
    log_warn "未检测到匹配成功，跳过后续测试"
fi

# 步骤6: 清理测试数据
log_info "步骤6: 清理测试数据"

# 用户2离开队列
leave_data2='{
    "Action": "LeaveQueue",
    "RequestUUID": "test-leave-2",
    "Mode": "'$MODE'"
}'

leave_response2=$(send_request "/" "$leave_data2" "$cookie2")
log_info "用户2离开队列响应: $leave_response2"

log_info "测试完成！" 