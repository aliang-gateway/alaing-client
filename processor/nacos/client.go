package nacos

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"nursor.org/nursorgate/common/logger"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
)

// InitializeClient initializes a Nacos config client from server URL
// Returns a config client instance or error
func InitializeClient(serverURL string) (config_client.IConfigClient, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("Nacos server URL is empty")
	}

	// Parse server URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Nacos server URL: %w", err)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "8848" // Nacos default port
	}

	// Create server config
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: host,
			Port:   parsePort(port),
		},
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create client config
	clientConfig := constant.ClientConfig{
		NamespaceId:         "", // Use default namespace
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              filepath.Join(homeDir, ".nonelane", "nacos", "log"),
		CacheDir:            filepath.Join(homeDir, ".nonelane", "nacos", "cache"),
		LogLevel:            "info",
	}

	// Create config client
	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Nacos client: %w", err)
	}

	logger.Info(fmt.Sprintf("Nacos client initialized successfully (server: %s:%s)", host, port))
	return client, nil
}

// parsePort converts port string to uint64
func parsePort(portStr string) uint64 {
	// Default to 8848 if parsing fails
	var port uint64 = 8848
	if portStr != "" {
		// Simple conversion
		for _, c := range portStr {
			if c >= '0' && c <= '9' {
				port = port*10 + uint64(c-'0')
			}
		}
	}
	return port
}

// InitializeFromConfig initializes Nacos client and ConfigManager
// This is the main entry point for startup integration (T061)
func InitializeFromConfig(nacosServerURL string) (*ConfigManager, error) {
	// Initialize Nacos client
	client, err := InitializeClient(nacosServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Nacos client: %w", err)
	}

	// Create ConfigManager
	manager := NewConfigManager(client)

	// Load initial configuration
	config, err := manager.LoadConfig()
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to load initial config from Nacos: %v", err))
		logger.Info("Using default configuration")
	}

	// Start listener if auto_update is enabled
	if config != nil && config.Settings.AutoUpdate {
		if err := manager.StartListening(); err != nil {
			logger.Error(fmt.Sprintf("Failed to start Nacos listener: %v", err))
			// Don't fail startup if listener fails
		}
	}

	logger.Info(fmt.Sprintf("Nacos configuration manager initialized (auto_update=%v)", manager.GetAutoUpdateStatus()))
	return manager, nil
}

// T062: Graceful shutdown helper
func GracefulShutdown(manager *ConfigManager) {
	if manager == nil {
		return
	}

	if err := manager.StopListening(); err != nil {
		logger.Error(fmt.Sprintf("Error stopping Nacos listener: %v", err))
	} else {
		logger.Info("Nacos listener stopped gracefully")
	}
}
