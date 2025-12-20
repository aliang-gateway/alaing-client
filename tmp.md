有以下几个问题：
1. @processor/rules/engine.go 下的func (e *RuleEngine) Initialize(config *model.RoutingRulesConfig) error 方法，仅仅在tun模式下被调用：initializeRuleEngineForTUN，正确的是不管那种模式，只要在路由相关的配置被加载的时候，都被调用；
2. app/http/handlers/config_handler.go 这个文件里，当post的方法的时候，是更新用户本地的配置，同时停止从nacos的server监听，你竟然在里边写入配置到nacos，这是万万不行的；
3. 这是为auto-update之后，我只看到了设置config.Settings.AutoUpdate = true，但是启动nacos的自动更新，有没有实现；
