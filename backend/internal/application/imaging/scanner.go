package imaging

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"image-dedup/internal/domain"

	"gorm.io/gorm"
)

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

// hashResult holds the result of a file hash computation
type hashResult struct {
	fi       fileInfo
	hash     string
	err      error
	existing *domain.ImageFile
}

// scanDirectory scans a directory for image files and updates the database.
// numWorkers controls the number of parallel goroutines used for file hashing.
func scanDirectory(db *gorm.DB, dirPath string, progressChan chan<- string, numWorkers int) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if numWorkers <= 0 {
		numWorkers = 1
	}

	// Phase 1: Collect all image files from the directory tree
	var allFiles []fileInfo
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			progressChan <- "Error accessing " + path + ": " + err.Error()
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !domain.IsImageFile(path) {
			return nil
		}
		allFiles = append(allFiles, fileInfo{
			path:           path,
			normalizedPath: filepath.ToSlash(path),
			size:           info.Size(),
			modTime:        info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return err
	}

	if len(allFiles) == 0 {
		return nil
	}

	// Phase 2: Batch query existing files from DB to build a cache map
	existingMap := make(map[string]domain.ImageFile)
	const dbBatchSize = 500
	for i := 0; i < len(allFiles); i += dbBatchSize {
		end := i + dbBatchSize
		if end > len(allFiles) {
			end = len(allFiles)
		}
		paths := make([]string, end-i)
		for j, fi := range allFiles[i:end] {
			paths[j] = fi.normalizedPath
		}
		var existingFiles []domain.ImageFile
		db.Where("path IN ?", paths).Find(&existingFiles)
		for _, ef := range existingFiles {
			existingMap[ef.Path] = ef
		}
	}

	// Phase 3: Separate cached (unchanged) files from files that need hashing
	var filesToHash []fileInfo
	for _, fi := range allFiles {
		if existing, ok := existingMap[fi.normalizedPath]; ok {
			if existing.ModTime.Equal(fi.modTime) && existing.Size == fi.size {
				progressChan <- "Skipping (cached): " + fi.path
				continue
			}
		}
		filesToHash = append(filesToHash, fi)
	}

	if len(filesToHash) == 0 {
		return nil
	}

	// Phase 4: Hash files in parallel using a worker pool
	jobs := make(chan fileInfo, numWorkers*2)
	results := make(chan hashResult, numWorkers*2)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fi := range jobs {
				hash, err := calculateFileHash(fi.path)
				var existing *domain.ImageFile
				if ef, ok := existingMap[fi.normalizedPath]; ok {
					existing = &ef
				}
				results <- hashResult{
					fi:       fi,
					hash:     hash,
					err:      err,
					existing: existing,
				}
			}
		}()
	}

	// Close results channel when all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Send jobs to workers
	go func() {
		for _, fi := range filesToHash {
			jobs <- fi
		}
		close(jobs)
	}()

	// Phase 5: Collect results and batch write to DB
	const writeBatchSize = 50
	var toCreate []domain.ImageFile
	var toUpdate []domain.ImageFile

	for result := range results {
		if result.err != nil {
			progressChan <- "Error hashing " + result.fi.path + ": " + result.err.Error()
			continue
		}

		progressChan <- "Processed: " + result.fi.path

		imageFile := domain.ImageFile{
			Path:    result.fi.normalizedPath,
			Size:    result.fi.size,
			Hash:    result.hash,
			ModTime: result.fi.modTime,
		}

		if result.existing != nil {
			imageFile.ID = result.existing.ID
			toUpdate = append(toUpdate, imageFile)
		} else {
			toCreate = append(toCreate, imageFile)
		}

		if len(toCreate)+len(toUpdate) >= writeBatchSize {
			flushDBBatch(db, &toCreate, &toUpdate)
		}
	}

	// Flush remaining
	flushDBBatch(db, &toCreate, &toUpdate)

	return nil
}

// flushDBBatch writes accumulated create/update records to the database and resets the slices
func flushDBBatch(db *gorm.DB, toCreate *[]domain.ImageFile, toUpdate *[]domain.ImageFile) {
	if len(*toCreate) > 0 {
		db.Create(toCreate)
		*toCreate = (*toCreate)[:0]
	}
	for _, f := range *toUpdate {
		db.Save(&f)
	}
	*toUpdate = (*toUpdate)[:0]
}

// findDuplicates finds all duplicate groups from the database
func findDuplicates(db *gorm.DB) ([]domain.DuplicateGroup, error) {
	type HashSizeCount struct {
		Hash  string
		Size  int64
		Count int64
	}

	var duplicateHashSizes []HashSizeCount
	result := db.Model(&domain.ImageFile{}).
		Select("hash, size, count(*) as count").
		Group("hash, size").
		Having("count(*) > 1").
		Scan(&duplicateHashSizes)

	if result.Error != nil {
		return nil, result.Error
	}

	var groups []domain.DuplicateGroup
	for _, hs := range duplicateHashSizes {
		var files []domain.ImageFile
		db.Where("hash = ? AND size = ?", hs.Hash, hs.Size).Find(&files)

		var existingFiles []domain.ImageFile
		for _, f := range files {
			if _, err := os.Stat(f.Path); err == nil {
				existingFiles = append(existingFiles, f)
			} else {
				db.Delete(&f)
			}
		}

		if len(existingFiles) > 1 {
			groups = append(groups, domain.DuplicateGroup{
				Hash:  hs.Hash,
				Size:  hs.Size,
				Files: existingFiles,
			})
		}
	}

	return groups, nil
}

// FindDuplicatesPaginated finds duplicate groups with pagination
func FindDuplicatesPaginated(db *gorm.DB, offset, limit int) ([]domain.DuplicateGroup, int, int, error) {
	type HashSizeCount struct {
		Hash  string
		Size  int64
		Count int64
	}

	var allDuplicateHashSizes []HashSizeCount
	result := db.Model(&domain.ImageFile{}).
		Select("hash, size, count(*) as count").
		Group("hash, size").
		Having("count(*) > 1").
		Order("size DESC").
		Scan(&allDuplicateHashSizes)

	if result.Error != nil {
		return nil, 0, 0, result.Error
	}

	totalGroups := len(allDuplicateHashSizes)

	totalFiles := 0
	for _, hs := range allDuplicateHashSizes {
		totalFiles += int(hs.Count)
	}

	if offset >= len(allDuplicateHashSizes) {
		return []domain.DuplicateGroup{}, totalGroups, totalFiles, nil
	}

	end := offset + limit
	if end > len(allDuplicateHashSizes) {
		end = len(allDuplicateHashSizes)
	}

	paginatedHashSizes := allDuplicateHashSizes[offset:end]

	var groups []domain.DuplicateGroup
	for _, hs := range paginatedHashSizes {
		var files []domain.ImageFile
		db.Where("hash = ? AND size = ?", hs.Hash, hs.Size).Find(&files)

		if len(files) > 1 {
			groups = append(groups, domain.DuplicateGroup{
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
	var files []domain.ImageFile
	db.Find(&files)

	for _, f := range files {
		if _, err := os.Stat(f.Path); os.IsNotExist(err) {
			progressChan <- fmt.Sprintf("Removing missing file from DB: %s", f.Path)
			db.Delete(&f)
		}
	}

	return nil
}
