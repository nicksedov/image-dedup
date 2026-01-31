package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ImageFile represents an image file in the database
type ImageFile struct {
	ID        uint      `gorm:"primaryKey"`
	Path      string    `gorm:"uniqueIndex;not null"`
	Size      int64     `gorm:"not null;index:idx_size_hash"`
	Hash      string    `gorm:"not null;index:idx_size_hash"`
	ModTime   time.Time `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DuplicateGroup represents a group of duplicate images
type DuplicateGroup struct {
	Hash  string
	Size  int64
	Files []ImageFile
}

// supportedExtensions contains all supported image file extensions
var supportedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
	".webp": true,
}

// isImageFile checks if a file is a supported image based on extension
func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

// calculateFileHash calculates MD5 hash of a file
func calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// scanDirectory scans a directory for image files and updates the database
func scanDirectory(db *gorm.DB, dirPath string, progressChan chan<- string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			progressChan <- fmt.Sprintf("Error accessing %s: %v", path, err)
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil
		}

		if !isImageFile(path) {
			return nil
		}

		// Normalize path separators to forward slashes for consistency
		normalizedPath := filepath.ToSlash(path)

		// Check if file already exists in database with same mod time
		var existing ImageFile
		result := db.Where("path = ?", normalizedPath).First(&existing)

		if result.Error == nil {
			// File exists in DB, check if it's been modified
			if existing.ModTime.Equal(info.ModTime()) && existing.Size == info.Size() {
				progressChan <- fmt.Sprintf("Skipping (cached): %s", path)
				return nil
			}
		}

		progressChan <- fmt.Sprintf("Processing: %s", path)

		// Calculate hash
		hash, err := calculateFileHash(path)
		if err != nil {
			progressChan <- fmt.Sprintf("Error hashing %s: %v", path, err)
			return nil
		}

		// Create or update record
		imageFile := ImageFile{
			Path:    normalizedPath,
			Size:    info.Size(),
			Hash:    hash,
			ModTime: info.ModTime(),
		}

		if result.Error == nil {
			// Update existing
			imageFile.ID = existing.ID
			db.Save(&imageFile)
		} else {
			// Create new
			db.Create(&imageFile)
		}

		return nil
	})
}

// findDuplicates finds all duplicate groups from the database
func findDuplicates(db *gorm.DB) ([]DuplicateGroup, error) {
	// Find hash+size combinations that appear more than once
	type HashSizeCount struct {
		Hash  string
		Size  int64
		Count int64
	}

	var duplicateHashSizes []HashSizeCount
	result := db.Model(&ImageFile{}).
		Select("hash, size, count(*) as count").
		Group("hash, size").
		Having("count(*) > 1").
		Scan(&duplicateHashSizes)

	if result.Error != nil {
		return nil, result.Error
	}

	var groups []DuplicateGroup
	for _, hs := range duplicateHashSizes {
		var files []ImageFile
		db.Where("hash = ? AND size = ?", hs.Hash, hs.Size).Find(&files)

		// Filter out files that no longer exist
		var existingFiles []ImageFile
		for _, f := range files {
			if _, err := os.Stat(f.Path); err == nil {
				existingFiles = append(existingFiles, f)
			} else {
				// Remove from database if file doesn't exist
				db.Delete(&f)
			}
		}

		if len(existingFiles) > 1 {
			groups = append(groups, DuplicateGroup{
				Hash:  hs.Hash,
				Size:  hs.Size,
				Files: existingFiles,
			})
		}
	}

	return groups, nil
}

// cleanupMissingFiles removes database entries for files that no longer exist
func cleanupMissingFiles(db *gorm.DB, progressChan chan<- string) error {
	var files []ImageFile
	db.Find(&files)

	for _, f := range files {
		if _, err := os.Stat(f.Path); os.IsNotExist(err) {
			progressChan <- fmt.Sprintf("Removing missing file from DB: %s", f.Path)
			db.Delete(&f)
		}
	}

	return nil
}
