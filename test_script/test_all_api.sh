#!/bin/bash

# 设置错误时退出
set -e

echo "=== 开始自动化API测试 ==="

#重启服务
echo "重启服务..."
./restart.sh
echo "重启完成"

# 生成以太坊密钥
echo "生成以太坊密钥..."
./test/api/tools/generate_eth_keys.sh
echo "密钥生成完成"

# 多用户登录
echo "执行多用户登录..."
./test_script/multi_login.sh
echo "多用户登录完成"

echo "=== 开始API测试 ==="

# 1.健康检查API
echo "1.测试健康检查API..."
echo "请求: HealthCheck"
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{
    "Action": "HealthCheck", 
    "CheckConnection": true
  }')
echo "响应: $response"
echo ""

# 2. 列出头像API
echo "2. 测试列出头像API..."
echo "请求: ListAvatars"
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{
    "Action": "ListAvatars"
  }')
echo "响应:"
echo "$response" | jq -C
echo ""

# 3. 获取卡片API
echo "3. 测试获取卡片API..."
echo "请求: GetCards"
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{
    "Action": "GetCards"
  }')
echo "响应:"
echo "$response" | jq -C
echo ""

# 4. 用户档案API测试
echo "4. 测试用户档案API..."

# 循环测试用户1-2的档案操作
for i in {1..2}; do
  echo "4.$i 用户$i档案操作..."
  user_address=$(cat ./test/api/users/user_$i/address.txt | tr -d '\n')
  echo "用户$i地址: $user_address"

  echo "4.$i.1 获取用户$i档案..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetUserProfile\",
      \"Address\": \"$user_address\"
    }")
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "4.$i.2 设置用户$i档案..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"SetUserProfile\",
      \"Name\": \"User${i}_set\",
      \"AvatarURL\": \"https://us3.example.com/avatars/default_avatar_${i}.png\"
    }" \
    -b ./test/api/users/user_$i/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "4.$i.3 再次获取用户$i档案..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetUserProfile\",
      \"Address\": \"$user_address\"
    }")
  echo "响应:"
  echo "$response" | jq -C
  echo ""
done

# 5. 每日奖励API测试
echo "5. 测试每日奖励API..."

# 循环测试用户1-2的每日奖励操作
for i in {1..2}; do
  echo "5.$i 用户$i每日奖励操作..."
  
  echo "5.$i.1 检查用户$i是否已领取每日奖励..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d '{
      "Action": "HasCollectedDailyReward"
    }' \
    -b ./test/api/users/user_$i/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "5.$i.2 领取用户$i每日奖励..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d '{
      "Action": "CollectDailyReward"
    }' \
    -b ./test/api/users/user_$i/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "5.$i.3 再次检查用户$i是否已领取每日奖励..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d '{
      "Action": "HasCollectedDailyReward"
    }' \
    -b ./test/api/users/user_$i/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""
done

# 6. SSE连接测试 - SubscribeGameInfo
echo "6. 测试SSE连接 - SubscribeGameInfo..."

# 创建日志目录
mkdir -p ./test_script/logs

# 清空之前的日志文件
echo "6.0 清空之前的日志文件..."
> ./test_script/logs/user1_sse.log
> ./test_script/logs/user2_sse.log
echo "日志文件已清空"

# 清理旧的SSE连接
echo "6.0.1 清理旧的SSE连接..."
pkill -f "curl.*sse" 2>/dev/null || true
echo "旧的SSE连接已清理"

# 获取用户地址
user1_temp_address="0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E"
user2_temp_address="0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c"

echo "6.1 启动用户1的SSE连接..."
echo "用户1地址: $user1_temp_address"
nohup curl --silent -N -X POST "http://localhost:8080/sse" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "{
    \"Action\": \"SubscribeGameInfo\",
    \"TempAddress\": \"$user1_temp_address\"
  }" \
  -b ./test/api/users/user_1/cookie.txt 2>&1  > ./test_script/logs/user1_sse.log &
user1_pid=$!
echo "用户1 SSE连接已启动，PID: $user1_pid，日志文件: ./test_script/logs/user1_sse.log"

echo "6.2 启动用户2的SSE连接..."
echo "用户2地址: $user2_temp_address"
nohup curl --silent -N -X POST "http://localhost:8080/sse" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "{
    \"Action\": \"SubscribeGameInfo\",
    \"TempAddress\": \"$user2_temp_address\"
  }" \
  -b ./test/api/users/user_2/cookie.txt 2>&1  > ./test_script/logs/user2_sse.log &
user2_pid=$!
echo "用户2 SSE连接已启动，PID: $user2_pid，日志文件: ./test_script/logs/user2_sse.log"

echo "6.3 等待SSE连接建立..."
sleep 3

echo "6.4 检查SSE连接状态..."
if ps -p $user1_pid > /dev/null; then
  echo "用户1 SSE连接运行中 (PID: $user1_pid)"
else
  echo "用户1 SSE连接已停止"
fi

if ps -p $user2_pid > /dev/null; then
  echo "用户2 SSE连接运行中 (PID: $user2_pid)"
else
  echo "用户2 SSE连接已停止"
fi

echo "6.5 显示最近的日志内容..."
echo "用户1 SSE日志 (最近10行):"
tail -n 10 ./test_script/logs/user1_sse.log
echo ""

echo "用户2 SSE日志 (最近10行):"
tail -n 10 ./test_script/logs/user2_sse.log
echo ""

# 6. 匹配和对战API测试
echo "6. 测试匹配和对战API..."

echo "6.1 用户1加入匹配队列..."
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"JoinQueue\",
    \"Mode\": \"PvP\",
    \"TempAddress\": \"$user2_temp_address\"
  }" \
  -b ./test/api/users/user_2/cookie.txt)
echo "响应:"
echo "$response" | jq -C
echo ""

echo "用户1离开匹配队列..."
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"ExitQueue\",
    \"Mode\": \"PvP\",
    \"TempAddress\": \"$user2_temp_address\"
  }" \
  -b ./test/api/users/user_2/cookie.txt)
echo "响应:"
echo "$response" | jq -C
echo ""

echo "6.2 用户2加入匹配队列..."
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"JoinQueue\",
    \"Mode\": \"PvP\",
    \"TempAddress\": \"$user1_temp_address\"
  }" \
  -b ./test/api/users/user_1/cookie.txt)
echo "响应:"
echo "$response" | jq -C
echo ""

echo "6.1 用户1再次加入匹配队列..."
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"JoinQueue\",
    \"Mode\": \"PvP\",
    \"TempAddress\": \"$user2_temp_address\"
  }" \
  -b ./test/api/users/user_2/cookie.txt)
echo "响应:"
echo "$response" | jq -C
echo ""

echo "6.3 等待匹配完成..."
sleep 5

echo "6.4 检查SSE连接状态..."
if ps -p $user1_pid > /dev/null; then
  echo "用户1 SSE连接运行中 (PID: $user1_pid)"
else
  echo "用户1 SSE连接已停止"
fi

if ps -p $user2_pid > /dev/null; then
  echo "用户2 SSE连接运行中 (PID: $user2_pid)"
else
  echo "用户2 SSE连接已停止"
fi

echo "6.4 获取用户1游戏阶段信息..."
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"GetGamePhase\",
    \"TempAddress\": \"$user1_temp_address\"
  }" \
  -b ./test/api/users/user_1/cookie.txt)
echo "响应:"
echo "$response" | jq -C
echo ""

echo "6.5 获取用户2游戏阶段信息..."
response=$(curl -s -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d "{
    \"Action\": \"GetGamePhase\",
    \"TempAddress\": \"$user2_temp_address\"
  }" \
  -b ./test/api/users/user_2/cookie.txt)
echo "响应:"
echo "$response" | jq -C
echo ""


# 从响应中提取GameID（如果存在）
game_id=$(echo "$response" | jq -r '.PvPInfo.GameID // empty')
if [ -n "$game_id" ] && [ "$game_id" != "null" ]; then
  echo "检测到游戏ID: $game_id"
  

  #round 1
  echo "6.5 用户1确认对战..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"ConfirmBattle\",
      \"GameID\": $game_id,
      \"Round\": 1,
      \"TempAddress\": \"$user1_temp_address\"
    }" \
    -b ./test/api/users/user_1/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "6.6 用户2确认对战..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"ConfirmBattle\",
      \"GameID\": $game_id,
      \"Round\": 1,
      \"TempAddress\": \"$user2_temp_address\"
    }" \
    -b ./test/api/users/user_2/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  sleep 3

  echo "查看用户1游戏阶段信息..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetGamePhase\",
      \"TempAddress\": \"$user1_temp_address\"
    }" \
    -b ./test/api/users/user_1/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "查看用户2游戏阶段信息..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetGamePhase\",
      \"TempAddress\": \"$user2_temp_address\"
    }" \
    -b ./test/api/users/user_2/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""
  
  # 从响应中提取ContractAddress
  contract_address=$(echo "$response" | jq -r '.PvPInfo.ContractAddress // empty')
  if [ -n "$contract_address" ] && [ "$contract_address" != "null" ]; then
    echo "检测到合约地址: $contract_address"
  else
    echo "未检测到合约地址"
  fi

  echo "6.7 用户1提交哈希..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
    -u "" \
    -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
    -n 1 \
    submit-hash 1 2 3
  echo ""

  echo "6.8 用户2提交哈希..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
    -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
    -n 1 \
    submit-hash 2 5 3
  echo ""

  echo "6.9 用户1提交卡牌..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
    -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
    -n 1 \
    submit-cards 1 2 3
  echo ""

  echo "6.10 用户2提交卡牌..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
    -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
    -n 1 \
    submit-cards 2 5 3
  echo ""

  sleep 5

  #round 2
    echo "6.5 用户1确认对战..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"ConfirmBattle\",
      \"GameID\": $game_id,
      \"Round\": 2,
      \"TempAddress\": \"$user1_temp_address\"
    }" \
    -b ./test/api/users/user_1/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "6.6 用户2确认对战..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"ConfirmBattle\",
      \"GameID\": $game_id,
      \"Round\": 2,
      \"TempAddress\": \"$user2_temp_address\"
    }" \
    -b ./test/api/users/user_2/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  sleep 3

  echo "查看用户1游戏阶段信息..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetGamePhase\",
      \"TempAddress\": \"$user1_temp_address\"
    }" \
    -b ./test/api/users/user_1/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "查看用户2游戏阶段信息..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetGamePhase\",
      \"TempAddress\": \"$user2_temp_address\"
    }" \
    -b ./test/api/users/user_2/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""
  
  # 从响应中提取ContractAddress
  contract_address=$(echo "$response" | jq -r '.PvPInfo.ContractAddress // empty')
  if [ -n "$contract_address" ] && [ "$contract_address" != "null" ]; then
    echo "检测到合约地址: $contract_address"
  else
    echo "未检测到合约地址"
  fi

  echo "6.7 用户1提交哈希..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
    -u "" \
    -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
    -n 2 \
    submit-hash 1 2 3
  echo ""

  echo "6.8 用户2提交哈希..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
    -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
    -n 2 \
    submit-hash 3 4 2
  echo ""

  echo "6.9 用户1提交卡牌..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
    -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
    -n 2 \
    submit-cards 1 2 3
  echo ""

  echo "6.10 用户2提交卡牌..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
    -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
    -n 2 \
    submit-cards 3 4 2
  echo ""

  sleep 5

  #round 3
    echo "6.5 用户1确认对战..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"ConfirmBattle\",
      \"GameID\": $game_id,
      \"Round\": 3,
      \"TempAddress\": \"$user1_temp_address\"
    }" \
    -b ./test/api/users/user_1/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "6.6 用户2确认对战..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"ConfirmBattle\",
      \"GameID\": $game_id,
      \"Round\": 3,
      \"TempAddress\": \"$user2_temp_address\"
    }" \
    -b ./test/api/users/user_2/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  sleep 3

  echo "查看用户1游戏阶段信息..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetGamePhase\",
      \"TempAddress\": \"$user1_temp_address\"
    }" \
    -b ./test/api/users/user_1/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  echo "查看用户2游戏阶段信息..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetGamePhase\",
      \"TempAddress\": \"$user2_temp_address\"
    }" \
    -b ./test/api/users/user_2/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""
  
  # 从响应中提取ContractAddress
  contract_address=$(echo "$response" | jq -r '.PvPInfo.ContractAddress // empty')
  if [ -n "$contract_address" ] && [ "$contract_address" != "null" ]; then
    echo "检测到合约地址: $contract_address"
  else
    echo "未检测到合约地址"
  fi

  echo "6.7 用户1提交哈希..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
    -u "" \
    -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
    -n 3 \
    submit-hash 1 2 3
  echo ""

  echo "6.8 用户2提交哈希..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
    -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
    -n 3 \
    submit-hash 2 5 3
  echo ""

  echo "6.9 用户1提交卡牌..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
    -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
    -n 3 \
    submit-cards 1 2 3
  echo ""

  echo "6.10 用户2提交卡牌..."
  ./bin/ele-apiserver submitter-test \
    -a "$contract_address" \
    -u "" \
    -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
    -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
    -n 3 \
    submit-cards 2 5 3
  echo ""

  sleep 5

  #end game
  echo "6.11 获取对战信息..."
  response=$(curl -s -X POST "http://localhost:8080/" \
    -H "Content-Type: application/json" \
    -d "{
      \"Action\": \"GetBattleInfo\",
      \"GameID\": $game_id,
      \"Round\": 3,
      \"TempAddress\": \"$user1_temp_address\"
    }" \
    -b ./test/api/users/user_1/cookie.txt)
  echo "响应:"
  echo "$response" | jq -C
  echo ""

  

  # 检查IsGameOver字段
  is_game_over=$(echo "$response" | jq -r '.RoundResult.IsGameOver // false')
  echo "游戏是否结束: $is_game_over"
  
  if [ "$is_game_over" = "true" ]; then
    echo "游戏已结束，执行ContinueGame测试..."
    
    echo "6.12 用户1继续游戏..."
    response=$(curl -s -X POST "http://localhost:8080/" \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"ContinueGame\",
        \"GameID\": $game_id,
        \"TempAddress\": \"$user1_temp_address\"
      }" \
      -b ./test/api/users/user_1/cookie.txt)
    echo "响应:"
    echo "$response" | jq -C
    echo ""

    echo "6.13 用户2继续游戏..."
    response=$(curl -s -X POST "http://localhost:8080/" \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"ContinueGame\",
        \"GameID\": $game_id,
        \"TempAddress\": \"$user2_temp_address\"
      }" \
      -b ./test/api/users/user_2/cookie.txt)
    echo "响应:"
    echo "$response" | jq -C
    echo ""

    echo "6.14 等待ContinueGame处理完成..."
    sleep 3

    echo "6.15 获取用户1新的游戏阶段信息..."
    response=$(curl -s -X POST "http://localhost:8080/" \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"GetGamePhase\",
        \"TempAddress\": \"$user1_temp_address\"
      }" \
      -b ./test/api/users/user_1/cookie.txt)
    echo "响应:"
    echo "$response" | jq -C
    echo ""

    echo "6.16 获取用户2新的游戏阶段信息..."
    response=$(curl -s -X POST "http://localhost:8080/" \
      -H "Content-Type: application/json" \
      -d "{
        \"Action\": \"GetGamePhase\",
        \"TempAddress\": \"$user2_temp_address\"
      }" \
      -b ./test/api/users/user_2/cookie.txt)
    echo "响应:"
    echo "$response" | jq -C
    echo ""

    # 从响应中提取新的GameID和ContractAddress
    new_game_id=$(echo "$response" | jq -r '.PvPInfo.GameID // empty')
    new_contract_address=$(echo "$response" | jq -r '.PvPInfo.ContractAddress // empty')
    
    if [ -n "$new_game_id" ] && [ "$new_game_id" != "null" ]; then
      echo "检测到新的游戏ID: $new_game_id"
      
      if [ -n "$new_contract_address" ] && [ "$new_contract_address" != "null" ]; then
        echo "检测到新的合约地址: $new_contract_address"
        

        echo "6.19 用户1提交哈希 (第二场)..."
        ./bin/ele-apiserver submitter-test \
          -a "$new_contract_address" \
          -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
          -u "" \
          -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
          -n 1 \
          submit-hash 1 2 3
        echo ""

        echo "6.20 用户2提交哈希 (第二轮)..."
        ./bin/ele-apiserver submitter-test \
          -a "$new_contract_address" \
          -u "" \
          -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
          -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
          -n 1 \
          submit-hash 2 5 4
        echo ""

        echo "6.21 用户1提交卡牌 (第二轮)..."
        ./bin/ele-apiserver submitter-test \
          -a "$new_contract_address" \
          -u "" \
          -t "0x098C0EE7Bd0DA785bBceE99a5ddb7CEbe697283E" \
          -p "940d59b61cc683c423f7727fc5e46f943067680965f5fe161b48c76c59b57a78" \
          -n 1 \
          submit-cards 1 2 3
        echo ""

        echo "6.22 用户2提交卡牌 (第二轮)..."
        ./bin/ele-apiserver submitter-test \
          -a "$new_contract_address" \
          -u "" \
          -t "0x16f30e7f6B8Ea4c75405cB9ad95B14CCf2Ac518c" \
          -p "5236f8d7223c6fa1a0087b4a28988974f5fff5a55cfa4140e6a7db279022024d" \
          -n 1 \
          submit-cards 2 5 4
        echo ""

        sleep 5

        echo "6.23 获取第二轮对战信息..."
        response=$(curl -s -X POST "http://localhost:8080/" \
          -H "Content-Type: application/json" \
          -d "{
            \"Action\": \"GetBattleInfo\",
            \"GameID\": $new_game_id,
            \"Round\": 1,
            \"TempAddress\": \"$user1_temp_address\"
          }" \
          -b ./test/api/users/user_1/cookie.txt)
        echo "响应:"
        echo "$response" | jq -C
        echo ""

        echo "第二轮游戏完成！"
      else
        echo "未检测到新的合约地址，跳过第二轮游戏"
      fi
    else
      echo "未检测到新的游戏ID，跳过第二轮游戏"
    fi
  else
    echo "游戏未结束，跳过ContinueGame测试"
  fi

else
  echo "未检测到游戏ID，跳过对战相关测试"
fi

echo "=== API测试完成 ==="
