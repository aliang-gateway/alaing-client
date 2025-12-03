# FFI Exports Documentation

本文档描述了通过 FFI (Foreign Function Interface) 暴露的核心功能。这些函数直接实现业务逻辑，不依赖于 HTTP 服务。

## 运行控制 API (Run Control)

### runStart(innerToken)
**功能**: 根据当前选择的模式启动对应的服务

**参数**:
- `innerToken` (char*): 内部令牌字符串

**返回值**: JSON 字符串
```json
{
  "status": "success|error",
  "message": "描述信息",
  "details": "详细信息",
  "user_id": 数字 // 当且仅当成功时
}
```

**行为**:
- **HTTP 模式**: 返回 HTTP 服务器已运行的提示（HTTP 在 swift 中自动启动）
- **TUN 模式**: 启动 TUN 服务，阻塞直到启动完成
- **未选择模式**: 返回错误

### runStop()
**功能**: 停止当前运行的服务

**参数**: 无

**返回值**: JSON 字符串
```json
{
  "status": "success|error",
  "message": "停止服务成功|失败",
  "stopped_mode": "http|tun",
  "details": "详细信息"
}
```

**行为**:
- 检查是否有服务在运行（tunRunning 标志）
- 根据当前模式停止对应的服务
- 清除 tunRunning 标志，但保留 currentMode（下次可继续启动相同模式）

### runStatus()
**功能**: 查询当前运行状态

**参数**: 无

**返回值**: JSON 字符串
```json
{
  "current_mode": "http|tun",
  "tun_running": true|false,
  "available_modes": ["http", "tun"],
  "status": "详细状态信息",
  "description": "模式描述"
}
```

**返回的状态**:
- 当前模式已选择并运行: "HTTP proxy server is running" / "TUN service is running"
- 当前模式已选择但未运行: "HTTP mode selected, service not running" / "TUN mode selected, service not running"

### runSwift(targetMode)
**功能**: 切换运行模式

**参数**:
- `targetMode` (char*): 目标模式，"http" 或 "tun"

**返回值**: JSON 字符串
```json
{
  "status": "switched|already_running|error",
  "target_mode": "http|tun",
  "message": "描述信息",
  "usage": "使用示例",
  "details": "详细信息",
  "next_action": "后续操作提示"
}
```

**工作流程**:
1. 验证目标模式有效性 ("http" 或 "tun")
2. 检查是否已处于目标模式
3. 如果切换模式且当前有服务在运行:
   - 调用 runStop() 停止当前服务
   - 等待服务完全停止
4. 设置新模式
5. 根据目标模式启动服务:
   - **HTTP 模式**: 在后台 goroutine 启动 HTTP 代理服务器
   - **TUN 模式**: 仅设置模式，不启动服务（需要后续调用 runStart）

### runSetUserInfo(innerToken, username, password, userUUID)
**功能**: 设置用户认证信息

**参数**:
- `innerToken` (char*): 内部令牌
- `username` (char*): 用户名
- `password` (char*): 密码
- `userUUID` (char*): 用户 UUID

**返回值**: JSON 字符串
```json
{
  "status": "success",
  "user_id": 用户ID数字
}
```

**行为**:
- 设置用户认证凭证
- 设置日志用户信息标签
- 返回当前用户 ID

## 代理管理 API

### 待实现
当前版本仅暴露了运行控制相关的 FFI 函数。以下功能可在后续版本中添加：

- proxyCurrentGet() - 获取当前代理
- proxyCurrentSet(name) - 设置当前代理
- proxyRegistryList() - 列出所有代理
- proxyRegistryGet(name) - 获取指定代理
- proxyRegistryRegister(name, configJSON) - 注册新代理
- proxyRegistryUnregister(name) - 注销代理
- proxyRegistrySetDefault(name) - 设置默认代理
- proxyRegistrySetDoor(name) - 设置门代理
- proxyRegistrySwitch(name) - 切换代理

## 实现细节

### 状态管理
所有状态都通过线程安全的全局变量管理，受 `sync.RWMutex` 保护：

- `currentMode`: 当前运行模式 ("http" 或 "tun")
- `tunRunning`: 当前是否有服务在运行

### 状态查询函数 (handlers 包中)
这些函数被 FFI 导出函数使用：

```go
// 获取当前模式
func GetCurrentMode() string

// 设置当前模式
func SetCurrentMode(mode string)

// 检查服务是否在运行
func IsTunRunning() bool

// 设置运行状态
func SetTunRunning(running bool)
```

### 错误处理
- 所有返回值都是 JSON 字符串，包含 "status" 字段
- "status" 为 "error" 时，查看 "message" 字段了解错误原因
- FFI 调用不会 panic，所有错误都会被捕获并返回

## FFI 调用示例

### C/C++ 示例
```c
#include <stdlib.h>

// 声明 FFI 函数
extern const char* runStatus(void);
extern const char* runSwift(const char* targetMode);
extern const char* runStart(const char* innerToken);

int main() {
    // 查询状态
    const char* status = runStatus();
    printf("Status: %s\n", status);
    free((void*)status);

    // 切换到 HTTP 模式
    const char* result = runSwift("http");
    printf("Result: %s\n", result);
    free((void*)result);

    return 0;
}
```

### 编译 FFI 库
```bash
# 编译为共享库
go build -o libnursorgate.so -buildmode=c-shared export.go
```

## 关键特性

✅ **无 HTTP 依赖**: FFI 调用直接访问核心逻辑，不需要启动 HTTP 服务

✅ **线程安全**: 所有状态访问都受互斥锁保护

✅ **异步服务启动**: HTTP 服务在后台 goroutine 启动，不阻塞 FFI 调用

✅ **模式隔离**: 同一时间仅一种模式处于运行状态

✅ **自动服务停止**: 切换模式时自动停止前一个服务

✅ **JSON 响应**: 所有返回值都是结构化的 JSON，便于解析

## 版本历史

- **v1.0.0** (2025-12-03)
  - 初始版本，暴露运行控制 API
  - 支持 HTTP/TUN 模式切换
  - 用户信息设置
  - 状态查询

## 许可

与 Nursor 项目主项目许可相同
