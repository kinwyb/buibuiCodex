package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type sqliteStorage struct {
	db *gorm.DB
}

func (s *sqliteStorage) GetDB() *gorm.DB {
	return s.db
}

// Close 关闭存储连接
func (s *sqliteStorage) Close() error {
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// NewSQLiteStorage 创建 SQLite 存储实例
// dbPath: 数据库文件路径，如 {workspace}/buibui.db
func NewSQLiteStorage(dbPath string) (IStorage, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// 连接 SQLite，启用 WAL 模式
	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	return &sqliteStorage{db: db}, nil
}
