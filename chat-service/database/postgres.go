package database

import (
	"chat-service/app/models"
	"chat-service/config"
	"fmt"
	"log"
	"os"

	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetDatabase(cfg config.Config) (*gorm.DB, error) {

	logger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			Colorful:                  true,
			IgnoreRecordNotFoundError: false,
			LogLevel:                  logger.Info,
		},
	)

	db, err := gorm.Open(postgres.Open(
		fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s timezone=%s",
			cfg.Database.DBHost, cfg.Database.DBPort, cfg.Database.DBUser, cfg.Database.DBName, cfg.Database.DBPassword, cfg.Database.DBTimezone)), &gorm.Config{
		Logger: logger,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if cfg.Migration {

		err := db.AutoMigrate(
			&models.Message{},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to make migration: " + err.Error())
		}
	}
	//
	// 	if cfg.Seeder {
	// 		if err := SeedAll(db); err != nil {
	// 			return nil, fmt.Errorf("failed to make seeder: " + err.Error())
	// 		}
	// 	}

	return db, nil
}
