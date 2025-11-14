package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	App      AppConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type CacheConfig struct {
	TemplateTTL time.Duration
	SessionTTL  time.Duration
	PostTTL     time.Duration
}

type AppConfig struct {
	BasePath    string
	Environment string
	IsProduction bool
}

// LoadConfig loads configuration from environment variables with fallbacks
func LoadConfig() *Config {
	env := getEnv("ENV", "development")
	isProd := env == "production"
	
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("PORT", 8080),
			ReadTimeout:  getEnvDuration("READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Path:            getEnv("DB_PATH", "server/database/database.db"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Cache: CacheConfig{
			TemplateTTL: getEnvDuration("CACHE_TEMPLATE_TTL", 1*time.Hour),
			SessionTTL:  getEnvDuration("CACHE_SESSION_TTL", 10*time.Minute),
			PostTTL:     getEnvDuration("CACHE_POST_TTL", 5*time.Minute),
		},
		App: AppConfig{
			BasePath:     getEnv("BASE_PATH", ""),
			Environment:  env,
			IsProduction: isProd,
		},
	}
	
	return cfg
}

// Helper functions to get environment variables with fallbacks

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}
