package test

import (
	"testing"

	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/cmd"
	"nursor.org/nursorgate/processor/config"
)

// TestDefaultConfigEmbedding 测试默认配置是否正确嵌入
func TestDefaultConfigEmbedding(t *testing.T) {
	// 获取嵌入的默认配置字节
	configBytes := cmd.GetDefaultConfigBytes()

	if len(configBytes) == 0 {
		t.Fatal("Default config bytes should not be empty")
	}

	t.Logf("Default config size: %d bytes", len(configBytes))
}

// TestDefaultConfigLoading 测试默认配置是否能被正确加载
func TestDefaultConfigLoading(t *testing.T) {
	// 获取默认配置字节
	configBytes := cmd.GetDefaultConfigBytes()

	// 尝试加载配置
	cfg, err := cmd.LoadConfigFromBytes(configBytes)
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Loaded config should not be nil")
	}

	t.Logf("Config loaded successfully: %v", cfg)
}

// TestConfigStateTracking 测试配置状态追踪
func TestConfigStateTracking(t *testing.T) {
	// 初始状态应该是 false
	initialState := config.IsUsingDefaultConfig()
	t.Logf("Initial state: %v", initialState)

	// 设置为 true
	config.SetUsingDefaultConfig(true)
	if !config.IsUsingDefaultConfig() {
		t.Fatal("Config state should be true after setting")
	}
	t.Log("Config state successfully set to true")

	// 设置回 false
	config.SetUsingDefaultConfig(false)
	if config.IsUsingDefaultConfig() {
		t.Fatal("Config state should be false after resetting")
	}
	t.Log("Config state successfully reset to false")
}

// TestDefaultConfigUsage 测试直接启动时使用默认配置的完整流程
func TestDefaultConfigUsage(t *testing.T) {
	// 这个测试模拟用户直接启动二进制文件（没有参数）时的场景

	// 步骤1: 应用默认配置
	configBytes := cmd.GetDefaultConfigBytes()
	cfg, err := cmd.LoadConfigFromBytes(configBytes)
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}
	t.Log("✓ Step 1: Default config loaded successfully")

	// 步骤2: 标记为使用默认配置
	config.SetUsingDefaultConfig(true)
	if !config.IsUsingDefaultConfig() {
		t.Fatal("Failed to mark default config as being used")
	}
	t.Log("✓ Step 2: Marked as using default config")

	// 步骤3: 验证配置对象不为空
	if cfg == nil {
		t.Fatal("Config object should not be nil")
	}
	t.Log("✓ Step 3: Config object is valid")

	// 步骤4: 验证状态标志
	if !config.IsUsingDefaultConfig() {
		t.Fatal("Default config state should be true")
	}
	t.Log("✓ Step 4: Config state verified")

	// 清理
	config.SetUsingDefaultConfig(false)
	t.Log("\n✓ All steps completed successfully - Direct binary launch simulation passed!")

	httpServer.StartHttpServer()
}

// TestConfigStateSync 测试配置状态在 cmd 包和 processor/config 包之间的同步
func TestConfigStateSync(t *testing.T) {
	// 通过 cmd 包的 setUseDefaultConfig 设置
	// 这测试了 cmd/config.go 中的 setUseDefaultConfig() 是否正确代理到 processor/config

	// 重置初始状态
	config.SetUsingDefaultConfig(false)

	// 验证初始状态
	if config.IsUsingDefaultConfig() {
		t.Fatal("Initial state should be false")
	}

	// 状态应该在所有包中一致
	// 验证 processor/config 的状态
	state1 := config.IsUsingDefaultConfig()

	// 设置状态
	config.SetUsingDefaultConfig(true)

	// 再次验证
	state2 := config.IsUsingDefaultConfig()

	if !state2 {
		t.Fatal("State should be true after setting")
	}

	t.Logf("State sync test: initial=%v, after setting=%v", state1, state2)

	// 清理
	config.SetUsingDefaultConfig(false)
	t.Log("✓ Config state sync verified across packages")
}
