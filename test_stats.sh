#!/bin/bash

# 高级测试脚本 - 支持多种测试选项

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 默认运行所有测试
TEST_TYPE=${1:-all}

case $TEST_TYPE in
    all)
        echo -e "${BLUE}=== 运行所有统计测试 ===${NC}"
        go test -v ./db -run "TestUpdateCardStatByPlayerIDs|TestUpdateUserStatByPlayerIDs"
        ;;
    card)
        echo -e "${BLUE}=== 运行卡牌统计测试 ===${NC}"
        go test -v ./db -run TestUpdateCardStatByPlayerIDs
        ;;
    user)
        echo -e "${BLUE}=== 运行用户统计测试 ===${NC}"
        go test -v ./db -run TestUpdateUserStatByPlayerIDs
        ;;
    card-new)
        echo -e "${BLUE}=== 运行卡牌统计新玩家测试 ===${NC}"
        go test -v ./db -run TestUpdateCardStatByPlayerIDs_NewPlayer
        ;;
    user-new)
        echo -e "${BLUE}=== 运行用户统计新玩家测试 ===${NC}"
        go test -v ./db -run TestUpdateUserStatByPlayerIDs_NewPlayer
        ;;
    user-multiple)
        echo -e "${BLUE}=== 运行用户统计多玩家测试 ===${NC}"
        go test -v ./db -run TestUpdateUserStatByPlayerIDs_MultiplePlayers
        ;;
    coverage)
        echo -e "${BLUE}=== 运行测试并生成覆盖率报告 ===${NC}"
        go test -v ./db -run "TestUpdateCardStatByPlayerIDs|TestUpdateUserStatByPlayerIDs" -coverprofile=coverage.out
        echo -e "${GREEN}覆盖率报告已生成: coverage.out${NC}"
        echo "使用以下命令查看 HTML 报告:"
        echo "  go tool cover -html=coverage.out"
        ;;
    help|--help|-h)
        echo -e "${YELLOW}用法: $0 [选项]${NC}"
        echo ""
        echo "选项:"
        echo "  all          - 运行所有统计测试（默认）"
        echo "  card         - 只运行卡牌统计测试"
        echo "  user         - 只运行用户统计测试"
        echo "  card-new     - 只运行卡牌统计新玩家测试"
        echo "  user-new     - 只运行用户统计新玩家测试"
        echo "  user-multiple - 只运行用户统计多玩家测试"
        echo "  coverage     - 运行测试并生成覆盖率报告"
        echo "  help         - 显示此帮助信息"
        echo ""
        echo "示例:"
        echo "  $0              # 运行所有测试"
        echo "  $0 card         # 只运行卡牌统计测试"
        echo "  $0 user         # 只运行用户统计测试"
        echo "  $0 coverage     # 生成覆盖率报告"
        ;;
    *)
        echo -e "${RED}错误: 未知选项 '$TEST_TYPE'${NC}"
        echo "使用 '$0 help' 查看帮助信息"
        exit 1
        ;;
esac

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=== 测试完成 ===${NC}"
else
    echo ""
    echo -e "${RED}=== 测试失败 ===${NC}"
    exit 1
fi

