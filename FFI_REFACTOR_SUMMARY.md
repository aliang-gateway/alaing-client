# FFI 导出函数重构总结

## 目标
将 HTTP API 处理程序的核心功能直接暴露为 FFI 导出函数，使得在 FFI 模式下可以直接调用业务逻辑，无需启动 HTTP 服务。

## 改动概览

### 1. 删除的内容
- ❌ 从 `export.go` 中删除了所有 HTTP 处理程序包装函数
- ❌ 删除了 `responseRecorder` HTTP 响应模拟实现
- ❌ 删除了 `createTestMux()` 路由创建函数
- ❌ 移除了不必要的 `net/http` 导入

### 2. 新增的内容

#### export.go 中的 FFI 导出函数
这些函数直接实现业务逻辑，不通过 HTTP 路由：

**运行控制 API**:
- `runStart(innerToken)` - 启动服务
- `runStop()` - 停止服务
- `runStatus()` - 查询状态
- `runSwift(targetMode)` - 切换模式
- `runSetUserInfo(...)` - 设置用户信息

#### app/http/handlers/run_handler.go 中的状态管理函数
新增了公开的状态管理接口供 FFI 调用：

```go
// GetCurrentMode() - 获取当前模式
// SetCurrentMode(mode) - 设置当前模式
// IsTunRunning() - 检查服务是否运行
// SetTunRunning(running) - 设置运行状态
```

这些函数都是线程安全的，受 `sync.RWMutex` 保护。

#### 公开的 HTTP 处理程序函数
将所有 HTTP 处理函数从私有改为公开（大写首字母），供 HTTP 路由使用：

**run_handler.go**:
- `HandleRunStart()` (was handleRun)
- `HandleRunStop()` (was handleStop)
- `HandleRunStatus()` (was handleStatus)
- `HandleRunSwift()` (was handleSwift)
- `HandleRunTUN()` (was handleRunTUN)
- `HandleRunUserInfo()` (was handleRunUserInfo)

**proxy_handler.go**:
- `HandleGetCurrentProxy()` (was handleGetCurrentProxy)
- `HandleSetCurrentProxy()` (was handleSetCurrentProxy)

**proxy_registry_handler.go**:
- `HandleProxyRegistryList()` (was handleProxyRegistryList)
- `HandleProxyRegistryGet()` (was handleProxyRegistryGet)
- `HandleProxyRegistryRegister()` (was handleProxyRegistryRegister)
- `HandleProxyRegistryUnregister()` (was handleProxyRegistryUnregister)
- `HandleProxyRegistrySetDefault()` (was handleProxyRegistrySetDefault)
- `HandleProxyRegistrySetDoor()` (was handleProxyRegistrySetDoor)
- `HandleProxyRegistrySwitch()` (was handleProxyRegistrySwitch)

**token_handler.go**:
- `HandleTokenSet()` (was handleTokenSet)
- `HandleTokenGet()` (was handleTokenGet)

### 3. 核心设计改变

#### 之前的架构
```
HTTP 请求 → HTTP 处理程序 → HTTP 响应
FFI 调用 → HTTP 处理程序包装 → HTTP 处理程序 → 响应
```

这种方式有两个问题：
1. FFI 调用需要经过 HTTP 处理程序，增加复杂性
2. HTTP 处理程序只能返回 http.ResponseWriter，不便于直接返回 JSON

#### 新的架构
```
HTTP 请求 → HTTP 处理程序 → HTTP 响应
FFI 调用 → 直接调用业务逻辑 → JSON 响应
```

好处：
1. ✅ FFI 调用直接访问核心逻辑，无需 HTTP 路由
2. ✅ 在 FFI 模式下不需要启动 HTTP 服务
3. ✅ 更高的性能（减少协议转换开销）
4. ✅ 更清晰的代码架构（业务逻辑与 HTTP 处理分离）

### 4. 实现细节

#### runStart 函数
```go
func runStart(innerToken *C.char) *C.char {
    // 设置令牌
    user.SetInnerToken(C.GoString(innerToken))

    // 根据 currentMode 启动服务
    switch handlers.GetCurrentMode() {
    case "http":
        // HTTP 已在 swift 中启动
        return "HTTP is already running"
    case "tun":
        // 启动 TUN 服务
        go runner2.Start()
        res := <-runner2.RunStatusChan
        return result
    }
}
```

#### runSwift 函数
```go
func runSwift(targetMode *C.char) *C.char {
    targetModeStr := C.GoString(targetMode)

    currentMode := handlers.GetCurrentMode()

    // 如果切换模式且当前有服务运行
    if currentMode != targetModeStr && handlers.IsTunRunning() {
        // 停止当前服务
        handlers.SetTunRunning(false)
        stopServiceCore(currentMode)
    }

    // 设置新模式
    handlers.SetCurrentMode(targetModeStr)

    // 根据模式启动服务
    if targetModeStr == "http" {
        go func() {
            handlers.SetTunRunning(true)
            httpServer.StartMitmHttp()
            handlers.SetTunRunning(false)
        }()
    }

    return result
}
```

#### runStop 函数
```go
func runStop() *C.char {
    currentMode := handlers.GetCurrentMode()

    // 只检查 tunRunning（是否有服务在运行）
    if !handlers.IsTunRunning() {
        return "No service is running"
    }

    handlers.SetTunRunning(false)

    // 根据 currentMode 停止对应的服务
    switch currentMode {
    case "http":
        httpServer.StopHttpProxy()
    case "tun":
        tun.Stop()
    }

    return result
}
```

### 5. 状态管理改进

#### 两个独立的状态
- **currentMode**: 当前选择的模式 ("http" 或 "tun")
  - 在 runSwift 中改变
  - 持久化，用于下次启动相同模式

- **tunRunning**: 服务是否在运行 (bool)
  - 在 runStart/runStop 中改变
  - 用于防止重复启动

#### 好处
这种分离设计有以下好处：

1. **模式持久化**: 停止服务后，下次可直接启动相同模式
2. **灵活的启动**: 可以在模式之间灵活切换
3. **线程安全**: 所有访问都受 RWMutex 保护
4. **清晰的语义**:
   - "当前模式是什么" 和 "现在有服务在运行吗" 是两个不同的问题

## 文件变更清单

### 新增文件
- ✅ `FFI_EXPORTS.md` - FFI 导出函数文档
- ✅ `FFI_REFACTOR_SUMMARY.md` - 本文件

### 修改的文件
- ✅ `export.go` - 替换为直接调用业务逻辑的函数
- ✅ `app/http/handlers/run_handler.go` - 新增状态管理函数，公开处理程序
- ✅ `app/http/handlers/proxy_handler.go` - 公开处理程序函数
- ✅ `app/http/handlers/proxy_registry_handler.go` - 公开处理程序函数
- ✅ `app/http/handlers/token_handler.go` - 公开处理程序函数

## 编译验证

✅ 代码编译成功，无错误或警告
```bash
$ go build -o /tmp/nursorgate ./cmd/nursor
# 编译成功
```

## 使用示例

### FFI 调用（无需 HTTP）
```c
// 查询状态
const char* status = runStatus();  // 直接调用，无需 HTTP 服务

// 切换模式并启动服务
runSwift("http");  // 直接启动 HTTP，无需分别调用 swift 和 start
```

### HTTP 调用（保持不变）
```bash
curl http://localhost:56431/run/status
curl -X POST http://localhost:56431/run/swift -d '{"target_mode":"http"}'
```

## 性能影响

- **FFI 模式**: 性能提升（减少 HTTP 协议处理开销）
- **HTTP 模式**: 性能不变（使用相同的处理程序）
- **内存占用**: 略微增加（新增状态管理函数，但非常小）

## 后续计划

### Phase 2: 代理管理 FFI 函数
可以在后续版本中添加：
- proxyCurrentGet()
- proxyCurrentSet()
- proxyRegistryList()
- proxyRegistryRegister()
- 等等...

### Phase 3: 日志管理 FFI 函数
- 扩展现有的日志相关导出函数
- 直接访问日志配置而不需要 HTTP

## 总结

这次重构实现了**两个调用方式共存**的架构：

1. **HTTP 调用**: 通过 HTTP 服务的处理程序（保持不变）
2. **FFI 调用**: 直接调用业务逻辑（新增）

优势：
- ✅ FFI 调用更高效（无需 HTTP 开销）
- ✅ 代码架构更清晰（业务逻辑与 HTTP 分离）
- ✅ 易于扩展（新增功能既可以通过 HTTP 也可以通过 FFI）
- ✅ 完全向后兼容（HTTP 调用不受影响）

---

**完成日期**: 2025-12-03
**状态**: ✅ 完成并通过编译验证
