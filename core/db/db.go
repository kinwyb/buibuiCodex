package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type IStorage interface {
	GetDB() *gorm.DB
	Close() error
}

type defaultStorage struct {
	db *gorm.DB
}

func (s *defaultStorage) GetDB() *gorm.DB {
	return s.db
}

// Close 关闭存储连接
func (s *defaultStorage) Close() error {
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

type Data struct {
	storage IStorage
	session ISession
	cron    ITask
}

func NewData(storage IStorage) *Data {
	ret := &Data{
		storage: storage,
	}
	if storage != nil && storage.GetDB() != nil {
		db := storage.GetDB()
		ret.session = newSessionStorage(db)
		ret.cron = newTaskStorage(db)
		ret.session.Init()
		ret.cron.Init()
	}
	return ret
}

func (d *Data) Session() ISession {
	if d.storage == nil {
		d.session = &sessionDefautStorage{}
	}
	return d.session
}

func (d *Data) Cron() ITask {
	if d.cron == nil {
		d.cron = &taskDefaultStorage{}
	}
	return d.cron
}

func (d *Data) Close() {
	if d.storage != nil {
		d.storage.Close()
	}
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

	return &defaultStorage{db: db}, nil
}

// NewMysqlStorage 创建 Mysql 存储实例
// dsn: 数据库连接
func NewMysqlStorage(username string, passwrod string, server string, dbname string) (IStorage, error) {
	// DSN格式: 用户名:密码@tcp(地址:端口)/数据库名?参数
	// parseTime=True: 解析时间类型; loc=Local: 使用本地时区; charset=utf8mb4: 支持emoji等
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, passwrod, server, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	// 获取底层 *sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns 设置打开数据库连接的最大数量
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime 设置了连接可复用的最大时间
	sqlDB.SetConnMaxLifetime(time.Hour)
	// 验证连接是否成功
	if err = sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return &defaultStorage{db: db}, nil
}
