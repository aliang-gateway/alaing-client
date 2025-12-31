# Run Control API 文档

HTTP服务器在 `127.0.0.1:56431` 提供了完整的运行控制接口。

## 接口概览

| 接口 | 方法 | 说明 |
|------|------|------|
| `/run/start` | POST | 启动TUN服务 |
| `/run/stop` | POST | 停止当前服务 |
| `/run/status` | GET | 查询当前运行状态 |
| `/run/swift` | POST | 切换运行模式（HTTP/TUN） |
| `/run/userInfo` | POST | 设置用户信息 |

---

## 1. 查询运行状态 - `/run/status`

**请求方法**: `GET`

**请求示例**:
```bash
curl http://127.0.0.1:56431/run/status
```

**响应示例（空闲状态）**:
```json
{
  "current_mode": "idle",
  "tun_running": false,
  "available_modes": ["http", "tun"],
  "status": "No service is currently running",
  "description": "Ready to start TUN or HTTP proxy service"
}
```

**响应示例（TUN运行中）**:
```json
{
  "current_mode": "tun",
  "tun_running": true,
  "available_modes": ["http", "tun"],
  "status": "TUN service is running",
  "description": "Transparent proxy mode via TUN interface"
}
```

**响应示例（HTTP模式）**:
```json
{
  "current_mode": "http",
  "tun_running": false,
  "available_modes": ["http", "tun"],
  "status": "HTTP proxy server is running",
  "description": "HTTP CONNECT proxy mode on port 56432"
}
```

---

## 2. 切换运行模式 - `/run/swift`

**请求方法**: `POST`

**请求头**: `Content-Type: application/json`

**说明**: `/run/swift` 实现完整的模式切换流程：
- 如果当前有其他模式的服务在运行，先**自动停止**当前服务
- 然后切换到目标模式
- 对于HTTP模式，**自动启动** HTTP代理服务器
- 对于TUN模式，仅切换模式，需要后续调用 `/run/start` 启动

### 切换到HTTP模式

**请求示例**:
```bash
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "http"}'
```

**响应示例**:
```json
{
  "status": "switched",
  "target_mode": "http",
  "message": "Switched to HTTP proxy mode. Server is starting on 127.0.0.1:56432",
  "usage": "curl -x http://127.0.0.1:56432 https://example.com",
  "details": "HTTP proxy server will be ready in a moment",
  "next_action": "HTTP service starts automatically, you can begin using it after 1 second"
}
```

**说明**:
- 切换到HTTP模式时，HTTP代理服务器会**自动启动**在后台
- 监听在 `127.0.0.1:56432`
- 1秒后可以开始使用HTTP CONNECT代理
- 如果之前有TUN服务在运行，会先**自动停止** TUN服务

### 切换到TUN模式

**请求示例**:
```bash
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "tun"}'
```

**响应示例**:
```json
{
  "status": "switched",
  "target_mode": "tun",
  "message": "Switched to TUN mode. Use /run/start to activate the TUN service",
  "usage": "POST /run/start with InnerToken",
  "next_step": "Call /run/start to initialize and start the TUN interface"
}
```

**说明**:
- 切换到TUN模式时，仅切换模式标志，不启动任何服务
- 如果之前有HTTP服务在运行，会先**自动停止** HTTP服务
- 需要后续调用 `/run/start` 来启动TUN接口

### 错误场景

**已处于该模式的响应**:
```json
{
  "status": "already_running",
  "current_mode": "http",
  "message": "Already running in http mode"
}
```

**无法切换（有服务在运行）**:
```json
{
  "code": 409,
  "message": "Cannot switch mode while running in tun mode. Please stop the current service first"
}
```

**无效的模式**:
```json
{
  "code": 400,
  "message": "Invalid target mode: xxx. Must be 'http' or 'tun'"
}
```

---

## 3. 启动服务 - `/run/start`

**请求方法**: `POST`

**请求头**: `Content-Type: application/json`

**请求示例**:
```bash
curl -X POST http://127.0.0.1:56431/run/start \
  -H "Content-Type: application/json" \
  -d '{"inner_token": "your_token_here"}'
```

**说明**: `/run/start` 的行为取决于当前选择的运行模式：
- **HTTP模式**: HTTP代理已在 `/run/swift` 中自动启动，此接口会返回已运行提示
- **TUN模式**: 启动TUN接口服务，这是必需的步骤

### TUN模式成功响应
```json
{
  "status": "success",
  "message": "TUN service started successfully"
}
```

### HTTP模式响应
```json
{
  "status": "success",
  "message": "HTTP proxy server is already running",
  "details": "HTTP proxy was started when you switched to HTTP mode",
  "port": "127.0.0.1:56432"
}
```

### 错误响应：未选择模式
```json
{
  "code": 400,
  "message": "No mode selected. Please use /run/swift to select HTTP or TUN mode first"
}
```

### 错误响应：服务已运行
```json
{
  "code": 409,
  "message": "tun service is already running"
}
```

### 错误响应：TUN启动失败
```json
{
  "status": "failed",
  "message": "具体的错误原因..."
}
```

---

## 4. 停止服务 - `/run/stop`

**请求方法**: `POST`

**请求示例**:
```bash
curl -X POST http://127.0.0.1:56431/run/stop
```

**成功响应（停止HTTP服务）**:
```json
{
  "status": "success",
  "message": "http service stopped successfully",
  "stopped_mode": "http",
  "details": "HTTP proxy server on 127.0.0.1:56432 has been stopped"
}
```

**成功响应（停止TUN服务）**:
```json
{
  "status": "success",
  "message": "tun service stopped successfully",
  "stopped_mode": "tun",
  "details": "TUN interface service has been stopped"
}
```

**无服务运行的响应**:
```json
{
  "code": 400,
  "message": "No service is currently running"
}
```

**说明**: `/run/stop` 会根据当前运行的服务类型自动停止相应的服务（HTTP或TUN）。

---

## 5. 设置用户信息 - `/run/userInfo`

**请求方法**: `POST`

**请求头**: `Content-Type: application/json`

**请求示例**:
```bash
curl -X POST http://127.0.0.1:56431/run/userInfo \
  -H "Content-Type: application/json" \
  -d '{
    "inner_token": "your_token",
    "username": "user@example.com",
    "password": "password",
    "user_uuid": "uuid-here"
  }'
```

**响应示例**:
```json
{
  "status": "success",
  "user_id": "12345"
}
```

---

## 完整工作流示例

### 场景1: 使用HTTP代理模式（推荐快速测试）

```bash
# 1. 查询当前状态
curl http://127.0.0.1:56431/run/status

# 2. 切换到HTTP模式（会自动启动HTTP服务器）
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "http"}'

# 3. 等待一秒，HTTP代理服务器已启动，可以在另一个终端测试代理
curl -x http://127.0.0.1:56432 https://www.google.com

# 4. 查询状态确认HTTP运行中
curl http://127.0.0.1:56431/run/status

# 5. 停止服务
curl -X POST http://127.0.0.1:56431/run/stop

# 6. 确认已停止
curl http://127.0.0.1:56431/run/status
```

**说明**: HTTP模式下，`/run/swift` 会自动启动HTTP CONNECT代理服务器，无需额外操作即可使用代理。

### 场景2: 切换到TUN模式

```bash
# 1. 查询状态
curl http://127.0.0.1:56431/run/status

# 2. 切换到TUN模式（仅切换模式，不启动服务）
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "tun"}'

# 3. 设置用户信息（如果需要）
curl -X POST http://127.0.0.1:56431/run/userInfo \
  -H "Content-Type: application/json" \
  -d '{
    "inner_token": "token_value",
    "username": "user",
    "password": "pass",
    "user_uuid": "uuid"
  }'

# 4. 启动TUN服务（需要调用/run/start）
curl -X POST http://127.0.0.1:56431/run/start \
  -H "Content-Type: application/json" \
  -d '{"inner_token": "token_value"}'

# 5. 查询状态确认TUN运行中
curl http://127.0.0.1:56431/run/status

# 6. 停止TUN服务
curl -X POST http://127.0.0.1:56431/run/stop
```

**说明**: TUN模式下，`/run/swift` 仅切换模式，还需要调用 `/run/start` 才能启动TUN服务。

### 场景3: 模式之间的自动切换

```bash
# 从HTTP模式切换到TUN模式（自动停止HTTP）

# 1. 当前处于HTTP模式
curl http://127.0.0.1:56431/run/status
# 返回: http mode running

# 2. 直接切换到TUN模式（自动停止HTTP服务）
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "tun"}'
# 自动停止HTTP服务，切换到TUN模式

# 3. 查询状态确认已切换
curl http://127.0.0.1:56431/run/status
# 返回: tun mode idle

# 4. 启动TUN服务
curl -X POST http://127.0.0.1:56431/run/start \
  -H "Content-Type: application/json" \
  -d '{"inner_token": "token"}'

# 5. 查询状态确认TUN运行中
curl http://127.0.0.1:56431/run/status
# 返回: tun mode running

# 6. 再次切换回HTTP（自动停止TUN）
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "http"}'
# 自动停止TUN服务，启动HTTP代理

# 7. 立即可用
sleep 1
curl -x http://127.0.0.1:56432 https://www.google.com
```

**说明**: `/run/swift` 会自动处理旧服务的停止，无需显式调用 `/run/stop`

---

## 状态机

```
┌─────────┐
│  IDLE   │ ← 初始状态，没有服务运行
└────┬────┘
     │
     ├─ swift(http) → HTTP MODE
     │     ↓
     │  [HTTP模式]
     │     ↓
     │  stop() → IDLE
     │
     └─ swift(tun) → TUN MODE
           ↓
        [TUN模式]
           ↓
        start() → TUN RUNNING
           ↓
        stop() → IDLE
```

---

## 返回状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误或非法操作 |
| 409 | 冲突（如已有服务在运行） |
| 500 | 服务器内部错误 |

---

## 注意事项

1. **模式互斥性**: 同时只能运行一种模式（HTTP或TUN），另一种模式处于待机状态
2. **切换前提**: 切换模式前必须停止当前运行的服务
3. **HTTP自动启动**: 切换到HTTP模式后，HTTP代理服务会自动启动，无需额外命令
4. **TUN需要启动**: 切换到TUN模式后，还需要调用 `/run/start` 来启动TUN服务
5. **线程安全**: 所有操作都使用互斥锁保护，支持并发请求
6. **端口**:
   - HTTP服务API: `127.0.0.1:56431`
   - HTTP代理: `127.0.0.1:56432`
