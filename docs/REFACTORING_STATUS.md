# 重构进度报告

## 已完成的工作

### 1. 目录结构创建 ✅
- 创建了新的目录结构：`inbound/`, `outbound/`, `processor/`, `server/`
- 创建了子目录：`outbound/proxy/{vless,shadowsocks,hysteria,http,direct}`
- 创建了处理器目录：`processor/{routing,http2,tls}`

### 2. 代码迁移 ✅
- **Proxy协议层**: `client/server/tun/proxy/*.go` → `outbound/proxy/*/`
  - vless.go → outbound/proxy/vless/
  - shadowsocks.go → outbound/proxy/shadowsocks/
  - hysteria.go → outbound/proxy/hysteria/
  - http.go → outbound/proxy/http/
  - direct.go → outbound/proxy/direct/
  - base.go, proxy.go, reject.go → outbound/proxy/
  - proto/proto.go → outbound/proxy/proto/

- **路由引擎**: `client/server/tun/engine/*.go` → `processor/routing/`
  - engine.go, parse.go, parse_unix.go, parse_windows.go, key.go

- **TLS/证书管理**: `client/server/helper/*.go` → `processor/tls/`
  - tls_sni_helper.go, wrapped_tls_watcher.go, watcher_settings.go

- **HTTP2处理**: `client/server/helper/*.go` → `processor/http2/`
  - http2_wrapper.go, http1_wrapper.go

- **TUN设备**: `client/server/tun/core/*` → `inbound/tun/`
  - 所有TUN设备相关代码

- **业务协调**:
  - `client/server/bussiness_server.go` → `server/bussiness_server.go`
  - `client/export.go` → `export.go`

### 3. 包名更新 ✅
- outbound/proxy/vless: `package proxy` → `package vless`
- outbound/proxy/shadowsocks: `package proxy` → `package shadowsocks`
- outbound/proxy/hysteria: `package proxy` → `package hysteria`
- outbound/proxy/http: `package proxy` → `package http`
- outbound/proxy/direct: `package proxy` → `package direct`
- processor/routing: `package engine` → `package routing`
- processor/tls: `package helper` → `package tls`
- processor/http2: `package helper` → `package http2`
- inbound/tun: `package core` → `package tun`

### 4. 导入路径更新 ✅
- `nursor.org/nursorgate/client/server/tun/proxy` → `nursor.org/nursorgate/outbound/proxy`
- `nursor.org/nursorgate/client/server/tun/engine` → `nursor.org/nursorgate/processor/routing`
- `nursor.org/nursorgate/client/server/helper` → `nursor.org/nursorgate/processor/tls`
- `nursor.org/nursorgate/client/server/tun/core` → `nursor.org/nursorgate/inbound/tun`
- `nursor.org/nursorgate/client/server` → `nursor.org/nursorgate/server`

## 当前问题

### 1. processor/tls 包冲突 ⚠️
**问题**:
- `tls_sni_helper.go` 和 `cert_manager.go` 中有重复的函数声明
- `parseSNIFromBuffer` 和 `ExtractSNI` 函数重复定义

**解决方案**: 需要合并或删除重复的函数定义

### 2. processor/tls 缺少依赖 ⚠️
**问题**:
- `wrapped_tls_watcher.go` 引用了未定义的 `http2Stream` 类型
- 缺少 `processH1ReqHeaders` 和 `processHttp2RequestFrame` 方法
- 缺少 `frameHeaderLen` 常量

**解决方案**: 需要从 `processor/http2` 导入相关类型和方法，或者将这些文件移到正确的位置

### 3. server 包缺少 TUN 接口 ⚠️
**问题**:
- `server/bussiness_server.go` 调用了 `tun.Start()`, `tun.RunStatusChan()`, `tun.Stop()`
- 但 `inbound/tun` 包可能没有导出这些函数

**解决方案**: 需要在 `inbound/tun` 中定义并导出这些接口函数

## 下一步工作

### 优先级1: 修复编译错误
1. **合并 processor/tls 中的重复函数**
   - 检查 `tls_sni_helper.go` 和 `cert_manager.go`
   - 删除重复的函数定义
   - 保留一个版本

2. **修复 processor/tls 的依赖问题**
   - 将 `wrapped_tls_watcher.go` 移到 `processor/http2` (因为它依赖HTTP2相关类型)
   - 或者在 `processor/tls` 中导入 `processor/http2` 的类型

3. **修复 inbound/tun 的接口**
   - 在 `inbound/tun` 中创建 `interfaces.go`
   - 定义并导出 `Start()`, `Stop()`, `RunStatusChan()` 函数

### 优先级2: 清理旧代码
1. 删除或归档 `client/server/tun/` 目录
2. 删除或归档 `client/server/helper/` 目录
3. 更新 `deprecated/` 目录的 README

### 优先级3: 测试和验证
1. 运行 `go build` 确保编译通过
2. 运行 `go test ./...` 确保测试通过
3. 更新测试文件中的导入路径

### 优先级4: 文档更新
1. 更新 CLAUDE.md 中的路径引用
2. 更新 README.md
3. 添加迁移指南

## 文件统计

### 已迁移文件数量
- outbound/proxy: ~10 个文件
- processor/routing: ~5 个文件
- processor/tls: ~3 个文件
- processor/http2: ~2 个文件
- inbound/tun: ~20+ 个文件
- server: 1 个文件

### 待处理文件
- client/test/*.go: 测试文件需要更新导入路径
- client/outbound/*.go: 部分outbound代码可能需要迁移
- client/inbound/*.go: 部分inbound代码可能需要迁移

## 建议

1. **立即修复编译错误**: 优先解决上述3个编译问题
2. **逐步清理**: 不要一次性删除所有旧代码，先确保新代码工作正常
3. **保持兼容**: 在过渡期间，可以在旧位置保留符号链接或转发函数
4. **增量测试**: 每修复一个问题就运行一次编译和测试

## 时间估算

- 修复编译错误: 1-2小时
- 清理旧代码: 30分钟
- 测试验证: 1小时
- 文档更新: 30分钟

**总计**: 约3-4小时可以完成剩余工作
