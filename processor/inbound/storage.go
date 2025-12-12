package inbound

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/crypto"
)

// SaveInboundsCache saves inbound configurations to local encrypted storage
func SaveInboundsCache(inbounds []InboundInfo) error {
	if inbounds == nil {
		return fmt.Errorf("inbounds cannot be nil")
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(inbounds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal inbounds: %w", err)
	}

	// Encrypt JSON data using crypto.EncryptField
	encryptedData, err := crypto.EncryptField(string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to encrypt inbounds: %w", err)
	}

	// Get cache file path
	cachePath, err := GetInboundsCachePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write encrypted data to file with 0600 permissions
	if err := os.WriteFile(cachePath, []byte(encryptedData), 0600); err != nil {
		return fmt.Errorf("failed to write inbounds cache file: %w", err)
	}

	logger.Info(fmt.Sprintf("Saved %d inbounds to encrypted cache", len(inbounds)))
	return nil
}

// LoadInboundsCache loads inbound configurations from local encrypted storage
func LoadInboundsCache() ([]InboundInfo, error) {
	cachePath, err := GetInboundsCachePath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("inbounds cache file does not exist: %s", cachePath)
	}

	// Read encrypted data from file
	encryptedData, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inbounds cache file: %w", err)
	}

	// Decrypt data using crypto.DecryptField
	jsonData, err := crypto.DecryptField(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt inbounds: %w", err)
	}

	// Parse JSON to inbounds
	var inbounds []InboundInfo
	if err := json.Unmarshal([]byte(jsonData), &inbounds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inbounds: %w", err)
	}

	logger.Info(fmt.Sprintf("Loaded %d inbounds from encrypted cache", len(inbounds)))
	return inbounds, nil
}

// GetInboundsCachePath returns the path to the inbounds cache file
func GetInboundsCachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".nursorgate")
	return filepath.Join(cacheDir, "inbounds.cache"), nil
}

// ClearInboundsCache removes the inbounds cache file
func ClearInboundsCache() error {
	cachePath, err := GetInboundsCachePath()
	if err != nil {
		return err
	}

	// Delete file, ignore "not found" errors
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete inbounds cache file: %w", err)
	}

	logger.Info("Inbounds cache cleared")
	return nil
}
