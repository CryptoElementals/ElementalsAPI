#!/bin/bash

# 测试GetGamePhase API功能和ConfirmBattle API功能
# 批量查询user_1到user_5的游戏状态，并为user_3,4,5进行战斗确认

# 配置
BASE_URL="http://localhost:8080"
COOKIE_DIR="./test/api/users"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
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

# 获取用户的MatchID
get_match_id() {
    local user=$1
    local cookie_file=$(check_cookie $user)
    
    data='{
        "Action": "GetGamePhase",
        "RequestUUID": "get-match-'$user'"
    }'
    
    response=$(send_request "/" "$data" "$cookie_file")
    
    # 从响应中提取MatchId
    match_id=$(echo "$response" | grep -o '"MatchId":"[^"]*"' | cut -d'"' -f4)
    echo "$match_id"
}

# 确认战斗
confirm_battle() {
    local user=$1
    local match_id=$2
    local cookie_file=$(check_cookie $user)
    
    if [ -z "$match_id" ]; then
        log_warn "${user}: 没有MatchID，跳过确认战斗"
        return 1
    fi
    
    log_info "${user}: 确认战斗，MatchID: $match_id"
    
    data='{
        "Action": "ConfirmBattle",
        "MatchId": "'$match_id'",
        "RequestUUID": "confirm-'$user'"
    }'
    
    response=$(send_request "/" "$data" "$cookie_file")
    log_info "${user}确认战斗响应: $response"
    
    if echo "$response" | grep -q '"RetCode":0'; then
        log_info "✓ ${user}: 确认战斗成功"
        return 0
    else
        log_error "✗ ${user}: 确认战斗失败"
        return 1
    fi
}

log_step "开始测试GetGamePhase和ConfirmBattle API功能..."

# 步骤1: 查看初始状态
log_step "步骤1: 查看所有用户的初始游戏状态"
echo "================================================"

for i in {1..5}; do
    USER="user_$i"
    log_info "=== 测试${USER}的游戏状态 ==="
    cookie=$(check_cookie $USER)
    data='{
        "Action": "GetGamePhase",
        "RequestUUID": "test-phase-'$USER'"
    }'
    response=$(send_request "/" "$data" "$cookie")
    log_info "${USER}游戏状态: $response"
    echo ""
done

# 步骤2: 为user_3,4,5进行战斗确认
log_step "步骤2: 为user_3,4,5进行战斗确认"
echo "================================================"

for i in {3..5}; do
    USER="user_$i"
    log_info "=== 为${USER}确认战斗 ==="
    
    # 获取MatchID
    match_id=$(get_match_id $USER)
    
    if [ -n "$match_id" ]; then
        # 确认战斗
        confirm_battle $USER $match_id
    else
        log_warn "${USER}: 没有找到MatchID，可能没有匹配记录"
    fi
    echo ""
done

# 步骤3: 查看确认后的状态
log_step "步骤3: 查看确认战斗后的状态"
echo "================================================"

for i in {1..5}; do
    USER="user_$i"
    log_info "=== 测试${USER}的游戏状态 ==="
    cookie=$(check_cookie $USER)
    data='{
        "Action": "GetGamePhase",
        "RequestUUID": "test-phase-after-'$USER'"
    }'
    response=$(send_request "/" "$data" "$cookie")
    log_info "${USER}游戏状态: $response"
    echo ""
done

log_info "批量测试完成！" 