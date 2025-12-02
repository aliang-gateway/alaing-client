# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a network proxy gateway that uses TUN device technology (utun on macOS) to intercept and route network traffic. The core functionality captures all client traffic through a virtual network interface, processes it according to configuration rules, and forwards it either directly or through proxy servers (including AI acceleration endpoints).

**Key Capabilities:**
- TUN-based全设备流量捕获（取代旧的HTTP代理方式）
- 多协议代理支持：VLESS、Shadowsocks、Hysteria、HTTP、Direct
- HTTPS流量解密（MITM）：生成证书对、安装到系统信任存储
- HTTP/2流量处理：自定义header字段注入
- 配置驱动的智能路由决策
- 

## Target Architecture (Post-Refactoring)

### Functional Module Organization

```
nursorgate2/
├── inbound/                    # 入站流量处理层
│   ├── tun/                   # TUN设备管理和流量捕获
│   │   ├── device/           # 平台相关设备实现 (darwin/linux/windows)
│   │   ├── stack/            # TCP/UDP协议栈处理
│   │   └── capture.go        # 流量捕获核心逻辑
│   └── listener/             # 其他入站监听器（可选，如HTTP代理入口）
│
├── outbound/                   # 出站流量处理层
│   └── proxy/                # 代理协议实现
│       ├── interfaces.go     # 通用代理接口定义
│       ├── vless/           # VLESS协议实现
│       ├── shadowsocks/     # Shadowsocks协议实现
│       ├── hysteria/        # Hysteria协议实现
│       ├── http/            # HTTP代理协议实现
│       └── direct/          # 直连（无代理）实现
│
├── processor/                  # 核心处理逻辑层
│   ├── routing/              # 流量路由决策
│   │   ├── engine.go        # 路由引擎核心
│   │   └── rules.go         # 路由规则管理
│   ├── http2/               # HTTP/2流量处理
│   │   ├── injector.go      # HTTP/2 header注入
│   │   └── modifier.go      # HTTP/2流量修改
│   └── tls/                 # TLS/证书管理
│       ├── cert_manager.go  # 证书生成
│       └── installer.go     # 系统证书安装
│
├── common/                     # 公共模块
│   ├── config/               # 配置管理
│   │   ├── app/             # 应用级配置
│   │   └── proxy/           # 代理路由配置
│   ├── logger/              # 日志系统 (支持HTTP/SQLite/Sentry)
│   └── model/               # 数据模型定义
│
├── server/                     # 业务协调层
│   └── business.go          # 主业务逻辑协调器
│
├── websocket/                  # WebSocket服务（用于Flutter前端日志展示）
│
└── deprecated/                 # 已弃用代码归档
    └── install/              # 旧HTTP代理 + Cursor JS修改逻辑
```

## Current State (Before Refactoring)

### Existing Structure Issues

```
client/
├── server/
│   ├── tun/                  # ⚠️ 混合了设备管理、路由引擎、代理协议
│   │   ├── core/            # TUN设备核心
│   │   ├── engine/          # 路由引擎（应移到processor）
│   │   └── proxy/           # 代理协议（应移到outbound）
│   ├── helper/              # ⚠️ 证书、HTTP2注入混在一起
│   ├── outbound/            # 部分outbound代码
│   └── install/             # ⚠️ 已弃用但未归档
└── ...

common/
└── config/                  # ⚠️ 应用配置和代理配置混在一起
```

**主要问题：**
1. **职责不清**：TUN设备管理、路由决策、代理协议混在 `client/server/tun/` 下
2. **弃用代码**：`client/install/` 中的旧HTTP代理和Cursor JS修改代码仍存在
3. **配置混乱**：应用配置和代理路由配置耦合在 `common/config/`
4. **Helper过载**：证书生成、安装、HTTP2注入功能缺乏清晰API边界

## Key Components & Traffic Flow

### Traffic Processing Pipeline

```
客户端应用
    ↓
【inbound/tun】捕获所有流量（虚拟网卡）
    ↓
【processor/routing】路由决策引擎（根据配置规则）
    ↓
【processor/http2】HTTP/2 header注入（可选）
    ↓
【processor/tls】HTTPS解密处理（可选，MITM证书）
    ↓
【outbound/proxy】代理协议转发
    ├── vless://      # VLESS协议
    ├── ss://         # Shadowsocks
    ├── hysteria://   # Hysteria
    ├── http://       # HTTP代理
    └── direct://     # 直连
    ↓
目标服务器
```

### Module Responsibilities

#### inbound/ - 入站流量层
**职责：** 捕获和接收流量，不做任何业务决策

- **TUN设备管理**：创建虚拟网卡（utun），读取IP包
- **TCP/UDP协议栈**：解析传输层协议
- **平台适配**：处理macOS/Linux/Windows差异

**关键文件（当前位置）：**
- `client/server/tun/core/` → 将移至 `inbound/tun/`
- `client/server/tun/core/device/` → 将移至 `inbound/tun/device/`

#### outbound/ - 出站代理层
**职责：** 实现各种代理协议的流量转发

- **统一接口**：所有代理协议实现通用 `Proxy` 接口
- **协议独立**：每个协议独立包，互不依赖
- **配置驱动**：根据配置实例化对应协议handler

**关键文件（当前位置）：**
- `client/server/tun/proxy/*.go` → 将移至 `outbound/proxy/*/`
- `client/outbound/http2_client.go` → 整合到 `outbound/`

#### processor/ - 核心处理层
**职责：** 路由决策、流量修改、证书管理等业务逻辑

**子模块：**
1. **routing/**：流量路由引擎
   - 解析IP包，提取目标地址、端口、协议
   - 根据规则决定使用哪个代理或直连
   - 当前位置：`client/server/tun/engine/`

2. **http2/**：HTTP/2流量处理
   - 注入自定义header字段（流量标识、修改）
   - HTTP/2帧解析和重组
   - 当前位置：`client/server/helper/` (部分)

3. **tls/**：TLS/证书管理
   - 生成MITM证书对（用于HTTPS解密）
   - 安装证书到系统信任存储（跨平台）
   - TLS SNI处理
   - 当前位置：`client/server/helper/tls_sni_helper.go`

#### common/ - 公共模块
**职责：** 跨模块共享的基础设施

- **config/app/**：应用级配置（日志级别、WebSocket端口等）
- **config/proxy/**：代理路由配置（规则、服务器列表）
- **logger/**：统一日志系统（支持多种输出后端）
- **model/**：数据模型定义

#### server/ - 业务协调层
**职责：** 组装各模块，协调整体业务流程

- 初始化各模块（inbound, processor, outbound）
- 启动TUN设备
- 建立模块间的数据流管道
- 处理优雅关闭和错误恢复

**关键文件（当前位置）：**
- `client/server/bussiness_server.go` → 将移至 `server/business.go`

## Development Commands

### Build
```bash
go build -o nursorgate2 .
```

### Run Tests
```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./inbound/tun/...
go test ./outbound/proxy/vless/...
go test ./processor/routing/...

# 运行特定测试文件（带详细输出）
go test -v ./inbound/tun/device_test.go
```

### Run Application
```bash
# 需要root权限创建TUN设备
sudo ./nursorgate2

# 带调试日志运行
sudo ./nursorgate2 --log-level=debug
```

### Platform-Specific Notes
```bash
# macOS: 检查TUN设备
ifconfig | grep utun

# 查看路由表（确认流量被TUN捕获）
netstat -rn

# 清理已安装的MITM证书（测试时）
# macOS: Keychain Access.app -> System -> 搜索证书名 -> 删除
```

## Refactoring Guide

### Refactoring Principles

1. **最小破坏原则**：逐步迁移，每步都能编译和测试
2. **接口先行**：先定义清晰的模块接口，再移动实现
3. **保留兼容**：暂时保留旧代码路径，直到新模块稳定
4. **测试驱动**：每次移动后立即运行相关测试
5. **文档同步**：代码移动后立即更新导入路径和文档

### Refactoring Steps (Recommended Order)

#### Phase 1: 创建新目录结构
```bash
mkdir -p inbound/tun/{device,stack}
mkdir -p outbound/proxy/{vless,shadowsocks,hysteria,http,direct}
mkdir -p processor/{routing,http2,tls}
mkdir -p common/config/{app,proxy}
mkdir -p server
mkdir -p websocket
mkdir -p deprecated
```

#### Phase 2: 定义模块接口
1. 创建 `outbound/proxy/interfaces.go`：定义通用 `Proxy` 接口
2. 创建 `processor/routing/interfaces.go`：定义路由引擎接口
3. 创建 `processor/tls/interfaces.go`：定义证书管理接口

#### Phase 3: 迁移 outbound（代理协议层）
**优先级：高** | **风险：中**

```bash
# 逐个协议迁移
client/server/tun/proxy/vless.go → outbound/proxy/vless/vless.go
client/server/tun/proxy/shadowsocks.go → outbound/proxy/shadowsocks/shadowsocks.go
# ... 其他协议类似
```

**迁移检查清单：**
- [ ] 更新包名和导入路径
- [ ] 实现统一的 `Proxy` 接口
- [ ] 运行协议相关测试
- [ ] 更新配置加载逻辑

#### Phase 4: 迁移 processor（处理逻辑层）
**优先级：高** | **风险：高**

```bash
# 路由引擎
client/server/tun/engine/ → processor/routing/

# 证书管理
client/server/helper/tls_sni_helper.go → processor/tls/cert_manager.go
# 证书安装逻辑拆分到 processor/tls/installer.go

# HTTP/2处理
client/server/helper/http*_*.go → processor/http2/
```

**注意事项：**
- 路由引擎是核心，改动需格外谨慎
- 证书管理涉及系统调用，需针对每个平台测试
- HTTP/2注入逻辑可能与代理协议有耦合，需仔细解耦

#### Phase 5: 迁移 inbound（TUN设备层）
**优先级：中** | **风险：高**

```bash
client/server/tun/core/ → inbound/tun/
client/server/tun/core/device/ → inbound/tun/device/
```

**警告：**
- TUN设备是底层核心，任何错误都会导致无法捕获流量
- 必须在每个目标平台（macOS/Linux/Windows）上测试
- 确保设备创建、读写、关闭流程完整

#### Phase 6: 拆分配置模块
**优先级：中** | **风险：低**

```bash
# 分离应用配置和代理配置
common/config/ → common/config/app/     # 日志、WebSocket等应用配置
common/config/ → common/config/proxy/   # 代理规则、服务器列表
```

#### Phase 7: 归档弃用代码
**优先级：低** | **风险：无**

```bash
client/install/ → deprecated/install/
# 添加 deprecated/README.md 说明归档原因
```

#### Phase 8: 更新主业务逻辑
**优先级：高** | **风险：中**

```bash
client/server/bussiness_server.go → server/business.go
# 更新所有模块导入路径
# 重构模块初始化和协调逻辑
```

### Critical Files Reference

| 原路径 | 新路径 | 优先级 | 功能 |
|-------|-------|-------|------|
| `client/server/tun/engine/engine.go` | `processor/routing/engine.go` | 高 | 路由决策核心 |
| `client/server/tun/proxy/vless.go` | `outbound/proxy/vless/vless.go` | 高 | VLESS协议 |
| `client/server/helper/tls_sni_helper.go` | `processor/tls/cert_manager.go` | 高 | 证书管理 |
| `client/server/bussiness_server.go` | `server/business.go` | 高 | 业务协调 |
| `client/server/tun/core/` | `inbound/tun/` | 中 | TUN设备管理 |
| `client/outbound/http2_client.go` | `outbound/http2/client.go` | 中 | HTTP/2客户端 |

## Protocol Implementation Guide

### Adding New Proxy Protocol

当需要添加新的代理协议时（例如：Trojan、VMess等）：

1. **在 `outbound/proxy/` 创建新包**
   ```
   outbound/proxy/trojan/
   ├── trojan.go          # 协议核心实现
   ├── config.go          # 协议配置结构
   └── trojan_test.go     # 单元测试
   ```

2. **实现 `Proxy` 接口**
   ```go
   type Proxy interface {
       // Dial 建立到目标的代理连接
       Dial(target string) (net.Conn, error)

       // Close 关闭代理连接
       Close() error

       // ProtocolName 返回协议名称
       ProtocolName() string
   }
   ```

3. **注册到路由引擎**
   ```go
   // processor/routing/engine.go
   func (e *Engine) RegisterProxy(name string, proxy Proxy) {
       e.proxies[name] = proxy
   }
   ```

4. **添加配置支持**
   ```go
   // common/config/proxy/config.go
   type ProxyConfig struct {
       Type   string                 // "vless", "trojan", etc.
       Config map[string]interface{} // 协议特定配置
   }
   ```

5. **编写测试**
   ```bash
   go test ./outbound/proxy/trojan/...
   ```

## Security Considerations

### MITM Certificate Management

**重要：** 此项目生成和安装MITM证书用于解密HTTPS流量，涉及安全敏感操作。

**最佳实践：**
1. **证书生成**：
   - 使用强加密算法（RSA 2048+或ECC P-256+）
   - 证书有效期不超过1年
   - 私钥严格保护，不可泄露

2. **证书安装**：
   - 明确告知用户正在安装根证书
   - 仅在必要时安装（用户主动启用HTTPS解密功能）
   - 提供卸载脚本

3. **流量处理**：
   - 解密后的流量数据不落盘（内存处理）
   - 日志中不记录敏感数据（密码、Token等）
   - 遵守用户隐私和当地法律法规

**平台差异：**
- **macOS**: 使用 `security` 命令操作Keychain
- **Linux**: 需要更新 `/etc/ssl/certs/` 和运行 `update-ca-certificates`
- **Windows**: 使用 `certutil` 或Windows API操作证书存储

## Testing Strategy

### Test Organization

```
nursorgate2/
├── inbound/tun/
│   ├── device_test.go         # 设备创建、读写测试
│   └── integration_test.go    # TUN设备集成测试（需root）
├── outbound/proxy/
│   ├── vless/vless_test.go    # VLESS协议单元测试
│   └── integration_test.go    # 多协议集成测试
├── processor/
│   ├── routing/engine_test.go # 路由决策测试
│   └── tls/cert_test.go       # 证书生成测试（不涉及系统安装）
```

### Running Tests

```bash
# 快速测试（单元测试，无需特殊权限）
go test ./outbound/... ./processor/...

# 完整测试（包括TUN设备，需要root）
sudo go test ./...

# 带覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 特定平台测试
GOOS=darwin go test ./inbound/tun/device/...
```

## Common Issues & Troubleshooting

### TUN Device Issues

**问题：无法创建TUN设备**
```
Error: failed to create TUN device: operation not permitted
```
**解决：** 需要root/管理员权限运行
```bash
sudo ./nursorgate2
```

**问题：macOS上TUN设备编号冲突**
```
Error: utun3 already in use
```
**解决：** 程序会自动寻找可用编号（utun0-utun9），检查是否有其他VPN软件占用

### Proxy Connection Issues

**问题：代理连接失败**
```
Error: dial proxy: connection refused
```
**检查清单：**
1. 代理服务器地址和端口配置是否正确
2. 代理服务器是否在线（ping测试）
3. 防火墙是否阻止连接
4. 代理协议配置是否匹配（UUID、密码等）

### Certificate Issues

**问题：HTTPS解密失败**
```
Error: certificate verify failed
```
**解决：**
1. 检查MITM证书是否正确安装到系统信任存储
2. macOS: Keychain Access -> System，确认证书状态为"信任"
3. 重新生成和安装证书：
   ```bash
   ./nursorgate2 --regen-cert
   ```

## Performance Optimization Tips

1. **连接池复用**：代理连接使用连接池，避免频繁建立TCP连接
2. **零拷贝**：在TUN设备读写时使用buffer池，减少内存分配
3. **协程池**：限制并发协程数量，避免过度并发导致系统资源耗尽
4. **配置缓存**：路由规则编译后缓存，避免每次请求都重新解析
5. **日志异步**：日志写入使用异步方式，避免阻塞主流程

## Contributing Guidelines

### Code Style
- 遵循 Go 官方代码规范：`gofmt`, `go vet`
- 使用有意义的变量名，避免单字母变量（除循环计数器外）
- 每个导出函数必须有清晰的注释说明用途、参数、返回值

### Commit Message Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type类型：**
- `feat`: 新功能
- `fix`: Bug修复
- `refactor`: 代码重构（不改变功能）
- `docs`: 文档更新
- `test`: 测试代码
- `chore`: 构建/工具链变更

**示例：**
```
feat(outbound): add Trojan protocol support

- Implement Trojan protocol handler
- Add Trojan configuration model
- Add unit tests for Trojan dialing

Closes #123
```

### Pull Request Checklist
- [ ] 代码通过 `go fmt` 和 `go vet` 检查
- [ ] 添加了相应的单元测试
- [ ] 测试覆盖率不低于80%
- [ ] 更新了相关文档（README.md, CLAUDE.md）
- [ ] 在至少一个目标平台上测试通过（macOS/Linux/Windows）

## References

- [tun2socks项目](https://github.com/xjasonlyu/tun2socks) - TUN设备实现参考
- [VLESS协议规范](https://github.com/XTLS/VLESS)
- [Shadowsocks协议](https://shadowsocks.org/)
- [Hysteria协议](https://hysteria.network/)
- Go网络编程最佳实践：https://golang.org/doc/effective_go#concurrency

---

**最后更新时间：** 2025-12-02
**维护者：** 项目团队
**文档版本：** 2.0 (Refactoring Edition)
