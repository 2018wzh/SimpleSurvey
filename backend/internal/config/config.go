package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                   string
	ServerPort               string
	MongoURI                 string
	MongoDatabase            string
	JWTSecret                string
	AccessTokenExpiresHours  int
	RefreshTokenExpiresHours int
	RedisAddr                string
	RedisPassword            string
	RedisDB                  int
	RedisKeyPrefix           string
	RequestTimeoutSec        int
	AllowedOrigins           []string
	AdminBootstrapUsername   string
	AdminBootstrapPassword   string
}

func Load() Config {
	_ = godotenv.Load()
	accessHours := getEnvInt("ACCESS_TOKEN_EXPIRES_HOURS", getEnvInt("JWT_EXPIRES_HOURS", 24))

	cfg := Config{
		AppEnv:                   getEnv("APP_ENV", "development"),
		ServerPort:               getEnv("SERVER_PORT", "8080"),
		MongoURI:                 getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase:            getEnv("MONGO_DATABASE", "simple_survey"),
		JWTSecret:                getEnv("JWT_SECRET", "change-me-in-production"),
		AccessTokenExpiresHours:  accessHours,
		RefreshTokenExpiresHours: getEnvInt("REFRESH_TOKEN_EXPIRES_HOURS", 24*7),
		RedisAddr:                getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:            getEnv("REDIS_PASSWORD", ""),
		RedisDB:                  getEnvInt("REDIS_DB", 0),
		RedisKeyPrefix:           getEnv("REDIS_KEY_PREFIX", "simple_survey"),
		RequestTimeoutSec:        getEnvInt("REQUEST_TIMEOUT_SECONDS", 10),
		AllowedOrigins:           getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
		AdminBootstrapUsername:   getEnv("ADMIN_BOOTSTRAP_USERNAME", ""),
		AdminBootstrapPassword:   getEnv("ADMIN_BOOTSTRAP_PASSWORD", ""),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvSlice(key string, fallback []string) []string {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
