#!/bin/bash

# 测试覆盖率工具脚本
# 用于生成和查看测试覆盖率报告

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Webhook 测试覆盖率工具 ===${NC}\n"

# 检查参数
MODE="${1:-all}"
OUTPUT_DIR="${2:-coverage}"

case "$MODE" in
  "all")
    echo -e "${YELLOW}运行所有测试并生成覆盖率报告...${NC}"
    go test -v -coverprofile=coverage.out -covermode=atomic ./...
    ;;
  "server")
    echo -e "${YELLOW}运行 server 包测试并生成覆盖率报告...${NC}"
    go test -v -coverprofile=coverage.out -covermode=atomic ./internal/server/...
    ;;
  "hook")
    echo -e "${YELLOW}运行 hook 包测试并生成覆盖率报告...${NC}"
    go test -v -coverprofile=coverage.out -covermode=atomic ./internal/hook/...
    ;;
  "critical")
    echo -e "${YELLOW}运行关键场景测试...${NC}"
    go test -v -coverprofile=coverage.out -covermode=atomic ./internal/server -run "TestConcurrent|TestCommand|TestPath|TestStress|TestLoad"
    ;;
  "html")
    echo -e "${YELLOW}生成 HTML 覆盖率报告...${NC}"
    if [ ! -f "coverage.out" ]; then
      echo -e "${RED}错误: coverage.out 文件不存在，请先运行测试${NC}"
      exit 1
    fi
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}HTML 报告已生成: coverage.html${NC}"
    ;;
  "func")
    echo -e "${YELLOW}生成函数级别覆盖率报告...${NC}"
    if [ ! -f "coverage.out" ]; then
      echo -e "${RED}错误: coverage.out 文件不存在，请先运行测试${NC}"
      exit 1
    fi
    go tool cover -func=coverage.out
    ;;
  "clean")
    echo -e "${YELLOW}清理覆盖率文件...${NC}"
    rm -f coverage.out coverage.html
    echo -e "${GREEN}已清理${NC}"
    ;;
  *)
    echo -e "${RED}用法: $0 [all|server|hook|critical|html|func|clean]${NC}"
    echo ""
    echo "模式说明:"
    echo "  all      - 运行所有测试并生成覆盖率报告（默认）"
    echo "  server   - 只运行 server 包测试"
    echo "  hook     - 只运行 hook 包测试"
    echo "  critical - 只运行关键场景测试"
    echo "  html     - 生成 HTML 覆盖率报告（需要先运行测试）"
    echo "  func     - 显示函数级别覆盖率（需要先运行测试）"
    echo "  clean    - 清理覆盖率文件"
    exit 1
    ;;
esac

# 如果生成了 coverage.out，显示简要统计
if [ -f "coverage.out" ] && [ "$MODE" != "html" ] && [ "$MODE" != "func" ] && [ "$MODE" != "clean" ]; then
  echo ""
  echo -e "${GREEN}=== 覆盖率统计 ===${NC}"
  go tool cover -func=coverage.out | tail -1
  echo ""
  echo -e "${YELLOW}提示: 运行 '$0 html' 生成 HTML 报告${NC}"
fi

