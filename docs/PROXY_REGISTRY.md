# 代理注册中心使用指南

## 概述

代理注册中心提供了一个统一的、线程安全的方式来管理多个代理实例。系统启动时会根据全局配置自动初始化代理，之后可以根据需要动态注册、切换和管理代理。

## 核心功能

### 1. 代理注册
- 支持通过配置注册代理
- 支持自定义代理名称
- 自动管理默认代理和门代理

### 2. 代理查询
- 根据名称获取代理
- 获取默认代理
- 获取门代理
- 列出所有已注册的代理

### 3. 代理切换
- 动态切换默认代理
- 动态切换门代理
- 支持运行时切换，无需重启

### 4. 线程安全
- 使用 `sync.RWMutex` 保证并发安全
- 所有操作都是线程安全的

## API 接口

### Go API

```go
import "nursor.org/nursorgate/processor/proxy"

// 获取注册中心实例
registry := proxy.GetRegistry()

// 注册代理（从配置）
cfg := &proxyConfig.ProxyConfig{
    Type: "vless",
    VLESS: &proxyConfig.VLESSConfig{...},
    IsDefault: true,
    IsDoorProxy: true,
}
err := registry.RegisterFromConfig("my-vless", cfg)

// 注册代理（直接）
p, _ := vless.NewVLESS("server:443", "uuid")
err := registry.Register("my-proxy", p)

// 获取代理
p, err := registry.Get("my-proxy")

// 获取默认代理
p, err := registry.GetDefault()

// 设置默认代理
err := registry.SetDefault("my-proxy")

// 列出所有代理
names := registry.List()
info := registry.ListWithInfo()
```

### HTTP API

#### 列出所有代理
```bash
GET /proxy/registry/list
```

响应：
```json
{
    "code": 0,
    "msg": "success",
    "data": {
        "proxies": {
            "vless-default": {
                "name": "vless-default",
                "type": "vless",
                "addr": "node1.nursor.org:35001",
                "is_default": true,
                "is_door_proxy": true
            }
        },
        "count": 1
    }
}
```

#### 获取指定代理
```bash
GET /proxy/registry/get?name=vless-default
```

#### 注册新代理
```bash
POST /proxy/registry/register
Content-Type: application/json

{
    "name": "my-vless",
    "config": {
        "type": "vless",
        "is_default": false,
        "is_door_proxy": false,
        "vless": {
            "server": "node1.nursor.org:35001",
            "uuid": "74cddcdd-6d48-41cf-8e62-902e7c943fe7",
            "sni": "www.microsoft.com",
            "reality_enabled": true,
            "public_key": "sAtJcW2xLIUWRE-_7KHGEAtvHx-P1sDbjrrgrt4_XCo"
        }
    }
}
```

#### 注销代理
```bash
POST /proxy/registry/unregister
Content-Type: application/json

{
    "name": "my-vless"
}
```

#### 设置默认代理
```bash
POST /proxy/registry/set-default
Content-Type: application/json

{
    "name": "my-vless"
}
```

#### 设置门代理
```bash
POST /proxy/registry/set-door
Content-Type: application/json

{
    "name": "my-vless"
}
```

#### 切换代理（设置默认并更新 tunnel）
```bash
POST /proxy/registry/switch
Content-Type: application/json

{
    "name": "my-vless"
}
```

### FFI 接口 (C)

```c
// 注册代理
void registerProxy(
    const char* name,
    const char* proxyType,  // "vless" 或 "shadowsocks"
    const char* server,
    const char* uuid,       // VLESS 需要
    const char* sni,        // 可选
    const char* publicKey   // REALITY 需要
);

// 切换代理
void switchProxy(const char* name);

// 列出所有代理（返回 JSON 字符串）
char* listProxies();
```

## 系统启动时的初始化

系统启动时（`server.StartHttpServer()`），会自动调用 `proxyRegistry.Initialize()`，从全局配置中加载代理：

1. 如果存在 VLESS 配置，创建 `vless-default` 代理
2. 如果存在 Shadowsocks 配置，创建 `shadowsocks-default` 代理
3. 自动设置第一个创建的代理为默认代理和门代理

## 使用示例

### 示例 1: 注册多个代理并切换

```go
// 注册第一个 VLESS 代理
cfg1 := &proxyConfig.ProxyConfig{
    Type: "vless",
    VLESS: &proxyConfig.VLESSConfig{
        Server: "server1.com:443",
        UUID: "uuid1",
        RealityEnabled: true,
        PublicKey: "key1",
    },
}
registry.RegisterFromConfig("vless-1", cfg1)

// 注册第二个 VLESS 代理
cfg2 := &proxyConfig.ProxyConfig{
    Type: "vless",
    VLESS: &proxyConfig.VLESSConfig{
        Server: "server2.com:443",
        UUID: "uuid2",
        RealityEnabled: true,
        PublicKey: "key2",
    },
}
registry.RegisterFromConfig("vless-2", cfg2)

// 切换到第二个代理
registry.SetDefault("vless-2")
```

### 示例 2: 通过 HTTP API 管理代理

```bash
# 1. 列出所有代理
curl http://127.0.0.1:56431/proxy/registry/list

# 2. 注册新代理
curl -X POST http://127.0.0.1:56431/proxy/registry/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "backup-proxy",
    "config": {
      "type": "vless",
      "vless": {
        "server": "backup.com:443",
        "uuid": "backup-uuid",
        "reality_enabled": true,
        "public_key": "backup-key"
      }
    }
  }'

# 3. 切换到备用代理
curl -X POST http://127.0.0.1:56431/proxy/registry/switch \
  -H "Content-Type: application/json" \
  -d '{"name": "backup-proxy"}'
```

## 注意事项

1. **代理名称唯一性**: 每个代理必须有唯一的名称，重复注册会覆盖之前的代理
2. **默认代理**: 系统必须至少有一个默认代理，否则会使用直连代理
3. **门代理**: 用于 DNS 解析等特殊场景，可以单独设置
4. **线程安全**: 所有操作都是线程安全的，可以在多个 goroutine 中并发使用
5. **配置持久化**: 当前配置不会自动持久化，重启后需要重新注册（可以通过配置文件实现持久化）

## 与配置管理器的关系

- **配置管理器** (`processor/config`): 管理代理配置（配置数据）
- **注册中心** (`processor/proxy`): 管理代理实例（运行时对象）

两者配合使用：
- 配置管理器存储配置信息
- 注册中心根据配置创建和管理代理实例
- 可以通过配置管理器设置配置，然后通过注册中心注册代理

