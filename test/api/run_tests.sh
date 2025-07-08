#!/bin/bash

# 测试运行示例脚本
echo "=== BeastRoyale Backend 测试运行示例 ==="
echo ""

echo "请选择要运行的测试类型："
echo "1. 单用户登录测试 (使用现有私钥)"
echo "2. 多用户登录测试 (使用新生成私钥)"
echo "3. 生成新的私钥和地址"
echo "4. 创建用户0文件夹 (移动现有文件)"
echo "5. 清理测试文件"
echo "6. 退出"
echo ""

read -p "请输入选择 (1-6): " choice

case $choice in
    1)
        echo ""
        echo "=== 运行单用户登录测试 ==="
        echo "使用用户0文件夹: test/api/users/user_0/"
        echo "使用固定地址: 0xD84bCB296882241EBc75770E7b47E08049aEcc3b"
        echo ""
        ./test/api/script/test_login.sh
        ;;
    2)
        echo ""
        echo "=== 运行多用户登录测试 ==="
        
        # 检查是否已生成私钥
        if [ ! -d "test/api/users" ] || [ ! -f "test/api/keys_config.sh" ]; then
            echo "✗ 未找到生成的用户文件夹，请先运行选项3生成私钥"
            exit 1
        fi
        
        echo "使用新生成的私钥: test/api/users/user_X/private.key"
        echo "每个用户都有独立的文件夹，包含所有相关文件"
        echo ""
        ./test/api/script/test_multi_login.sh
        ;;
    3)
        echo ""
        echo "=== 生成新的私钥和地址 ==="
        echo "这将生成5个新的以太坊私钥和对应的地址"
        echo "每个用户将有自己的文件夹: test/api/users/user_X/"
        echo ""
        ./test/api/generate_eth_keys.sh
        echo ""
        echo "=== 更新配置文件 ==="
        ./test/api/update_config.sh
        ;;
    4)
        echo ""
        echo "=== 创建用户0文件夹 ==="
        echo "将现有的单用户测试文件移动到用户0文件夹"
        echo ""
        ./test/api/create_user_0.sh
        ;;
    5)
        echo ""
        echo "=== 清理测试文件 ==="
        ./test/api/cleanup_test_files.sh
        ;;
    6)
        echo "退出"
        exit 0
        ;;
    *)
        echo "无效选择，请重新运行脚本"
        exit 1
        ;;
esac

echo ""
echo "=== 测试完成 ===" 