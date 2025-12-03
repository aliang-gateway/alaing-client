# HTTP Proxy Test Guide

## 概述
项目中的 `test/proxy_test.go` 提供了两个test函数来启动和测试HTTP代理服务器：

- `TestHTTPProxyWithConfig`: 使用指定的配置文件启动代理
- `TestHTTPProxyDefault`: 使用默认配置启动代理

## 使用方法

### 1. 指定配置文件的方式（推荐）

```bash
cd /Users/mac/MyProgram/GoProgram/nursor/nursorgate2

# 方式1: 使用默认配置文件 (config.test.json)
go test -v -run TestHTTPProxyWithConfig ./test

# 方式2: 通过环境变量指定自定义配置文件
CONFIG_PATH=/path/to/your/config.json go test -v -run TestHTTPProxyWithConfig ./test

# 方式3: 不限制超时时间运行测试（推荐）
go test -v -run TestHTTPProxyWithConfig ./test -timeout 0
```

### 2. 使用默认配置启动

```bash
go test -v -run TestHTTPProxyDefault ./test -timeout 0
```

## 配置文件格式

如果需要自定义配置，创建配置文件（如 `config.test.json`）：

```json
{
  "engine": {
    "mtu": 1500,
    "fwmark": 0,
    "restapi": "127.0.0.1:56431",
    "device": "utun0",
    "loglevel": "debug",
    "interface": "127.0.0.1",
    "tcp-moderate-receive-buffer": false,
    "tcp-send-buffer-size": "2097152",
    "tcp-receive-buffer-size": "2097152",
    "multicast-groups": "",
    "tun-pre-up": "",
    "tun-post-up": "",
    "udp-timeout": "60s"
  },
  "currentProxy": "direct",
  "coreServer": "https://api2.nursor.org:12235",
  "proxies": {
    "direct": {
      "type": "direct",
      "settings": {}
    }
  }
}
```

## 测试HTTP代理

当你启动了test后，HTTP代理服务器会监听在 `http://127.0.0.1:56432`

### 测试命令

#### 1. 测试HTTP CONNECT（HTTPS隧道）- 最常见的使用方式
```bash
curl -x http://127.0.0.1:56432 https://www.google.com
```

#### 2. 使用详细输出查看连接过程
```bash
curl -x http://127.0.0.1:56432 -v https://www.example.com
```

#### 3. 测试HTTP透明代理
```bash
curl -x http://127.0.0.1:56432 http://www.example.com
```

#### 4. 测试不同的网站
```bash
curl -x http://127.0.0.1:56432 https://www.github.com
curl -x http://127.0.0.1:56432 https://www.baidu.com
```

#### 5. 测试请求头和其他HTTP方法
```bash
curl -x http://127.0.0.1:56432 -H "User-Agent: Mozilla/5.0" https://www.example.com
```

## 预期行为

### 成功的连接
- 服务器应该输出调试日志
- curl应该成功建立连接并获取响应
- 应该看到 `200 Connection Established` 响应

### 日志输出示例
```
2025/12/03 18:46:37 CONNECT tunnel: hostname=www.example.com, port=443, dstIP=93.184.216.34, srcIP=127.0.0.1
2025/12/03 18:46:37 Routing CONNECT through TCP handler for www.example.com:443
2025/12/03 18:46:37 Connected to target server: 93.184.216.34:443
```

## 关闭测试

按 `Ctrl+C` 组合键来优雅地关闭服务器。

## 故障排除

### 连接被拒绝
```
curl: (7) Failed to connect to 127.0.0.1 port 56432
```
确保测试正在运行，并检查防火墙设置。

### SSL/TLS错误
```
curl: (35) LibreSSL SSL_connect: SSL_ERROR_SYSCALL
```
这可能是DNS解析问题或目标服务器无法访问。检查日志输出。

### 配置文件找不到
```
[WARN] Config file not found: ./config.test.json, using minimal defaults
```
这是正常的，程序会使用默认配置继续运行。

## 实时日志监控

Test运行时会输出详细的调试日志。你可以：

1. 查看连接元数据提取
2. 追踪TCP处理流程
3. 观察DNS解析
4. 监控双向数据中继

## 同时修改代码和测试

1. 在编辑器中修改代码
2. 停止当前测试（Ctrl+C）
3. 运行新的测试
4. 重复步骤1-3

这样可以快速迭代开发和测试。
