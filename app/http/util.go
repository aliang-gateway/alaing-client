package http

import (
	"io/ioutil"
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
	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, []byte(port[1:]), 0644) // 去掉冒号，只写 56431
}
