# 项目重构方案

## 项目概况

本项目基于Tun设备技术（macOS用utun）实现网络代理网关。通过接受来自客户端的所有网络流量，将其捕获、解析、修改后，依据配置选择直连或通过多种代理协议（VLESS、Shadowsocks、Hysteria等）转发，支持HTTPS流量解密（证书管理）和HTTP2头部注入。还通过FFI导出接口和HTTP服务与Flutter等客户端交互。

---

## 重构后的目录结构

```
cursor-proxy/
├── cmd/                      # 可执行程序入口
│   ├── main.go              # 主程序入口
│   └── export.go            # FFI导出函数
├── inbound/                  # 入站流量处理
│   ├── tun/                 # TUN设备管理
│   │   ├── device.go        # TUN设备创建和管理
│   │   ├── packet.go        # 数据包读写
│   │   └── stack.go         # 网络协议栈
│   └── listener.go          # 流量监听接口
├── outbound/                 # 出站代理协议
│   ├── direct/              # 直连
│   ├── vless/               # VLESS协议
│   ├── shadowsocks/         # Shadowsocks协议
│   ├── hysteria/            # Hysteria协议
│   └── interface.go         # 代理协议统一接口
├── processor/                # 流量处理器
│   ├── router.go            # 路由规则引擎
│   ├── cert/                # 证书管理
│   │   ├── generator.go     # 证书生成
│   │   ├── installer.go     # 证书安装
│   │   └── store.go         # 证书存储
│   ├── http2/               # HTTP2处理
│   │   ├── header.go        # Header注入
│   │   └── parser.go        # HTTP2解析
│   └── modifier.go          # 流量修改器
├── common/                   # 通用组件
│   ├── config/              # 配置管理
│   │   ├── app.go           # 应用配置
│   │   ├── proxy.go         # 代理配置
│   │   └── loader.go        # 配置加载器
│   ├── log/                 # 日志系统
│   │   ├── logger.go        # 日志接口
│   │   └── formatter.go     # 日志格式化
│   └── models/              # 数据模型
├── integration/              # 外部集成
│   ├── httpserver/          # HTTP API服务
│   │   ├── server.go        # HTTP服务器
│   │   ├── handlers.go      # API处理器
│   │   └── routes.go        # 路由定义
│   └── ffi/                 # FFI导出层
│       ├── export.go        # 导出函数定义
│       └── callbacks.go     # 回调函数
├── legacy/                   # 已废弃代码（归档）
│   ├── js/                  # JS处理相关
│   └── http_proxy/          # 旧HTTP代理
├── tests/                    # 测试
│   ├── integration/         # 集成测试
│   └── fixtures/            # 测试数据
├── scripts/                  # 构建脚本
│   ├── build_dll.sh         # Windows DLL构建
│   ├── build_so.sh          # Linux SO构建
│   └── build_dylib.sh       # macOS DYLIB构建
├── docs/                     # 文档
│   ├── architecture.md      # 架构说明
│   └── refactoring-plan.md  # 本文档
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── CLAUDE.md
```

---

## 核心架构设计

### 1. Inbound层（入站流量）

```go
// proxyserver/listener.go
type Listener interface {
    Start() error
    Stop() error
    GetConnections() <-chan Connection
}

// proxyserver/tun/device.go
type TunDevice struct {
    name string
    mtu  int
    // TUN设备管理
}
```

**职责：**
- TUN设备的创建、配置和管理
- 原始IP数据包的读取
- 将数据包解析为连接（TCP/UDP）
- 向Processor层传递连接

### 2. Processor层（流量处理）

```go
// processor/router.go
type Router struct {
    rules []Rule
    // 路由决策引擎
}

func (r *Router) Route(conn Connection) (Action, Outbound) {
    // 根据规则决定：直连/代理/拒绝
}

// processor/modifier.go
type Modifier interface {
    Process(data []byte) ([]byte, error)
}

// processor/cert/generator.go
type CertManager struct {
    // 证书生成、安装、管理
}

// processor/http2/header.go
type HeaderInjector struct {
    // HTTP2头部注入逻辑
}
```

**职责：**
- 路由决策（基于规则匹配）
- HTTPS流量解密（证书管理）
- HTTP2头部注入
- 流量修改和处理
- 日志记录和监控

### 3. Outbound层（出站代理）

```go
// outbound/interface.go
type Outbound interface {
    Name() string
    Dial(target string) (net.Conn, error)
    DialUDP(target string) (net.PacketConn, error)
}

// outbound/vless/client.go
type VlessOutbound struct {
    // VLESS协议实现
}

// outbound/shadowsocks/client.go
type ShadowsocksOutbound struct {
    // Shadowsocks协议实现
}
```

**职责：**
- 各种代理协议的独立实现
- 统一的Outbound接口
- 连接池管理
- 协议特定的加密和传输

### 4. Common层（通用组件）

```go
// common/config/app.go
type AppConfig struct {
    LogLevel    string
    TunName     string
    TunMTU      int
    HTTPPort    int
}

// common/config/proxy.go
type ProxyConfig struct {
    Inbounds  []InboundConfig
    Outbounds []OutboundConfig
    Rules     []RuleConfig
}

// common/log/logger.go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}
```

**职责：**
- 配置文件加载和验证
- 统一日志接口（支持输出到Flutter）
- 通用数据模型和工具函数

### 5. Integration层（外部集成）

```go
// integration/httpserver/server.go
type APIServer struct {
    port int
}

func (s *APIServer) Start() error {
    // 启动HTTP API服务
}

// integration/ffi/export.go
//export StartProxy
func StartProxy(configPath *C.char) C.int {
    // FFI导出函数
}

//export StopProxy
func StopProxy() C.int {
    // 停止代理
}

//export GetStatus
func GetStatus() *C.char {
    // 获取运行状态
}
```

**职责：**
- HTTP API服务（供Flutter等调用）
- FFI导出接口
- 状态查询和控制接口

---

## 数据流向

```
Client Traffic
      ↓
[Inbound/TUN]  ← 捕获所有流量
      ↓
[Processor/Router]  ← 路由决策
      ↓
[Processor/Cert]  ← HTTPS解密（如需要）
      ↓
[Processor/HTTP2]  ← HTTP2头部注入（如需要）
      ↓
[Processor/Modifier]  ← 流量修改
      ↓
[Outbound/Protocol]  ← 选择代理协议
      ↓
Destination Server
```

---

## 具体重构步骤

### 阶段1：代码清理（1-2天）
1. 将JS相关代码移至`legacy/js/`
2. 将旧HTTP代理移至`legacy/http_proxy/`
3. 从构建中排除legacy目录
4. 清理无用依赖

### 阶段2：创建新目录结构（1天）
1. 创建inbound/outbound/processor/integration目录
2. 定义各层接口（interface.go文件）
3. 设计数据模型

### 阶段3：迁移Inbound层（2-3天）
1. 将`client/server/tun`代码迁移到`inbound/tun/`
2. 重构TUN设备管理逻辑
3. 实现统一的Listener接口
4. 编写单元测试

### 阶段4：重构Processor层（3-4天）
1. 提取路由逻辑到`processor/router.go`
2. 重构`client/server/helper`为`processor/cert/`和`processor/http2/`
3. 实现流量修改器接口
4. 确保证书生成和HTTP2注入功能完整
5. 编写单元测试

### 阶段5：模块化Outbound层（3-4天）
1. 将每个协议拆分到独立目录
2. 实现统一的Outbound接口
3. 重构VLESS、Shadowsocks、Hysteria等协议
4. 添加连接池管理
5. 编写协议测试

### 阶段6：完善Common层（1-2天）
1. 拆分配置为app.go和proxy.go
2. 统一日志接口
3. 提取通用工具函数

### 阶段7：Integration层实现（2-3天）
1. 实现HTTP API服务器
2. 定义API路由和处理器
3. 实现FFI导出函数
4. 编写API文档

### 阶段8：构建系统（1-2天）
1. 编写跨平台构建脚本
2. 配置DLL/SO/DYLIB编译
3. 测试各平台打包

### 阶段9：测试与文档（2-3天）
1. 编写集成测试
2. 更新CLAUDE.md
3. 编写README和架构文档
4. API使用示例

---

## 关键设计原则

1. **接口优先**：每一层都定义清晰的接口
2. **单一职责**：每个模块职责明确，避免耦合
3. **依赖注入**：通过接口注入依赖，便于测试
4. **配置驱动**：行为由配置文件控制，不硬编码
5. **错误处理**：统一错误处理和日志记录
6. **测试覆盖**：关键路径必须有测试

---

## 配置文件示例

```yaml
# config.yaml
app:
  log_level: info
  tun_name: utun3
  tun_mtu: 1500
  http_api_port: 8080

proxy:
  inbounds:
    - type: tun
      listen: 10.0.0.1

  outbounds:
    - name: direct
      type: direct
    - name: my-vless
      type: vless
      server: example.com
      port: 443
      uuid: xxx

  rules:
    - domain_suffix: ".google.com"
      outbound: my-vless
    - geoip: cn
      outbound: direct
    - default: my-vless

processor:
  cert:
    enabled: true
    ca_path: ./ca.crt
  http2:
    inject_headers:
      X-Custom-Field: "value"
```

---

## FFI导出接口设计

```go
// integration/ffi/export.go

//export InitProxy
func InitProxy(configPath *C.char) C.int

//export StartProxy
func StartProxy() C.int

//export StopProxy
func StopProxy() C.int

//export GetStatus
func GetStatus() *C.char  // 返回JSON状态

//export UpdateConfig
func UpdateConfig(configJSON *C.char) C.int

//export GetLogs
func GetLogs(count C.int) *C.char  // 返回最近N条日志

//export SetLogCallback
func SetLogCallback(callback unsafe.Pointer)  // 实时日志回调
```

---

## HTTP API设计

```
GET  /api/status          # 获取运行状态
POST /api/start           # 启动代理
POST /api/stop            # 停止代理
GET  /api/config          # 获取配置
PUT  /api/config          # 更新配置
GET  /api/logs            # 获取日志
GET  /api/connections     # 获取当前连接
POST /api/cert/generate   # 生成证书
POST /api/cert/install    # 安装证书
```

---

## 构建命令

```makefile
# Makefile

.PHONY: build test clean

# 构建可执行文件
build:
	go build -o cursor-proxy cmd/main.go

# 构建Windows DLL
build-dll:
	GOOS=windows GOARCH=amd64 go build -buildmode=c-shared -o cursor-proxy.dll cmd/export.go

# 构建Linux SO
build-so:
	GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o libcursor-proxy.so cmd/export.go

# 构建macOS DYLIB
build-dylib:
	GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o libcursor-proxy.dylib cmd/export.go

# 运行测试
test:
	go test ./... -v -cover

# 清理
clean:
	rm -f cursor-proxy cursor-proxy.dll libcursor-proxy.so libcursor-proxy.dylib
```

---

## 总结

这个重构方案：

✅ 去除了client/server结构，改为清晰的inbound/outbound/processor架构
✅ 保留了common层，统一管理配置和日志
✅ 分离了证书管理和HTTP2处理到processor层
✅ 独立的integration层支持FFI和HTTP API
✅ 归档legacy代码，保持主代码库简洁
✅ 支持多平台打包（DLL/SO/DYLIB）
✅ 清晰的数据流向和模块职责
