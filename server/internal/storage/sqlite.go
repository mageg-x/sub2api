package storage

import (
	"os"
	"path/filepath"

	"sub2api/server/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return nil, err
	}
	if err := db.Exec("PRAGMA foreign_keys=ON;").Error; err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.APIKey{},
		&model.Account{},
		&model.ModelPrice{},
		&model.PaymentOrder{},
		&model.Coupon{},
		&model.Announcement{},
		&model.ErrorLog{},
		&model.SystemMetric{},
		&model.UsageLog{},
		&model.OAuthSession{},
	); err != nil {
		return nil, err
	}
	return db, nil
}
