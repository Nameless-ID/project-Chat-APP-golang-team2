package database

import (
	"auth-service/config"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectDB(cfg config.Config) (*gorm.DB, error) {
	// Configure the database logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: false,       // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color output
		},
	)

	db, err := gorm.Open(postgres.Open(makePostgresString(cfg)), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err = createCustomDBTypes(db); err != nil {
		return nil, err
	}

	// Call Migrate function to auto-migrate database schemas
	if cfg.DB.Migrate {
		err = Migrate(db)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return db, nil
}

// makePostgresString creates the PostgreSQL DSN (Data Source Name)
func makePostgresString(cfg config.Config) string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Name, cfg.DB.Password)
}

func createCustomDBTypes(db *gorm.DB) error {
	var err error

	return err
}
