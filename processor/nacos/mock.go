package nacos

import (
	nacosModel "github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

// MockConfigClient is a mock implementation of Nacos config client for testing
// This is exported so it can be used in tests from other packages
type MockConfigClient struct {
	Configs  map[string]string
	Listener func(namespace, group, dataId, data string)
}

// NewMockConfigClient creates a new mock Nacos config client for testing
func NewMockConfigClient() *MockConfigClient {
	return &MockConfigClient{
		Configs: make(map[string]string),
	}
}

func (m *MockConfigClient) GetConfig(param vo.ConfigParam) (string, error) {
	key := RoutingRulesDataID
	if content, ok := m.Configs[key]; ok {
		return content, nil
	}
	return "", nil
}

func (m *MockConfigClient) PublishConfig(param vo.ConfigParam) (bool, error) {
	// Store the content for testing
	m.Configs[param.DataId] = param.Content
	return true, nil
}

func (m *MockConfigClient) ListenConfig(param vo.ConfigParam) error {
	// Store the listener callback for testing
	if param.OnChange != nil {
		m.Listener = param.OnChange
	}
	return nil
}

func (m *MockConfigClient) CancelListenConfig(param vo.ConfigParam) error {
	m.Listener = nil
	return nil
}

func (m *MockConfigClient) DeleteConfig(param vo.ConfigParam) (bool, error) {
	delete(m.Configs, param.DataId)
	return true, nil
}

func (m *MockConfigClient) SearchConfig(param vo.SearchConfigParam) (*nacosModel.ConfigPage, error) {
	return &nacosModel.ConfigPage{}, nil
}

func (m *MockConfigClient) PublishAggr(param vo.ConfigParam) (bool, error) {
	m.Configs[param.DataId] = param.Content
	return true, nil
}
