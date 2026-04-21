package config

import (
	"os"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	AppPort   string
	RedisAddr string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		AppPort:   getEnv("APP_PORT", "8080"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
