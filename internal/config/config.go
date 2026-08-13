package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP     HTTPConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Redis    RedisConfig
	Security SecurityConfig
}
type DatabaseConfig struct{ URL string }
type StorageConfig struct {
	Endpoint, AccessKey, SecretKey, Bucket string
	Secure                                 bool
}
type RedisConfig struct{ Address string }
type HTTPConfig struct {
	Address                                string
	AllowedOrigin                          string
	ReadTimeout, WriteTimeout, IdleTimeout time.Duration
}
type SecurityConfig struct {
	AdminAPIKey       string
	MaxIdentifyBytes  int64
	IdentifyPerMinute int
}

func Load() (Config, error) {
	c := Config{HTTP: HTTPConfig{Address: value("LYRA_HTTP_ADDRESS", ":8080"), AllowedOrigin: value("LYRA_ALLOWED_ORIGIN", "http://localhost:5173"), ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}, Database: DatabaseConfig{URL: os.Getenv("DATABASE_URL")}, Storage: StorageConfig{Endpoint: os.Getenv("S3_ENDPOINT"), AccessKey: os.Getenv("S3_ACCESS_KEY"), SecretKey: os.Getenv("S3_SECRET_KEY"), Bucket: value("S3_BUCKET", "lyra-reference"), Secure: os.Getenv("S3_SECURE") == "true"}, Redis: RedisConfig{Address: value("REDIS_ADDR", "localhost:6379")}, Security: SecurityConfig{AdminAPIKey: os.Getenv("LYRA_ADMIN_API_KEY"), MaxIdentifyBytes: 10 << 20, IdentifyPerMinute: 30}}
	if raw := os.Getenv("LYRA_MAX_IDENTIFY_BYTES"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c, fmt.Errorf("LYRA_MAX_IDENTIFY_BYTES: %w", err)
		}
		c.Security.MaxIdentifyBytes = n
	}
	if c.Security.MaxIdentifyBytes <= 0 {
		return c, fmt.Errorf("LYRA_MAX_IDENTIFY_BYTES must be positive")
	}
	if c.Database.URL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if c.Security.AdminAPIKey == "" {
		return c, fmt.Errorf("LYRA_ADMIN_API_KEY is required")
	}
	return c, nil
}
func value(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
