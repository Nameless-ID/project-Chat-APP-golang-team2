package config

import (
	"flag"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	AppDebug        bool
	DB              DatabaseConfig
	Email           EmailConfig
	RedisConfig     RedisConfig
	ServerPort      string
	ShutdownTimeout int

	PrivateKey string
	PublicKey  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string

	Migrate bool
	Seeding bool
}

type EmailConfig struct {
	ApiKey    string
	FromName  string
	FromEmail string
}

type RedisConfig struct {
	Url      string
	Password string
	Prefix   string
}

func LoadConfig() (Config, error) {
	// Set default values
	setDefaultValues()

	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AddConfigPath("../..")
	viper.SetConfigType("dotenv")
	viper.SetConfigName(".env")

	// Allow Viper to read environment variables
	viper.AutomaticEnv()

	// Read the configuration file
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Error reading config file: %s, using default values or environment variables", err)
	}

	// readFlags()

	// add value to the config
	config := Config{
		DB:    loadDatabaseConfig(),
		Email: loadEmailConfig(),

		AppDebug:        viper.GetBool("APP_DEBUG"),
		ServerPort:      viper.GetString("SERVER_PORT"),
		ShutdownTimeout: viper.GetInt("SHUTDOWN_TIMEOUT"),
		PrivateKey:      viper.GetString("PRIVATE_KEY"),
		PublicKey:       viper.GetString("PUBLIC_KEY"),

		RedisConfig: loadRedisConfig(),
	}
	return config, nil
}

func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:     viper.GetString("DB_HOST"),
		Port:     viper.GetString("DB_PORT"),
		User:     viper.GetString("DB_USER"),
		Password: viper.GetString("DB_PASSWORD"),
		Name:     viper.GetString("DB_NAME"),
		Migrate:  viper.GetBool("DB_MIGRATE"),
		Seeding:  viper.GetBool("DB_SEEDING"),
	}
}

func loadEmailConfig() EmailConfig {
	return EmailConfig{
		ApiKey:    viper.GetString("MAILERSEND_API_KEY"),
		FromName:  viper.GetString("MAILERSEND_FROM_NAME"),
		FromEmail: viper.GetString("MAILERSEND_FROM_EMAIL"),
	}
}

func loadRedisConfig() RedisConfig {
	return RedisConfig{
		Url:      viper.GetString("REDIS_URL"),
		Password: viper.GetString("REDIS_PASSWORD"),
		Prefix:   viper.GetString("REDIS_PREFIX"),
	}
}

func setDefaultValues() {
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "admin")
	viper.SetDefault("DB_NAME", "database")
	viper.SetDefault("APP_DEBUG", true)
	viper.SetDefault("APP_SECRET", "team-2")
	viper.SetDefault("SERVER_PORT", ":8080")
	viper.SetDefault("SHUTDOWN_TIMEOUT", 5)

	viper.SetDefault("DB_MIGRATE", false)
	viper.SetDefault("DB_SEEDING", false)
}

func readFlags() {
	migrateDb := flag.Bool("m", false, "use this flag to migrate database")
	seedDb := flag.Bool("s", false, "use this flag to seed database")
	flag.Parse()
	if *migrateDb {
		viper.Set("DB_MIGRATE", true)
	}

	if *seedDb {
		viper.Set("DB_SEEDING", true)
	}
}
