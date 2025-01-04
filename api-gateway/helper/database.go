package helper

import (
	"api-gateway/config"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() error {
	cfg, err := config.SetConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		return err
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.Database.DBHost, cfg.Database.DBUser, cfg.Database.DBPassword, cfg.Database.DBName, cfg.Database.DBPort)
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
		return err
	}

	// Periksa apakah db nil
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return nil
}
