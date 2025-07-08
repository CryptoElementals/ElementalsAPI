#!/bin/bash

# 生成以太坊私钥和地址的脚本
echo "=== 生成以太坊私钥和地址 ==="

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
USERS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/users"

# 生成5个私钥和对应的地址
for i in {1..5}; do
    echo "生成第 $i 个私钥..."
    
    # 为每个用户创建独立的文件夹
    USER_DIR="$USERS_DIR/user_$i"
    mkdir -p "$USER_DIR"
    
    # 生成64位随机十六进制私钥
    PRIVATE_KEY=$(openssl rand -hex 32)
    
    # 保存私钥到用户文件夹
    echo "$PRIVATE_KEY" > "$USER_DIR/private.key"
    
    # 使用Go脚本计算地址
    echo "  计算地址..."
    cd "$PROJECT_ROOT"
    ADDRESS_OUTPUT=$(go run test/api/tools/calculate_address.go -private "$PRIVATE_KEY" 2>/dev/null)
    ADDRESS=$(echo "$ADDRESS_OUTPUT" | grep "地址:" | cut -d':' -f2 | tr -d ' ')
    
    if [ -n "$ADDRESS" ]; then
        # 保存地址到用户文件夹
        echo "$ADDRESS" > "$USER_DIR/address.txt"
        
        echo "  私钥: ${PRIVATE_KEY:0:20}..."
        echo "  地址: $ADDRESS"
        echo "  用户文件夹: $USER_DIR"
        echo "  ├── private.key"
        echo "  └── address.txt"
    else
        echo "  ✗ 地址计算失败"
        echo "  调试信息: $ADDRESS_OUTPUT"
    fi
    
    echo ""
done

echo "=== 生成完成 ==="
echo "用户文件夹位置: test/api/users/"
echo ""
echo "注意：这些是测试用的私钥，请勿在生产环境中使用！" 