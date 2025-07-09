## AiGate

macos的pf的常见操作

```

开启
sudo pfctl -e

加载配置
sudo pfctl -v -f /Users/liang/MyProgram/goprogram/nursor/nursorgate/client/utils/macos/443pf.conf

清空pf
sudo pfctl -F all

查看pf规则
sudo pfctl -sr

开启forward
sudo ipfw add 100 fwd 127.0.0.1,56433 tcp from any to any 443
```
