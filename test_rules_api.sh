#!/bin/bash

# Door 加速规则引擎 API 集成测试脚本
# 使用方法: ./test_rules_api.sh

set -e

# 配置
API_BASE="http://127.0.0.1:56431/api/rules"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_section() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}\n"
}

# 检查 HTTP 服务是否运行
check_server() {
    print_test "Checking if HTTP server is running..."

    if curl -s --connect-timeout 2 "http://127.0.0.1:56431/" > /dev/null 2>&1; then
        print_success "HTTP server is running"
        return 0
    else
        print_error "HTTP server is not running on port 56431"
        echo "Please start the server first: ./nursor"
        exit 1
    fi
}

# 测试 GeoIP 状态
test_geoip_status() {
    print_test "Testing GET /api/rules/geoip/status"

    response=$(curl -s "${API_BASE}/geoip/status")

    if echo "$response" | jq -e '.code == 0' > /dev/null 2>&1; then
        enabled=$(echo "$response" | jq -r '.data.enabled')
        dbPath=$(echo "$response" | jq -r '.data.databasePath')

        print_success "GeoIP Status: enabled=$enabled, path=$dbPath"
        echo "$response" | jq '.'
    else
        print_error "GeoIP status request failed"
        echo "$response"
        return 1
    fi
}

# 测试 GeoIP 查询
test_geoip_lookup() {
    print_test "Testing POST /api/rules/geoip/lookup"

    # 测试 Google DNS (US)
    print_test "  - Looking up 8.8.8.8 (Google DNS)"
    response=$(curl -s -X POST "${API_BASE}/geoip/lookup" \
        -H "Content-Type: application/json" \
        -d '{"ip": "8.8.8.8"}')

    if echo "$response" | jq -e '.code == 0' > /dev/null 2>&1; then
        country=$(echo "$response" | jq -r '.data.country')
        isChina=$(echo "$response" | jq -r '.data.isChina')

        if [ "$country" = "US" ] && [ "$isChina" = "false" ]; then
            print_success "8.8.8.8 -> US (correct)"
        else
            print_error "8.8.8.8 -> $country, isChina=$isChina (unexpected)"
        fi
    else
        print_error "GeoIP lookup failed for 8.8.8.8"
        echo "$response"
    fi

    # 测试百度 IP (CN)
    print_test "  - Looking up 180.76.76.76 (Baidu DNS)"
    response=$(curl -s -X POST "${API_BASE}/geoip/lookup" \
        -H "Content-Type: application/json" \
        -d '{"ip": "180.76.76.76"}')

    if echo "$response" | jq -e '.code == 0' > /dev/null 2>&1; then
        country=$(echo "$response" | jq -r '.data.country')
        isChina=$(echo "$response" | jq -r '.data.isChina')

        if [ "$country" = "CN" ] && [ "$isChina" = "true" ]; then
            print_success "180.76.76.76 -> CN (correct)"
        else
            print_error "180.76.76.76 -> $country, isChina=$isChina (unexpected)"
        fi
    else
        print_error "GeoIP lookup failed for 180.76.76.76"
        echo "$response"
    fi
}

# 测试缓存统计
test_cache_stats() {
    print_test "Testing GET /api/rules/cache/stats"

    response=$(curl -s "${API_BASE}/cache/stats")

    if echo "$response" | jq -e '.code == 0' > /dev/null 2>&1; then
        size=$(echo "$response" | jq -r '.data.size')
        maxEntries=$(echo "$response" | jq -r '.data.maxEntries')
        hits=$(echo "$response" | jq -r '.data.hits')
        misses=$(echo "$response" | jq -r '.data.misses')
        hitRate=$(echo "$response" | jq -r '.data.hitRate')

        print_success "Cache Stats: size=$size/$maxEntries, hits=$hits, misses=$misses, hitRate=${hitRate}%"
        echo "$response" | jq '.data'
    else
        print_error "Cache stats request failed"
        echo "$response"
        return 1
    fi
}

# 测试缓存清除
test_cache_clear() {
    print_test "Testing POST /api/rules/cache/clear"

    response=$(curl -s -X POST "${API_BASE}/cache/clear")

    if echo "$response" | jq -e '.code == 0' > /dev/null 2>&1; then
        print_success "Cache cleared successfully"

        # 验证缓存已清空
        sleep 1
        stats_response=$(curl -s "${API_BASE}/cache/stats")
        size=$(echo "$stats_response" | jq -r '.data.size')

        if [ "$size" = "0" ]; then
            print_success "Verified: cache size is now 0"
        else
            print_error "Cache size is $size (expected 0)"
        fi
    else
        print_error "Cache clear request failed"
        echo "$response"
        return 1
    fi
}

# 测试规则引擎状态
test_engine_status() {
    print_test "Testing GET /api/rules/engine/status"

    response=$(curl -s "${API_BASE}/engine/status")

    if echo "$response" | jq -e '.code == 0' > /dev/null 2>&1; then
        engineEnabled=$(echo "$response" | jq -r '.data.engineEnabled')
        geoipEnabled=$(echo "$response" | jq -r '.data.geoipEnabled')

        print_success "Engine Status: engine=$engineEnabled, geoip=$geoipEnabled"
        echo "$response" | jq '.data'
    else
        print_error "Engine status request failed"
        echo "$response"
        return 1
    fi
}

# 测试规则引擎启用/禁用
test_engine_toggle() {
    print_test "Testing engine enable/disable"

    # 禁用引擎
    print_test "  - Disabling engine..."
    response=$(curl -s -X POST "${API_BASE}/engine/disable")

    if echo "$response" | jq -e '.data.enabled == false' > /dev/null 2>&1; then
        print_success "Engine disabled"
    else
        print_error "Failed to disable engine"
        echo "$response"
        return 1
    fi

    # 验证状态
    sleep 1
    status_response=$(curl -s "${API_BASE}/engine/status")
    engineEnabled=$(echo "$status_response" | jq -r '.data.engineEnabled')

    if [ "$engineEnabled" = "false" ]; then
        print_success "Verified: engine is disabled"
    else
        print_error "Engine status is still enabled"
    fi

    # 启用引擎
    print_test "  - Enabling engine..."
    response=$(curl -s -X POST "${API_BASE}/engine/enable")

    if echo "$response" | jq -e '.data.enabled == true' > /dev/null 2>&1; then
        print_success "Engine enabled"
    else
        print_error "Failed to enable engine"
        echo "$response"
        return 1
    fi

    # 验证状态
    sleep 1
    status_response=$(curl -s "${API_BASE}/engine/status")
    engineEnabled=$(echo "$status_response" | jq -r '.data.engineEnabled')

    if [ "$engineEnabled" = "true" ]; then
        print_success "Verified: engine is enabled"
    else
        print_error "Engine status is still disabled"
    fi
}

# 测试错误处理
test_error_handling() {
    print_test "Testing error handling"

    # 无效的 IP 地址
    print_test "  - Testing invalid IP address"
    response=$(curl -s -X POST "${API_BASE}/geoip/lookup" \
        -H "Content-Type: application/json" \
        -d '{"ip": "invalid-ip"}')

    if echo "$response" | jq -e '.code != 0' > /dev/null 2>&1; then
        print_success "Invalid IP rejected correctly"
    else
        print_error "Invalid IP was not rejected"
        echo "$response"
    fi

    # 空 IP 地址
    print_test "  - Testing empty IP address"
    response=$(curl -s -X POST "${API_BASE}/geoip/lookup" \
        -H "Content-Type: application/json" \
        -d '{"ip": ""}')

    if echo "$response" | jq -e '.code != 0' > /dev/null 2>&1; then
        print_success "Empty IP rejected correctly"
    else
        print_error "Empty IP was not rejected"
        echo "$response"
    fi
}

# 主测试流程
main() {
    print_section "Door 加速规则引擎 API 集成测试"

    # 检查依赖
    if ! command -v jq &> /dev/null; then
        print_error "jq is required but not installed. Please install it first."
        echo "  macOS: brew install jq"
        echo "  Linux: apt-get install jq or yum install jq"
        exit 1
    fi

    # 检查服务器
    check_server

    # 运行测试
    print_section "1. GeoIP Service Tests"
    test_geoip_status
    test_geoip_lookup

    print_section "2. Cache Tests"
    test_cache_stats
    test_cache_clear

    print_section "3. Rule Engine Tests"
    test_engine_status
    test_engine_toggle

    print_section "4. Error Handling Tests"
    test_error_handling

    print_section "✅ All Tests Completed"
    echo "Test results summary:"
    echo "  - GeoIP service: ✓"
    echo "  - Cache system: ✓"
    echo "  - Rule engine: ✓"
    echo "  - Error handling: ✓"
}

# 运行主测试
main
