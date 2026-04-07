package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"
)

// init is invoked before main()
func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	// Load configuration
	config := LoadConfig()

	fmt.Printf("Image Dedup - API Server\n")
	fmt.Printf("========================\n\n")

	// Initialize database
	fmt.Println("Connecting to PostgreSQL database...")
	db, err := initDatabase(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	fmt.Println("Database connected successfully!")

	// Create scan manager (reads gallery folders from DB dynamically)
	scanManager := NewScanManager(db)

	// Start web server
	server := NewServer(db, scanManager, config)
	router := server.SetupRouter()

	fmt.Printf("\nStarting API server on http://%s:%s\n", config.ServerHost, config.ServerPort)
	fmt.Printf("CORS allowed origins: %s\n", strings.Join(config.CORSOrigins, ", "))
	fmt.Println("Configure gallery folders via the web UI Settings tab.")
	fmt.Println("Press Ctrl+C to stop the server")

	if err := router.Run(fmt.Sprintf("%s:%s", config.ServerHost, config.ServerPort)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
