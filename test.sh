#!/bin/bash

# Quick test script for HTTP proxy server

set -e

PROJECT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$PROJECT_DIR"

# Get config path from argument or use default
CONFIG_PATH="${1:-./config.test.json}"

echo ""
echo "========================================================================"
echo "HTTP Proxy Server Test Script"
echo "========================================================================"
echo ""
echo "Configuration file: $CONFIG_PATH"
echo ""

# Check if config file exists
if [ ! -f "$CONFIG_PATH" ]; then
    echo "Warning: Config file '$CONFIG_PATH' not found"
    echo "Using default configuration..."
    echo ""
fi

# Run the test with environment variable and no timeout
echo "Starting HTTP proxy server on http://127.0.0.1:56432"
echo ""
echo "Test commands:"
echo "  curl -x http://127.0.0.1:56432 https://www.google.com"
echo "  curl -x http://127.0.0.1:56432 -v https://www.example.com"
echo ""
echo "Press Ctrl+C to stop"
echo "========================================================================"
echo ""

CONFIG_PATH="$CONFIG_PATH" go test -v -run TestHTTPProxyWithConfig ./test -timeout 0
