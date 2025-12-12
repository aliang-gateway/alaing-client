package user

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptionKey 生成或获取加密密钥
// 为了确保稳定性，使用固定的密钥（长度32字节用于AES-256）
// 实际应用中，可以从环境变量或安全存储中获取
var encryptionKey = []byte("nursorgate-secret-key-2025-token!")[:32]

// EncryptField 加密单个字段
func EncryptField(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	ciphertext, err := encryptBytes([]byte(plaintext))
	if err != nil {
		return "", err
	}

	// Base64编码以便存储为JSON字符串
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptField 解密单个字段
func DecryptField(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64解码
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	plaintext, err := decryptBytes(decoded)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// encryptBytes 使用AES-256-GCM加密字节数据
func encryptBytes(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptBytes 使用AES-256-GCM解密字节数据
func decryptBytes(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 提取nonce（前gcm.NonceSize()字节）
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}
