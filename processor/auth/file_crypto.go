package user

import (
	"encoding/json"
	"fmt"
)

// EncryptUserInfoFile serializes and encrypts the legacy whole-file auth payload.
func EncryptUserInfoFile(userInfo *legacyStoredUserInfo) ([]byte, error) {
	if userInfo == nil {
		return nil, fmt.Errorf("user info cannot be nil")
	}

	jsonData, err := json.Marshal(userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user info: %w", err)
	}

	encrypted, err := EncryptField(string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt user info: %w", err)
	}

	return []byte(encrypted), nil
}

// DecryptUserInfoFile decrypts the legacy whole-file auth payload.
func DecryptUserInfoFile(encryptedData []byte) (*legacyStoredUserInfo, error) {
	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("encrypted data cannot be empty")
	}

	decrypted, err := DecryptField(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt user info: %w", err)
	}

	userInfo := &legacyStoredUserInfo{}
	if err := json.Unmarshal([]byte(decrypted), userInfo); err != nil {
		return nil, fmt.Errorf("failed to deserialize user info: %w", err)
	}

	return userInfo, nil
}

// DetectOldFormat reports whether a file looks like the old field-based JSON format.
func DetectOldFormat(data []byte) bool {
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return false
	}

	_, hasAccessToken := rawData["AccessToken"]
	_, hasRefreshToken := rawData["RefreshToken"]
	return hasAccessToken && hasRefreshToken
}

// MigrateFromOldFormat parses the older JSON payload used before whole-file encryption.
func MigrateFromOldFormat(oldFormatData []byte) (*legacyStoredUserInfo, error) {
	userInfo := &legacyStoredUserInfo{}
	if err := json.Unmarshal(oldFormatData, userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse old format data: %w", err)
	}

	return userInfo, nil
}
