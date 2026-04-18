package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"image-dedup/internal/application/auth"
	"image-dedup/internal/application/imaging"
	"image-dedup/internal/infrastructure/config"
	"image-dedup/internal/infrastructure/database"
	"image-dedup/internal/infrastructure/geocoder"
	"image-dedup/internal/interfaces/handler"
	"image-dedup/internal/interfaces/middleware"
)

// init is invoked before main()
func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	fmt.Printf("Image Dedup - API Server\n")
	fmt.Printf("========================\n\n")

	// Initialize database
	fmt.Println("Connecting to PostgreSQL database...")
	db, err := database.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	fmt.Println("Database connected successfully!")

	// Initialize offline geocoder
	fmt.Println("Initializing offline geocoder...")
	geoc := geocoder.NewGeocoder()
	if geoc != nil {
		fmt.Println("Geocoder initialized successfully!")
	} else {
		fmt.Println("Geocoder unavailable, geolocation will be disabled.")
	}

	// Create scan manager (reads gallery folders from DB dynamically)
	scanManager := imaging.NewScanManager(db, cfg.ScanWorkers)

	// Create metadata manager (background EXIF extraction)
	metadataManager := imaging.NewMetadataManager(db, geoc, cfg.MetadataWorkers, cfg.MetadataIntervalMin)
	defer metadataManager.Stop()

	// Wire scan complete callback to trigger metadata extraction
	scanManager.OnScanComplete = func() {
		if err := metadataManager.StartExtraction(); err != nil {
			log.Printf("Metadata extraction not started: %v", err)
		}
	}

	// Initialize authentication components
	sessionConfig := &auth.SessionConfig{
		IdleTimeout:     time.Duration(cfg.SessionIdleHours) * time.Hour,
		AbsoluteTimeout: time.Duration(cfg.SessionAbsoluteDays) * 24 * time.Hour,
		CookieMaxAge:    cfg.SessionIdleHours * 60 * 60,
		TokenLength:     64,
	}

	sessionRepo := auth.NewSessionRepository(db, sessionConfig)
	bootstrap := auth.NewBootstrapService(db, cfg.BootstrapLogin, cfg.BootstrapPassword)
	loginLimiter := auth.NewLoginRateLimiter(10, 15*time.Minute, 30*time.Minute)
	authService := auth.NewAuthService(db, bootstrap, sessionRepo, loginLimiter)
	userService := auth.NewUserService(db, sessionRepo)
	authMiddleware := middleware.NewAuthMiddleware(sessionRepo, authService)
	csrfProtection := middleware.NewCSRFProtection()
	authHandlers := handler.NewAuthHandlers(authService, bootstrap, userService, sessionRepo, db)

	// Start session cleanup job
	sessionCleanup := auth.NewSessionCleanupJob(sessionRepo, 1*time.Hour)
	sessionCleanup.Start()
	defer sessionCleanup.Stop()

	fmt.Println("Authentication system initialized!")

	// Start web server
	server := handler.NewServer(db, scanManager, metadataManager, cfg)
	router := server.SetupRouter(authMiddleware, csrfProtection, authHandlers)

	fmt.Printf("\nStarting API server on http://%s:%s\n", cfg.ServerHost, cfg.ServerPort)
	fmt.Printf("Scan workers: %d\n", cfg.ScanWorkers)
	fmt.Printf("Metadata workers: %d, interval: %d min\n", cfg.MetadataWorkers, cfg.MetadataIntervalMin)
	fmt.Printf("CORS allowed origins: %s\n", strings.Join(cfg.CORSOrigins, ", "))
	fmt.Println("Configure gallery folders via the web UI Settings tab.")
	fmt.Println("Press Ctrl+C to stop the server")

	if err := router.Run(fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
