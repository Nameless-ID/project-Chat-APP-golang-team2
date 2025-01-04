package helper

import (
	"api-gateway/config"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	cfg, err := config.SetConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		return err
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.Database.DBHost, cfg.Database.DBUser, cfg.Database.DBPassword, cfg.Database.DBName, cfg.Database.DBPort)
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{}) // Inisialisasi variabel global DB
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
		return err
	}

	return nil
}
