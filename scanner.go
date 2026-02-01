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

// fileInfo holds file information collected during directory walk
type fileInfo struct {
	path           string
	normalizedPath string
	size           int64
	modTime        time.Time
}

// progressBuffer accumulates progress messages for batch output
type progressBuffer struct {
	messages []string
	limit    int
	channel  chan<- string
}

func newProgressBuffer(ch chan<- string, limit int) *progressBuffer {
	return &progressBuffer{
		messages: make([]string, 0, limit),
		limit:    limit,
		channel:  ch,
	}
}

func (pb *progressBuffer) add(msg string) {
	pb.messages = append(pb.messages, msg)
	if len(pb.messages) >= pb.limit {
		pb.flush()
	}
}

func (pb *progressBuffer) flush() {
	if len(pb.messages) == 0 {
		return
	}
	// Join all messages with newline and send as single message
	var sb strings.Builder
	for i, msg := range pb.messages {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(msg)
	}
	pb.channel <- sb.String()
	pb.messages = pb.messages[:0]
}

// scanDirectory scans a directory for image files and updates the database
func scanDirectory(db *gorm.DB, dirPath string, progressChan chan<- string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	const batchSize = 50
	const progressBufferSize = 100
	var batch []fileInfo
	progress := newProgressBuffer(progressChan, progressBufferSize)

	// Process a batch of files
	processBatch := func(batch []fileInfo) {
		if len(batch) == 0 {
			return
		}

		// Collect all normalized paths for batch query
		paths := make([]string, len(batch))
		pathToInfo := make(map[string]fileInfo)
		for i, fi := range batch {
			paths[i] = fi.normalizedPath
			pathToInfo[fi.normalizedPath] = fi
		}

		// Batch query: get all existing files in one query
		var existingFiles []ImageFile
		db.Where("path IN ?", paths).Find(&existingFiles)

		// Create map of existing files by path
		existingMap := make(map[string]ImageFile)
		for _, ef := range existingFiles {
			existingMap[ef.Path] = ef
		}

		// Process each file in batch
		var toCreate []ImageFile
		var toUpdate []ImageFile

		for _, fi := range batch {
			existing, exists := existingMap[fi.normalizedPath]

			if exists {
				// File exists in DB, check if it's been modified
				if existing.ModTime.Equal(fi.modTime) && existing.Size == fi.size {
					progress.add("Skipping (cached): " + fi.path)
					continue
				}
			}

			progress.add("Processing: " + fi.path)

			// Calculate hash
			hash, err := calculateFileHash(fi.path)
			if err != nil {
				progress.add("Error hashing " + fi.path + ": " + err.Error())
				continue
			}

			imageFile := ImageFile{
				Path:    fi.normalizedPath,
				Size:    fi.size,
				Hash:    hash,
				ModTime: fi.modTime,
			}

			if exists {
				imageFile.ID = existing.ID
				toUpdate = append(toUpdate, imageFile)
			} else {
				toCreate = append(toCreate, imageFile)
			}
		}

		// Batch create new files
		if len(toCreate) > 0 {
			db.Create(&toCreate)
		}

		// Update existing files (need to do one by one for proper ID handling)
		for _, f := range toUpdate {
			db.Save(&f)
		}

		// Flush progress after each batch
		progress.flush()
	}

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			progress.add("Error accessing " + path + ": " + err.Error())
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

		batch = append(batch, fileInfo{
			path:           path,
			normalizedPath: normalizedPath,
			size:           info.Size(),
			modTime:        info.ModTime(),
		})

		// Process batch when it reaches batchSize
		if len(batch) >= batchSize {
			processBatch(batch)
			batch = batch[:0] // Reset batch
		}

		return nil
	})

	// Process remaining files in the last batch
	if len(batch) > 0 {
		processBatch(batch)
	}

	// Final flush of any remaining progress messages
	progress.flush()

	return err
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

// countDuplicateGroups returns the total number of duplicate groups
func countDuplicateGroups(db *gorm.DB) (int, error) {
	var count int64
	result := db.Model(&ImageFile{}).
		Select("count(distinct hash || '-' || cast(size as text))").
		Where("hash IN (SELECT hash FROM image_files GROUP BY hash, size HAVING count(*) > 1)").
		Count(&count)

	if result.Error != nil {
		// Fallback to a simpler count
		type HashSizeCount struct {
			Hash  string
			Size  int64
			Count int64
		}
		var duplicates []HashSizeCount
		result = db.Model(&ImageFile{}).
			Select("hash, size, count(*) as count").
			Group("hash, size").
			Having("count(*) > 1").
			Scan(&duplicates)
		if result.Error != nil {
			return 0, result.Error
		}
		return len(duplicates), nil
	}

	return int(count), nil
}

// findDuplicatesPaginated finds duplicate groups with pagination (no file existence check)
// Returns: groups for current page, total groups count, total files count, error
func findDuplicatesPaginated(db *gorm.DB, offset, limit int) ([]DuplicateGroup, int, int, error) {
	// Find hash+size combinations that appear more than once
	type HashSizeCount struct {
		Hash  string
		Size  int64
		Count int64
	}

	// Get all duplicate hash+size combinations
	var allDuplicateHashSizes []HashSizeCount
	result := db.Model(&ImageFile{}).
		Select("hash, size, count(*) as count").
		Group("hash, size").
		Having("count(*) > 1").
		Order("size DESC").
		Scan(&allDuplicateHashSizes)

	if result.Error != nil {
		return nil, 0, 0, result.Error
	}

	totalGroups := len(allDuplicateHashSizes)

	// Calculate total files across all groups
	totalFiles := 0
	for _, hs := range allDuplicateHashSizes {
		totalFiles += int(hs.Count)
	}

	// Apply pagination to hash+size list
	if offset >= len(allDuplicateHashSizes) {
		return []DuplicateGroup{}, totalGroups, totalFiles, nil
	}

	end := offset + limit
	if end > len(allDuplicateHashSizes) {
		end = len(allDuplicateHashSizes)
	}

	paginatedHashSizes := allDuplicateHashSizes[offset:end]

	// Fetch files only for the paginated groups
	var groups []DuplicateGroup
	for _, hs := range paginatedHashSizes {
		var files []ImageFile
		db.Where("hash = ? AND size = ?", hs.Hash, hs.Size).Find(&files)

		if len(files) > 1 {
			groups = append(groups, DuplicateGroup{
				Hash:  hs.Hash,
				Size:  hs.Size,
				Files: files,
			})
		}
	}

	return groups, totalGroups, totalFiles, nil
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
