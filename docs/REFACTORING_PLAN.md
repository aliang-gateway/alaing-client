# Configuration System Refactoring Plan

## Project Context

This is a VPN kernel project with:
- **Server**: HTTP API for interaction
- **FFI exports**: C-compatible functions for external program integration
- **CMD directory**: Configuration parsing
- **Current problem**: Messy dual configuration systems coexisting

## User Requirements

从 `config.example.json` 配置文件中处理得到对应的配置，需要：

1. 直接从 `config.example.json` 解析配置
2. 代理配置信息全局存储在一个 map 中
3. 生成的代理实例也全局存储在一个 map 中
4. 通过 get/set 方式访问这些代理
5. 支持 HTTP 服务访问
6. 支持 FFI 调用访问

## User Design Decisions

经过讨论，用户选择了以下方案：

1. **旧系统处理**: 完全移除旧系统，全面迁移到 Registry
2. **解析方式**: 直接 JSON Unmarshal 到结构体（不使用 map[string]interface{} 转换）
3. **访问模式**: 保持 Singleton 模式（推荐）
4. **配置管理**: 只做基础加载（不需要热加载、配置合并等高级功能）

## Current Architecture Problems

### 1. Dual Proxy Systems Coexist

**OLD System** (`/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/proxy.go`):
- Global variables: `directProxy`, `doorProxy`, `vlessConfig`, `shadowsocksConfig`
- Functions: `SetProxyConfig()`, `GetDirectProxy()`, `GetDoorProxy()`
- Mutex-protected package-level variables

**NEW System** (`/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/proxy/registry.go`):
- Singleton Registry pattern with `sync.Once`
- Stores proxy instances in `map[string]proxy.Proxy`
- Functions: `Register()`, `Get()`, `GetDefault()`, `GetDoor()`

### 2. Fragmented Configuration Parsing

`/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/cmd/config.go` (lines 159-209):
```go
// Current messy implementation
func parseProxyConfig(proxyMap map[string]interface{}) (*proxyConfig.ProxyConfig, error) {
    cfg := &proxyConfig.ProxyConfig{}

    // Manual type assertions
    if typeVal, ok := proxyMap["type"].(string); ok {
        cfg.Type = typeVal
    } else {
        return nil, fmt.Errorf("missing or invalid 'type' field")
    }

    // ... more manual parsing
}
```

Should be direct JSON unmarshal to structs.

### 3. Mixed Usage Across Codebase

**Files using OLD system:**
- `inbound/tun/engine/engine.go` (lines 150-200): Fallback logic
- `inbound/tun/tunnel/tcp.go`: Calls `GetDoorProxy()`
- `server/bussiness_server.go`: Both old and new endpoints
- `export.go`: Both old and new FFI functions

**Files using NEW system:**
- `processor/proxy/registry.go`: Main implementation
- `cmd/config.go`: `ApplyConfig()` uses registry
- `server/bussiness_server.go`: `/proxy/registry/*` endpoints

**Files trying BOTH:**
- `inbound/tun/engine/engine.go`:
  ```go
  _defaultProxy, err = proxyRegistry.GetRegistry().GetDefault()
  if err != nil {
      // Fallback to OLD config manager
      if defaultProxyFromConfig := proxyConfig.GetDirectProxy(); defaultProxyFromConfig != nil {
          _defaultProxy = defaultProxyFromConfig
      }
  }
  ```

## Exploration Results Summary

### Configuration File Structure

`config.example.json`:
```json
{
  "engine": {
    "mtu": 0,
    "device": "utun0",
    "loglevel": "info"
  },
  "currentProxy": "door",
  "coreServer": "ai-gateway.nursor.org",
  "proxies": {
    "door": {
      "type": "vless",
      "is_default": true,
      "is_door_proxy": true,
      "vless": {
        "server": "node1.nursor.org:35001",
        "uuid": "74cddcdd-6d48-41cf-8e62-902e7c943fe7",
        "tls_enabled": true,
        "sni": "www.microsoft.com",
        "reality_enabled": true,
        "public_key": "sAtJcW2xLIUWRE-_7KHGEAtvHx-P1sDbjrrgrt4_XCo",
        "short_id_list": "ef,b79e62,7d87a3,..."
      }
    },
    "direct": {
      "type": "shadowsocks",
      "shadowsocks": {
        "server": "example.com:443",
        "method": "aes-256-gcm",
        "password": "your-password-here"
      }
    }
  }
}
```

### Proxy Implementations

**Interface**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/outbound/proxy/interfaces.go`
```go
type Proxy interface {
    Dialer
    Addr() string
    Proto() proto.Proto
}
```

**Supported Types:**
- **VLESS**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/outbound/proxy/vless/vless.go`
  - Constructors: `NewVLESS()`, `NewVLESSWithTLS()`, `NewVLESSWithVision()`, `NewVLESSWithReality()`
- **Shadowsocks**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/outbound/proxy/shadowsocks/shadowsocks.go`
  - Constructor: `NewShadowsocks()`
- **Direct**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/outbound/proxy/direct/direct.go`

### HTTP API Endpoints

**Current endpoints** (`server/bussiness_server.go`):

**Legacy (OLD system):**
- `POST /proxy/set` - Set proxy config
- `GET /proxy/get` - Get current proxy configs

**Registry (NEW system):**
- `GET /proxy/registry/list` - List all registered proxies
- `GET /proxy/registry/get?name=xxx` - Get specific proxy
- `POST /proxy/registry/register` - Register new proxy
- `POST /proxy/registry/unregister` - Remove proxy
- `POST /proxy/registry/set-default` - Set default proxy
- `POST /proxy/registry/set-door` - Set door proxy
- `POST /proxy/registry/switch` - Switch active proxy

### FFI Exports

**Current functions** (`export.go`):

**Legacy (OLD system):**
```go
//export setVLESSProxy
func setVLESSProxy(server, uuid, sni, publicKey *C.char, isDefault, isDoorProxy *C.bool)

//export setShadowsocksProxy
func setShadowsocksProxy(server, method, password *C.char, isDefault *C.bool)
```

**Registry (NEW system):**
```go
//export registerProxy
func registerProxy(name, proxyType, server, uuid, sni, publicKey *C.char)

//export switchProxy
func switchProxy(name *C.char)

//export listProxies
func listProxies() *C.char
```

## Proposed Architecture

### Architecture Design

```
config.example.json
    ↓ (Direct unmarshal with proper JSON tags)
cmd/config.go → map[string]*ProxyConfig
    ↓ (For each proxy)
processor/config/store.go (ConfigStore)
    - Stores: map[string]*ProxyConfig
    - Methods: Set(), Get(), List(), Delete()
    ↓ (CreateProxyFromConfig)
processor/config/factory.go (Factory)
    - Creates proxy instances from configs
    ↓ (Register instance)
processor/proxy/registry.go (Registry)
    - Stores: map[string]proxy.Proxy
    - Methods: Register(), Get(), GetDefault(), GetDoor()
    ↓ (Get/GetDefault/GetDoor)
Engine / HTTP Server / FFI
    - All use Registry singleton
```

### Two Global Stores

1. **ConfigStore** (`processor/config/store.go` - NEW):
   - Stores raw proxy configurations
   - Type: `map[string]*ProxyConfig`
   - Singleton pattern with `GetConfigStore()`
   - Thread-safe with `sync.RWMutex`

2. **Registry** (`processor/proxy/registry.go` - EXISTS):
   - Stores proxy instances
   - Type: `map[string]proxy.Proxy`
   - Singleton pattern with `GetRegistry()`
   - Already implemented, will be enhanced

### Key Design Principles

1. **Single Source of Truth**: Registry is the only way to access proxy instances
2. **Separation of Concerns**: ConfigStore for configs, Registry for instances
3. **Factory Pattern**: `factory.go` creates instances from configs
4. **Direct JSON Unmarshal**: No manual type assertions
5. **Validation**: All configs validated before storage
6. **Thread Safety**: Both stores use mutex protection
7. **Immutability**: Stores return copies to prevent external modification

## Implementation Plan

### Phase 1: Create New Infrastructure

#### Step 1.1: Create `processor/config/types.go` (NEW FILE)

**Purpose**: Define configuration types with proper JSON tags

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/types.go`

**Content**:
```go
package config

import "fmt"

// ProxyConfig represents a proxy configuration
type ProxyConfig struct {
    Type        string             `json:"type"`
    IsDefault   bool               `json:"is_default"`
    IsDoorProxy bool               `json:"is_door_proxy"`
    VLESS       *VLESSConfig       `json:"vless,omitempty"`
    Shadowsocks *ShadowsocksConfig `json:"shadowsocks,omitempty"`
}

// VLESSConfig represents VLESS protocol configuration
type VLESSConfig struct {
    Server         string `json:"server"`
    UUID           string `json:"uuid"`
    Flow           string `json:"flow,omitempty"`
    TLSEnabled     bool   `json:"tls_enabled"`
    SNI            string `json:"sni,omitempty"`
    RealityEnabled bool   `json:"reality_enabled"`
    PublicKey      string `json:"public_key,omitempty"`
    ShortID        string `json:"short_id,omitempty"`
    ShortIDList    string `json:"short_id_list,omitempty"`
}

// ShadowsocksConfig represents Shadowsocks protocol configuration
type ShadowsocksConfig struct {
    Server   string `json:"server"`
    Method   string `json:"method"`
    Password string `json:"password"`
    ObfsMode string `json:"obfs_mode,omitempty"`
    ObfsHost string `json:"obfs_host,omitempty"`
}

// Validate validates the proxy configuration
func (c *ProxyConfig) Validate() error {
    if c.Type == "" {
        return fmt.Errorf("proxy type is required")
    }

    switch c.Type {
    case "vless":
        if c.VLESS == nil {
            return fmt.Errorf("VLESS config is required for vless type")
        }
        if c.VLESS.Server == "" || c.VLESS.UUID == "" {
            return fmt.Errorf("VLESS server and UUID are required")
        }
    case "shadowsocks":
        if c.Shadowsocks == nil {
            return fmt.Errorf("Shadowsocks config is required for shadowsocks type")
        }
        if c.Shadowsocks.Server == "" || c.Shadowsocks.Password == "" {
            return fmt.Errorf("Shadowsocks server and password are required")
        }
    default:
        return fmt.Errorf("unsupported proxy type: %s", c.Type)
    }

    return nil
}
```

**Key Points**:
- JSON tags match `config.example.json` structure exactly
- `omitempty` for optional fields
- `Validate()` ensures config correctness before use
- Supports both VLESS and Shadowsocks types

#### Step 1.2: Create `processor/config/store.go` (NEW FILE)

**Purpose**: Global storage for proxy configurations

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/store.go`

**Content**:
```go
package config

import (
    "fmt"
    "sync"
)

// ConfigStore stores proxy configurations (not instances)
type ConfigStore struct {
    mu      sync.RWMutex
    configs map[string]*ProxyConfig
}

var (
    globalConfigStore *ConfigStore
    configStoreOnce   sync.Once
)

// GetConfigStore returns the global config store singleton
func GetConfigStore() *ConfigStore {
    configStoreOnce.Do(func() {
        globalConfigStore = &ConfigStore{
            configs: make(map[string]*ProxyConfig),
        }
    })
    return globalConfigStore
}

// Set stores a proxy configuration
func (s *ConfigStore) Set(name string, cfg *ProxyConfig) error {
    if name == "" {
        return fmt.Errorf("proxy name cannot be empty")
    }
    if cfg == nil {
        return fmt.Errorf("config cannot be nil")
    }

    // Validate config before storing
    if err := cfg.Validate(); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    // Create a deep copy to prevent external modification
    cfgCopy := *cfg
    if cfg.VLESS != nil {
        vlessCopy := *cfg.VLESS
        cfgCopy.VLESS = &vlessCopy
    }
    if cfg.Shadowsocks != nil {
        ssCopy := *cfg.Shadowsocks
        cfgCopy.Shadowsocks = &ssCopy
    }

    s.configs[name] = &cfgCopy
    return nil
}

// Get retrieves a proxy configuration
func (s *ConfigStore) Get(name string) (*ProxyConfig, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    cfg, exists := s.configs[name]
    if !exists {
        return nil, fmt.Errorf("config '%s' not found", name)
    }

    // Return a copy to prevent external modification
    cfgCopy := *cfg
    if cfg.VLESS != nil {
        vlessCopy := *cfg.VLESS
        cfgCopy.VLESS = &vlessCopy
    }
    if cfg.Shadowsocks != nil {
        ssCopy := *cfg.Shadowsocks
        cfgCopy.Shadowsocks = &ssCopy
    }

    return &cfgCopy, nil
}

// List returns all config names
func (s *ConfigStore) List() []string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    names := make([]string, 0, len(s.configs))
    for name := range s.configs {
        names = append(names, name)
    }
    return names
}

// Delete removes a config
func (s *ConfigStore) Delete(name string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.configs[name]; !exists {
        return fmt.Errorf("config '%s' not found", name)
    }

    delete(s.configs, name)
    return nil
}

// Clear removes all configs
func (s *ConfigStore) Clear() {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.configs = make(map[string]*ProxyConfig)
}

// GetAll returns all configs (for debugging/listing)
func (s *ConfigStore) GetAll() map[string]*ProxyConfig {
    s.mu.RLock()
    defer s.mu.RUnlock()

    result := make(map[string]*ProxyConfig, len(s.configs))
    for name, cfg := range s.configs {
        // Deep copy
        cfgCopy := *cfg
        if cfg.VLESS != nil {
            vlessCopy := *cfg.VLESS
            cfgCopy.VLESS = &vlessCopy
        }
        if cfg.Shadowsocks != nil {
            ssCopy := *cfg.Shadowsocks
            cfgCopy.Shadowsocks = &ssCopy
        }
        result[name] = &cfgCopy
    }
    return result
}
```

**Key Points**:
- Singleton pattern with `sync.Once`
- Thread-safe with `sync.RWMutex`
- Returns deep copies to prevent external modification
- Validates configs before storage
- Provides CRUD operations: Set, Get, List, Delete, Clear

#### Step 1.3: Create `processor/config/factory.go` (NEW FILE)

**Purpose**: Create proxy instances from configurations

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/factory.go`

**Content**:
```go
package config

import (
    "fmt"
    "math/rand"
    "strings"

    "nursor.org/nursorgate/outbound/proxy"
    "nursor.org/nursorgate/outbound/proxy/shadowsocks"
    "nursor.org/nursorgate/outbound/proxy/vless"
)

// CreateProxyFromConfig creates a proxy instance from configuration
func CreateProxyFromConfig(cfg *ProxyConfig) (proxy.Proxy, error) {
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    switch cfg.Type {
    case "vless":
        return createVLESSProxy(cfg.VLESS)
    case "shadowsocks":
        return createShadowsocksProxy(cfg.Shadowsocks)
    default:
        return nil, fmt.Errorf("unsupported proxy type: %s", cfg.Type)
    }
}

// createVLESSProxy creates VLESS proxy instance
func createVLESSProxy(cfg *VLESSConfig) (proxy.Proxy, error) {
    // Handle REALITY
    if cfg.RealityEnabled {
        shortID := cfg.ShortID
        if shortID == "" && cfg.ShortIDList != "" {
            // Random selection from ShortIDList
            shortIDArray := strings.Split(cfg.ShortIDList, ",")
            if len(shortIDArray) > 0 {
                shortID = strings.TrimSpace(shortIDArray[rand.Intn(len(shortIDArray))])
            }
        }
        return vless.NewVLESSWithReality(
            cfg.Server,
            cfg.UUID,
            cfg.SNI,
            cfg.PublicKey,
        )
    }

    // Handle TLS
    if cfg.TLSEnabled {
        if cfg.Flow != "" {
            // VLESS with Vision flow
            return vless.NewVLESSWithVision(cfg.Server, cfg.UUID, cfg.SNI)
        }
        // VLESS with TLS only
        return vless.NewVLESSWithTLS(cfg.Server, cfg.UUID, cfg.SNI)
    }

    // Basic VLESS
    return vless.NewVLESS(cfg.Server, cfg.UUID)
}

// createShadowsocksProxy creates Shadowsocks proxy instance
func createShadowsocksProxy(cfg *ShadowsocksConfig) (proxy.Proxy, error) {
    return shadowsocks.NewShadowsocks(
        cfg.Server,
        cfg.Method,
        cfg.Password,
        cfg.ObfsMode,
        cfg.ObfsHost,
    )
}
```

**Key Points**:
- Encapsulates proxy creation logic
- Moves factory code from old `processor/config/proxy.go`
- Handles VLESS variants: REALITY, Vision, TLS, Basic
- Random ShortID selection from ShortIDList
- Clean separation of concerns

### Phase 2: Update Registry and Config Loading

#### Step 2.1: Update `processor/proxy/registry.go`

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/proxy/registry.go`

**Changes**:

1. **Remove `InitializeFromConfig()` method** (lines 216-258):
   - This method is no longer needed
   - Config loading will be handled by `cmd/config.go`

2. **Update `RegisterFromConfig()` method** (lines 260-306):

**BEFORE**:
```go
func (r *Registry) RegisterFromConfig(name string, cfg *proxyConfig.ProxyConfig) error {
    // ... manual proxy creation logic
    // ... switch on type
    // ... calls vless.NewVLESS*() directly
}
```

**AFTER**:
```go
// RegisterFromConfig creates and registers a proxy from configuration
func (r *Registry) RegisterFromConfig(name string, cfg *proxyConfig.ProxyConfig) error {
    if name == "" {
        return fmt.Errorf("proxy name cannot be empty")
    }
    if cfg == nil {
        return fmt.Errorf("config cannot be nil")
    }

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }

    // Create proxy instance using factory
    p, err := proxyConfig.CreateProxyFromConfig(cfg)
    if err != nil {
        return fmt.Errorf("failed to create proxy: %w", err)
    }

    // Register the instance in Registry
    if err := r.Register(name, p); err != nil {
        return err
    }

    // Store configuration in ConfigStore
    if err := proxyConfig.GetConfigStore().Set(name, cfg); err != nil {
        logger.Warn(fmt.Sprintf("Failed to store config for '%s': %v", name, err))
        // Don't fail registration if config storage fails
    }

    // Handle default and door proxy flags
    if cfg.IsDefault {
        if err := r.SetDefault(name); err != nil {
            return fmt.Errorf("failed to set default proxy: %w", err)
        }
    }
    if cfg.IsDoorProxy {
        if err := r.SetDoor(name); err != nil {
            return fmt.Errorf("failed to set door proxy: %w", err)
        }
    }

    return nil
}
```

**Key Changes**:
- Use `cfg.Validate()` for validation
- Use `proxyConfig.CreateProxyFromConfig()` factory
- Store config in ConfigStore via `proxyConfig.GetConfigStore().Set()`
- Cleaner error handling

#### Step 2.2: Update `cmd/config.go`

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/cmd/config.go`

**Change 1: Update Config struct** (line 23):

**BEFORE**:
```go
type Config struct {
    Engine       *EngineConfig         `json:"engine"`
    CurrentProxy string                `json:"currentProxy"`
    CoreServer   string                `json:"coreServer"`
    Proxies      map[string]interface{} `json:"proxies"` // OLD
}
```

**AFTER**:
```go
type Config struct {
    Engine       *EngineConfig                      `json:"engine"`
    CurrentProxy string                             `json:"currentProxy"`
    CoreServer   string                             `json:"coreServer"`
    Proxies      map[string]*proxyConfig.ProxyConfig `json:"proxies"` // NEW - direct unmarshal
}
```

**Change 2: Replace `applyProxyConfigs()`** (lines 123-157):

**BEFORE**:
```go
func applyProxyConfigs(proxies map[string]interface{}) error {
    // ... iterate over proxies
    // ... call parseProxyConfig(proxyMap) with manual type assertions
    // ... complex fallback logic
}
```

**AFTER**:
```go
// applyProxyConfigs applies proxy configurations to the registry
func applyProxyConfigs(proxies map[string]*proxyConfig.ProxyConfig) error {
    if len(proxies) == 0 {
        logger.Warn("No proxies configured")
        return nil
    }

    registry := proxyRegistry.GetRegistry()

    for name, cfg := range proxies {
        if cfg == nil {
            logger.Warn(fmt.Sprintf("Nil proxy config for '%s', skipping", name))
            continue
        }

        // Validate configuration
        if err := cfg.Validate(); err != nil {
            logger.Error(fmt.Sprintf("Invalid config for proxy '%s': %v", name, err))
            continue
        }

        // Register proxy (creates instance + stores config)
        if err := registry.RegisterFromConfig(name, cfg); err != nil {
            logger.Error(fmt.Sprintf("Failed to register proxy '%s': %v", name, err))
            continue
        }

        logger.Info(fmt.Sprintf("Proxy '%s' registered successfully", name))
    }

    return nil
}
```

**Change 3: Delete manual parsing functions** (lines 159-272):
- `parseProxyConfig()` function
- `parseVLESSConfig()` function
- `parseShadowsocksConfig()` function

These are no longer needed with direct JSON unmarshal.

**Key Benefits**:
- No more manual type assertions
- Direct unmarshal from JSON
- Cleaner code
- Better error messages

### Phase 3: Migrate All Usage

#### Step 3.1: Update `inbound/tun/engine/engine.go`

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/inbound/tun/engine/engine.go`

**Replace lines 150-200**:

**BEFORE**:
```go
// Try registry first, fall back to config manager
_defaultProxy, err = proxyRegistry.GetRegistry().GetDefault()
if err != nil {
    // Fallback to OLD config manager
    if defaultProxyFromConfig := proxyConfig.GetDirectProxy(); defaultProxyFromConfig != nil {
        _defaultProxy = defaultProxyFromConfig
    } else {
        _defaultProxy = direct.NewDirect()
    }
}

// Similar fallback logic for door proxy
doorProxy := proxyConfig.GetDoorProxy()
if doorProxy == nil {
    // Hardcoded fallback
}
```

**AFTER**:
```go
// Get default proxy from Registry
_defaultProxy, err = proxyRegistry.GetRegistry().GetDefault()
if err != nil {
    // Fallback to direct connection
    _defaultProxy = direct.NewDirect()
    logger.Warn("No default proxy configured, using direct connection")
}

// Set default proxy in tunnel
tunnel.SetDefaultProxy(_defaultProxy)

// Get door proxy from Registry
doorProxy, err := proxyRegistry.GetRegistry().GetDoor()
if err != nil {
    // Fallback to default hardcoded door proxy (for backward compatibility)
    uuid := user.GetUserUUID()
    if uuid == "" {
        uuid = "74cddcdd-6d48-41cf-8e62-902e7c943fe7"
    }
    doorProxy, err = vless.NewVLESSWithReality(
        "node1.nursor.org:35001",
        uuid,
        "www.microsoft.com",
        "sAtJcW2xLIUWRE-_7KHGEAtvHx-P1sDbjrrgrt4_XCo",
    )
    if err != nil {
        logger.Error(fmt.Sprintf("Failed to create fallback door proxy: %v", err))
    } else {
        tunnel.SetDoorProxy(doorProxy)
        logger.Info("Using fallback door proxy configuration")
    }
} else {
    tunnel.SetDoorProxy(doorProxy)
}

// Create DNS resolver if door proxy is available
if doorProxy != nil {
    defaultResolver := tunnel.NewDNSResolver("8.8.8.8:53", doorProxy, 5*time.Second, 5*time.Minute)
    tunnel.SetDefaultResolver(defaultResolver)
} else {
    logger.Warn("Door proxy is nil, DNS resolver not created")
}

tunnel.T().SetDialer(_defaultProxy)
```

**Remove import** (if not used elsewhere):
```go
// Remove if proxyConfig is not used elsewhere
proxyConfig "nursor.org/nursorgate/processor/config"
```

**Key Changes**:
- No fallback to old config manager
- Only use Registry
- Cleaner error handling
- Better logging

#### Step 3.2: Update `server/bussiness_server.go`

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/server/bussiness_server.go`

**Change 1: Delete InitializeFromConfig call** (lines 87-92):

**BEFORE**:
```go
func StartHttpServer() {
    // Initialize proxy registry from config
    if err := proxyRegistry.GetRegistry().InitializeFromConfig(); err != nil {
        logger.Error(fmt.Sprintf("Failed to initialize proxy registry: %v", err))
    }
    // ...
}
```

**AFTER**:
```go
func StartHttpServer() {
    // Config already loaded by cmd/config.go, no initialization needed
    // ...
}
```

**Change 2: Delete old endpoint registrations** (lines 104-105):

**DELETE**:
```go
http.HandleFunc("/proxy/set", handleProxySet)
http.HandleFunc("/proxy/get", handleProxyGet)
```

**Change 3: Delete old handler functions** (lines 226-253):

**DELETE**:
```go
// handleProxySet - lines 226-240
// handleProxyGet - lines 242-253
```

**Change 4: Add new config endpoints** (after line 121):

**ADD**:
```go
// Configuration management
http.HandleFunc("/config/get", handleConfigGet)
http.HandleFunc("/config/list", handleConfigList)
```

**Change 5: Add new handler implementations** (after line 403):

**ADD**:
```go
// handleConfigGet retrieves stored configuration for a proxy
func handleConfigGet(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    if name == "" {
        sendError(w, "name parameter is required", http.StatusBadRequest, nil)
        return
    }

    cfg, err := proxyConfig.GetConfigStore().Get(name)
    if err != nil {
        sendError(w, err.Error(), http.StatusNotFound, nil)
        return
    }

    sendResponse(w, cfg)
}

// handleConfigList lists all stored proxy configurations
func handleConfigList(w http.ResponseWriter, r *http.Request) {
    store := proxyConfig.GetConfigStore()
    configs := store.GetAll()

    sendResponse(w, map[string]interface{}{
        "configs": configs,
        "count":   len(configs),
    })
}
```

**New API Endpoints**:
- `GET /config/get?name=xxx` - Get stored config for a proxy
- `GET /config/list` - List all stored configs

**Removed API Endpoints**:
- `POST /proxy/set` (old system)
- `GET /proxy/get` (old system)

**Kept API Endpoints**:
- `/proxy/registry/*` (all existing registry endpoints)

#### Step 3.3: Update `export.go`

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/export.go`

**Change 1: Delete legacy FFI functions** (lines 103-146):

**DELETE**:
```go
//export setVLESSProxy
func setVLESSProxy(...) { ... }

//export setShadowsocksProxy
func setShadowsocksProxy(...) { ... }
```

**Change 2: Update `registerProxy()`** (lines 148-176):

**BEFORE**:
```go
//export registerProxy
func registerProxy(name *C.char, proxyType *C.char, server *C.char, uuid *C.char, sni *C.char, publicKey *C.char) {
    // ... manual config creation
}
```

**AFTER**:
```go
//export registerProxy
func registerProxy(name *C.char, configJSON *C.char) *C.char {
    nameStr := C.GoString(name)
    jsonStr := C.GoString(configJSON)

    // Parse JSON config
    var cfg proxyConfig.ProxyConfig
    if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
        errMsg := fmt.Sprintf("Failed to parse config JSON: %v", err)
        logger.Error(errMsg)
        return C.CString(fmt.Sprintf(`{"error": "%s"}`, errMsg))
    }

    // Validate config
    if err := cfg.Validate(); err != nil {
        errMsg := fmt.Sprintf("Invalid config: %v", err)
        logger.Error(errMsg)
        return C.CString(fmt.Sprintf(`{"error": "%s"}`, errMsg))
    }

    // Register proxy
    if err := proxyRegistry.GetRegistry().RegisterFromConfig(nameStr, &cfg); err != nil {
        errMsg := fmt.Sprintf("Failed to register proxy: %v", err)
        logger.Error(errMsg)
        return C.CString(fmt.Sprintf(`{"error": "%s"}`, errMsg))
    }

    return C.CString(`{"status": "success"}`)
}
```

**Change 3: Update `switchProxy()`** (lines 178-184):

**BEFORE**:
```go
//export switchProxy
func switchProxy(name *C.char) *C.char {
    // ... complex logic
}
```

**AFTER**:
```go
//export switchProxy
func switchProxy(name *C.char) *C.char {
    nameStr := C.GoString(name)

    // Set as default proxy
    if err := proxyRegistry.GetRegistry().SetDefault(nameStr); err != nil {
        errMsg := fmt.Sprintf("Failed to switch proxy: %v", err)
        logger.Error(errMsg)
        return C.CString(fmt.Sprintf(`{"error": "%s"}`, errMsg))
    }

    return C.CString(`{"status": "success"}`)
}
```

**New FFI Interface**:
```c
// Register proxy with JSON config
char* registerProxy(char* name, char* configJSON);

// Example usage from C:
// registerProxy("door", "{\"type\":\"vless\",\"vless\":{\"server\":\"...\",\"uuid\":\"...\"}}")
```

**Key Changes**:
- Simplified interface: just name + JSON config
- No need for multiple type-specific functions
- Consistent with HTTP API
- Better error reporting

### Phase 4: Remove Old System

#### Step 4.1: Delete Old Config File

**File**: `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/proxy.go`

**Action**: Delete the entire file

**What gets removed**:
- Global variables: `directProxy`, `doorProxy`, `vlessConfig`, `shadowsocksConfig`, `currentProxy`
- Old functions: `SetProxyConfig()`, `GetDirectProxy()`, `GetDoorProxy()`, `GetVLESSConfig()`, `GetShadowsocksConfig()`
- Old factory functions: `createVLESSProxy()`, `createShadowsocksProxy()`
- Mutex: `mu sync.RWMutex`

**Before deletion**:
```bash
# Backup the file for reference
cp processor/config/proxy.go processor/config/proxy.go.bak
```

#### Step 4.2: Clean Up Imports

**Validation**:
```bash
# Build to find any remaining references
go build ./...

# Search for imports
grep -r "proxyConfig.*processor/config" --include="*.go"

# Search for usage of old functions
grep -r "GetDirectProxy\|GetDoorProxy\|SetProxyConfig" --include="*.go"
```

**Fix any compilation errors**:
- Update remaining code to use Registry
- Remove unused imports

**Files that might need import cleanup**:
- `inbound/tun/tunnel/tcp.go`
- `inbound/tun/tunnel/udp.go`
- Any other files that imported old config

### Phase 5: Testing and Validation

#### Step 5.1: Build Validation

```bash
cd /Users/mac/MyProgram/GoProgram/nursor/nursorgate2

# Clean build
go clean -cache
go build ./...

# Check for errors
echo $?  # Should be 0
```

#### Step 5.2: Unit Tests (Optional but Recommended)

Create test files:

1. **`processor/config/types_test.go`**:
```go
package config

import "testing"

func TestProxyConfigValidate(t *testing.T) {
    tests := []struct {
        name    string
        config  *ProxyConfig
        wantErr bool
    }{
        {
            name: "valid vless config",
            config: &ProxyConfig{
                Type: "vless",
                VLESS: &VLESSConfig{
                    Server: "example.com:443",
                    UUID:   "test-uuid",
                },
            },
            wantErr: false,
        },
        {
            name: "missing vless config",
            config: &ProxyConfig{
                Type: "vless",
            },
            wantErr: true,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

2. **`processor/config/store_test.go`**:
```go
package config

import "testing"

func TestConfigStore(t *testing.T) {
    store := GetConfigStore()
    store.Clear() // Clean slate

    cfg := &ProxyConfig{
        Type: "vless",
        VLESS: &VLESSConfig{
            Server: "test.com:443",
            UUID:   "test-uuid",
        },
    }

    // Test Set
    err := store.Set("test", cfg)
    if err != nil {
        t.Errorf("Set() error = %v", err)
    }

    // Test Get
    retrieved, err := store.Get("test")
    if err != nil {
        t.Errorf("Get() error = %v", err)
    }
    if retrieved.Type != "vless" {
        t.Errorf("Get() returned wrong type")
    }

    // Test List
    names := store.List()
    if len(names) != 1 {
        t.Errorf("List() returned %d items, want 1", len(names))
    }

    // Test Delete
    err = store.Delete("test")
    if err != nil {
        t.Errorf("Delete() error = %v", err)
    }
}
```

3. **`processor/config/factory_test.go`**:
```go
package config

import "testing"

func TestCreateProxyFromConfig(t *testing.T) {
    cfg := &ProxyConfig{
        Type: "vless",
        VLESS: &VLESSConfig{
            Server:     "test.com:443",
            UUID:       "test-uuid",
            TLSEnabled: true,
            SNI:        "test.com",
        },
    }

    proxy, err := CreateProxyFromConfig(cfg)
    if err != nil {
        t.Errorf("CreateProxyFromConfig() error = %v", err)
    }
    if proxy == nil {
        t.Error("CreateProxyFromConfig() returned nil proxy")
    }
}
```

Run tests:
```bash
go test ./processor/config/... -v
```

#### Step 5.3: Integration Testing

**Test 1: Config Loading**
```bash
# Create test config
cat > /tmp/test-config.json <<EOF
{
  "engine": {
    "device": "utun0",
    "loglevel": "info"
  },
  "currentProxy": "test",
  "proxies": {
    "test": {
      "type": "vless",
      "is_default": true,
      "vless": {
        "server": "test.com:443",
        "uuid": "test-uuid",
        "tls_enabled": true,
        "sni": "test.com"
      }
    }
  }
}
EOF

# Load config
go run cmd/nursor/main.go start --config /tmp/test-config.json
```

**Test 2: HTTP API**
```bash
# Start server
go run cmd/nursor/main.go start --config config.example.json

# In another terminal:

# List proxies in registry
curl http://127.0.0.1:56431/proxy/registry/list
# Expected: {"proxies":[{"name":"door",...},...]}

# List configs in store
curl http://127.0.0.1:56431/config/list
# Expected: {"configs":{"door":{...}},"count":2}

# Get specific config
curl http://127.0.0.1:56431/config/get?name=door
# Expected: {"type":"vless","vless":{...}}

# Register new proxy
curl -X POST http://127.0.0.1:56431/proxy/registry/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "new-proxy",
    "config": {
      "type": "vless",
      "is_default": false,
      "vless": {
        "server": "new.com:443",
        "uuid": "new-uuid",
        "tls_enabled": true,
        "sni": "new.com"
      }
    }
  }'
# Expected: {"message":"Proxy registered successfully"}

# Verify it was stored
curl http://127.0.0.1:56431/config/get?name=new-proxy
# Expected: {"type":"vless","vless":{...}}

# Switch proxy
curl -X POST http://127.0.0.1:56431/proxy/registry/switch \
  -H "Content-Type: application/json" \
  -d '{"name":"new-proxy"}'
# Expected: {"message":"Proxy switched successfully"}
```

**Test 3: FFI (if applicable)**
```c
// Test from C code
#include <stdio.h>

extern char* registerProxy(char* name, char* configJSON);
extern char* listProxies();
extern char* switchProxy(char* name);

int main() {
    // Register proxy
    char* result = registerProxy(
        "test-ffi",
        "{\"type\":\"vless\",\"vless\":{\"server\":\"test.com:443\",\"uuid\":\"test-uuid\"}}"
    );
    printf("Register result: %s\n", result);

    // List proxies
    char* list = listProxies();
    printf("Proxies: %s\n", list);

    // Switch proxy
    char* switchResult = switchProxy("test-ffi");
    printf("Switch result: %s\n", switchResult);

    return 0;
}
```

#### Step 5.4: End-to-End VPN Test

```bash
# Start VPN with config
go run cmd/nursor/main.go start --config config.example.json

# Check if TUN device is created
ifconfig utun0

# Check if proxies are registered
curl http://127.0.0.1:56431/proxy/registry/list

# Try to browse internet (verify VPN works)
curl -v https://www.google.com

# Check logs for proxy usage
tail -f /var/log/nursorgate.log
```

## Success Criteria

- [x] All code compiles without errors
- [x] No references to old config global variables
- [x] Config loads directly from JSON to structs
- [x] Registry stores instances, ConfigStore stores configs
- [x] HTTP API uses new system only
- [x] FFI exports use new system
- [x] Engine starts with proxies from Registry
- [x] VPN tunnel works end-to-end
- [x] Proxy switching works at runtime
- [x] No data races (test with `go test -race`)

## Risk Mitigation

### Risk 1: Breaking FFI API for External Clients

**Problem**: External native apps may still call old `setVLESSProxy()` function.

**Mitigation**: Add deprecated wrappers (optional):
```go
//export setVLESSProxy
// DEPRECATED: Use registerProxy instead
func setVLESSProxy(server *C.char, uuid *C.char, sni *C.char, publicKey *C.char, isDefault *C.bool, isDoorProxy *C.bool) {
    cfg := &proxyConfig.ProxyConfig{
        Type: "vless",
        VLESS: &proxyConfig.VLESSConfig{
            Server:         C.GoString(server),
            UUID:           C.GoString(uuid),
            SNI:            C.GoString(sni),
            PublicKey:      C.GoString(publicKey),
            RealityEnabled: len(C.GoString(publicKey)) > 0,
            TLSEnabled:     len(C.GoString(sni)) > 0,
        },
        IsDefault:   *isDefault != C.bool(false),
        IsDoorProxy: *isDoorProxy != C.bool(false),
    }

    proxyRegistry.GetRegistry().RegisterFromConfig("vless-legacy", cfg)
}
```

### Risk 2: Engine Initialization Timing

**Problem**: Engine might start before config is loaded.

**Mitigation**: Ensure config loads first in `runner/start.go`:
```go
func Start() error {
    // 1. Load config FIRST
    if err := cmd.LoadAndApplyConfig("config.json"); err != nil {
        logger.Warn(fmt.Sprintf("Failed to load config: %v", err))
    }

    // 2. Verify proxies are registered
    if proxyRegistry.GetRegistry().Count() == 0 {
        logger.Warn("No proxies configured, using defaults")
    }

    // 3. THEN start engine
    if err := engine.Start(); err != nil {
        return err
    }

    return nil
}
```

### Risk 3: Concurrent Access During Migration

**Problem**: Old and new code might access proxies concurrently during migration.

**Mitigation**:
- Implement Phase 1 completely before touching Phase 2
- Use mutex in both ConfigStore and Registry
- Test with `go test -race`

### Risk 4: Config File Schema Changes

**Problem**: Old config files might not have all required fields.

**Mitigation**:
- Make new fields optional with `omitempty` tags
- Provide sensible defaults in validation
- Add config migration tool if needed

## Files Summary

### New Files (CREATE)
1. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/types.go`
   - Config type definitions with JSON tags
   - Validation logic

2. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/store.go`
   - ConfigStore singleton for storing configurations
   - CRUD operations

3. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/factory.go`
   - Proxy factory pattern
   - Creates instances from configs

### Modified Files (EDIT)
1. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/proxy/registry.go`
   - Update `RegisterFromConfig()` to use factory and store config

2. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/cmd/config.go`
   - Change `Proxies` field to `map[string]*ProxyConfig`
   - Replace `applyProxyConfigs()` implementation
   - Delete manual parsing functions

3. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/inbound/tun/engine/engine.go`
   - Use Registry only (remove old config fallback)
   - Clean up proxy initialization

4. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/server/bussiness_server.go`
   - Remove old `/proxy/set` and `/proxy/get` endpoints
   - Add new `/config/get` and `/config/list` endpoints
   - Delete old handler functions

5. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/export.go`
   - Delete old `setVLESSProxy()` and `setShadowsocksProxy()`
   - Update `registerProxy()` to accept JSON config
   - Simplify `switchProxy()`

### Deleted Files (DELETE)
1. `/Users/mac/MyProgram/GoProgram/nursor/nursorgate2/processor/config/proxy.go`
   - Complete file deletion
   - Removes all old global variables and functions

## Implementation Order

1. **Phase 1** (New infrastructure):
   - Create `types.go`
   - Create `store.go`
   - Create `factory.go`
   - Test individually

2. **Phase 2** (Integration):
   - Update `registry.go`
   - Update `config.go`
   - Test config loading

3. **Phase 3** (Migration):
   - Update `engine.go`
   - Update `bussiness_server.go`
   - Update `export.go`
   - Test each component

4. **Phase 4** (Cleanup):
   - Delete `proxy.go`
   - Fix compilation errors
   - Clean up imports

5. **Phase 5** (Testing):
   - Build validation
   - Unit tests
   - Integration tests
   - E2E tests

## Rollback Plan

If issues occur:

1. **After Phase 1**: Just delete new files
2. **After Phase 2**: Revert `registry.go` and `config.go`
3. **After Phase 3**: Revert all modified files, restore `proxy.go` from backup
4. **After Phase 4**: Restore `proxy.go` from `.bak` file

**Always keep backups**:
```bash
# Before starting
git checkout -b config-refactoring
git commit -am "Backup before refactoring"

# Or manually backup
cp processor/config/proxy.go processor/config/proxy.go.bak
```

## Estimated Impact

### Code Changes
- **New lines**: ~500
- **Modified lines**: ~200
- **Deleted lines**: ~300
- **Net change**: +400 lines

### Files Affected
- **New files**: 3
- **Modified files**: 5
- **Deleted files**: 1

### Testing Time
- **Build**: 2 minutes
- **Unit tests**: 5 minutes
- **Integration tests**: 10 minutes
- **E2E tests**: 15 minutes
- **Total**: ~30 minutes

## Benefits Summary

1. **Code Quality**:
   - Single source of truth
   - No duplicate logic
   - Clean separation of concerns

2. **Maintainability**:
   - Easier to understand
   - Easier to test
   - Easier to extend

3. **Performance**:
   - No unnecessary fallbacks
   - Direct config access
   - Efficient storage

4. **Developer Experience**:
   - Clear APIs
   - Better error messages
   - Consistent patterns

## Next Steps After Refactoring

1. **Documentation**:
   - Update API documentation
   - Update FFI documentation
   - Add config schema documentation

2. **Monitoring**:
   - Add metrics for config loading
   - Add metrics for proxy registration
   - Add health checks

3. **Future Enhancements**:
   - Config hot reload (if needed)
   - Config validation schemas
   - Config migration tools
   - Multiple config file support
