# Nursorgate - TUN-Based Intelligent Proxy System

English | [中文](./README.zh.md)

> A high-performance TUN-based transparent proxy system with intelligent routing, DNS caching, and real-time network traffic management.

## 🎯 Project Overview

**Nursorgate** (formerly Nursor) is a sophisticated network proxy system built for macOS and Linux that intercepts and intelligently routes network traffic through TUN (Tunnel) devices. It combines transparent packet interception with intelligent rule-based routing to provide a seamless, privacy-focused networking experience.

### Key Characteristics

- **TUN-Based Architecture**: Operates at the OS kernel level using TUN devices for transparent traffic interception
- **Intelligent Routing**: SNI allowlist → MITM to Aliang; otherwise SOCKS (if configured) → Direct
- **Real-Time DNS Caching**: Automatic domain-IP relationship discovery and caching
- **HTTPS MITM Support**: Optional transparent HTTPS interception with custom CA certificates
- **Multi-Protocol Proxy Support**: SOCKS5, VLESS, Shadowsocks, and custom protocols
- **Live Dashboard**: Web-based management interface with real-time statistics
- **Cross-Platform**: Builds for macOS (arm64/amd64), Linux (amd64/arm64), and Windows

---

## 📊 Recent Development Achievements

### Phase 4 ✅ - Complete DNS Binding Storage Layer (December 2024)

Successfully implemented automatic DNS binding persistence:

- **DNS Binding Capture**: Automatically captures domain-IP relationships from:
  - TLS SNI extraction (HTTPS connections)
  - HTTP Host headers
  - CONNECT requests

- **Automatic Storage**: DNS bindings stored to cache immediately after successful relay
- **Route Decision Tracking**: Each binding stores the routing decision used (RouteToCursor/RouteToDoor/RouteDirect)

**Files Modified:**
- `inbound/tun/metadata/metadata.go` - Added Route field for tracking final routing decision
- `processor/rules/engine.go` - Implemented StoreBinding() method for DNS cache persistence
- `processor/tcp/handler.go` - Integrated binding storage after successful relay

**Impact**: Enables cache-first routing optimization - repeated connections to the same IP can now skip expensive SNI extraction by looking up domain from cache.

---

### Phase 3 ✅ - Real-Time DNS Cache API & Dashboard (December 2024)

Comprehensive DNS cache visibility with real-time web interface:

- **7 REST API Endpoints**: Full CRUD operations for DNS cache management
  - `GET /api/dns/cache` - List cache entries with pagination
  - `GET /api/dns/stats` - Cache statistics (hits, misses, hit rate, unique domains/IPs)
  - `GET /api/dns/hotspots` - Top 20 most frequently accessed domains and IPs
  - `GET /api/dns/cache/query` - Domain lookup
  - `GET /api/dns/cache/reverse` - Reverse IP lookup
  - `DELETE /api/dns/cache/{domain}` - Remove specific entry
  - `DELETE /api/dns/cache` - Clear all entries

- **Integrated Dashboard**: DNS cache panel in main web dashboard with:
  - 4 Stat Cards: Total entries, unique domains, unique IPs, hit rate
  - Hot Domains Table: Top 20 with hit counts and binding sources
  - Hot IPs Table: Top 20 with associated domains
  - Search Interface: Domain/IP search with delete functionality
  - Color-Coded Badges: Binding source identification (SNI=blue, HTTP=yellow, CONNECT=red, DNS=green)
  - Auto-refresh: 5-second update interval when viewing DNS cache page

**Files Created:**
- `app/http/handlers/dns_cache.go` - REST API implementation
- Web UI integrated into `app/website/index.html` and `app/website/assets/app.js`

---

### Phase 2 ✅ - IP↔Domain Bidirectional Mapping (December 2024)

Complete reverse lookup capability with statistical aggregation:

- **Forward Index**: Domain → IP mapping
- **Reverse Index**: IP → Multiple domains mapping
- **Query Methods**:
  - GetByDomain(domain) - Find IPs for a domain
  - GetByIP(ip) - Find domains for an IP
  - GetIPStatistics(ip) - Aggregated stats for an IP
  - GetHotspotIPs() - Top N IPs by access frequency
  - GetAll() - All valid (non-expired) entries

- **Bidirectional Sync**: Both indexes stay consistent through all cache mutations

**Files Modified:**
- `processor/cache/ipdomain.go` - Extended with reverse index and query methods

---

### Phase 1 ✅ - Multi-Source DNS Binding (December 2024)

Automatic domain capture from multiple sources:

- **SNI Extraction**: Automatically extract domain from TLS ClientHello
- **HTTP Header Extraction**: Capture domain from Host header and CONNECT requests
- **Binding Metadata**: Track source, timestamp, and TTL for each binding

**Data Structure Enhancement:**
```go
type DNSInfo struct {
    BindingSource BindingSource  // Source: SNI, HTTP, DNS, CONNECT
    BindingTime   time.Time      // When binding was captured
    CacheTTL      time.Duration  // How long to keep this binding
    ShouldCache   bool           // Whether to persist this binding
}
```

---

### Dashboard Display Fixes ✅ (December 2024)

Fixed three critical dashboard display issues:

| Issue | Root Cause | Fix |
|-------|-----------|-----|
| **Hit Count = 0** | Get() method wasn't updating individual entry HitCount | Added `entry.HitCount++` in Get() method |
| **Hit Rate = 0** | Stats() calculated correctly but missing data from cache usage | Fixed data flow with StoreBinding() implementation |
| **Wrong Unique Counts** | Stats() returned maxEntries (capacity) instead of uniqueDomains; JS mapped hits instead of uniqueIPs | Added uniqueDomains and uniqueIPs calculation; Fixed JS mapping |

**Files Modified:**
- `processor/cache/ipdomain.go:Get()` - Update HitCount on cache hit
- `processor/cache/ipdomain.go:Stats()` - Calculate and return uniqueDomains and uniqueIPs
- `app/website/assets/app.js` - Correct field mapping for dashboard display

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                       TUN Device Layer                           │
│         Intercepts all TCP/UDP packets at kernel level           │
└────────────────┬────────────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────────────┐
│                    Protocol Detection                            │
│   Determines: TLS (443) | HTTP (80) | Other (custom handling)   │
└────────────────┬────────────────────────────────────────────────┘
                 │
         ┌───────┴─────────┬──────────────┐
         │                 │              │
    ┌────▼─────┐      ┌────▼────┐   ┌───▼──┐
    │  TLS/443 │      │ HTTP/80 │   │ Other│
    └────┬─────┘      └────┬────┘   └───┬──┘
         │                 │            │
    ┌────▼──────────────────▼────────────▼─────┐
    │        Metadata Extraction & Caching     │
    │  • SNI domain extraction from TLS        │
    │  • HTTP Host header extraction           │
    │  • CONNECT request parsing               │
    │  • Automatic DNS binding storage         │
    └────┬────────────────────────────────────┘
         │
    ┌────▼──────────────────────────────────────┐
    │       Routing Decision Engine             │
    │  Priority: Bypass → Cache → Rules → GeoIP│
    │  Returns: Route decision + domain info    │
    └────┬──────────────────────────────────────┘
         │
    ┌────▼──────────────────────────────────────┐
    │          Route Execution                  │
    │  RouteToCursor (MITM) → Aliang Proxy   │
    │  RouteToDoor (Forward) → VLESS/SS Proxy  │
    │  RouteDirect → Direct TCP Connection     │
    └────┬──────────────────────────────────────┘
         │
    ┌────▼──────────────────────────────────────┐
    │      Data Relay & Statistics              │
    │  • Bidirectional data forwarding           │
    │  • Connection tracking & stats collection │
    │  • DNS binding persistence                │
    └──────────────────────────────────────────┘
```

### DNS Caching System

```
HTTP Metadata         TCP Handler           DNS Cache
  Extraction          (port 443)            Storage
     │                    │                    │
     ├─ CONNECT request   ├─ SNI extraction   │
     │  → DNSInfo (10m)   │  → DNSInfo (5m)  │
     │                    │                    │
     └─ HTTP Host header  └─ Create Route     │
        → DNSInfo (10m)      Decision         │
                                 │
                                 ▼
                          ┌─────────────┐
                          │StoreBinding()
                          │   (New!)    │
                          └─────┬───────┘
                                │
                    ┌───────────▼───────────┐
                    │  IPDomainCache        │
                    │  ├─ Forward: Domain→IP
                    │  ├─ Reverse: IP→Domain
                    │  ├─ HitCount tracking
                    │  └─ LRU eviction
                    └───────────┬───────────┘
                                │
                        ┌───────▼────────┐
                        │  Next Request  │
                        │ (Same IP?)     │
                        │  → Cache HIT!  │
                        │ Skip SNI extract
                        └────────────────┘
```

---

## 🚀 Quick Start

### Build

```bash
# Standard binary build
go build -o nursorgate ./cmd/nursor

# Optimized for size (with symbol stripping)
go build -ldflags="-s -w" -o nursorgate ./cmd/nursor

# Cross-compile for different platforms
./build.sh  # See build scripts below
```

### Cross-Platform Build Scripts

**macOS (arm64 - Apple Silicon):**
```bash
export CGO_ENABLED=1
export GOOS=darwin
export GOARCH=arm64
go build -ldflags="-s -w" -tags=with_utls -o nursorgate-darwin-arm64 ./cmd/nursor
```

**macOS (amd64 - Intel):**
```bash
export CGO_ENABLED=1
export GOOS=darwin
export GOARCH=amd64
go build -ldflags="-s -w" -o nursorgate-darwin-amd64 ./cmd/nursor
```

**Linux (amd64):**
```bash
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
go build -ldflags="-s -w" -o nursorgate-linux-amd64 ./cmd/nursor
```

**Linux (arm64):**
```bash
export GOOS=linux
export GOARCH=arm64
go build -ldflags="-s -w" -o nursorgate-linux-arm64 ./cmd/nursor
```

**Windows (amd64):**
```bash
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o nursorgate-win-amd64.exe ./cmd/nursor
```

### Run

```bash
# Start the proxy
./nursorgate --config config.json

# View management dashboard
# Open browser to: http://localhost:56431
```

---

## 📚 Development Documentation

### DNS Caching System Design

The DNS caching system solves a fundamental architectural challenge: **TUN devices capture only TCP/UDP layer traffic, never seeing DNS queries at the application layer**.

This creates a "hostname metadata vacuum" where domain resolution context available at query time is completely lost by the time TCP connections are captured.

**Solution:** Multi-source domain binding through:

1. **SNI Extraction** (HTTPS): Auto-extract domain from TLS ClientHello
2. **HTTP Headers**: Capture domain from Host header
3. **CONNECT Requests**: Extract domain from CONNECT method
4. **System DNS Interception** (Optional): Capture full DNS queries at network layer

Each binding is automatically stored to cache with:
- Domain name and destination IP
- Binding source (SNI/HTTP/CONNECT/DNS)
- Route decision used
- Expiration time (TTL varies by source)

**Cache Usage:**
- First connection: Expensive SNI extraction or header parsing
- Subsequent connections: Cache hit → skip extraction → faster routing

### HTTP CONNECT Handling

Important implementation detail for HTTP tunneling:

```
Client → Proxy:  CONNECT example.com:443
         HTTP/1.1 200 Connection Established
         (metadata extraction happens here)

         ↓ (metadata + route decision)

Proxy → Remote: Transparent TCP connection
```

The proxy must:
1. Return `HTTP/1.1 200 Connection Established` before routing
2. Extract domain from CONNECT request for cache
3. Switch to transparent TCP relay mode

---

## 📝 Development Notes

### HTTP/2 Frame Handling

When processing HTTP/2 traffic:

1. **Header Frames**: Must extract `priority` field from payload before parsing headers
2. **Header Assembly**: After modifying headers, `priority` must be placed back in payload
3. **Important**: Envoy may force HTTP→H2 conversion, requiring proper priority handling
4. **Cursor Compatibility**: Improper priority handling breaks Cursor website loading

### Certificate Authority Setup

For HTTPS interception:
- Cannot use system CA certificates
- Must explicitly trust `mitm-ca.pem` certificate
- Certificate pinning in some applications may prevent interception

### GeoIP Routing

The system can route traffic based on IP geolocation:

```
IP Address → GeoIP Lookup → Country/City → Rule Evaluation → Route
```

This enables country-based routing rules without application involvement.

---

## 📊 Development Journal

### December 10, 2024

**DNS Cache Storage Implementation**
- ✅ Added Route field to Metadata struct
- ✅ Implemented StoreBinding() in RuleEngine
- ✅ Integrated storage into TCP handler
- ✅ Fixed dashboard display issues (HitCount, Hit Rate, unique counts)

**Achievement**: Complete end-to-end DNS caching system now operational. DNS bindings are automatically captured, stored, and reused for cache-first routing optimization.

### December 8-9, 2024

**Dashboard Display Bug Fixes**
- 🐛 Issue: Hit count always showing 0
  - Root Cause: Get() method not updating individual entry HitCount
  - Fix: Added `entry.HitCount++` in Get() method

- 🐛 Issue: Hit rate always 0%
  - Root Cause: Cache wasn't being queried, so hits=0, misses=0
  - Context: Not a bug but reflection of cache usage pattern

- 🐛 Issue: Wrong unique IP/domain display
  - Root Cause: Backend missing uniqueDomains and uniqueIPs calculation
  - Root Cause: Frontend incorrectly mapped stats.maxEntries and stats.hits
  - Fix: Added calculation to Stats(); corrected JS field mapping

**Achievement**: Dashboard now accurately reflects cache statistics and performance metrics.

### December 2-7, 2024

**Real-Time DNS Cache Dashboard**
- ✅ Created 7 REST API endpoints for DNS cache operations
- ✅ Integrated DNS cache panel into main web dashboard
- ✅ Implemented live statistics with 5-second refresh
- ✅ Added hot domains/IPs tables with color-coded source badges
- ✅ Implemented search, delete, and clear functions

**Achievement**: Complete visibility into DNS cache operations with real-time statistics and management capabilities.

### August 4, 2024

**HTTP/2 Frame Processing**
1. Header frame priority field must be extracted from payload before header parsing
2. After header modification, priority must be restored to payload
3. Envoy may convert HTTP to H2, requiring robust priority handling
4. Missing priority restoration breaks Cursor website loading

---

## 🛠️ Development Commands

### Build & Test

```bash
# Build individual packages
go build ./processor/tcp
go build ./processor/cache
go build ./processor/rules
go build ./app/http/handlers

# Run tests
go test -v ./processor/cache
go test -v ./processor/rules

# Clean build
go clean -cache
go build -o nursorgate ./cmd/nursor
```

### Debugging

```bash
# Check DNS cache API response
curl http://localhost:56431/api/dns/stats | jq

# View hotspots
curl http://localhost:56431/api/dns/hotspots | jq

# Query specific domain
curl "http://localhost:56431/api/dns/cache/query?domain=example.com" | jq

# Clear cache
curl -X DELETE http://localhost:56431/api/dns/cache
```

---

## 📦 Key Dependencies

- **gVisor** (github.com/sagernet/gvisor) - User-space network stack
- **sing-box** (github.com/sagernet/sing-box) - Protocol implementations
- **SNI Allowlist** - Local domain list for MITM routing to Aliang
- **GeoIP2** (oschwald/geoip2-golang) - IP geolocation
- **tun2socks** (xjasonlyu/tun2socks/v2) - TUN device integration
- **miekg/dns** - DNS protocol support

---

## 🔒 Security Considerations

- HTTPS MITM requires system CA trust
- SNI extraction operates at TCP layer (TLS plaintext handshake)
- GeoIP database updates recommended quarterly
- System DNS reconfiguration for full DNS interception

---

## 📄 License & Attribution

See LICENSE file for project licensing information.

---

## 🤝 Contributing

Development focuses on:
1. Cache performance optimization
2. Protocol compatibility improvements
3. Dashboard UX/UX enhancements
4. Cross-platform stability

---

**Last Updated**: December 10, 2024
**Latest Version**: Phase 4 - Complete DNS Caching System
**Module**: nursor.org/nursorgate
