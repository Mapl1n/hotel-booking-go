package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	ServerPort string
	AppEnv     string

	// Database
	DBDriver   string // mysql | sqlite
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPath     string // SQLite file path

	// Redis (optional for local dev)
	RedisEnabled  bool
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// JWT
	JWTSecretKey   string
	JWTExpireHours int

	// SMS
	SMSProvider  string
	SMSMockCode  string

	// Payment
	PaymentMode string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		AppEnv:         getEnv("APP_ENV", "development"),
		DBDriver:       getEnv("DB_DRIVER", "mysql"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "hotel_db"),
		DBPath:         getEnv("DB_PATH", "hotel.db"),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getEnvInt("REDIS_DB", 0),
		JWTSecretKey:   getEnv("JWT_SECRET_KEY", "default-secret-change-me"),
		JWTExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 24),
		SMSProvider:    getEnv("SMS_PROVIDER", "mock"),
		SMSMockCode:    getEnv("SMS_MOCK_CODE", "888888"),
		PaymentMode:    getEnv("PAYMENT_MODE", "mock"),
	}

	// Redis is enabled if a host is explicitly configured
	cfg.RedisEnabled = os.Getenv("REDIS_HOST") != ""

	return cfg
}

func (c *Config) DSN() string {
	if c.DBDriver == "sqlite" {
		return c.DBPath
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
