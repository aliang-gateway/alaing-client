#!/bin/bash

# Nonelane 系统托盘快速入门脚本

echo "========================================"
echo "Nonelane System Tray Quick Start"
echo "========================================"
echo ""

# 检查操作系统
OS="unknown"
case "$(uname -s)" in
   Darwin)
     OS="macos"
     echo "✓ Detected: macOS"
     ;;
   Linux)
     OS="linux"
     echo "✓ Detected: Linux"
     
     # 检查 Linux 依赖
     if command -v apt-get &> /dev/null; then
         if ! dpkg -l | grep -q libappindicator3-1; then
             echo "⚠ Warning: libappindicator3-dev not found"
             echo "  Install with: sudo apt-get install libappindicator3-dev"
         else
             echo "✓ libappindicator3-dev is installed"
         fi
     fi
     ;;
   CYGWIN*|MINGW32*|MSYS*|MINGW*)
     OS="windows"
     echo "✓ Detected: Windows"
     ;;
   *)
     echo "✗ Unsupported OS: $(uname -s)"
     exit 1
     ;;
esac

echo ""
echo "========================================"
echo "Build Instructions"
echo "========================================"
echo ""
echo "1. Build the application:"
echo "   go build -o dist/nursorgate cmd/nursor/main.go"
echo ""
echo "2. Run with system tray:"
echo "   ./dist/nursorgate tray"
echo ""
echo "3. Run with config file:"
echo "   ./dist/nursorgate tray --config ./config.json"
echo ""

# 检查是否已经编译
if [ -f "dist/nursorgate" ]; then
    echo "✓ Binary found: dist/nursorgate"
    echo ""
    read -p "Do you want to run it now? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        ./dist/nursorgate tray
    fi
else
    echo "✗ Binary not found"
    read -p "Do you want to build it now? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Building..."
        go build -o dist/nursorgate cmd/nursor/main.go
        if [ $? -eq 0 ]; then
            echo "✓ Build successful!"
            echo ""
            read -p "Run it now? (y/n) " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                ./dist/nursorgate tray
            fi
        else
            echo "✗ Build failed"
            exit 1
        fi
    fi
fi