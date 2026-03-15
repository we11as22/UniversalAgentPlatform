package config

import "os"

type ServiceConfig struct {
	ServiceName string
	HTTPPort    string
	DatabaseURL string
	RedisURL    string
}

func Load(serviceName string, defaultPort string) ServiceConfig {
	return ServiceConfig{
		ServiceName: serviceName,
		HTTPPort:    envOrDefault("HTTP_PORT", defaultPort),
		DatabaseURL: envOrDefault("DATABASE_URL", "postgres://uap:uap@localhost:5432/uap?sslmode=disable"),
		RedisURL:    envOrDefault("REDIS_URL", "redis://localhost:6379/0"),
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

