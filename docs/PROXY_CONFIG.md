# 代理配置系统使用指南

## 概述

代理配置系统提供了一个统一的接口来管理 VLESS 和 Shadowsocks 代理配置，支持通过 HTTP API 和 FFI 接口进行设置。

## 配置结构

### VLESS 配置

```go
type VLESSConfig struct {
    Server          string  // 服务器地址，格式: host:port
    UUID            string  // UUID
    Flow            string  // 流控类型，如: xtls-rprx-vision
    TLSEnabled      bool    // 是否启用 TLS
    SNI             string  // SNI 服务器名称
    RealityEnabled  bool    // 是否启用 REALITY
    PublicKey       string  // REALITY 公钥
    ShortID         string  // REALITY ShortID（可选）
    ShortIDList     string  // ShortID 列表，逗号分隔
}
```

### Shadowsocks 配置

```go
type ShadowsocksConfig struct {
    Server   string  // 服务器地址，格式: host:port
    Method   string  // 加密方法，如: aes-256-gcm
    Password string  // 密码
    ObfsMode string  // 混淆模式: tls, http, 或空
    ObfsHost string  // 混淆主机
}
```

## HTTP API

### 设置代理配置

**端点**: `POST /proxy/set`

**请求体**:
```json
{
    "type": "vless",
    "is_default": true,
    "is_door_proxy": false,
    "vless": {
        "server": "node1.nursor.org:35001",
        "uuid": "74cddcdd-6d48-41cf-8e62-902e7c943fe7",
        "sni": "www.microsoft.com",
        "reality_enabled": true,
        "public_key": "sAtJcW2xLIUWRE-_7KHGEAtvHx-P1sDbjrrgrt4_XCo",
        "short_id_list": "ef,b79e62,7d87a3"
    }
}
```

或 Shadowsocks:
```json
{
    "type": "shadowsocks",
    "is_default": true,
    "is_door_proxy": false,
    "shadowsocks": {
        "server": "example.com:443",
        "method": "aes-256-gcm",
        "password": "your-password",
        "obfs_mode": "tls",
        "obfs_host": "example.com"
    }
}
```

**响应**:
```json
{
    "code": 0,
    "msg": "success",
    "data": {
        "status": "success"
    }
}
```

### 获取代理配置

**端点**: `GET /proxy/get`

**响应**:
```json
{
    "code": 0,
    "msg": "success",
    "data": {
        "vless": {
            "server": "node1.nursor.org:35001",
            "uuid": "74cddcdd-6d48-41cf-8e62-902e7c943fe7",
            ...
        },
        "shadowsocks": {
            "server": "example.com:443",
            "method": "aes-256-gcm",
            ...
        }
    }
}
```

## FFI 接口

### 设置 VLESS 代理

```c
void setVLESSProxy(
    const char* server,
    const char* uuid,
    const char* sni,
    const char* publicKey,
    bool isDefault,
    bool isDoorProxy
);
```

**参数说明**:
- `server`: 服务器地址
- `uuid`: UUID
- `sni`: SNI 服务器名称（如果启用 TLS/REALITY）
- `publicKey`: REALITY 公钥（如果启用 REALITY）
- `isDefault`: 是否为默认代理
- `isDoorProxy`: 是否为门代理（用于 DNS 等）

### 设置 Shadowsocks 代理

```c
void setShadowsocksProxy(
    const char* server,
    const char* method,
    const char* password,
    const char* obfsMode,
    const char* obfsHost,
    bool isDefault,
    bool isDoorProxy
);
```

**参数说明**:
- `server`: 服务器地址
- `method`: 加密方法
- `password`: 密码
- `obfsMode`: 混淆模式（可选）
- `obfsHost`: 混淆主机（可选）
- `isDefault`: 是否为默认代理
- `isDoorProxy`: 是否为门代理

## 使用示例

### HTTP API 示例

```bash
# 设置 VLESS 代理
curl -X POST http://127.0.0.1:56431/proxy/set \
  -H "Content-Type: application/json" \
  -d '{
    "type": "vless",
    "is_default": true,
    "is_door_proxy": true,
    "vless": {
      "server": "node1.nursor.org:35001",
      "uuid": "74cddcdd-6d48-41cf-8e62-902e7c943fe7",
      "sni": "www.microsoft.com",
      "reality_enabled": true,
      "public_key": "sAtJcW2xLIUWRE-_7KHGEAtvHx-P1sDbjrrgrt4_XCo"
    }
  }'

# 获取配置
curl http://127.0.0.1:56431/proxy/get
```

### FFI 示例 (Go)

```go
import "C"

// 设置 VLESS 代理
C.setVLESSProxy(
    C.CString("node1.nursor.org:35001"),
    C.CString("74cddcdd-6d48-41cf-8e62-902e7c943fe7"),
    C.CString("www.microsoft.com"),
    C.CString("sAtJcW2xLIUWRE-_7KHGEAtvHx-P1sDbjrrgrt4_XCo"),
    C.bool(true),  // isDefault
    C.bool(true),  // isDoorProxy
)
```

## 向后兼容性

如果未通过 API 或 FFI 设置配置，系统会使用默认的硬编码配置（向后兼容）。建议尽快迁移到新的配置系统。

## 注意事项

1. 配置是线程安全的，可以在运行时动态更新
2. 设置配置后，新的连接将使用新配置
3. 如果同时设置了 `is_default` 和 `is_door_proxy`，代理将同时作为默认代理和门代理使用
4. REALITY 的 ShortID 如果未提供，将从 `short_id_list` 中随机选择，如果都没有提供，将使用内置的默认列表

