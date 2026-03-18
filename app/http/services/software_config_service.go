package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/app/http/storage"
	"nursor.org/nursorgate/common/cache"
)

type SoftwareConfigService struct {
	store      *storage.SoftwareConfigStore
	httpClient *http.Client
}

func NewSoftwareConfigService() *SoftwareConfigService {
	client := &http.Client{Timeout: 30 * time.Second}
	return &SoftwareConfigService{
		store:      storage.NewSoftwareConfigStore(),
		httpClient: client,
	}
}

func NewSoftwareConfigServiceWithStore(store *storage.SoftwareConfigStore, client *http.Client) *SoftwareConfigService {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &SoftwareConfigService{store: store, httpClient: client}
}

func (s *SoftwareConfigService) Save(req models.SaveSoftwareConfigRequest) (*models.SoftwareConfig, error) {
	cfg, err := s.normalizeSaveRequest(req)
	if err != nil {
		return nil, err
	}

	if err := s.store.Upsert(*cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *SoftwareConfigService) Activate(req models.ActivateSoftwareConfigRequest) (*models.SoftwareConfig, error) {
	if strings.TrimSpace(req.FilePath) == "" {
		return nil, errors.New("file_path is required")
	}

	cfg, err := s.normalizeActivateRequest(req)
	if err != nil {
		return nil, err
	}

	if err := writeConfigFile(cfg.FilePath, cfg.Content); err != nil {
		return nil, err
	}

	if err := s.store.Activate(*cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (s *SoftwareConfigService) List() ([]models.SoftwareConfig, error) {
	return s.store.List()
}

func (s *SoftwareConfigService) PushToCloud(req models.CloudPushRequest) (*models.CloudPushResponse, error) {
	cloudURL := strings.TrimSpace(req.CloudURL)
	if cloudURL == "" {
		return nil, errors.New("cloud_url is required")
	}

	configs, err := s.store.List()
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(models.CloudConfigBatch{Configs: configs})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, cloudURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(req.AuthToken) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(req.AuthToken))
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloud push failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return &models.CloudPushResponse{PushedCount: len(configs)}, nil
}

func (s *SoftwareConfigService) PullFromCloud(req models.CloudPullRequest) (*models.CloudPullResponse, error) {
	cloudURL := strings.TrimSpace(req.CloudURL)
	if cloudURL == "" {
		return nil, errors.New("cloud_url is required")
	}

	httpReq, err := http.NewRequest(http.MethodGet, cloudURL, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.AuthToken) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(req.AuthToken))
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloud pull failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var batch models.CloudConfigBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil, err
	}

	now := time.Now()
	for i := range batch.Configs {
		if strings.TrimSpace(batch.Configs[i].UUID) == "" {
			batch.Configs[i].UUID = uuid.NewString()
		}
		if batch.Configs[i].CreatedAt.IsZero() {
			batch.Configs[i].CreatedAt = now
		}
		if batch.Configs[i].UpdatedAt.IsZero() {
			batch.Configs[i].UpdatedAt = now
		}
	}

	inserted, updated, kept, err := s.store.MergeByLatest(batch.Configs)
	if err != nil {
		return nil, err
	}

	return &models.CloudPullResponse{
		PulledCount:       len(batch.Configs),
		InsertedCount:     inserted,
		UpdatedFromCloud:  updated,
		KeptLocalNewerCnt: kept,
	}, nil
}

func (s *SoftwareConfigService) normalizeSaveRequest(req models.SaveSoftwareConfigRequest) (*models.SoftwareConfig, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("name is required")
	}
	if strings.TrimSpace(req.FilePath) == "" {
		return nil, errors.New("file_path is required")
	}

	format, err := validateFormat(req.Format)
	if err != nil {
		return nil, err
	}
	if err := validateContentByFormat(format, req.Content); err != nil {
		return nil, err
	}

	now := time.Now()
	createdAt, err := parseTimeOrError(req.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTimeOrError(req.UpdatedAt)
	if err != nil {
		return nil, err
	}

	id := strings.TrimSpace(req.UUID)
	if id == "" {
		id = uuid.NewString()
	}

	path, err := cache.ExpandHomePath(strings.TrimSpace(req.FilePath))
	if err != nil {
		return nil, err
	}

	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}

	if strings.TrimSpace(req.UUID) != "" {
		existing, found, findErr := s.store.FindByUUID(id)
		if findErr != nil {
			return nil, findErr
		}
		if found && strings.TrimSpace(req.CreatedAt) == "" {
			createdAt = existing.CreatedAt
		}
	}

	return &models.SoftwareConfig{
		UUID:      id,
		Name:      strings.TrimSpace(req.Name),
		FilePath:  path,
		Version:   strings.TrimSpace(req.Version),
		InUse:     req.InUse,
		Format:    format,
		Content:   req.Content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (s *SoftwareConfigService) normalizeActivateRequest(req models.ActivateSoftwareConfigRequest) (*models.SoftwareConfig, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("name is required")
	}
	if strings.TrimSpace(req.FilePath) == "" {
		return nil, errors.New("file_path is required")
	}

	format, err := validateFormat(req.Format)
	if err != nil {
		return nil, err
	}
	if err := validateContentByFormat(format, req.Content); err != nil {
		return nil, err
	}

	now := time.Now()
	createdAt, err := parseTimeOrError(req.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTimeOrError(req.UpdatedAt)
	if err != nil {
		return nil, err
	}

	id := strings.TrimSpace(req.UUID)
	if id == "" {
		id = uuid.NewString()
	}

	path, err := cache.ExpandHomePath(strings.TrimSpace(req.FilePath))
	if err != nil {
		return nil, err
	}

	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}

	if strings.TrimSpace(req.UUID) != "" {
		existing, found, findErr := s.store.FindByUUID(id)
		if findErr != nil {
			return nil, findErr
		}
		if found && strings.TrimSpace(req.CreatedAt) == "" {
			createdAt = existing.CreatedAt
		}
	}

	return &models.SoftwareConfig{
		UUID:      id,
		Name:      strings.TrimSpace(req.Name),
		FilePath:  path,
		Version:   strings.TrimSpace(req.Version),
		InUse:     true,
		Format:    format,
		Content:   req.Content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func validateFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))
	if format == "yml" {
		format = models.ConfigFormatYAML
	}
	if format != models.ConfigFormatJSON && format != models.ConfigFormatYAML {
		return "", errors.New("format must be json or yaml")
	}
	return format, nil
}

func validateContentByFormat(format string, content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("content is required")
	}

	switch format {
	case models.ConfigFormatJSON:
		if !json.Valid([]byte(content)) {
			return errors.New("content is not valid json")
		}
	case models.ConfigFormatYAML:
		var out interface{}
		if err := yaml.Unmarshal([]byte(content), &out); err != nil {
			return errors.New("content is not valid yaml")
		}
	}
	return nil
}

func parseTimeOrError(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, errors.New("time fields must be RFC3339")
	}
	return t, nil
}

func writeConfigFile(filePath string, content string) error {
	if filePath == "" {
		return errors.New("file_path is required")
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}
