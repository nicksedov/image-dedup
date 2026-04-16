package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// initDatabase initializes the database connection and runs migrations
func initDatabase(config *AppConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(&ImageFile{}, &GalleryFolder{}, &AppSettings{}, &ImageMetadata{}, &User{}, &Session{}, &AuditLog{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Seed default settings row if not exists
	var count int64
	db.Model(&AppSettings{}).Count(&count)
	if count == 0 {
		db.Create(&AppSettings{ID: 1, Theme: "light", Language: "en"})
	}

	return db, nil
}
