#!/bin/bash

# 快速部署向导 Bug 修复验证脚本

echo "=================================="
echo "  快速部署向导 - Bug修复验证"
echo "=================================="
echo ""

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}修复的问题:${NC}"
echo ""
echo "  1. ✓ 路由模式选择逻辑混乱"
echo "     - 改用 '是'/'否' 按钮"
echo "     - 提示文字更清晰"
echo ""
echo "  2. ✓ 无法重复部署（页面ID冲突）"  
echo "     - 每个对话框使用唯一ID"
echo "     - 可以无限次重复部署"
echo ""
echo "  3. ✓ 配置展示增加颜色区分"
echo "     - 全局模式：紫色"
echo "     - 分流模式：青色"
echo ""

echo -e "${YELLOW}测试检查清单:${NC}"
echo ""
echo "□ 测试1: 路由模式选择"
echo "   1. 启动向导，配置到步骤3"
echo "   2. 选择 '是' → 应该设置全局模式"
echo "   3. 查看日志确认显示 '全局'"
echo ""
echo "□ 测试2: 重复部署"
echo "   1. 完成第一次部署"
echo "   2. 再次进入向导"
echo "   3. 配置并确认部署"
echo "   4. 应该正常执行，不会显示'已取消'"
echo ""
echo "□ 测试3: 中途取消再重新开始"
echo "   1. 启动向导，在步骤2按ESC取消"
echo "   2. 再次进入向导"
echo "   3. 应该能正常工作"
echo ""

read -p "按 Enter 开始测试..."

# 检查二进制文件
BIN="/config/Desktop/test1/tls-vpn/bin/tls-vpn"
if [ ! -f "$BIN" ]; then
    echo -e "${RED}✗ 未找到二进制文件${NC}"
    exit 1
fi

# 显示文件信息
echo ""
echo -e "${GREEN}二进制文件信息:${NC}"
ls -lh "$BIN" | awk '{print "  大小: " $5 "  修改: " $6 " " $7 " " $8}'
echo ""

read -p "按 Enter 启动程序进行测试..."

# 启动程序
if [ "$EUID" -eq 0 ]; then
    "$BIN"
else
    echo ""
    echo -e "${YELLOW}提示: 服务端部署需要 root 权限${NC}"
    echo "请使用: sudo $0"
    echo ""
    read -p "以普通用户继续（仅能测试客户端）? [y/N]: " cont
    if [ "$cont" = "y" ] || [ "$cont" = "Y" ]; then
        "$BIN"
    fi
fi

echo ""
echo -e "${GREEN}测试完成${NC}"
echo ""
echo "请确认:"
echo "  ✓ 路由模式选择是否正确"
echo "  ✓ 是否能重复部署"
echo "  ✓ 配置信息是否有颜色"
