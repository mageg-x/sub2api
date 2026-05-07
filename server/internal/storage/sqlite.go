package storage

import (
	"os"
	"path/filepath"

	"sub2api/server/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open 打开SQLite数据库连接并执行必要的初始化
// 参数:
//   - path: 数据库文件路径
//
// 返回:
//   - *gorm.DB: 数据库连接实例
//   - error: 初始化过程中的错误
func Open(path string) (*gorm.DB, error) {
	// 确保数据库文件所在的目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	// 打开SQLite数据库连接
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 设置WAL（Write-Ahead Logging）模式以提高并发性能
	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return nil, err
	}

	// 启用外键约束
	if err := db.Exec("PRAGMA foreign_keys=ON;").Error; err != nil {
		return nil, err
	}

	// 自动迁移数据库表结构
	// 根据model包中的结构体创建/更新表
	if err := db.AutoMigrate(
		&model.User{},         // 用户表
		&model.APIKey{},       // API密钥表
		&model.Account{},      // AI账号表
		&model.ModelPrice{},   // 模型价格表
		&model.PaymentOrder{}, // 支付订单表
		&model.Coupon{},       // 优惠券表
		&model.Announcement{}, // 公告表
		&model.ErrorLog{},     // 错误日志表
		&model.SystemMetric{}, // 系统指标表
		&model.UsageLog{},     // 使用日志表
		&model.OAuthSession{}, // OAuth会话表
	); err != nil {
		return nil, err
	}

	return db, nil
}
