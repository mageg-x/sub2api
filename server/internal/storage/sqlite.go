package storage

import (
	"os"
	"path/filepath"

	"sub2api/server/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open 打开SQLite数据库连接并初始化
// 参数：
//   - path: 数据库文件路径
//
// 返回：数据库连接和错误
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

	// 设置SQLite性能优化PRAGMA
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",   // WAL模式：提高并发读写性能
		"PRAGMA foreign_keys=ON;",    // 启用外键约束
		"PRAGMA busy_timeout=5000;",  // 忙等待超时5秒：避免写冲突立即报错
		"PRAGMA synchronous=NORMAL;", // 同步模式NORMAL：WAL下安全且比FULL快
		"PRAGMA cache_size=-8000;",   // 页面缓存8MB：减少磁盘IO
		"PRAGMA temp_store=MEMORY;",  // 临时表存内存：避免临时文件IO
	}
	for _, p := range pragmas {
		if err := db.Exec(p).Error; err != nil {
			return nil, err
		}
	}

	// 自动迁移数据库表结构
	if err := db.AutoMigrate(
		&model.User{},                  // 用户表
		&model.APIKey{},                // API密钥表
		&model.Account{},               // AI账号表
		&model.ModelPrice{},            // 模型价格表
		&model.PaymentOrder{},          // 支付订单表
		&model.Coupon{},                // 优惠券表
		&model.Announcement{},          // 公告表
		&model.ErrorLog{},              // 错误日志表
		&model.SystemMetric{},          // 系统指标表
		&model.UsageLog{},              // 使用日志表
		&model.UserUsageMinute{},       // 用户分钟级用量汇总表
		&model.UserUsageHour{},         // 用户小时级用量汇总表
		&model.UserUsageDay{},          // 用户天级用量汇总表
		&model.UserUsageDayDimension{}, // 用户天级维度用量汇总表
		&model.OAuthSession{},          // OAuth会话表
	); err != nil {
		return nil, err
	}

	// 配置连接池：SQLite单文件写入，限制连接数避免锁竞争
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(2)    // 最大打开连接数2：SQLite写锁互斥，多连接无益
	sqlDB.SetMaxIdleConns(2)    // 最大空闲连接数2：保持连接复用
	sqlDB.SetConnMaxLifetime(0) // 连接不过期：SQLite本地文件无需回收

	return db, nil
}
