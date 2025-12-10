# 综合DNS本地缓存解决方案规划

## 用户需求
建立尽可能全面的DNS本地缓存能力，解决以下问题：
- TUN截获的TCP流量只有TCP层信息，缺少DNS信息
- SNI提取的域名应该与IP进行关联和缓存
- 需要IP反查功能来恢复丢失的域名信息
- 建立完整的DNS结果缓存系统

---

## 方案分析

### 方案1: TUN层DNS截获与本地DNS服务
**文件**: `/inbound/tun/tunnel/udp.go` 中实现

#### 当前状态
- ✅ 被动截获：所有UDP/53流量被`dnsLoggingPacketConn`包装，记录查询域名
- ❌ 缺陷：仅记录，不拦截，无法修改查询或响应

#### 目标功能
1. **本地DNS代理**：在127.0.0.1:53创建本地DNS服务器
2. **拦截系统DNS**：将系统DNS客户端重定向到本地代理
3. **DNS缓存查询**：先查本地缓存，未命中则转发上游
4. **系统无关**：不依赖系统DNS配置

#### 实现难度
- **高**：需要实现DNS协议处理、缓存管理、本地监听等
- 涉及修改文件：
  - `inbound/tun/tunnel/udp.go` - 添加DNS拦截处理器
  - `processor/dns/` - 新建DNS服务器模块
  - `inbound/tun/engine/engine.go` - DNS服务器初始化

#### 优缺点
| 优点 | 缺点 |
|------|------|
| 全局DNS缓存 | 需要重定向系统DNS |
| 可控的DNS策略 | 实现复杂度高 |
| 支持域名黑白名单 | 需要处理DNSSEC |
| 可以拦截DNS污染 | macOS/Windows需要不同方案 |

#### 预期效果
- 可以缓存所有系统DNS查询结果
- 减少DNS延迟（本地缓存命中）
- 可以检测和修改特定域名的DNS结果

---

### 方案2: SNI自动关联到DNS缓存
**文件**: `/processor/tcp/handler.go` 和 `/processor/cache/`

#### 当前状态
- ✅ SNI提取：`handleTLS()` 从TLS ClientHello中提取域名
- ✅ IP-Domain缓存：LRU缓存存储IP↔Domain映射
- ❌ 缺陷：缓存仅用于路由决策，不用于DNS结果缓存；反向查询不完整

#### 目标功能
1. **自动DNS绑定**：SNI提取时，自动绑定IP→Domain
2. **路由决策同步**：将路由决策与DNS结果同时缓存
3. **缓存统一化**：统一的数据结构存储DNS结果和路由
4. **TTL管理**：根据实际连接有效期管理缓存

#### 实现难度
- **低**：在现有缓存基础上扩展
- 涉及修改文件：
  - `inbound/tun/metadata/metadata.go` - 扩展Metadata字段
  - `processor/tcp/handler.go` - SNI绑定到缓存
  - `processor/cache/ipdomain.go` - 增强缓存数据结构
  - `processor/rules/engine.go` - 缓存查询增强

#### 优缺点
| 优点 | 缺点 |
|------|------|
| 实现简单 | 仅缓存已访问的域名 |
| 利用现有代码 | 新访问首次仍需SNI提取 |
| 无系统依赖 | 不能拦截不访问的域名 |
| 立竿见影 | 需要长连接才能有效 |

#### 预期效果
- 避免重复的SNI提取（同一域名第二次访问快速）
- 建立IP→Domain映射库（便于分析）
- 支持缓存查询和统计

---

### 方案3: IP反查能力（Reverse DNS + GeoIP）
**文件**: `/common/geoip/` 和 `/processor/dns/`

#### 当前状态
- ✅ GeoIP查询：有GeoIP数据库（Country判断）
- ❌ 缺陷：只能查询地理位置，不能查询域名

#### 目标功能
1. **Reverse DNS查询**：通过PTR记录从IP查询域名
2. **缓存聚合**：将所有已知IP与其可能的域名关联
3. **黑名单反查**：通过IP识别恶意域名
4. **热点统计**：识别常访问的IP段和域名

#### 实现难度
- **中等**：需要集成Reverse DNS库或实现PTR查询
- 涉及修改文件：
  - `processor/dns/` - 新建Reverse DNS模块
  - `processor/cache/ipdomain.go` - 扩展反向查询能力
  - `common/geoip/` - 添加Reverse DNS功能

#### 优缺点
| 优点 | 缺点 |
|------|------|
| 补充丢失的域名信息 | 准确度不高（PTR记录很少） |
| 低成本扩展 | 查询延迟（互联网查询） |
| 可选的离线库 | 需要额外维护 |
| 支持离线IP库 | 库更新频率问题 |

#### 预期效果
- 通过IP反查恢复部分域名信息
- 建立IP与域名的多对多关系
- 支持域名溯源

---

### 方案4: 其他补充方案

#### 4a. HTTP头解析（已部分实现）
**现状**：HTTP/HTTPS请求中的Host头可以提供域名
**扩展**：
- 增强HTTP代理的域名提取
- 从HTTP/2伪头部提取`:authority`
- 存储到缓存

#### 4b. SOCKS5协议解析
**目标**：从SOCKS5请求中提取目标域名
**说明**：某些代理工具使用SOCKS5，其中包含域名信息
**用途**：识别某些特殊应用的目标域名

#### 4c. DNS over HTTPS (DoH) 拦截
**现状**：已识别DoH提供商（dns.google, 1.1.1.1等）
**扩展**：
- 拦截HTTPS/443上的DoH请求
- 提取DoH中的域名查询参数
- 缓存DoH查询结果

#### 4d. 本地DNS缓存API
**目标**：HTTP接口查看和管理缓存
**路径**：`GET /api/dns/cache`、`DELETE /api/dns/cache/{domain}`
**用途**：实时监控和调试

---

## 推荐的综合方案（分阶段实施）

### 第一阶段（推荐先做）：SNI自动关联 + HTTP头提取增强
**优点**：投入少，效果明显，基础牢固
**预期缓存覆盖**：所有已访问的HTTPS域名
**工作量**：2-3小时
**优先级**：⭐⭐⭐⭐⭐ 最优先

### 第二阶段（必做）：IP反查能力 + 倒排索引
**优点**：建立完整的IP↔Domain双向映射
**预期缓存覆盖**：所有已访问IP对应的域名（包括直连IP）
**工作量**：3-4小时
**优先级**：⭐⭐⭐⭐⭐ 紧跟第一阶段

### 第三阶段（必做）：DNS缓存管理API + 可视化
**优点**：实时监控缓存，查看热点域名和IP
**预期功能**：
- `GET /api/dns/cache` - 查看所有缓存条目
- `GET /api/dns/stats` - 缓存统计信息
- `GET /api/dns/hotspots` - 热点域名和IP
- `DELETE /api/dns/cache/{domain}` - 清除缓存
- 管理后台展示缓存状态
**工作量**：2-3小时
**优先级**：⭐⭐⭐⭐⭐ 必须有

### 第四阶段（可选增强）：本地DNS拦截服务
**优点**：全局覆盖，系统级缓存，支持DNS策略
**预期缓存覆盖**：所有应用的DNS查询（包括后台应用）
**工作量**：4-6小时
**优先级**：⭐⭐⭐ 有余力再做

### 第五阶段（可选增强）：DoH拦截 + DNS黑名单
**优点**：捕获DNS over HTTPS查询，支持域名黑名单
**工作量**：2-3小时
**优先级**：⭐⭐ 最后考虑

---

## 详细实现路径

### 第一阶段：SNI自动关联

#### Step 1: 扩展Metadata数据结构
**文件**: `/inbound/tun/metadata/metadata.go`

```go
type Metadata struct {
    // 现有字段...

    // 新增DNS关联字段
    DNSInfo *DNSInfo  // DNS相关信息
}

type DNSInfo struct {
    // 域名绑定来源
    BindingSource BindingSource    // "sni" / "http_host" / "dns" / "connect"
    BindingTime   time.Time        // 绑定时间

    // 缓存管理
    CacheTTL      time.Duration    // 建议的缓存TTL
    ShouldCache   bool             // 是否应该缓存
}

type BindingSource string
const (
    BindingSourceSNI     BindingSource = "sni"
    BindingSourceHTTP    BindingSource = "http_host"
    BindingSourceDNS     BindingSource = "dns"
    BindingSourceCONNECT BindingSource = "connect"
)
```

#### Step 2: 修改SNI提取处理
**文件**: `/processor/tcp/handler.go` 的 `handleTLS()` 方法

修改第125行后的代码：
```go
sni, sniBuf, err = h.tlsHandler.ExtractSNI(ctx, originConn)
metadata.HostName = sni

// 新增：记录SNI绑定信息
if sni != "" {
    metadata.DNSInfo = &M.DNSInfo{
        BindingSource: M.BindingSourceSNI,
        BindingTime:   time.Now(),
        CacheTTL:      5 * time.Minute,  // 默认5分钟
        ShouldCache:   true,
    }
}
```

#### Step 3: 增强缓存存储
**文件**: `/processor/cache/ipdomain.go`

修改`CacheEntry`结构：
```go
type CacheEntry struct {
    Domain        string
    IP            netip.Addr
    Route         RouteDecision
    BindingSources []BindingSource   // 新增：可能来自多个来源
    ExpiresAt     time.Time
    CreatedAt     time.Time
}
```

修改缓存存储逻辑：
```go
// 在规则引擎(engine.go)的cacheResult()方法中
if metadata.DNSInfo != nil && metadata.DNSInfo.ShouldCache {
    entry := &cache.CacheEntry{
        Domain:        metadata.HostName,
        IP:            metadata.DstIP,
        Route:         route,
        BindingSources: []BindingSource{metadata.DNSInfo.BindingSource},
        CreatedAt:     time.Now(),
    }
    e.ipDomainCache.SetWithTTL(
        metadata.HostName,
        entry,
        metadata.DNSInfo.CacheTTL,
    )
}
```

#### Step 4: 增强HTTP元数据提取
**文件**: `/inbound/http/metadata_extractor.go`

在HTTP Host头提取时添加绑定信息：
```go
func ExtractMetadataFromHTTP(...) {
    // ... 现有代码 ...

    metadata.DNSInfo = &M.DNSInfo{
        BindingSource: M.BindingSourceHTTP,
        BindingTime:   time.Now(),
        CacheTTL:      10 * time.Minute,
        ShouldCache:   true,
    }
}
```

---

### 第二阶段：IP反查能力 + 倒排索引

#### 核心设计
在IP-Domain缓存基础上建立**倒排索引**：
```
正向查询:  Domain → {IPs, Route, Sources}
反向查询:  IP → {Domains, Routes, SourceCount}

例如:
  正向: "www.google.com" → [1.2.3.4, 1.2.3.5] (from SNI, HTTP)
  反向: "1.2.3.4" → ["www.google.com", "google.com"] (2个来源)
```

#### Step 1: 扩展缓存数据结构
**文件**: `/processor/cache/ipdomain.go`

添加倒排索引：
```go
type IPDomainCache struct {
    mu         sync.RWMutex
    entries    map[string]*list.Element       // 正向索引: domain → entry
    ipIndex    map[string][]*CacheEntry      // 反向索引: IP → entries
    lru        *list.List
    maxEntries int
    defaultTTL time.Duration
}

type CacheEntry struct {
    Domain         string
    IPs            []netip.Addr           // 支持一个域名多个IP
    Route          RouteDecision
    BindingSources []BindingSource
    ExpiresAt      time.Time
    CreatedAt      time.Time
}
```

#### Step 2: 实现双向查询接口
添加到`IPDomainCache`：

```go
// 正向查询：域名 → IP列表
func (c *IPDomainCache) GetByDomain(domain string) (*CacheEntry, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if elem, ok := c.entries[domain]; ok {
        entry := elem.Value.(*CacheEntry)
        if time.Now().Before(entry.ExpiresAt) {
            return entry, true
        }
    }
    return nil, false
}

// 反向查询：IP → 域名列表
func (c *IPDomainCache) GetByIP(ip netip.Addr) []*CacheEntry {
    c.mu.RLock()
    defer c.mu.RUnlock()

    ipStr := ip.String()
    if entries, ok := c.ipIndex[ipStr]; ok {
        var validEntries []*CacheEntry
        now := time.Now()

        for _, entry := range entries {
            if now.Before(entry.ExpiresAt) {
                validEntries = append(validEntries, entry)
            }
        }
        return validEntries
    }
    return nil
}

// 批量反向查询：多个IP → 域名集合
func (c *IPDomainCache) GetDomainsForIPs(ips []netip.Addr) map[string][]string {
    result := make(map[string][]string)

    for _, ip := range ips {
        entries := c.GetByIP(ip)
        for _, entry := range entries {
            result[ip.String()] = append(result[ip.String()], entry.Domain)
        }
    }

    return result
}
```

#### Step 3: 修改缓存存储逻辑
**文件**: `/processor/cache/ipdomain.go`

修改`Set`方法以维护倒排索引：

```go
func (c *IPDomainCache) SetWithTTL(key string, entry *CacheEntry, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 正向索引
    if elem, exists := c.entries[key]; exists {
        c.lru.MoveToFront(elem)
        elem.Value = entry
    } else {
        if len(c.entries) >= c.maxEntries {
            c.evict()  // LRU淘汰
        }
        elem := c.lru.PushFront(entry)
        c.entries[key] = elem
    }

    // 倒排索引：为每个IP维护entry列表
    for _, ip := range entry.IPs {
        ipStr := ip.String()

        // 移除旧的entry（如果存在）
        oldEntries := c.ipIndex[ipStr]
        var newEntries []*CacheEntry
        for _, e := range oldEntries {
            if e.Domain != entry.Domain {
                newEntries = append(newEntries, e)
            }
        }

        // 添加新entry
        newEntries = append(newEntries, entry)
        c.ipIndex[ipStr] = newEntries
    }
}
```

#### Step 4: IP热点统计
添加到`IPDomainCache`：

```go
type IPStatistics struct {
    IP            netip.Addr
    AssociatedDomains []string    // 关联的所有域名
    HitCount      uint64          // IP被访问次数
    SourceCount   int             // 信息来源数
    FirstSeen     time.Time       // 首次出现
    LastSeen      time.Time       // 最后出现
    IsHotspot     bool            // 是否热点IP
}

func (c *IPDomainCache) GetIPStatistics(ip netip.Addr) *IPStatistics {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entries := c.ipIndex[ip.String()]
    if len(entries) == 0 {
        return nil
    }

    var domains []string
    var hitCount uint64
    var sources = make(map[BindingSource]bool)
    var firstSeen, lastSeen time.Time

    for _, entry := range entries {
        domains = append(domains, entry.Domain)
        hitCount += entry.HitCount
        firstSeen = entry.CreatedAt
        lastSeen = entry.CreatedAt

        for _, source := range entry.BindingSources {
            sources[source] = true
        }
    }

    return &IPStatistics{
        IP:                    ip,
        AssociatedDomains:     domains,
        HitCount:              hitCount,
        SourceCount:           len(sources),
        FirstSeen:             firstSeen,
        LastSeen:              lastSeen,
        IsHotspot:             hitCount > 100,  // 可配置阈值
    }
}

// 获取所有热点IP
func (c *IPDomainCache) GetHotspotIPs(limit int) []*IPStatistics {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var stats []*IPStatistics
    for ipStr := range c.ipIndex {
        ip, _ := netip.ParseAddr(ipStr)
        stat := c.GetIPStatistics(ip)
        if stat != nil && stat.IsHotspot {
            stats = append(stats, stat)
        }
    }

    // 按HitCount排序
    sort.Slice(stats, func(i, j int) bool {
        return stats[i].HitCount > stats[j].HitCount
    })

    if len(stats) > limit {
        stats = stats[:limit]
    }

    return stats
}
```

---

### 第三阶段：DNS缓存管理API + 可视化

#### Step 1: 创建DNS缓存API处理器
**新建文件**: `/app/http/handlers/dns_cache.go`

```go
type DNSCacheHandler struct {
    cache *cache.IPDomainCache
}

// GET /api/dns/cache - 查看所有缓存
func (h *DNSCacheHandler) GetCacheEntries(w http.ResponseWriter, r *http.Request) {
    page := r.URL.Query().Get("page")      // 分页支持
    limit := r.URL.Query().Get("limit")
    sortBy := r.URL.Query().Get("sortBy")  // domain, hits, recent

    entries := h.cache.GetAll()            // 获取所有条目

    // 分页和排序
    sorted := sortEntries(entries, sortBy)
    paginated := paginateEntries(sorted, page, limit)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "entries": paginated,
        "total":   len(entries),
        "page":    page,
        "limit":   limit,
    })
}

// GET /api/dns/stats - 缓存统计信息
func (h *DNSCacheHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
    stats := h.cache.Stats()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "totalEntries":   stats.TotalEntries,
        "uniqueDomains":  stats.DomainCount,
        "uniqueIPs":      stats.IPCount,
        "cacheHits":      stats.CacheHits,
        "cacheMisses":    stats.CacheMisses,
        "hitRate":        stats.HitRate,
        "oldestEntry":    stats.OldestEntry,
        "newestEntry":    stats.NewestEntry,
    })
}

// GET /api/dns/hotspots - 热点域名和IP
func (h *DNSCacheHandler) GetHotspots(w http.ResponseWriter, r *http.Request) {
    topCount := 20  // 可配置

    hotDomains := h.cache.GetHotspotDomains(topCount)
    hotIPs := h.cache.GetHotspotIPs(topCount)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "topDomains": hotDomains,  // 访问最频繁的域名
        "topIPs":     hotIPs,      // 访问最频繁的IP
    })
}

// GET /api/dns/cache/query?domain=example.com - 查询单个域名
func (h *DNSCacheHandler) QueryDomain(w http.ResponseWriter, r *http.Request) {
    domain := r.URL.Query().Get("domain")

    entry, found := h.cache.GetByDomain(domain)
    if !found {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{"error": "domain not found"})
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(entry)
}

// GET /api/dns/cache/reverse?ip=1.2.3.4 - IP反查
func (h *DNSCacheHandler) ReverseQuery(w http.ResponseWriter, r *http.Request) {
    ipStr := r.URL.Query().Get("ip")
    ip, err := netip.ParseAddr(ipStr)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "invalid IP"})
        return
    }

    entries := h.cache.GetByIP(ip)
    stats := h.cache.GetIPStatistics(ip)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "domains":    entries,
        "statistics": stats,
    })
}

// DELETE /api/dns/cache/{domain} - 清除特定缓存
func (h *DNSCacheHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
    domain := r.PathValue("domain")

    h.cache.Delete(domain)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// DELETE /api/dns/cache - 清除所有缓存
func (h *DNSCacheHandler) ClearAll(w http.ResponseWriter, r *http.Request) {
    h.cache.Clear()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}
```

#### Step 2: 在HTTP路由中注册
**文件**: `/app/http/server.go`

```go
func setupDNSCacheRoutes(mux *http.ServeMux, cache *cache.IPDomainCache) {
    handler := &handlers.DNSCacheHandler{Cache: cache}

    // 缓存查询API
    mux.HandleFunc("GET /api/dns/cache", handler.GetCacheEntries)
    mux.HandleFunc("GET /api/dns/stats", handler.GetStatistics)
    mux.HandleFunc("GET /api/dns/hotspots", handler.GetHotspots)
    mux.HandleFunc("GET /api/dns/cache/query", handler.QueryDomain)
    mux.HandleFunc("GET /api/dns/cache/reverse", handler.ReverseQuery)

    // 缓存管理API
    mux.HandleFunc("DELETE /api/dns/cache/{domain}", handler.DeleteEntry)
    mux.HandleFunc("DELETE /api/dns/cache", handler.ClearAll)
}
```

#### Step 3: 前端可视化仪表板
**新建文件**: `/web/dashboard/dns-cache.html`

```html
<!DOCTYPE html>
<html>
<head>
    <title>DNS缓存仪表板</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', sans-serif; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }

        .stat-value { font-size: 2.5em; font-weight: bold; color: #2196F3; }
        .stat-label { color: #666; margin-top: 10px; }

        .table-section {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }

        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 15px;
        }

        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #eee;
        }

        th { background: #f9f9f9; font-weight: 600; color: #333; }

        .hotspot { background: #fff3cd; }
        .recent { background: #e7f3ff; }

        .controls {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
        }

        button {
            padding: 10px 20px;
            background: #2196F3;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }

        button:hover { background: #1976D2; }

        .search-box {
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            width: 300px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>DNS缓存仪表板</h1>

        <!-- 统计卡片 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value" id="totalEntries">-</div>
                <div class="stat-label">缓存条目</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="uniqueDomains">-</div>
                <div class="stat-label">唯一域名</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="uniqueIPs">-</div>
                <div class="stat-label">唯一IP</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="hitRate">-</div>
                <div class="stat-label">命中率</div>
            </div>
        </div>

        <!-- 热点表格 -->
        <div class="table-section">
            <h2>热点域名 Top 20</h2>
            <table id="hotDomains">
                <thead>
                    <tr>
                        <th>排名</th>
                        <th>域名</th>
                        <th>访问次数</th>
                        <th>IP地址</th>
                        <th>来源</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody></tbody>
            </table>
        </div>

        <!-- 热点IP表格 -->
        <div class="table-section">
            <h2>热点IP地址 Top 20</h2>
            <table id="hotIPs">
                <thead>
                    <tr>
                        <th>排名</th>
                        <th>IP地址</th>
                        <th>访问次数</th>
                        <th>关联域名</th>
                        <th>来源数</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody></tbody>
            </table>
        </div>

        <!-- 缓存查询 -->
        <div class="table-section">
            <h2>缓存查询</h2>
            <div class="controls">
                <input type="text" class="search-box" id="searchBox" placeholder="搜索域名或IP...">
                <button onclick="searchCache()">搜索</button>
                <button onclick="clearAll()">清空所有缓存</button>
                <button onclick="refreshStats()">刷新</button>
            </div>
            <table id="searchResults">
                <thead>
                    <tr>
                        <th>域名</th>
                        <th>IP地址</th>
                        <th>路由决策</th>
                        <th>来源</th>
                        <th>命中次数</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody></tbody>
            </table>
        </div>
    </div>

    <script>
        async function refreshStats() {
            // 获取统计信息
            const statsResp = await fetch('/api/dns/stats');
            const stats = await statsResp.json();

            document.getElementById('totalEntries').textContent = stats.totalEntries;
            document.getElementById('uniqueDomains').textContent = stats.uniqueDomains;
            document.getElementById('uniqueIPs').textContent = stats.uniqueIPs;
            document.getElementById('hitRate').textContent = (stats.hitRate * 100).toFixed(1) + '%';

            // 获取热点数据
            const hotspotsResp = await fetch('/api/dns/hotspots');
            const hotspots = await hotspotsResp.json();

            renderHotspots(hotspots);
        }

        function renderHotspots(hotspots) {
            // 渲染热点域名
            const domainsBody = document.querySelector('#hotDomains tbody');
            domainsBody.innerHTML = '';

            hotspots.topDomains.forEach((domain, idx) => {
                const row = document.createElement('tr');
                row.className = 'hotspot';
                row.innerHTML = `
                    <td>${idx + 1}</td>
                    <td>${domain.domain}</td>
                    <td>${domain.hitCount}</td>
                    <td>${domain.ips.join(', ')}</td>
                    <td>${domain.sources.join(', ')}</td>
                    <td><button onclick="deleteDomain('${domain.domain}')">删除</button></td>
                `;
                domainsBody.appendChild(row);
            });

            // 渲染热点IP
            const ipsBody = document.querySelector('#hotIPs tbody');
            ipsBody.innerHTML = '';

            hotspots.topIPs.forEach((ip, idx) => {
                const row = document.createElement('tr');
                row.className = 'hotspot';
                row.innerHTML = `
                    <td>${idx + 1}</td>
                    <td>${ip.ip}</td>
                    <td>${ip.hitCount}</td>
                    <td>${ip.associatedDomains.join(', ')}</td>
                    <td>${ip.sourceCount}</td>
                    <td><button onclick="reverseQuery('${ip.ip}')">查询</button></td>
                `;
                ipsBody.appendChild(row);
            });
        }

        async function searchCache() {
            const query = document.getElementById('searchBox').value;
            if (!query) return;

            // 先尝试作为域名查询
            let resp = await fetch(`/api/dns/cache/query?domain=${encodeURIComponent(query)}`);

            // 如果失败，尝试作为IP反查
            if (!resp.ok) {
                resp = await fetch(`/api/dns/cache/reverse?ip=${encodeURIComponent(query)}`);
            }

            if (resp.ok) {
                const result = await resp.json();
                renderSearchResults(result);
            }
        }

        async function deleteDomain(domain) {
            await fetch(`/api/dns/cache/${encodeURIComponent(domain)}`, { method: 'DELETE' });
            refreshStats();
        }

        async function clearAll() {
            if (confirm('确定要清除所有DNS缓存吗？')) {
                await fetch('/api/dns/cache', { method: 'DELETE' });
                refreshStats();
            }
        }

        async function reverseQuery(ip) {
            const resp = await fetch(`/api/dns/cache/reverse?ip=${encodeURIComponent(ip)}`);
            const result = await resp.json();
            renderSearchResults(result);
        }

        // 初始加载
        refreshStats();
        setInterval(refreshStats, 5000); // 每5秒自动刷新
    </script>
</body>
</html>
```

---

### 第四阶段：本地DNS拦截服务

#### Step 1: 创建DNS服务器模块
**新建文件**: `/processor/dns/server.go`

```go
type DNSServer struct {
    listener   net.PacketConn
    resolver   *DNSResolver
    cache      *DNSCache
    upstream   string  // 上游DNS (8.8.8.8:53)
    rules      *DNSRules
}

func NewDNSServer(addr string, upstream string) *DNSServer {
    // 创建本地DNS服务器
    // 监听127.0.0.1:53或自定义地址
}

func (s *DNSServer) Start() error {
    // 启动UDP监听
    // 处理DNS查询
    // 查询缓存或转发上游
}

func (s *DNSServer) HandleQuery(query *dns.Msg) *dns.Msg {
    // 1. 检查本地缓存
    if cached := s.cache.Get(query.Question[0].Name); cached != nil {
        return cached
    }

    // 2. 检查IP-Domain缓存
    if domainInfo := s.lookupIPDomain(ip); domainInfo != nil {
        return s.createResponse(query, domainInfo)
    }

    // 3. 转发上游DNS
    resp, _ := s.forwardUpstream(query)

    // 4. 缓存结果
    s.cache.Set(query.Question[0].Name, resp, resp.Ttl)

    return resp
}
```

#### Step 2: 创建DNS缓存管理
**新建文件**: `/processor/dns/cache.go`

```go
type DNSCache struct {
    mu       sync.RWMutex
    entries  map[string]*DNSCacheEntry
    maxSize  int
    ttl      time.Duration
}

type DNSCacheEntry struct {
    Query      string        // 查询名称
    Response   *dns.Msg      // DNS响应
    TTL        time.Duration // 实际TTL
    CreatedAt  time.Time
    ExpiresAt  time.Time
    HitCount   uint64        // 命中次数
    Sources    []string      // 来源（sni/http/dns）
}

func (c *DNSCache) Get(name string) *dns.Msg {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if entry, ok := c.entries[name]; ok {
        if time.Now().Before(entry.ExpiresAt) {
            entry.HitCount++
            return entry.Response
        }
        delete(c.entries, name)
    }
    return nil
}
```

#### Step 3: 在引擎初始化时启动DNS服务器
**文件**: `/inbound/tun/engine/engine.go`

在`Start()`方法中：
```go
// 启动本地DNS服务器
dnsServer := dns_module.NewDNSServer(
    "127.0.0.1:53",     // 监听地址
    "8.8.8.8:53",       // 上游DNS
)

if err := dnsServer.Start(); err != nil {
    logger.Warn(fmt.Sprintf("Failed to start DNS server: %v", err))
    // 继续运行（DNS服务器可选）
} else {
    logger.Info("DNS server started on 127.0.0.1:53")
}

// 设置系统DNS（macOS/Linux）
configureSystemDNS("127.0.0.1")  // 需要root/管理员权限
```

#### Step 4: 修改UDP处理以支持DNS拦截
**文件**: `/inbound/tun/tunnel/udp.go`

```go
func handleDNSQuery(uc adapter.UDPConn, metadata *M.Metadata) {
    // 如果本地DNS服务器已启动，由它处理
    if tunnel.GetDNSServer() != nil {
        // DNS流量已被重定向到本地DNS服务器
        // UDP处理器中的DNS查询会自动路由到它
        return
    }

    // 否则使用传统方式（转发+日志）
    // ...
}
```

---

### 第三阶段：IP反查能力

#### Step 1: 创建反向DNS模块
**新建文件**: `/processor/dns/reverse.go`

```go
type ReverseResolver struct {
    // 本地IP→Domain映射表
    localCache map[string][]string

    // 从IP-Domain缓存构建的倒排索引
    ipToDomains map[string][]string
}

func (r *ReverseResolver) QueryDomains(ip netip.Addr) []string {
    // 1. 查询本地缓存
    if domains, ok := r.ipToDomains[ip.String()]; ok {
        return domains
    }

    // 2. 尝试PTR查询（可选，互联网查询）
    if domains, err := r.queryPTR(ip); err == nil {
        return domains
    }

    // 3. 查询GeoIP库
    if country := r.geoipLookup(ip); country != "" {
        return []string{country + ":unknown"}
    }

    return nil
}

func (r *ReverseResolver) queryPTR(ip netip.Addr) ([]string, error) {
    // 执行DNS PTR反向查询
    // 例如: 查询 1.2.3.4 → 4.3.2.1.in-addr.arpa
}
```

#### Step 2: 集成到DNS服务器
在`/processor/dns/server.go`中：

```go
func (s *DNSServer) HandleQuery(query *dns.Msg) *dns.Msg {
    // ...

    // 支持PTR查询（IP反查）
    if query.Question[0].Qtype == dns.TypePTR {
        domains := s.reverseResolver.QueryDomains(ip)
        return s.createPTRResponse(query, domains)
    }

    // ...
}
```

---

## 核心数据结构总结

### 统一的DNS缓存条目
```go
type CacheEntry struct {
    Domain         string                  // 域名
    IPs            []netip.Addr           // 可能的多个IP
    Route          cache.RouteDecision    // 路由决策
    BindingSources []BindingSource        // 来源多个
    CreatedAt      time.Time
    ExpiresAt      time.Time
    TTL            time.Duration          // 实际TTL
    HitCount       uint64                 // 命中次数

    // DNS特定信息
    DNSRecords     *DNSRecords            // A/AAAA记录详情
    QueryTime      time.Duration          // DNS查询耗时
}

type DNSRecords struct {
    ARecords       []string               // IPv4地址列表
    AAAARecords    []string               // IPv6地址列表
    CNAMEs         []string               // CNAME列表
    Authorities    []string               // 权威记录
    Additionals    []string               // 附加记录
}
```

### 缓存查询接口
```go
// 统一的缓存查询接口
type CacheQuery interface {
    // 按域名查询
    QueryByDomain(domain string) (*CacheEntry, error)

    // 按IP查询
    QueryByIP(ip netip.Addr) ([]*CacheEntry, error)

    // 按来源查询
    QueryBySource(source BindingSource) ([]*CacheEntry, error)

    // 统计查询
    Stats() CacheStats
}

type CacheStats struct {
    TotalEntries    uint64
    DomainCount     uint64
    IPCount         uint64
    CacheHits       uint64
    CacheMisses     uint64
    AverageHitRate  float64
    OldestEntry     time.Time
    NewestEntry     time.Time
}
```

---

## 预期成果

### 缓存覆盖范围
| 来源 | 覆盖范围 | 覆盖率 |
|------|---------|--------|
| SNI提取 | HTTPS已访问 | ~60% |
| HTTP头提取 | HTTP已访问 | ~20% |
| DNS拦截 | 所有DNS查询 | ~100%（需要第二阶段） |
| IP反查 | 直连IP的近似域名 | ~30%（不准确） |
| DoH拦截 | DoH提供商查询 | ~5%（补充） |
| 合计 | 所有网络访问 | ~95%+ |

### 性能改进
- 重复访问域名：SNI提取延迟从10-50ms降至0ms（缓存命中）
- DNS查询延迟：从50-200ms降至<5ms（本地缓存）
- 全局延迟：页面加载时间降低10-30%

### 功能能力
- ✅ 域名→IP正向映射缓存
- ✅ IP→域名反向映射缓存
- ✅ 多源绑定追踪（知道域名来自哪里）
- ✅ TTL管理和过期清理
- ✅ 缓存统计和分析
- ✅ 路由决策与DNS结果同步

---

## 文件修改清单

### 需要创建的新文件
- `/processor/dns/server.go` - DNS服务器实现
- `/processor/dns/cache.go` - DNS缓存管理
- `/processor/dns/reverse.go` - 反向DNS解析
- `/processor/dns/types.go` - DNS数据类型
- `/processor/dns/rules.go` - DNS规则引擎（可选）

### 需要修改的现有文件
- `/inbound/tun/metadata/metadata.go` - 扩展DNSInfo字段
- `/processor/tcp/handler.go` - SNI自动绑定
- `/processor/cache/ipdomain.go` - 增强缓存结构
- `/processor/rules/engine.go` - 缓存同步
- `/inbound/http/metadata_extractor.go` - HTTP头绑定
- `/inbound/tun/engine/engine.go` - DNS服务器初始化
- `/processor/config/types.go` - DNS服务器配置

### 不需要修改的文件
- `/processor/tcp/tls.go` - 现有SNI提取逻辑保持不变
- `/processor/cache/ipdomain_test.go` - 补充新测试
- `/common/geoip/` - 可选集成

---

## 优先级建议（已按用户需求调整）

1. **最优先 ⭐⭐⭐⭐⭐**：第一阶段（SNI自动关联 + HTTP头提取）
   - 最小投入获得最高收益
   - 为后续阶段打好基础
   - 可独立运行
   - **工作量：2-3小时**

2. **紧跟第一阶段 ⭐⭐⭐⭐⭐**：第二阶段（IP反查能力 + 倒排索引）
   - 用户特别强调"很重要，需要完整的IP→Domain映射"
   - 建立双向查询能力（Domain→IP和IP→Domain）
   - 支持IP热点统计
   - **工作量：3-4小时**

3. **必须做 ⭐⭐⭐⭐⭐**：第三阶段（DNS缓存管理API + 可视化）
   - 用户要求"必须有，需要实时看到缓存状态"
   - 包含完整的HTTP REST API
   - 包含管理后台仪表板
   - **工作量：2-3小时**

4. **有余力再做 ⭐⭐⭐**：第四阶段（本地DNS拦截服务）
   - 全局缓存覆盖
   - 可选增强功能
   - **工作量：4-6小时**

5. **最后考虑 ⭐⭐**：第五阶段（DoH拦截 + DNS黑名单）
   - 补充功能，可选
   - **工作量：2-3小时**

---

## 注意事项

### 技术限制
1. **macOS DNS设置**：需要root权限，可能与系统DNS冲突
2. **Linux DNS设置**：需要修改/etc/resolv.conf或使用systemd-resolved
3. **Windows DNS设置**：需要管理员权限
4. **Docker/VM环境**：可能无法修改DNS

### 向后兼容性
- 所有新功能都是可选的（有优雅降级）
- 现有的路由决策逻辑保持不变
- 缓存系统完全向后兼容

### 安全考虑
- DNS缓存不存储用户敏感信息（仅域名和IP）
- 本地DNS服务器仅监听127.0.0.1（不暴露到网络）
- PTR反查结果仅用于分析，不做特殊处理


用户补充：
在判断代理要走哪个路由的时候，先提取sni，根据sni判断是否需要走door代理，还是去RouteToCursor；如果没有识别到，再执行根据geoip判断走向的逻辑：检查engine是否开启，是否在中国等等；