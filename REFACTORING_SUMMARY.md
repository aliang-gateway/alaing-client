# Swift接口完整重构总结

## 概述

已完成 `/run/swift` 和 `/run/start` 接口的**完整重构**，实现更加智能和用户友好的运行模式管理：

- ✅ `handleSwift()` - 自动停止旧服务，启动新服务
- ✅ `handleRun()` - 根据当前模式启动对应的服务
- ✅ `handleRunTUN()` - 新增TUN启动逻辑处理函数
- ✅ `stopService()` - 新增通用停止函数，支持所有模式

---

## 核心改动

### 1. handleRun() 重构

**原逻辑**: 直接启动TUN服务

**新逻辑**:
```go
func handleRun(w http.ResponseWriter, r *http.Request) {
    // 1. 检查当前模式 (不能是ModeIdle)
    // 2. 检查服务是否已运行 (tunRunning)
    // 3. 根据currentMode调用不同的启动函数
    //    ├─ ModeTUN: 调用 handleRunTUN()
    //    └─ ModeHTTP: 返回已运行提示
}
```

**改进点**:
- ✅ 不再硬编码为TUN模式
- ✅ 支持HTTP模式（虽然HTTP已自动启动）
- ✅ 清晰的错误提示（必须先选择模式）
- ✅ 提取TUN启动逻辑到 `handleRunTUN()`

### 2. handleSwift() 完全重构

**原逻辑**:
- 检查模式冲突
- 切换到新模式
- 启动新服务

**新逻辑**:
```go
func handleSwift(w http.ResponseWriter, r *http.Request) {
    // 1. 检查目标模式有效性
    // 2. 检查是否已处于目标模式
    // 3. 如果运行着其他模式的服务
    //    ├─ 调用 stopService(previousMode)
    //    └─ 等待服务停止
    // 4. 设置新模式
    // 5. 启动新模式的服务
    //    ├─ HTTP: 在后台启动 StartMitmHttp()
    //    └─ TUN: 仅设置模式，不启动
}
```

**改进点**:
- ✅ **自动停止前一个服务** - 不需要用户显式调用 `/run/stop`
- ✅ **无缝切换** - HTTP/TUN可以直接相互切换
- ✅ **智能启动** - 根据模式判断是否需要启动
- ✅ **状态一致** - 自动处理状态清理

### 3. 新增 handleRunTUN() 函数

```go
func handleRunTUN(w http.ResponseWriter, innerToken string) {
    // 处理TUN模式的启动逻辑
    // 1. 设置 tunRunning = true
    // 2. 启动 runner2.Start()
    // 3. 监听启动结果
    // 4. 根据结果更新状态
}
```

**用途**:
- 从 `handleRun()` 调用，启动TUN服务
- 保持启动逻辑集中化
- 便于未来扩展其他启动模式

### 4. 新增 stopService() 函数

```go
func stopService(mode RunMode) {
    switch mode {
    case ModeHTTP:
        httpServer.StopHttpProxy()
    case ModeTUN:
        tun.Stop()
    case ModeIdle:
        // Nothing
    }
}
```

**用途**:
- 从 `handleSwift()` 调用，停止前一个服务
- 从 `handleStop()` 调用，停止当前服务
- 统一的停止逻辑，减少代码重复

---

## 工作流程对比

### 旧工作流（HTTP→TUN）
```
用户在HTTP模式
    ↓
调用 /run/stop 停止HTTP
    ↓
调用 /run/swift 切换到TUN
    ↓
调用 /run/start 启动TUN
```

### 新工作流（HTTP→TUN）
```
用户在HTTP模式
    ↓
调用 /run/swift ({"target_mode": "tun"})
    ├─ 自动停止HTTP
    ├─ 切换到TUN模式
    └─ 返回响应
    ↓
调用 /run/start 启动TUN
```

**改进**: 减少一次API调用，无需显式停止前一个服务

---

## 新增的API行为

### `/run/swift` - 三合一接口

现在 `/run/swift` 实现了**三合一**功能：
1. **检查冲突** - 验证目标模式
2. **自动停止** - 如果有其他模式运行，自动停止
3. **自动启动** - 对于HTTP模式自动启动

### `/run/start` - 模式感知启动

现在 `/run/start` 会：
- 检查当前选择的模式
- 对于HTTP: 返回已运行提示
- 对于TUN: 启动TUN服务
- 对于Idle: 返回错误，提示先选择模式

---

## 状态管理改进

### 细粒度的tunRunning标志

新增了更精确的运行状态管理：

```go
// 在 handleSwift 中
if currentMode != ModeIdle && currentMode != targetMode {
    tunRunning = false  // 立即标记为停止
    modeChangeMutex.Unlock()
    stopService(previousMode)  // 停止服务
    modeChangeMutex.Lock()
}

// 在后台启动中
go func() {
    modeChangeMutex.Lock()
    tunRunning = true  // 服务开始运行
    modeChangeMutex.Unlock()

    httpServer.StartMitmHttp()  // 阻塞直到服务停止

    modeChangeMutex.Lock()
    tunRunning = false  // 服务已停止
    modeChangeMutex.Unlock()
}()
```

**优势**:
- ✅ 状态同步 - 准确反映服务运行状态
- ✅ 原子性 - 使用互斥锁保护状态
- ✅ 准确性 - 区分"模式选择"和"服务运行"

---

## 使用示例

### 示例1: 快速切换HTTP代理
```bash
# 一条命令开启HTTP代理
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "http"}'

# 1秒后开始使用
sleep 1
curl -x http://127.0.0.1:56432 https://www.google.com
```

### 示例2: HTTP和TUN无缝切换
```bash
# 从HTTP切换到TUN（自动停止HTTP）
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "tun"}'

# 启动TUN服务
curl -X POST http://127.0.0.1:56431/run/start \
  -H "Content-Type: application/json" \
  -d '{"inner_token": "token"}'

# 从TUN切换回HTTP（自动停止TUN）
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "http"}'

# 立即可用
sleep 1
curl -x http://127.0.0.1:56432 https://www.example.com
```

---

## 测试场景

### ✅ 场景1: 模式冲突检测
```bash
# 在HTTP模式下尝试再次切换到HTTP
curl -X POST http://127.0.0.1:56431/run/swift \
  -H "Content-Type: application/json" \
  -d '{"target_mode": "http"}'
# 返回: already_running
```

### ✅ 场景2: 未选择模式时启动
```bash
# 在Idle状态调用 /run/start
curl -X POST http://127.0.0.1:56431/run/start \
  -H "Content-Type: application/json" \
  -d '{"inner_token": "token"}'
# 返回: 400 No mode selected
```

### ✅ 场景3: 服务已运行时重复启动
```bash
# 在TUN运行中调用 /run/start
curl -X POST http://127.0.0.1:56431/run/start \
  -H "Content-Type: application/json" \
  -d '{"inner_token": "token"}'
# 返回: 409 tun service is already running
```

### ✅ 场景4: 无服务运行时停止
```bash
# 在Idle状态调用 /run/stop
curl -X POST http://127.0.0.1:56431/run/stop
# 返回: 400 No service is currently running
```

---

## 代码质量改进

### ✅ 减少代码重复
- `stopService()` 统一处理HTTP和TUN停止
- `handleRunTUN()` 专门处理TUN启动

### ✅ 提高可读性
- 每个函数职责明确
- 添加详细的注释说明
- 清晰的错误提示

### ✅ 增强可维护性
- 易于添加新的运行模式
- 停止/启动逻辑集中化
- 状态管理更加精确

### ✅ 提升用户体验
- 减少API调用次数
- 自动处理服务停止
- 清晰的错误消息

---

## 编译验证

```bash
go build -o /tmp/nursorgate ./cmd/nursor 2>&1
# ✅ 编译成功，无警告或错误
```

---

## 文档更新

已更新以下文档：
- ✅ `RUN_API.md` - API使用文档
- ✅ `SWIFT_IMPLEMENTATION.md` - 实现细节
- ✅ `REFACTORING_SUMMARY.md` - 本文档

---

## 总结

通过这次重构，我们实现了：

1. **更智能的模式切换** - 自动停止旧服务
2. **更少的用户交互** - 减少API调用次数
3. **更好的错误处理** - 清晰的提示和验证
4. **更清晰的代码** - 职责明确，易于维护
5. **更完整的功能** - 支持所有运行模式的完整生命周期管理

现在用户可以通过简单的API调用实现HTTP和TUN模式之间的无缝切换！
