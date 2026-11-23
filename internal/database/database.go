// Package database: koneksi PostgreSQL, migrasi, dan seeder.
package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"app/internal/config"
	"app/internal/features/auth"
	"app/internal/features/post"
)

// Connect membuka koneksi PostgreSQL.
func Connect(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
}

// Migrate membuat/menyesuaikan tabel users dan posts.
func Migrate(db *gorm.DB) error { return db.AutoMigrate(&auth.User{}, &post.Post{}) }

// Seed membuat akun admin/member bawaan dan contoh artikel (idempoten).
func Seed(db *gorm.DB, cfg config.Config) error {
	svc := auth.NewService(auth.NewRepository(db), cfg.AppKey)
	if err := svc.EnsureUser(cfg.AdminUsername, cfg.AdminUsername+"@delcom.local", cfg.AdminPassword, "admin"); err != nil {
		return err
	}
	if err := svc.EnsureUser("member", "member@delcom.local", "member123", "member"); err != nil {
		return err
	}
	author, err := auth.NewRepository(db).FindByLogin("member")
	if err != nil {
		return err
	}
	return post.NewService(post.NewRepository(db)).SeedSamples(author)
}
