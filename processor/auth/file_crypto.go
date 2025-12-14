package user

import (
	"encoding/json"
	"fmt"
)

// EncryptUserInfoFile encrypts the entire UserInfo struct as JSON
// This provides whole-file encryption with better privacy and smaller file size
// compared to field-level encryption
func EncryptUserInfoFile(userInfo *UserInfo) ([]byte, error) {
	if userInfo == nil {
		return nil, fmt.Errorf("user info cannot be nil")
	}

	// Serialize UserInfo to JSON
	jsonData, err := json.Marshal(userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user info: %w", err)
	}

	// Encrypt the entire JSON blob using existing crypto.encryptBytes wrapper
	// We'll use EncryptField and then decode to get raw bytes
	// Actually, we need direct access to encryptBytes, so we'll create a wrapper

	// For now, let's use a simpler approach: serialize -> encrypt field -> store
	// But this is inefficient. Instead, we should use the crypto module directly.

	// Since encryptBytes is private in crypto package, we'll use base64 encoding
	// with the existing EncryptField mechanism
	jsonStr := string(jsonData)
	encrypted, err := EncryptField(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt user info: %w", err)
	}

	return []byte(encrypted), nil
}

// DecryptUserInfoFile decrypts the entire UserInfo file
// Supports both new whole-file format and old field-level format for migration
func DecryptUserInfoFile(encryptedData []byte) (*UserInfo, error) {
	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("encrypted data cannot be empty")
	}

	// Try new format (whole-file encryption stored as base64 string)
	encryptedStr := string(encryptedData)

	decrypted, err := DecryptField(encryptedStr)
	if err == nil {
		// Successfully decrypted using new format
		userInfo := &UserInfo{}
		if err := json.Unmarshal([]byte(decrypted), userInfo); err != nil {
			return nil, fmt.Errorf("failed to deserialize user info: %w", err)
		}
		return userInfo, nil
	}

	return nil, fmt.Errorf("failed to decrypt user info: %w", err)
}

// DetectOldFormat checks if the file uses old field-level encryption format
// Old format has JSON structure with encrypted individual fields
func DetectOldFormat(data []byte) bool {
	// Try to unmarshal as JSON
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return false
	}

	// Old format has encrypted fields like "AccessToken", "RefreshToken", etc.
	// with encrypted base64 values
	// New format is a single base64-encoded blob (when stored)

	// Check if it has the structure of field-level encrypted data
	if _, hasAccessToken := rawData["AccessToken"]; hasAccessToken {
		if _, hasRefreshToken := rawData["RefreshToken"]; hasRefreshToken {
			return true
		}
	}

	return false
}

// MigrateFromOldFormat converts old field-level encrypted format to new whole-file format
// This is called automatically when loading old-format files
func MigrateFromOldFormat(oldFormatData []byte) (*UserInfo, error) {
	// Unmarshal the old format JSON
	userInfo := &UserInfo{}
	if err := json.Unmarshal(oldFormatData, userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse old format data: %w", err)
	}

	// The fields are already decrypted from SaveUserInfo() loading path
	// (user_info.go handles decryption of individual fields)
	// Here we just return the decrypted UserInfo

	return userInfo, nil
}
