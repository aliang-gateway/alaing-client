#!/bin/bash

# 测试脚本：验证 VLESS REALITY 代理是否正确转发流量

echo "=========================================="
echo "测试 VLESS REALITY 代理"
echo "=========================================="

PROXY="http://127.0.0.1:56432"

echo ""
echo "1. 测试访问 www.microsoft.com (REALITY SNI 目标)"
echo "------------------------------------------"
curl -v -x "$PROXY" https://www.microsoft.com 2>&1 | head -30

echo ""
echo ""
echo "2. 测试访问 www.google.com (非 REALITY SNI 目标)"
echo "------------------------------------------"
curl -v -x "$PROXY" https://www.google.com 2>&1 | head -30

echo ""
echo ""
echo "3. 测试访问 www.github.com (另一个非 REALITY SNI 目标)"
echo "------------------------------------------"
curl -v -x "$PROXY" https://www.github.com 2>&1 | head -30

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="

