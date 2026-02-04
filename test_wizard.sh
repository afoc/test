#!/bin/bash

# 快速部署向导测试脚本
# 用于验证快速部署功能的改进

echo "=================================="
echo "  快速部署向导 - 功能测试"
echo "=================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 检查是否以root运行
if [ "$EUID" -ne 0 ]; then 
    echo -e "${YELLOW}提示: 服务端部署需要 root 权限${NC}"
    echo -e "${YELLOW}请使用: sudo $0${NC}"
    echo ""
fi

# 检查二进制文件
BIN_PATH="/config/Desktop/test1/tls-vpn/bin/tls-vpn"
if [ ! -f "$BIN_PATH" ]; then
    echo -e "${RED}✗ 未找到二进制文件: $BIN_PATH${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 找到二进制文件: $BIN_PATH${NC}"
echo ""

# 检查版本信息
echo "二进制文件信息:"
ls -lh "$BIN_PATH" | awk '{print "  大小: " $5 "  修改时间: " $6 " " $7 " " $8}'
echo ""

# 测试菜单
echo "请选择测试项目:"
echo ""
echo "  ${CYAN}1)${NC} 测试服务端快速部署"
echo "  ${CYAN}2)${NC} 测试客户端快速配置"
echo "  ${CYAN}3)${NC} 查看更新说明"
echo "  ${CYAN}4)${NC} 清理测试数据"
echo "  ${CYAN}5)${NC} 退出"
echo ""
read -p "请选择 [1-5]: " choice

case $choice in
    1)
        echo ""
        echo -e "${CYAN}=================================="
        echo "  测试服务端快速部署"
        echo -e "==================================${NC}"
        echo ""
        echo "测试步骤:"
        echo "  1. 程序启动后，选择 '4) 快速向导'"
        echo "  2. 选择 '1) 服务端快速部署'"
        echo "  3. 按提示输入配置:"
        echo "     - 端口: 8080 (或自定义)"
        echo "     - 网段: 10.8.0.0/24 (或自定义)"
        echo "     - 模式: 选择 '否' 使用分流模式"
        echo "     - 确认: 选择 '是' 开始部署"
        echo "  4. 观察部署过程和完成提示"
        echo ""
        echo -e "${YELLOW}关键检查点:${NC}"
        echo "  ✓ 每个步骤显示 '步骤 X/4'"
        echo "  ✓ 端口验证 (输入 99999 应该报错)"
        echo "  ✓ 网段验证 (输入 'abc' 应该报错)"
        echo "  ✓ 部署完成后显示下一步提示"
        echo ""
        read -p "按 Enter 启动程序..."
        sudo "$BIN_PATH"
        ;;
        
    2)
        echo ""
        echo -e "${CYAN}=================================="
        echo "  测试客户端快速配置"
        echo -e "==================================${NC}"
        echo ""
        echo "测试步骤:"
        echo "  1. 程序启动后，选择 '4) 快速向导'"
        echo "  2. 选择 '2) 客户端快速配置'"
        echo "  3. 按提示输入配置:"
        echo "     - 服务器: localhost 或服务器IP"
        echo "     - 端口: 8080 (或自定义)"
        echo "  4. 观察证书检测和后续提示"
        echo ""
        echo -e "${YELLOW}关键检查点:${NC}"
        echo "  ✓ 每个步骤显示 '步骤 X/2'"
        echo "  ✓ 地址不能为空的验证"
        echo "  ✓ 端口验证 (输入 0 应该报错)"
        echo "  ✓ 无证书时显示申请步骤"
        echo "  ✓ 有证书时提供连接选项"
        echo ""
        read -p "按 Enter 启动程序..."
        "$BIN_PATH"
        ;;
        
    3)
        echo ""
        echo -e "${CYAN}查看更新说明${NC}"
        echo ""
        if [ -f "/config/Desktop/test1/tls-vpn/WIZARD_UPDATE.md" ]; then
            less "/config/Desktop/test1/tls-vpn/WIZARD_UPDATE.md"
        else
            echo -e "${RED}✗ 未找到更新说明文件${NC}"
        fi
        ;;
        
    4)
        echo ""
        echo -e "${YELLOW}清理测试数据${NC}"
        echo ""
        echo "将清理以下内容:"
        echo "  - config.json"
        echo "  - certs/ 目录"
        echo "  - tokens/ 目录中的测试Token"
        echo ""
        read -p "确认清理? [y/N]: " confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            cd /config/Desktop/test1/tls-vpn
            [ -f "config.json" ] && rm config.json && echo -e "${GREEN}✓ 已删除 config.json${NC}"
            [ -d "certs" ] && rm -rf certs && echo -e "${GREEN}✓ 已删除 certs/${NC}"
            [ -d "source/tokens" ] && find source/tokens -name "*.json" -type f -mtime -1 -delete && echo -e "${GREEN}✓ 已清理测试Token${NC}"
            echo ""
            echo -e "${GREEN}清理完成！${NC}"
        else
            echo "已取消"
        fi
        ;;
        
    5)
        echo ""
        echo "退出测试"
        exit 0
        ;;
        
    *)
        echo ""
        echo -e "${RED}无效选择${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}测试完成${NC}"
