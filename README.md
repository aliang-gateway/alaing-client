# NursorDoor

减小体积
```
go build -ldflags="-s -w" -o myapp
```

## 开发日记：
#### 3月16日：
现在发现`1. ai提问，2. 本地的代码编写时的提示`是没有问题的，但是当提问完成之后，需要将变更应用到文件时，就会出现client已经close的情况，导致应用不过来。这个bug也需要解决,有些新人用户就喜欢在旁边指挥着ai写代码；

中间人开发日记：
目前有两大错误一直没有解决，实在是找不到原因：
```
2025/03/17 03:26:04 http2 to server error : decoding error: invalid indexed representation index 73
2025/03/17 03:26:04 failure to read frame, error is read tcp 172.16.91.103:56407->52.86.249.123:443: use of closed network connection
failure to read frame, error is read tcp 172.16.91.103:56407->52.86.249.123:443: use of closed network connection
2025/03/17 03:26:04 decoding error: invalid indexed representation index 73
decoding error: invalid indexed representation index 73
2025/03/17 03:26:04 http2 to client error : read tcp 172.16.91.103:56407->52.86.249.123:443: use of closed network connection
2025/03/17 03:26:04 Received CONNECT request for marketplace.cursorapi.com:443
2025/03/17 03:26:04 Detected HTTP/2 connection
1
2
2025/03/17 03:26:05 http2 处理头部帧
2025/03/17 03:26:05 http2: 发现认证信息: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhdXRoMHx1c2VyXzAxSlBGMzhHRjNQUTMxRk5KR0ZIRDVTVEcwIiwidGltZSI6IjE3NDIxMTYwMjMiLCJyYW5kb21uZXNzIjoiMDcyYmU2ZGQtYzJkOC00N2E5IiwiZXhwIjo0MzM0MTE2MDIzLCJpc3MiOiJodHRwczovL2F1dGhlbnRpY2F0aW9uLmN1cnNvci5zaCIsInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgZW1haWwgb2ZmbGluZV9hY2Nlc3MiLCJhdWQiOiJodHRwczovL2N1cnNvci5jb20ifQ.ZMkSjPgJPF6M4fz8EvXuNn-Qi66aiEqyI7ieU1gylI8
3
2025/03/17 03:26:08 http2 处理头部帧
2025/03/17 03:26:11 Received CONNECT request for api2.cursor.sh:443
2025/03/17 03:26:11 Handling POST /aiserver.v1.AiService/SlashEdit, Proto: HTTP/1.1
2025/03/17 03:26:11 Proxying POST https://api2.cursor.sh:443/aiserver.v1.AiService/SlashEdit, Proto: HTTP/1.1
2025/03/17 03:26:23 http1: Response Body: &
$
"common/test/n
2025/03/17 03:26:23 write tcp 127.0.0.1:56432->127.0.0.1:56410: write: broken pipe
write tcp 127.0.0.1:56432->127.0.0.1:56410: write: broken pipe

```

2. 
```
2025/03/17 02:49:56 http2 处理头部帧
2025/03/17 02:49:56 http2 处理数据帧
98
2025/03/17 02:49:56 failure to read frame, error is connection error: PROTOCOL_ERROR
failure to read frame, error is connection error: PROTOCOL_ERROR
2025/03/17 02:49:56 http2 to server error : connection error: PROTOCOL_ERROR
2025/03/17 02:49:56 failure to read frame, error is read tcp 172.16.91.103:54689->13.248.241.7:443: use of closed network connection
failure to read frame, error is read tcp 172.16.91.103:54689->13.248.241.7:443: use of closed network connection
2025/03/17 02:49:56 connection error: PROTOCOL_ERROR
connection error: PROTOCOL_ERROR
2025/03/17 02:49:56 http2 to client error : read tcp 172.16.91.103:54689->13.248.241.7:443: use of closed network connection
2025/03/17 02:50:24 Received CONNECT request for api2.cursor.sh:443

```
3.
```
2025/03/17 03:56:18 http2 to server error : decoding error: invalid indexed representation index 69
2025/03/17 03:56:18 decoding error: invalid indexed representation index 69
decoding error: invalid indexed representation index 69
2025/03/17 03:56:18 Received CONNECT request for api2.cursor.sh:443
1
2
2025/03/17 03:56:18 http2 处理头部帧
2025/03/17 03:56:18 http2: 发现认证信息: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhdXRoMHx1c2VyXzAxSlBGMzhHRjNQUTMxRk5KR0ZIRDVTVEcwIiwidGltZSI6IjE3NDIxMTYwMjMiLCJyYW5kb21uZXNzIjoiMDcyYmU2ZGQtYzJkOC00N2E5IiwiZXhwIjo0MzM0MTE2MDIzLCJpc3MiOiJodHRwczovL2F1dGhlbnRpY2F0aW9uLmN1cnNvci5zaCIsInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgZW1haWwgb2ZmbGluZV9hY2Nlc3MiLCJhdWQiOiJodHRwczovL2N1cnNvci5jb20ifQ.ZMkSjPgJPF6M4fz8EvXuNn-Qi66aiEqyI7ieU1gylI8

2025/03/17 03:56:19 http2 处理头部帧
2025/03/17 03:56:19 http2 to server error : decoding error: invalid indexed representation index 77
2025/03/17 03:56:19 decoding error: invalid indexed representation index 77
decoding error: invalid indexed representation index 77
2025/03/17 03:56:19 Received CONNECT request for api2.cursor.sh:443
```
一共就三种错误，但是找不到原因，斯密达啊



直接相信ca证书不行，还是要只相信mitm-ca.pem 才可以


```
# Cgo for darwin/arm64
export CGO_ENABLED=1
export GOOS=darwin
export GOARCH=arm64
go build -ldflags="-s -w" -tags=with_utls -buildmode=c-shared -o nursor-core-arm64.dylib

# Cgo for darwin/amd64  
export CGO_ENABLED=1
export GOOS=darwin
export GOARCH=amd64
go build -ldflags="-s -w" -tags=with_utls -buildmode=c-shared -o nursor-core-amd64.dylib

# Cgo for linux/amd64
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
go build -ldflags="-s -w" -tags=with_utls -buildmode=c-shared -o nursor-core-amd64.so

# Cgo for linux/arm64
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=arm64
go build -ldflags="-s -w" -buildmode=c-shared -o nursor-core-arm64.so

# Cgo for windows/amd64
export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64
go build -ldflags="-s -w -X 'nursor.org/nursorgate/common/logger.LogSilent=true'" -tags=with_utls -buildmode=c-shared -o nursor-core-amd64.dll



# runbanle编译

export GOOS=darwin
export GOARCH=arm64
go build -ldflags="-s -w" -tags=with_utls -o core-darwin-arm

export CGO_ENABLED=1
export GOOS=darwin
export GOARCH=amd64
go build -ldflags="-s -w"  -o core-darwin-amd64

export GOOS=linux
export GOARCH=amd64
go build -ldflags="-s -w"  -o core-linux-amd64

set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w"  -o core-win-amd64



```



## 8月4日
1. http2的header的frame，priority要从payload中提取出来，剩余的payload才能被解析；
2. payload中的header被组装后，priority要放回去，不然要出问题，cursor官网无法加载；
3. envoy也可能会将http强行转换成h2，所以这里也要注意




# 开发日记：

runner中config和转换的部分需要区分开