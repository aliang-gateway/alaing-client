package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/common/cache"
)

var (
	softwareConfigDBOnce sync.Once
	softwareConfigDB     *gorm.DB
	softwareConfigDBErr  error
)

const legacySoftwareConfigDBFile = "software_configs.db"

type SoftwareConfigStore struct {
	db      *gorm.DB
	initErr error
}

func NewSoftwareConfigStore() *SoftwareConfigStore {
	db, err := getSoftwareConfigDB()
	return &SoftwareConfigStore{db: db, initErr: err}
}

func NewSoftwareConfigStoreWithDBPath(dbPath string) (*SoftwareConfigStore, error) {
	db, err := openSoftwareConfigDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &SoftwareConfigStore{db: db}, nil
}

func getSoftwareConfigDB() (*gorm.DB, error) {
	softwareConfigDBOnce.Do(func() {
		dbPath, err := cache.GetUnifiedDataDBPath()
		if err != nil {
			softwareConfigDBErr = err
			return
		}
		softwareConfigDB, softwareConfigDBErr = openUnifiedSoftwareConfigDB(dbPath)
	})
	return softwareConfigDB, softwareConfigDBErr
}

// InitializeSoftwareConfigStore ensures the shared software-config tables are ready.
func InitializeSoftwareConfigStore() error {
	_, err := getSoftwareConfigDB()
	return err
}

// ResetSoftwareConfigDBForTest clears the package singleton so tests can isolate db path resolution.
func ResetSoftwareConfigDBForTest() {
	softwareConfigDB = nil
	softwareConfigDBErr = nil
	softwareConfigDBOnce = sync.Once{}
	cache.ResetCacheDirForTest()
}

func openSoftwareConfigDB(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		return nil, errors.New("software config db path is empty")
	}

	absPath, err := cache.ExpandHomePath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve db path: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(absPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.AutoMigrate(&models.SoftwareConfig{}, &models.SoftwareConfigOperationLog{}, &models.SoftwareEffectiveConfigSnapshot{}); err != nil {
		return nil, fmt.Errorf("failed to migrate software_configs table: %w", err)
	}

	if err := db.Exec("UPDATE software_configs SET software = 'opencode' WHERE software IS NULL OR software = ''").Error; err != nil {
		return nil, fmt.Errorf("failed to backfill software column: %w", err)
	}

	return db, nil
}

func openUnifiedSoftwareConfigDB(dbPath string) (*gorm.DB, error) {
	db, err := openSoftwareConfigDB(dbPath)
	if err != nil {
		return nil, err
	}
	if err := migrateLegacySoftwareConfigDB(db, dbPath); err != nil {
		return nil, err
	}
	return db, nil
}

func migrateLegacySoftwareConfigDB(db *gorm.DB, sharedDBPath string) error {
	legacyPath, err := cache.GetCacheFile(legacySoftwareConfigDBFile)
	if err != nil {
		return err
	}

	sharedAbs, err := cache.ExpandHomePath(sharedDBPath)
	if err != nil {
		return fmt.Errorf("failed to resolve shared db path: %w", err)
	}
	legacyAbs, err := cache.ExpandHomePath(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to resolve legacy db path: %w", err)
	}
	if filepath.Clean(sharedAbs) == filepath.Clean(legacyAbs) {
		return nil
	}
	if _, err := os.Stat(legacyAbs); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to inspect legacy software config db: %w", err)
	}

	var configCount int64
	var logCount int64
	var snapshotCount int64
	if err := db.Model(&models.SoftwareConfig{}).Count(&configCount).Error; err != nil {
		return err
	}
	if err := db.Model(&models.SoftwareConfigOperationLog{}).Count(&logCount).Error; err != nil {
		return err
	}
	if err := db.Model(&models.SoftwareEffectiveConfigSnapshot{}).Count(&snapshotCount).Error; err != nil {
		return err
	}
	if configCount > 0 || logCount > 0 || snapshotCount > 0 {
		return nil
	}

	legacyDB, err := gorm.Open(sqlite.Open(legacyAbs), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open legacy software config db: %w", err)
	}

	var configs []models.SoftwareConfig
	if err := legacyDB.Order("updated_at ASC").Find(&configs).Error; err != nil {
		if !isMissingSQLiteTableError(err) {
			return fmt.Errorf("failed to read legacy software configs: %w", err)
		}
	}

	var logs []models.SoftwareConfigOperationLog
	if err := legacyDB.Order("id ASC").Find(&logs).Error; err != nil {
		if !isMissingSQLiteTableError(err) {
			return fmt.Errorf("failed to read legacy software config logs: %w", err)
		}
	}

	var snapshots []models.SoftwareEffectiveConfigSnapshot
	if err := legacyDB.Order("id ASC").Find(&snapshots).Error; err != nil {
		if !isMissingSQLiteTableError(err) {
			return fmt.Errorf("failed to read legacy software config snapshots: %w", err)
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, cfg := range configs {
			if err := tx.Save(&cfg).Error; err != nil {
				return err
			}
		}
		for _, log := range logs {
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}
		for _, snapshot := range snapshots {
			if err := tx.Create(&snapshot).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func isMissingSQLiteTableError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such table")
}

func (s *SoftwareConfigStore) ensureReady() error {
	if s == nil {
		return errors.New("software config store is nil")
	}
	if s.initErr != nil {
		return s.initErr
	}
	if s.db == nil {
		return errors.New("software config store db is nil")
	}
	return nil
}

func (s *SoftwareConfigStore) Upsert(cfg models.SoftwareConfig) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.db.Save(&cfg).Error
}

func (s *SoftwareConfigStore) Activate(cfg models.SoftwareConfig) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if cfg.Software == "" {
		return errors.New("software is required")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.SoftwareConfig{}).Where("software = ? AND in_use = ?", cfg.Software, true).Update("in_use", false).Error; err != nil {
			return err
		}
		cfg.InUse = true
		return tx.Save(&cfg).Error
	})
}

func (s *SoftwareConfigStore) List() ([]models.SoftwareConfig, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	var configs []models.SoftwareConfig
	if err := s.db.Order("updated_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *SoftwareConfigStore) ListBySoftware(software string) ([]models.SoftwareConfig, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	var configs []models.SoftwareConfig
	if err := s.db.Where("software = ?", software).Order("updated_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *SoftwareConfigStore) ListSelectedBySoftware(software string) ([]models.SoftwareConfig, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	var configs []models.SoftwareConfig
	query := s.db.Where("selected = ?", true)
	if software != "" {
		query = query.Where("software = ?", software)
	}
	if err := query.Order("updated_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *SoftwareConfigStore) ListByUUIDs(ids []string) ([]models.SoftwareConfig, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []models.SoftwareConfig{}, nil
	}

	var configs []models.SoftwareConfig
	if err := s.db.Where("uuid IN ?", ids).Order("updated_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *SoftwareConfigStore) GetByUUID(id string) (*models.SoftwareConfig, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	var cfg models.SoftwareConfig
	if err := s.db.First(&cfg, "uuid = ?", id).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *SoftwareConfigStore) FindByUUID(id string) (*models.SoftwareConfig, bool, error) {
	if err := s.ensureReady(); err != nil {
		return nil, false, err
	}

	var cfg models.SoftwareConfig
	err := s.db.First(&cfg, "uuid = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &cfg, true, nil
}

func (s *SoftwareConfigStore) DeleteByUUID(id string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.db.Delete(&models.SoftwareConfig{}, "uuid = ?", id).Error
}

func (s *SoftwareConfigStore) SetSelected(id string, selected bool) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.db.Model(&models.SoftwareConfig{}).Where("uuid = ?", id).Update("selected", selected).Error
}

func (s *SoftwareConfigStore) SaveOperationLog(log models.SoftwareConfigOperationLog) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.db.Create(&log).Error
}

func (s *SoftwareConfigStore) SaveEffectiveConfigSnapshot(snapshot models.SoftwareEffectiveConfigSnapshot) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.db.Create(&snapshot).Error
}

func (s *SoftwareConfigStore) GetLatestEffectiveConfigSnapshot() (*models.SoftwareEffectiveConfigSnapshot, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	var snapshot models.SoftwareEffectiveConfigSnapshot
	if err := s.db.Order("created_at DESC, id DESC").First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *SoftwareConfigStore) GetLatestEffectiveConfigSnapshotBySoftwareAndName(software string, configName string) (*models.SoftwareEffectiveConfigSnapshot, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	var snapshot models.SoftwareEffectiveConfigSnapshot
	if err := s.db.
		Where("software = ? AND config_name = ?", software, configName).
		Order("created_at DESC, id DESC").
		First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *SoftwareConfigStore) MergeByLatest(incoming []models.SoftwareConfig) (inserted int, updated int, keptLocalNewer int, err error) {
	if err = s.ensureReady(); err != nil {
		return 0, 0, 0, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, remoteCfg := range incoming {
			var localCfg models.SoftwareConfig
			err := tx.First(&localCfg, "uuid = ?", remoteCfg.UUID).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if remoteCfg.InUse {
					if clearErr := tx.Model(&models.SoftwareConfig{}).Where("software = ? AND in_use = ?", remoteCfg.Software, true).Update("in_use", false).Error; clearErr != nil {
						return clearErr
					}
				}
				if createErr := tx.Create(&remoteCfg).Error; createErr != nil {
					return createErr
				}
				inserted++
				continue
			}
			if err != nil {
				return err
			}

			if remoteCfg.UpdatedAt.After(localCfg.UpdatedAt) {
				if remoteCfg.InUse {
					if clearErr := tx.Model(&models.SoftwareConfig{}).Where("software = ? AND in_use = ?", remoteCfg.Software, true).Update("in_use", false).Error; clearErr != nil {
						return clearErr
					}
				}
				if saveErr := tx.Save(&remoteCfg).Error; saveErr != nil {
					return saveErr
				}
				updated++
			} else {
				keptLocalNewer++
			}
		}
		return nil
	})

	return inserted, updated, keptLocalNewer, err
}
