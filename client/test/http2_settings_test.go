package test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"nursor.org/nursorgate/client/server/helper"
)

func TestHttp2SettingsParsing(t *testing.T) {
	// 创建一个模拟的 WatcherWrapConn
	watcher := &helper.WatcherWrapConn{}

	// 创建模拟的 SETTINGS 帧
	// SETTINGS_HEADER_TABLE_SIZE = 4096
	// SETTINGS_MAX_CONCURRENT_STREAMS = 100
	// SETTINGS_INITIAL_WINDOW_SIZE = 65535

	settings := []struct {
		identifier uint16
		value      uint32
	}{
		{helper.SETTINGS_HEADER_TABLE_SIZE, 4096},
		{helper.SETTINGS_MAX_CONCURRENT_STREAMS, 100},
		{helper.SETTINGS_INITIAL_WINDOW_SIZE, 65535},
	}

	// 构建 SETTINGS 帧的 payload
	var payload bytes.Buffer
	for _, setting := range settings {
		binary.Write(&payload, binary.BigEndian, setting.identifier)
		binary.Write(&payload, binary.BigEndian, setting.value)
	}

	// 调用解析函数
	watcher.ParseSettingsFrame(payload.Bytes())

	// 验证解析结果
	for _, setting := range settings {
		value, exists := watcher.GetHttp2Setting(setting.identifier)
		if !exists {
			t.Errorf("SETTINGS %d not found", setting.identifier)
		}
		if value != setting.value {
			t.Errorf("SETTINGS %d expected %d, got %d", setting.identifier, setting.value, value)
		}
	}

	// 测试获取所有 SETTINGS
	allSettings := watcher.GetAllHttp2Settings()
	if len(allSettings) != len(settings) {
		t.Errorf("Expected %d settings, got %d", len(settings), len(allSettings))
	}

	t.Log("HTTP/2 SETTINGS parsing test passed")
}
