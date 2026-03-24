package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/uuid"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/app/http/storage"
	"nursor.org/nursorgate/common/cache"
	"nursor.org/nursorgate/processor/config"
)

const customerConfigFilePath = "~/.aliang/config.json"

type CustomerConfigService struct {
	store *storage.SoftwareConfigStore
}

type CustomerConfigCommitResult struct {
	Version  uint64
	Customer map[string]interface{}
}

type CustomerConfigValidationError struct {
	err error
}

func (e *CustomerConfigValidationError) Error() string {
	if e == nil || e.err == nil {
		return "invalid customer config"
	}
	return e.err.Error()
}

func (e *CustomerConfigValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func NewCustomerConfigService() *CustomerConfigService {
	return &CustomerConfigService{store: storage.NewSoftwareConfigStore()}
}

func (s *CustomerConfigService) GetPresetAIRuleProviders() []config.AIRuleProviderPreset {
	return config.PresetAIRuleProviders
}

func (s *CustomerConfigService) GetCommittedCustomerConfig() (map[string]interface{}, uint64, error) {
	coordinator := config.GetEffectiveConfigCommitCoordinator()
	if committed := coordinator.LastCommittedSnapshot(); committed != nil && strings.TrimSpace(committed.Content) != "" {
		customer, err := extractCustomerFromConfigContent(committed.Content)
		if err != nil {
			return nil, 0, err
		}
		return customer, coordinator.Version(), nil
	}

	globalCfg := config.GetGlobalConfig()
	if globalCfg == nil {
		return map[string]interface{}{}, coordinator.Version(), nil
	}

	raw, err := json.Marshal(globalCfg)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal global config: %w", err)
	}

	customer, err := extractCustomerFromConfigContent(string(raw))
	if err != nil {
		return nil, 0, err
	}
	return customer, coordinator.Version(), nil
}

func (s *CustomerConfigService) UpdateCommittedCustomerConfig(payload []byte) (*CustomerConfigCommitResult, error) {
	if len(payload) == 0 {
		return nil, &CustomerConfigValidationError{err: errors.New("request body is empty")}
	}

	globalCfg, err := resolveBaseConfigForCustomerUpdate()
	if err != nil {
		return nil, err
	}

	// Ensure api_server is present so full config validation passes
	// even when the committed snapshot has an empty value.
	if strings.TrimSpace(globalCfg.APIServer) == "" {
		globalCfg.APIServer = "https://api.example.com"
	}

	mergedContent, nextCfg, err := mergeCustomerPayload(globalCfg, payload)
	if err != nil {
		return nil, &CustomerConfigValidationError{err: err}
	}

	if err := nextCfg.Validate(); err != nil {
		return nil, &CustomerConfigValidationError{err: err}
	}

	nextCfgRaw, err := json.Marshal(nextCfg)
	if err != nil {
		return nil, fmt.Errorf("marshal validated config: %w", err)
	}

	nextSnapshot := &config.EffectiveConfigSnapshot{
		UUID:     uuid.NewString(),
		Software: "runtime",
		Name:     "customer-config",
		FilePath: customerConfigFilePath,
		Version:  "",
		Format:   models.ConfigFormatJSON,
		Content:  string(nextCfgRaw),
	}

	coordinator := config.GetEffectiveConfigCommitCoordinator()
	commitResult, err := coordinator.Commit(
		nextSnapshot,
		func(snapshot *config.EffectiveConfigSnapshot) error {
			if snapshot == nil {
				return errors.New("file snapshot is required")
			}
			var fileCfg config.Config
			if err := json.Unmarshal([]byte(snapshot.Content), &fileCfg); err != nil {
				return fmt.Errorf("decode snapshot content for file persist: %w", err)
			}
			expandedPath, err := cache.ExpandHomePath(snapshot.FilePath)
			if err != nil {
				return fmt.Errorf("expand file path: %w", err)
			}
			return config.SaveConfigToFile(&fileCfg, expandedPath)
		},
		func(snapshot *config.EffectiveConfigSnapshot) error {
			if snapshot == nil {
				return errors.New("db snapshot is required")
			}
			if s == nil || s.store == nil {
				return errors.New("customer config store is not initialized")
			}
			if err := s.store.SaveEffectiveConfigSnapshot(models.SoftwareEffectiveConfigSnapshot{
				Software:       snapshot.Software,
				ConfigUUID:     snapshot.UUID,
				ConfigName:     snapshot.Name,
				ConfigFilePath: snapshot.FilePath,
				ConfigVersion:  snapshot.Version,
				ConfigFormat:   snapshot.Format,
				SnapshotJSON:   snapshot.Content,
			}); err != nil {
				return err
			}

			var committedCfg config.Config
			if err := json.Unmarshal([]byte(snapshot.Content), &committedCfg); err != nil {
				return fmt.Errorf("decode snapshot content for memory commit: %w", err)
			}
			config.SetGlobalConfig(&committedCfg)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	customer, err := extractCustomerFromConfigContent(mergedContent)
	if err != nil {
		return nil, err
	}

	return &CustomerConfigCommitResult{
		Version:  commitResult.Version,
		Customer: customer,
	}, nil
}

func IsCustomerConfigValidationError(err error) bool {
	var target *CustomerConfigValidationError
	return errors.As(err, &target)
}

func mergeCustomerPayload(baseCfg *config.Config, payload []byte) (string, *config.Config, error) {
	if baseCfg == nil {
		return "", nil, errors.New("global config is not initialized")
	}

	customerRaw, err := normalizeCustomerPayload(payload)
	if err != nil {
		return "", nil, err
	}

	baseRaw, err := json.Marshal(baseCfg)
	if err != nil {
		return "", nil, fmt.Errorf("marshal current config: %w", err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(baseRaw, &root); err != nil {
		return "", nil, fmt.Errorf("decode current config: %w", err)
	}
	root["customer"] = customerRaw

	mergedRaw, err := json.Marshal(root)
	if err != nil {
		return "", nil, fmt.Errorf("marshal merged config: %w", err)
	}

	var nextCfg config.Config
	if err := json.Unmarshal(mergedRaw, &nextCfg); err != nil {
		return "", nil, err
	}

	return string(mergedRaw), &nextCfg, nil
}

func normalizeCustomerPayload(payload []byte) (json.RawMessage, error) {
	var rawRoot map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawRoot); err != nil {
		return nil, fmt.Errorf("invalid json format: %w", err)
	}

	if len(rawRoot) == 0 {
		return nil, errors.New("customer config payload is required")
	}

	if rawCustomer, ok := rawRoot["customer"]; ok {
		if len(rawRoot) > 1 {
			forbidden := make([]string, 0, len(rawRoot)-1)
			for key := range rawRoot {
				if key == "customer" {
					continue
				}
				forbidden = append(forbidden, key)
			}
			sort.Strings(forbidden)
			return nil, fmt.Errorf("customer.%s is forbidden: editable customer fields are [proxy ai_rules proxy_rules]", forbidden[0])
		}
		return rawCustomer, nil
	}

	return json.RawMessage(payload), nil
}

func extractCustomerFromConfigContent(content string) (map[string]interface{}, error) {
	if strings.TrimSpace(content) == "" {
		return map[string]interface{}{}, nil
	}

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		return nil, fmt.Errorf("decode config content: %w", err)
	}

	customer, ok := root["customer"].(map[string]interface{})
	if !ok || customer == nil {
		return map[string]interface{}{}, nil
	}

	return customer, nil
}

func resolveBaseConfigForCustomerUpdate() (*config.Config, error) {
	if globalCfg := config.GetGlobalConfig(); globalCfg != nil {
		return globalCfg, nil
	}

	coordinator := config.GetEffectiveConfigCommitCoordinator()
	if committed := coordinator.LastCommittedSnapshot(); committed != nil && strings.TrimSpace(committed.Content) != "" {
		var committedCfg config.Config
		if err := json.Unmarshal([]byte(committed.Content), &committedCfg); err != nil {
			return nil, fmt.Errorf("decode committed snapshot content: %w", err)
		}
		return &committedCfg, nil
	}

	configPath, err := cache.ExpandHomePath(customerConfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("expand config file path: %w", err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return bootstrapBaseConfigForCustomerUpdate(), nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var fileCfg config.Config
	if err := json.Unmarshal(raw, &fileCfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	return &fileCfg, nil
}

func bootstrapBaseConfigForCustomerUpdate() *config.Config {
	return &config.Config{
		APIServer:    "https://api.example.com",
		CurrentProxy: "direct",
	}
}
