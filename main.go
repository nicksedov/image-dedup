package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// Parse command line arguments
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()

	// Get directories from remaining arguments
	dirs := flag.Args()
	if len(dirs) == 0 {
		fmt.Println("Usage: image-dedup [options] <directory1> [directory2] ...")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nEnvironment variables:")
		fmt.Println("  DB_HOST     PostgreSQL host (default: localhost)")
		fmt.Println("  DB_PORT     PostgreSQL port (default: 5432)")
		fmt.Println("  DB_USER     PostgreSQL user (default: postgres)")
		fmt.Println("  DB_PASSWORD PostgreSQL password (default: postgres)")
		fmt.Println("  DB_NAME     Database name (default: image_dedup)")
		os.Exit(1)
	}

	// Validate directories
	var validDirs []string
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			log.Printf("Warning: Cannot access directory %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			log.Printf("Warning: %s is not a directory", dir)
			continue
		}
		validDirs = append(validDirs, dir)
	}

	if len(validDirs) == 0 {
		log.Fatal("No valid directories provided")
	}

	fmt.Printf("Image Dedup - Duplicate Image Manager\n")
	fmt.Printf("======================================\n\n")

	// Initialize database
	fmt.Println("Connecting to PostgreSQL database...")
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	fmt.Println("Database connected successfully!")

	// Initial scan
	fmt.Printf("\nScanning directories: %s\n", strings.Join(validDirs, ", "))
	progressChan := make(chan string, 100)

	go func() {
		for msg := range progressChan {
			fmt.Printf("  %s\n", msg)
		}
	}()

	// Cleanup missing files first
	cleanupMissingFiles(db, progressChan)

	// Scan all directories
	for _, dir := range validDirs {
		fmt.Printf("\nScanning: %s\n", dir)
		if err := scanDirectory(db, dir, progressChan); err != nil {
			log.Printf("Error scanning %s: %v", dir, err)
		}
	}
	close(progressChan)

	// Find duplicates and show summary
	groups, _ := findDuplicates(db)
	fmt.Printf("\n======================================\n")
	fmt.Printf("Scan complete! Found %d duplicate groups.\n", len(groups))

	// Start web server
	server := NewServer(db, validDirs)
	router := server.SetupRouter()

	fmt.Printf("\nStarting web server on http://localhost:%d\n", *port)
	fmt.Println("Press Ctrl+C to stop the server")

	if err := router.Run(fmt.Sprintf(":%d", *port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
