#!/bin/bash

# 测试GetMatchInfo API功能
# 通过GetGamePhase获取MatchId，然后调用GetMatchInfo获取匹配详情

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

# 获取匹配信息
get_match_info() {
    local user=$1
    local match_id=$2
    local cookie_file=$(check_cookie $user)
    
    if [ -z "$match_id" ]; then
        log_warn "${user}: 没有MatchID，无法获取匹配信息"
        return 1
    fi
    
    log_info "${user}: 获取匹配信息，MatchID: $match_id"
    
    data='{
        "Action": "GetMatchInfo",
        "MatchId": "'$match_id'",
        "RequestUUID": "get-match-info-'$user'"
    }'
    
    response=$(send_request "/" "$data" "$cookie_file")
    log_info "${user}匹配信息响应: $response"
    
    if echo "$response" | grep -q '"RetCode":0'; then
        log_info "✓ ${user}: 获取匹配信息成功"
        return 0
    else
        log_error "✗ ${user}: 获取匹配信息失败"
        return 1
    fi
}

# 测试非参与者访问匹配信息
test_unauthorized_access() {
    local user=$1
    local match_id=$2
    local cookie_file=$(check_cookie $user)
    
    if [ -z "$match_id" ]; then
        log_warn "${user}: 没有MatchID，跳过未授权访问测试"
        return 1
    fi
    
    log_info "${user}: 测试未授权访问匹配信息，MatchID: $match_id"
    
    data='{
        "Action": "GetMatchInfo",
        "MatchId": "'$match_id'",
        "RequestUUID": "unauthorized-test-'$user'"
    }'
    
    response=$(send_request "/" "$data" "$cookie_file")
    log_info "${user}未授权访问响应: $response"
    
    if echo "$response" | grep -q '"RetCode":1003'; then
        log_info "✓ ${user}: 未授权访问被正确拒绝"
        return 0
    else
        log_error "✗ ${user}: 未授权访问测试失败"
        return 1
    fi
}

log_step "开始测试GetMatchInfo API功能..."

# 步骤1: 查看所有用户的游戏状态
log_step "步骤1: 查看所有用户的游戏状态"
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

# 步骤2: 获取有匹配记录用户的MatchId并测试GetMatchInfo
log_step "步骤2: 获取匹配信息"
echo "================================================"

# 存储第一个有效的MatchId用于后续测试
first_match_id=""

for i in {1..5}; do
    USER="user_$i"
    log_info "=== 测试${USER}的匹配信息 ==="
    
    # 获取MatchID
    match_id=$(get_match_id $USER)
    
    if [ -n "$match_id" ]; then
        # 保存第一个有效的MatchId
        if [ -z "$first_match_id" ]; then
            first_match_id="$match_id"
        fi
        
        # 获取匹配信息
        get_match_info $USER $match_id
    else
        log_warn "${USER}: 没有找到MatchID，可能没有匹配记录"
    fi
    echo ""
done

# 步骤3: 测试未授权访问（使用user_1访问user_2的匹配信息）
log_step "步骤3: 测试未授权访问"
echo "================================================"

if [ -n "$first_match_id" ]; then
    # 使用user_1访问user_2的匹配信息（假设user_1不是该匹配的参与者）
    log_info "使用user_1访问MatchId: $first_match_id"
    test_unauthorized_access "user_1" "$first_match_id"
else
    log_warn "没有找到有效的MatchId，跳过未授权访问测试"
fi

# 步骤4: 测试无效的MatchId
log_step "步骤4: 测试无效的MatchId"
echo "================================================"

USER="user_1"
cookie=$(check_cookie $USER)
invalid_match_id="invalid-match-id-12345"

log_info "${USER}: 测试无效MatchId: $invalid_match_id"

data='{
    "Action": "GetMatchInfo",
    "MatchId": "'$invalid_match_id'",
    "RequestUUID": "test-invalid-match-'$USER'"
}'

response=$(send_request "/" "$data" "$cookie")
log_info "${USER}无效MatchId响应: $response"

if echo "$response" | grep -q '"RetCode":1002'; then
    log_info "✓ ${USER}: 无效MatchId被正确拒绝"
else
    log_error "✗ ${USER}: 无效MatchId测试失败"
fi

log_info "GetMatchInfo API测试完成！" 