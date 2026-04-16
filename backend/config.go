package main

import (
	"os"
	"runtime"
	"strconv"
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

	ScanWorkers         int
	MetadataWorkers     int
	MetadataIntervalMin int

	// Auth configuration
	BootstrapLogin      string
	BootstrapPassword   string
	SessionIdleHours    int
	SessionAbsoluteDays int
}

// LoadConfig reads configuration from environment variables
func LoadConfig() *AppConfig {
	originsStr := getEnv("CORS_ORIGINS", "http://localhost:5173")
	origins := strings.Split(originsStr, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	scanWorkers := runtime.NumCPU()
	if v := getEnv("SCAN_WORKERS", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			scanWorkers = n
		}
	}

	metadataWorkers := runtime.NumCPU() / 2
	if metadataWorkers < 1 {
		metadataWorkers = 1
	}
	if v := getEnv("METADATA_WORKERS", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			metadataWorkers = n
		}
	}

	metadataInterval := 30
	if v := getEnv("METADATA_INTERVAL_MINUTES", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			metadataInterval = n
		}
	}

	return &AppConfig{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", "postgres"),
		DBName:              getEnv("DB_NAME", "image_dedup"),
		ServerHost:          getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:          getEnv("SERVER_PORT", "5170"),
		CORSOrigins:         origins,
		ScanWorkers:         scanWorkers,
		MetadataWorkers:     metadataWorkers,
		MetadataIntervalMin: metadataInterval,
		BootstrapLogin:      getEnv("BOOTSTRAP_LOGIN", "admin"),
		BootstrapPassword:   getEnv("BOOTSTRAP_PASSWORD", "admin"),
		SessionIdleHours:    getEnvInt("SESSION_IDLE_HOURS", 720),   // 30 days
		SessionAbsoluteDays: getEnvInt("SESSION_ABSOLUTE_DAYS", 90), // 90 days
	}
}

// getEnvInt gets environment variable as int with a default value
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return defaultValue
}

// getEnv gets environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
