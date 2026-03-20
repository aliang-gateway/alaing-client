package models

import "time"

// ConfigGetRequest is the request to get specific config
type ConfigGetRequest struct {
	Name string `json:"name"`
}

// ConfigInfo represents configuration information
type ConfigInfo struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

const (
	ConfigFormatJSON = "json"
	ConfigFormatYAML = "yaml"
)

type SoftwareConfig struct {
	UUID      string    `json:"uuid" gorm:"type:varchar(64);primaryKey"`
	Software  string    `json:"software" gorm:"type:varchar(128);not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null;index"`
	FilePath  string    `json:"file_path" gorm:"type:text;not null"`
	Version   string    `json:"version" gorm:"type:varchar(128)"`
	InUse     bool      `json:"in_use" gorm:"not null;default:false"`
	Format    string    `json:"format" gorm:"type:varchar(16);not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime;index"`
}

func (SoftwareConfig) TableName() string {
	return "software_configs"
}

type SaveSoftwareConfigRequest struct {
	UUID      string `json:"uuid"`
	Software  string `json:"software"`
	Name      string `json:"name"`
	FilePath  string `json:"file_path"`
	Version   string `json:"version"`
	InUse     bool   `json:"in_use"`
	Format    string `json:"format"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ActivateSoftwareConfigRequest struct {
	UUID      string `json:"uuid"`
	Software  string `json:"software"`
	Name      string `json:"name"`
	FilePath  string `json:"file_path"`
	Version   string `json:"version"`
	Format    string `json:"format"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type CloudPushRequest struct {
	CloudURL  string `json:"cloud_url"`
	AuthToken string `json:"auth_token,omitempty"`
}

type CloudPullRequest struct {
	CloudURL  string `json:"cloud_url"`
	AuthToken string `json:"auth_token,omitempty"`
}

type CloudConfigBatch struct {
	Configs []SoftwareConfig `json:"configs"`
}

type CloudPushResponse struct {
	PushedCount int `json:"pushed_count"`
}

type CloudPullResponse struct {
	PulledCount       int `json:"pulled_count"`
	InsertedCount     int `json:"inserted_count"`
	UpdatedFromCloud  int `json:"updated_from_cloud"`
	KeptLocalNewerCnt int `json:"kept_local_newer"`
}

type CloudSyncResponse struct {
	SyncedCount  int    `json:"synced_count"`
	LastSyncedAt string `json:"last_synced_at"`
}
