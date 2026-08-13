package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP          HTTPConfig
	Database      DatabaseConfig
	Storage       StorageConfig
	Redis         RedisConfig
	Security      SecurityConfig
	Observability ObservabilityConfig
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
	AdminUsername     string
	AdminPasswordHash string
	AdminCookieSecure bool
	MaxIdentifyBytes  int64
	IdentifyPerMinute int
}
type ObservabilityConfig struct {
	LogLevel  string
	LogFormat string
}

func Load() (Config, error) {
	c := Config{HTTP: HTTPConfig{Address: value("LYRA_HTTP_ADDRESS", ":8080"), AllowedOrigin: value("LYRA_ALLOWED_ORIGIN", "http://localhost:5173"), ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}, Database: DatabaseConfig{URL: os.Getenv("DATABASE_URL")}, Storage: StorageConfig{Endpoint: os.Getenv("S3_ENDPOINT"), AccessKey: os.Getenv("S3_ACCESS_KEY"), SecretKey: os.Getenv("S3_SECRET_KEY"), Bucket: value("S3_BUCKET", "lyra-reference"), Secure: os.Getenv("S3_SECURE") == "true"}, Redis: RedisConfig{Address: value("REDIS_ADDR", "localhost:6379")}, Security: SecurityConfig{AdminUsername: value("LYRA_ADMIN_USERNAME", "admin"), AdminPasswordHash: os.Getenv("LYRA_ADMIN_PASSWORD_HASH"), AdminCookieSecure: os.Getenv("LYRA_ADMIN_COOKIE_SECURE") == "true", MaxIdentifyBytes: 10 << 20, IdentifyPerMinute: 30}, Observability: ObservabilityConfig{LogLevel: value("LYRA_LOG_LEVEL", "info"), LogFormat: value("LYRA_LOG_FORMAT", "text")}}
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
	if c.Security.AdminPasswordHash == "" {
		return c, fmt.Errorf("LYRA_ADMIN_PASSWORD_HASH is required")
	}
	if !oneOf(c.Observability.LogLevel, "debug", "info", "warn", "warning", "error") {
		return c, fmt.Errorf("LYRA_LOG_LEVEL must be debug, info, warn, or error")
	}
	if !oneOf(c.Observability.LogFormat, "text", "json") {
		return c, fmt.Errorf("LYRA_LOG_FORMAT must be text or json")
	}
	return c, nil
}
func value(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
