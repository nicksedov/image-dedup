package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"image-toolkit/internal/application/auth"
	"image-toolkit/internal/application/imaging"
	"image-toolkit/internal/infrastructure/config"
	"image-toolkit/internal/infrastructure/database"
	"image-toolkit/internal/infrastructure/geocoder"
	"image-toolkit/internal/infrastructure/ocr"
	"image-toolkit/internal/interfaces/handler"
	"image-toolkit/internal/interfaces/middleware"
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

	// Initialize OCR classifier client and health check
	var ocrCheckInterval int
	if cfg.OCREnabled {
		fmt.Println("Initializing OCR classifier client...")
		ocrCheckInterval = cfg.OCRCheckInterval
		fmt.Printf("OCR classifier enabled: host=%s, port=%s, check interval=%ds\n", cfg.OCRHost, cfg.OCRPort, ocrCheckInterval)
	} else {
		fmt.Println("OCR classifier integration disabled")
		ocrCheckInterval = 0
	}

	// Create scan manager (reads gallery folders from DB dynamically)
	scanManager := imaging.NewScanManager(db, cfg.ScanWorkers)

	// Create metadata manager (background EXIF extraction)
	metadataManager := imaging.NewMetadataManager(db, geoc, cfg.MetadataWorkers, cfg.MetadataIntervalMin)
	defer metadataManager.Stop()

	// Create OCR manager (background classification)
	var ocrManager *imaging.OcrManager
	if cfg.OCREnabled {
		ocrClient := ocr.NewClient(cfg.OCRHost, cfg.OCRPort)
		ocrManager = imaging.NewOcrManager(db, ocrClient, cfg.ScanWorkers)
		fmt.Printf("OCR manager initialized: workers=%d\n", cfg.ScanWorkers)
	}

	// Wire scan complete callback to trigger metadata extraction and OCR classification
	scanManager.OnScanComplete = func() {
		if err := metadataManager.StartExtraction(); err != nil {
			log.Printf("Metadata extraction not started: %v", err)
		}
		if cfg.OCREnabled && ocrManager != nil {
			if err := ocrManager.StartClassification(); err != nil {
				log.Printf("OCR classification not started: %v", err)
			}
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

	// Create LLM OCR service
	llmOcrService := imaging.NewLlmOcrService(db)
	fmt.Println("LLM OCR service initialized")

	// Start web server
	server := handler.NewServer(db, scanManager, metadataManager, ocrManager, llmOcrService, cfg)
	router := server.SetupRouter(authMiddleware, csrfProtection, authHandlers)

	// Start OCR health check if enabled
	server.StartOCRHealthCheck()
	defer server.StopOCRHealthCheck()

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
