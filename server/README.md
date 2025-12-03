# Server Package 模块化架构

## 📋 目录

- [概述](#概述)
- [文件结构](#文件结构)
- [模块说明](#模块说明)
- [快速查找](#快速查找)
- [添加新功能](#添加新功能)
- [最佳实践](#最佳实践)

## 概述

`server` 包是 Nursor VPN 的 HTTP 服务层，负责处理所有 API 请求。

### 设计哲学

采用 **单一职责原则 (Single Responsibility Principle)** 进行模块化：
- ✅ 每个文件专注于一个业务功能
- ✅ 清晰的模块边界
- ✅ 易于维护和扩展
- ✅ 便于团队协作和并行开发

### 版本历史

**重构前**:
- 单一文件: `bussiness_server.go` (474 行)
- 多个业务混杂，难以维护

**重构后** (当前):
- 10 个专业化模块
- 清晰的职责划分
- 平均文件大小: 50 行
- 完全向后兼容

## 文件结构

```
server/
├─ README.md                         📖 本文件

├─ 🚀 启动入口
│  ├─ http_server.go                HTTP服务启动 + 统一路由注册 ⭐
│  └─ websocket_server.go           WebSocket服务
│
├─ 🔌 功能模块 (Handler)
│  ├─ token_handler.go              Token管理 (/token/*)
│  ├─ run_handler.go                VPN运行控制 (/run/*)
│  ├─ config_handler.go             代理配置查询 (/config/*)
│  ├─ proxy_handler.go              当前代理管理 (/proxy/current/*)
│  ├─ proxy_registry_handler.go      代理注册表管理 (/proxy/registry/*)
│  │
│  └─ 日志相关 (拆分自 inbound/http/logger_api.go)
│     ├─ logger_handler.go          日志处理器
│     ├─ logger_stream.go           WebSocket日志流
│     ├─ logger_service.go          日志业务逻辑
│     ├─ logger_config_service.go   日志配置管理
│     ├─ logger_types.go            日志数据结构
│     ├─ logger_util.go             日志工具函数
│     ├─ logger_routes.go           日志路由注册
│
├─ 🛠️  支持模块
│  ├─ http_types.go                 HTTP类型定义
│  └─ http_util.go                  HTTP工具函数
│
└─ 📦 向后兼容
   └─ bussiness_server.go           (空文件，仅说明用途)
```

## 模块说明

### 🚀 启动入口

#### **http_server.go** (42 行)
**职责**: HTTP 服务器启动和统一路由注册

```go
func StartHttpServer()
// 启动 HTTP 服务器 (127.0.0.1:56431)
// 调用 registerAllRoutes() 注册所有路由
// 启动 WebSocket 服务器

func registerAllRoutes()
// 统一路由注册入口
// 按模块调用各个 RegisterXxxRoutes()
```

**使用**: `main.go` 或 `runner.go` 调用 `StartHttpServer()`

---

#### **websocket_server.go** (58 行)
**职责**: WebSocket 服务器

```go
func StartWebSocketServer()
// 在 127.0.0.1:56433 启动 WebSocket 服务
// 管理客户端连接和断开

func handleWebSocket(w http.ResponseWriter, r *http.Request)
// 处理 /ws 连接
```

**路由**: `WebSocket /ws`

---

### 🔌 功能模块 (Handler)

#### **token_handler.go** (37 行)
**职责**: Token 管理

**函数**:
- `handleTokenSet()` - 保存 outbound token
- `handleTokenGet()` - 获取 outbound token
- `RegisterTokenRoutes()` - 注册路由

**路由**:
```
POST   /token/set          设置token
GET    /token/get          获取token
```

**导入**:
```go
"nursor.org/nursorgate/outbound"
```

---

#### **run_handler.go** (54 行)
**职责**: VPN 运行控制

**函数**:
- `handleRun()` - 启动 VPN
- `handleStop()` - 停止 VPN
- `handleRunUserInfo()` - 设置用户信息
- `RegisterRunRoutes()` - 注册路由

**路由**:
```
POST   /run/start          启动VPN
POST   /run/stop           停止VPN
POST   /run/userInfo       设置用户信息
```

**导入**:
```go
"nursor.org/nursorgate/common/logger"
tun "nursor.org/nursorgate/inbound/tun/engine"
user "nursor.org/nursorgate/processor/auth"
"nursor.org/nursorgate/runner"
```

---

#### **config_handler.go** (35 行)
**职责**: 代理配置查询

**函数**:
- `handleConfigGet()` - 获取指定配置
- `handleConfigList()` - 列出所有配置
- `RegisterConfigRoutes()` - 注册路由

**路由**:
```
GET    /config/get?name=xxx         获取特定代理配置
GET    /config/list                 列出所有配置
```

**导入**:
```go
proxyConfig "nursor.org/nursorgate/processor/config"
```

---

#### **proxy_handler.go** (57 行)
**职责**: 当前代理管理

**函数**:
- `handleGetCurrentProxy()` - 获取当前代理
- `handleSetCurrentProxy()` - 设置当前代理
- `RegisterProxyRoutes()` - 注册路由

**路由**:
```
GET    /proxy/current/get           获取当前代理
POST   /proxy/current/set           设置当前代理
```

**导入**:
```go
proxyRegistry "nursor.org/nursorgate/processor/proxy"
```

---

#### **proxy_registry_handler.go** (168 行) ⭐ 最大模块
**职责**: 代理注册表管理

**函数**:
- `handleProxyRegistryList()` - 列表
- `handleProxyRegistryGet()` - 获取
- `handleProxyRegistryRegister()` - 注册
- `handleProxyRegistryUnregister()` - 注销
- `handleProxyRegistrySetDefault()` - 设置默认
- `handleProxyRegistrySetDoor()` - 设置门代理
- `handleProxyRegistrySwitch()` - 切换
- `RegisterProxyRegistryRoutes()` - 注册路由

**路由**:
```
GET    /proxy/registry/list         列出所有代理
GET    /proxy/registry/get?name=xxx 获取指定代理
POST   /proxy/registry/register     注册代理
POST   /proxy/registry/unregister   注销代理
POST   /proxy/registry/set-default  设置默认
POST   /proxy/registry/set-door     设置门代理
POST   /proxy/registry/switch       切换代理
```

**导入**:
```go
proxyConfig "nursor.org/nursorgate/processor/config"
proxyRegistry "nursor.org/nursorgate/processor/proxy"
```

---

### 📊 日志相关模块

这些模块从 `inbound/http/logger_api.go` 拆分而来。

#### **logger_handler.go**
HTTP 请求处理层

#### **logger_stream.go**
WebSocket 实时日志流

#### **logger_service.go**
日志查询和管理业务逻辑

#### **logger_config_service.go**
日志配置管理业务逻辑

#### **logger_types.go**
日志相关数据结构

#### **logger_util.go**
日志工具函数和错误定义

#### **logger_routes.go**
日志路由统一注册

---

### 🛠️ 支持模块

#### **http_types.go** (15 行)
**职责**: HTTP 通用类型定义

```go
type Response struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data"`
}

type LoginRequest struct {
    RefreshToken string `json:"refreshToken"`
    AccessToken  string `json:"accessToken"`
    SubId        string `json:"subId"`
}
```

---

#### **http_util.go** (52 行)
**职责**: HTTP 工具函数

**函数**:
- `decodeRequest()` - 解析 JSON 请求体
- `sendResponse()` - 发送成功响应
- `sendError()` - 发送错误响应
- `writePortToFile()` - 写入端口到文件

**使用**: 所有 handler 都应使用这些函数

---

## 快速查找

### 我要修改...

| 功能 | 文件 |
|-----|------|
| Token 相关逻辑 | **token_handler.go** |
| VPN 启动/停止 | **run_handler.go** |
| 代理配置查询 | **config_handler.go** |
| 当前代理操作 | **proxy_handler.go** |
| 代理注册表操作 | **proxy_registry_handler.go** |
| HTTP 工具函数 | **http_util.go** |
| 响应类型定义 | **http_types.go** |
| 日志处理 | **logger_handler.go** |
| WebSocket 日志流 | **logger_stream.go** |

### 端点速查表

| 路径 | 方法 | 文件 | 说明 |
|-----|-----|------|------|
| /token/set | POST | token_handler.go | 设置 token |
| /token/get | GET | token_handler.go | 获取 token |
| /run/start | POST | run_handler.go | 启动 VPN |
| /run/stop | POST | run_handler.go | 停止 VPN |
| /run/userInfo | POST | run_handler.go | 设置用户信息 |
| /config/get | GET | config_handler.go | 获取配置 |
| /config/list | GET | config_handler.go | 列表配置 |
| /proxy/current/get | GET | proxy_handler.go | 获取当前代理 |
| /proxy/current/set | POST | proxy_handler.go | 设置当前代理 |
| /proxy/registry/list | GET | proxy_registry_handler.go | 列表代理 |
| /proxy/registry/get | GET | proxy_registry_handler.go | 获取代理 |
| /proxy/registry/register | POST | proxy_registry_handler.go | 注册代理 |
| /proxy/registry/unregister | POST | proxy_registry_handler.go | 注销代理 |
| /proxy/registry/set-default | POST | proxy_registry_handler.go | 设置默认 |
| /proxy/registry/set-door | POST | proxy_registry_handler.go | 设置门代理 |
| /proxy/registry/switch | POST | proxy_registry_handler.go | 切换代理 |
| /api/logs | GET | logger_handler.go | 获取日志 |
| /api/logs/clear | POST | logger_handler.go | 清空日志 |
| /api/logs/level | POST | logger_handler.go | 设置日志级别 |
| /api/logs/config | GET/POST | logger_handler.go | 日志配置 |
| /api/logs/stream | WS | logger_stream.go | 日志流 |
| /ws | WS | websocket_server.go | WebSocket |

## 添加新功能

### 步骤 1: 创建 Handler 文件

创建 `new_feature_handler.go`:

```go
package server

import (
    "net/http"
    // 导入需要的包
)

// handleNewFeature 处理新功能请求
func handleNewFeature(w http.ResponseWriter, r *http.Request) {
    var req struct {
        // 请求体结构
    }

    if err := decodeRequest(r, &req); err != nil {
        sendError(w, "Invalid request body", http.StatusBadRequest, nil)
        return
    }

    // 业务逻辑

    sendResponse(w, result)
}

// RegisterNewFeatureRoutes 注册新功能路由
func RegisterNewFeatureRoutes() {
    http.HandleFunc("/new-feature/endpoint", handleNewFeature)
}
```

### 步骤 2: 在 http_server.go 中注册

修改 `registerAllRoutes()`:

```go
func registerAllRoutes() {
    RegisterTokenRoutes()
    RegisterRunRoutes()
    // ... 其他路由
    RegisterNewFeatureRoutes()  // ← 添加这一行
}
```

### 步骤 3: 编译和测试

```bash
go build ./...
go test ./server/...
```

## 最佳实践

### ✅ 应该做的

```go
// ✅ 在对应 handler 中修改逻辑
// token_handler.go
func handleTokenSet(w http.ResponseWriter, r *http.Request) {
    // 修改这里
}

// ✅ 使用 http_util.go 的工具函数
sendResponse(w, result)
sendError(w, "Error message", http.StatusBadRequest, nil)

// ✅ 添加 RegisterXxxRoutes() 函数
func RegisterNewFeatureRoutes() {
    http.HandleFunc("/new-feature", handleNewFeature)
}

// ✅ 在 http_server.go 中调用
func registerAllRoutes() {
    RegisterNewFeatureRoutes()
}
```

### ❌ 避免做的

```go
// ❌ 不要在多个 handler 中复制工具函数
func handleTokenSet(w http.ResponseWriter, r *http.Request) {
    resp := Response{...}  // 复制了 sendResponse 的逻辑
}

// ❌ 不要混合多个功能在一个 handler
func handleMixedLogic(w http.ResponseWriter, r *http.Request) {
    // Token 逻辑
    // VPN 控制逻辑
    // 代理操作逻辑
}

// ❌ 不要直接修改 http_server.go 的业务逻辑
func StartHttpServer() {
    // ❌ 不要在这里放业务逻辑
}

// ❌ 不要在 main.go 中注册路由
// main.go
http.HandleFunc("/token/set", handleTokenSet)  // ❌ 错误！
```

## 设计原则

### 单一职责原则 (SRP)

每个文件只负责一个功能:

```
token_handler.go       → Token 相关
run_handler.go         → VPN 运行
proxy_handler.go       → 代理管理
http_util.go           → 工具函数
```

### 开闭原则 (OCP)

对扩展开放，对修改关闭:

```go
// ✅ 添加新功能：创建新文件
new_feature_handler.go
  └─ RegisterNewFeatureRoutes()

// 修改 http_server.go 的 registerAllRoutes()
function registerAllRoutes() {
    RegisterNewFeatureRoutes()  // ← 只需添加这一行
}
```

### 依赖倒转原则 (DIP)

通过接口而不是具体实现:

```go
// handler 接受接口而不是具体类型
func handleProxyRegistryRegister(w http.ResponseWriter, r *http.Request) {
    registry := proxyRegistry.GetRegistry()  // ← 获取单例
    // 而不是: var registry = &Registry{}
}
```

## 导入依赖

### 按模块分离导入

**token_handler.go**:
```go
import "nursor.org/nursorgate/outbound"
```

**run_handler.go**:
```go
import (
    "nursor.org/nursorgate/common/logger"
    tun "nursor.org/nursorgate/inbound/tun/engine"
    user "nursor.org/nursorgate/processor/auth"
    "nursor.org/nursorgate/runner"
)
```

**config_handler.go**:
```go
import proxyConfig "nursor.org/nursorgate/processor/config"
```

**proxy_handler.go**:
```go
import proxyRegistry "nursor.org/nursorgate/processor/proxy"
```

### 优点

✅ 清晰的依赖关系
✅ 容易看出每个模块依赖什么
✅ 减少循环依赖风险
✅ 便于单元测试 (Mock 依赖)

## 路由注册流程

```
main.go
  ↓
runner.go or cmd/
  ↓
StartHttpServer()  [http_server.go]
  ↓
registerAllRoutes()  [http_server.go]
  ├─ RegisterTokenRoutes()  [token_handler.go]
  ├─ RegisterRunRoutes()  [run_handler.go]
  ├─ RegisterConfigRoutes()  [config_handler.go]
  ├─ RegisterProxyRoutes()  [proxy_handler.go]
  ├─ RegisterProxyRegistryRoutes()  [proxy_registry_handler.go]
  ├─ RegisterLoggerRoutes()  [logger_routes.go]
  └─ StartWebSocketServer()  [websocket_server.go]
```

## 代码风格

### 命名规范

**函数命名**:
```go
func handle<ModuleName><Action>(w http.ResponseWriter, r *http.Request)
// 例如:
// handleTokenSet
// handleProxyRegistryRegister
// handleRunUserInfo
```

**Register 函数**:
```go
func Register<ModuleName>Routes()
// 例如:
// RegisterTokenRoutes
// RegisterProxyRegistryRoutes
```

**文件命名**:
```
<feature>_handler.go
// 例如:
// token_handler.go
// run_handler.go
// proxy_registry_handler.go
```

### 代码组织

```go
package server

import (
    // 标准库
    "encoding/json"
    "net/http"

    // 外部包
    "nursor.org/nursorgate/common/logger"
)

// 处理器函数
func handleXxx(w http.ResponseWriter, r *http.Request) {
    // 验证
    // 业务逻辑
    // 响应
}

// 路由注册函数
func RegisterXxxRoutes() {
    http.HandleFunc("/path", handleXxx)
}
```

## 常见问题

**Q: 怎么修改 /token/set 的逻辑?**
A: 修改 `token_handler.go` 中的 `handleTokenSet()` 函数

**Q: 怎么添加新的代理管理端点?**
A: 在 `proxy_registry_handler.go` 中添加新函数，修改 `RegisterProxyRegistryRoutes()`

**Q: 为什么 http_server.go 只有 42 行?**
A: 因为它只负责启动和路由注册，具体业务逻辑都在各个 handler 中

**Q: 原来的 bussiness_server.go 为什么还在?**
A: 保留用于向后兼容和说明，所有代码都已拆分到各个专业模块

**Q: 怎么为某个 handler 写单元测试?**
A: 创建 `xxx_handler_test.go`，导入对应 handler 进行测试 (因为现在是独立的小文件)

## 文件统计

| 指标 | 数值 |
|------|------|
| 原始文件数 | 1 (bussiness_server.go) |
| 拆分后文件数 | 18 (+ logger 相关) |
| 原始代码行数 | 474 |
| 拆分后总行数 | ~1000+ |
| 平均文件大小 | ~55 行 |
| 最小文件 | http_types.go (15 行) |
| 最大文件 | proxy_registry_handler.go (168 行) |
| 编译状态 | ✅ 通过 |

## 相关文档

- [Common Logger 文档](../common/logger/README.md)
- [Processor Config 文档](../processor/config/README.md)
- [Processor Proxy 文档](../processor/proxy/README.md)

## 版本记录

### v1.0 (2025-12-03)
- ✅ 拆分 bussiness_server.go 为 10 个模块
- ✅ 分离 logger API 为 7 个模块
- ✅ 实现统一路由注册
- ✅ 保持 100% 向后兼容
- ✅ 编译通过，零错误

---

**维护者**: Nursor Team
**最后更新**: 2025-12-03
