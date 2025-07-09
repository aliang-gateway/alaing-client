package utils

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// KVStore 定义存储结构
type KVStore struct {
	db     *sql.DB
	dbPath string
}

var kvStore *KVStore

// NewKVStore 创建新的存储实例
func NewKVStore() *KVStore {
	if kvStore == nil {
		kvStore = &KVStore{}
	}
	return kvStore
}

// SetDBPath 设置数据库路径并初始化
func (k *KVStore) SetDBPath(path string) error {
	k.dbPath = path

	// 打开数据库连接
	db, err := sql.Open("sqlite3", k.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// 创建表
	// createTableSQL := `
	// CREATE TABLE IF NOT EXISTS ItemTable (
	//     key TEXT PRIMARY KEY,
	//     value TEXT NOT NULL
	// );`

	// db.Exec(createTableSQL)
	// if err != nil {
	// 	return fmt.Errorf("failed to create table: %v", err)
	// }

	k.db = db
	return nil
}

// Set 插入或更新键值对
func (k *KVStore) Set(key, value any) error {
	query := `INSERT OR REPLACE INTO ItemTable (key, value) VALUES (?, ?)`
	_, err := k.db.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set value: %v", err)
	}
	return nil
}

// Read 读取键对应的值
func (k *KVStore) Read(key string) (string, error) {
	var value string
	query := `SELECT value FROM ItemTable WHERE key = ?`
	err := k.db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", err
		}
		return "", fmt.Errorf("failed to read value: %v", err)
	}
	return value, nil
}

// Update 更新已有键的值
func (k *KVStore) Update(key, value string) error {
	query := `UPDATE ItemTable SET value = ? WHERE key = ?`
	result, err := k.db.Exec(query, value, key)
	if err != nil {
		return fmt.Errorf("failed to update value: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	return nil
}

// Delete 删除键值对
func (k *KVStore) Delete(key string) error {
	query := `DELETE FROM ItemTable WHERE key = ?`
	result, err := k.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	return nil
}

// Close 关闭数据库连接
func (k *KVStore) Close() error {
	if k.db != nil {
		return k.db.Close()
	}
	return nil
}
