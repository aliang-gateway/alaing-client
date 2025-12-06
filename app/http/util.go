package http

import (
	"os"
	"path/filepath"
)

// WritePortToFile 写入端口到文件
func WritePortToFile(port string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	filePath := filepath.Join(homeDir, ".cursor", "nursor")
	err = os.MkdirAll(filepath.Dir(filePath), 0700)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(port[1:]), 0600) // 去掉冒号，只写 56431
}
