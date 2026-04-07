package main

import (
	"os"
	"strings"
)

// AppConfig holds all application configuration
type AppConfig struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	ServerHost  string
	ServerPort  string
	CORSOrigins []string
}

// LoadConfig reads configuration from environment variables
func LoadConfig() *AppConfig {
	originsStr := getEnv("CORS_ORIGINS", "http://localhost:5173")
	origins := strings.Split(originsStr, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &AppConfig{
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", "postgres"),
		DBName:      getEnv("DB_NAME", "image_dedup"),
		ServerHost:  getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:  getEnv("SERVER_PORT", "5170"),
		CORSOrigins: origins,
	}
}

// getEnv gets environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
