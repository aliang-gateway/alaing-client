# Nonelane core
以前名字交nursor，现在改名字到nonelane来了；

减小体积
```
go build -ldflags="-s -w" -o myapp
```

## 开发日记：
#### 3月16日：
现在发现`1. ai提问，2. 本地的代码编写时的提示`是没有问题的，但是当提问完成之后，需要将变更应用到文件时，就会出现client已经close的情况，导致应用不过来。这个bug也需要解决,有些新人用户就喜欢在旁边指挥着ai写代码；


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