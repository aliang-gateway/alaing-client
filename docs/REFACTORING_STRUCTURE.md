# 重构目录结构设计

## 完整的重构后目录结构

```
nursorgate2/
├── inbound/                          # 入站流量采集与接收层
│   ├── tun/                         # TUN虚拟网卡相关代码
│   │   ├── device/                  # 平台相关设备实现（darwin/macOS, linux, windows）
│   │   │   ├── device_darwin.go
│   │   │   ├── device_linux.go
│   │   │   └── device_windows.go
│   │   ├── stack/                   # TCP/UDP协议栈处理（拆包、重组）
│   │   │   ├── tcp.go
│   │   │   └── udp.go
│   │   └── capture.go               # 流量捕获核心，设备创建和读写调度
│   └── listener/                   # 其他入站协议监听器（未来扩展，如HTTP代理入口）
│       └── http_listener.go
│
├── outbound/                        # 出站代理协议实现
│   └── proxy/
│       ├── interfaces.go           # 代理协议统一接口定义
│       ├── vless/
│       │   ├── vless.go            # VLESS协议核心实现
│       │   └── vless_test.go
│       ├── shadowsocks/
│       │   ├── shadowsocks.go      # Shadowsocks协议实现
│       │   └── shadowsocks_test.go
│       ├── hysteria/
│       │   ├── hysteria.go
│       │   └── hysteria_test.go
│       ├── http/
│       │   ├── http.go             # HTTP代理实现
│       │   └── http_test.go
│       └── direct/
│           ├── direct.go           # 直连转发实现
│           └── direct_test.go
│
├── processor/                      # 核心业务处理层
│   ├── routing/                   # 路由决策引擎及规则管理
│   │   ├── engine.go              # 路由决策引擎核心
│   │   ├── rules.go               # 路由规则定义与管理
│   │   ├── interfaces.go          # 路由引擎接口定义
│   │   └── engine_test.go
│   ├── http2/                     # HTTP/2相关流量处理（header注入、改写）
│   │   ├── injector.go
│   │   ├── modifier.go
│   │   └── http2_test.go
│   └── tls/                       # TLS / HTTPS证书管理
│       ├── cert_manager.go        # 证书创建与密钥管理
│       ├── installer.go           # 系统证书安装接口及实现
│       ├── interfaces.go          # 证书管理相关接口
│       └── tls_test.go
│
├── common/                        # 公共基础设施
│   ├── config/
│   │   ├── app/                   # 应用层配置（日志、metrics、UI等）
│   │   └── proxy/                 # 代理相关配置（路由、服务器列表、规则）
│   ├── logger/                   # 日志系统（支持HTTP、文件、SQLite、Sentry）
│   │   ├── logger.go
│   │   └── logger_test.go
│   └── model/                    # 数据模型定义（用户、缓存、连接状态等）
│       ├── user.go
│       ├── cache.go
│       └── model_test.go
│
├── server/                       # 应用顶层与业务协调层
│   ├── business.go              # 主业务协调逻辑，模块初始化与调度
│   ├── server.go                # 程序入口及服务生命周期管理
│   └── server_test.go
│
├── websocket/                   # WebSocket服务（用于前端Flutter日志展示）
│   ├── ws_server.go
│   └── ws_handler.go
│
└── deprecated/                  # 废弃代码归档（保留旧实现，方便审查）
    └── install/
        ├── old_http_proxy.go
        ├── cursor_js_modification.go
        └── README.md             # 说明该目录归档目的及内容
```

## 模块职责说明

### inbound/ - 入站流量层
**职责：** 捕获和接收流量，不做任何业务决策
- TUN设备管理：创建虚拟网卡（utun），读取IP包
- TCP/UDP协议栈：解析传输层协议
- 平台适配：处理macOS/Linux/Windows差异

### outbound/ - 出站代理层
**职责：** 实现各种代理协议的流量转发
- 统一接口：所有代理协议实现通用 Proxy 接口
- 协议独立：每个协议独立包，互不依赖
- 配置驱动：根据配置实例化对应协议handler

### processor/ - 核心处理层
**职责：** 路由决策、流量修改、证书管理等业务逻辑
- routing/：流量路由引擎
- http2/：HTTP/2流量处理
- tls/：TLS/证书管理

### common/ - 公共模块
**职责：** 跨模块共享的基础设施
- config/app/：应用级配置
- config/proxy/：代理路由配置
- logger/：统一日志系统
- model/：数据模型定义

### server/ - 业务协调层
**职责：** 组装各模块，协调整体业务流程
- 初始化各模块（inbound, processor, outbound）
- 启动TUN设备
- 建立模块间的数据流管道
- 处理优雅关闭和错误恢复

## 迁移映射表

| 原路径 | 新路径 | 优先级 |
|-------|-------|-------|
| `client/server/tun/core/` | `inbound/tun/` | 中 |
| `client/server/tun/core/device/` | `inbound/tun/device/` | 中 |
| `client/server/tun/engine/` | `processor/routing/` | 高 |
| `client/server/tun/proxy/*.go` | `outbound/proxy/*/` | 高 |
| `client/server/helper/tls_sni_helper.go` | `processor/tls/cert_manager.go` | 高 |
| `client/server/helper/http*.go` | `processor/http2/` | 中 |
| `client/server/bussiness_server.go` | `server/business.go` | 高 |
| `client/outbound/http2_client.go` | `outbound/http2/client.go` | 中 |
| `client/install/` | `deprecated/install/` | 低 |

## 重构步骤

1. ✅ 创建新目录结构
2. ⏳ 定义模块接口（interfaces.go）
3. ⏳ 迁移 outbound 代理协议代码
4. ⏳ 迁移 processor 处理层代码
5. ⏳ 迁移 inbound TUN设备代码
6. ⏳ 迁移 server 业务协调代码
7. ⏳ 归档 deprecated 旧代码
8. ⏳ 更新 go.mod 和导入路径
9. ⏳ 运行测试验证重构
