package main

import (
	"fmt"
	"log"
	"strings"
	"time"

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

	// Initialize offline geocoder
	fmt.Println("Initializing offline geocoder...")
	geocoder := NewGeocoder()
	if geocoder != nil {
		fmt.Println("Geocoder initialized successfully!")
	} else {
		fmt.Println("Geocoder unavailable, geolocation will be disabled.")
	}

	// Create scan manager (reads gallery folders from DB dynamically)
	scanManager := NewScanManager(db, config.ScanWorkers)

	// Create metadata manager (background EXIF extraction)
	metadataManager := NewMetadataManager(db, geocoder, config.MetadataWorkers, config.MetadataIntervalMin)
	defer metadataManager.Stop()

	// Wire scan complete callback to trigger metadata extraction
	scanManager.OnScanComplete = func() {
		if err := metadataManager.StartExtraction(); err != nil {
			log.Printf("Metadata extraction not started: %v", err)
		}
	}

	// Initialize authentication components
	sessionConfig := &SessionConfig{
		IdleTimeout:     time.Duration(config.SessionIdleHours) * time.Hour,
		AbsoluteTimeout: time.Duration(config.SessionAbsoluteDays) * 24 * time.Hour,
		CookieMaxAge:    config.SessionIdleHours * 60 * 60,
		TokenLength:     64,
	}

	sessionRepo := NewSessionRepository(db, sessionConfig)
	bootstrap := NewBootstrapService(db, config.BootstrapLogin, config.BootstrapPassword)
	loginLimiter := NewLoginRateLimiter(10, 15*time.Minute, 30*time.Minute)
	authService := NewAuthService(db, bootstrap, sessionRepo, loginLimiter)
	userService := NewUserService(db, sessionRepo)
	authMiddleware := NewAuthMiddleware(sessionRepo, authService)
	csrfProtection := NewCSRFProtection()
	authHandlers := NewAuthHandlers(authService, bootstrap, userService, sessionRepo, db)

	// Start session cleanup job
	sessionCleanup := NewSessionCleanupJob(sessionRepo, 1*time.Hour)
	sessionCleanup.Start()
	defer sessionCleanup.Stop()

	fmt.Println("Authentication system initialized!")

	// Start web server
	server := NewServer(db, scanManager, metadataManager, config)
	router := server.SetupRouter(authMiddleware, csrfProtection, authHandlers)

	fmt.Printf("\nStarting API server on http://%s:%s\n", config.ServerHost, config.ServerPort)
	fmt.Printf("Scan workers: %d\n", config.ScanWorkers)
	fmt.Printf("Metadata workers: %d, interval: %d min\n", config.MetadataWorkers, config.MetadataIntervalMin)
	fmt.Printf("CORS allowed origins: %s\n", strings.Join(config.CORSOrigins, ", "))
	fmt.Println("Configure gallery folders via the web UI Settings tab.")
	fmt.Println("Press Ctrl+C to stop the server")

	if err := router.Run(fmt.Sprintf("%s:%s", config.ServerHost, config.ServerPort)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
