package imaging

import (
	"fmt"
	"sync"

	"image-dedup/internal/domain"

	"gorm.io/gorm"
)

// ScanStatusResponse is the JSON response for GET /api/status
type ScanStatusResponse struct {
	Scanning       bool   `json:"scanning"`
	Progress       string `json:"progress"`
	FilesProcessed int    `json:"filesProcessed"`
}

// ScanManager manages asynchronous directory scanning
type ScanManager struct {
	mu             sync.RWMutex
	isScanning     bool
	progress       string
	filesProcessed int
	db             *gorm.DB
	scanWorkers    int
	OnScanComplete func() // called after each scan finishes (if non-nil)
}

// NewScanManager creates a new ScanManager
func NewScanManager(db *gorm.DB, scanWorkers int) *ScanManager {
	return &ScanManager{
		db:          db,
		scanWorkers: scanWorkers,
	}
}

// getGalleryDirs reads current gallery folder paths from the database
func (sm *ScanManager) getGalleryDirs() []string {
	var folders []domain.GalleryFolder
	sm.db.Find(&folders)
	dirs := make([]string, len(folders))
	for i, f := range folders {
		dirs[i] = f.Path
	}
	return dirs
}

// StartScan launches an asynchronous scan of all gallery directories
func (sm *ScanManager) StartScan() error {
	sm.mu.Lock()
	if sm.isScanning {
		sm.mu.Unlock()
		return fmt.Errorf("scan already in progress")
	}
	sm.isScanning = true
	sm.progress = "Starting scan..."
	sm.filesProcessed = 0
	sm.mu.Unlock()

	go func() {
		progressChan := make(chan string, 200)

		go func() {
			count := 0
			for msg := range progressChan {
				count++
				sm.mu.Lock()
				sm.progress = msg
				sm.filesProcessed = count
				sm.mu.Unlock()
			}
		}()

		// Cleanup missing files first
		sm.mu.Lock()
		sm.progress = "Cleaning up missing files..."
		sm.mu.Unlock()
		cleanupMissingFiles(sm.db, progressChan)

		// Read gallery dirs from DB at scan time
		scanDirs := sm.getGalleryDirs()

		// Scan all directories
		for _, dir := range scanDirs {
			sm.mu.Lock()
			sm.progress = fmt.Sprintf("Scanning: %s", dir)
			sm.mu.Unlock()
			scanDirectory(sm.db, dir, progressChan, sm.scanWorkers)
		}

		close(progressChan)

		sm.mu.Lock()
		sm.isScanning = false
		sm.progress = "Scan complete"
		sm.mu.Unlock()

		if sm.OnScanComplete != nil {
			sm.OnScanComplete()
		}
	}()

	return nil
}

// ScanSingleDir launches an asynchronous scan of a single directory
func (sm *ScanManager) ScanSingleDir(dirPath string) error {
	sm.mu.Lock()
	if sm.isScanning {
		sm.mu.Unlock()
		return fmt.Errorf("scan already in progress")
	}
	sm.isScanning = true
	sm.progress = fmt.Sprintf("Scanning: %s", dirPath)
	sm.filesProcessed = 0
	sm.mu.Unlock()

	go func() {
		progressChan := make(chan string, 200)

		go func() {
			count := 0
			for msg := range progressChan {
				count++
				sm.mu.Lock()
				sm.progress = msg
				sm.filesProcessed = count
				sm.mu.Unlock()
			}
		}()

		scanDirectory(sm.db, dirPath, progressChan, sm.scanWorkers)

		close(progressChan)

		sm.mu.Lock()
		sm.isScanning = false
		sm.progress = "Scan complete"
		sm.mu.Unlock()

		if sm.OnScanComplete != nil {
			sm.OnScanComplete()
		}
	}()

	return nil
}

// GetStatus returns the current scan status
func (sm *ScanManager) GetStatus() ScanStatusResponse {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return ScanStatusResponse{
		Scanning:       sm.isScanning,
		Progress:       sm.progress,
		FilesProcessed: sm.filesProcessed,
	}
}

// IsScanning returns whether a scan is currently running
func (sm *ScanManager) IsScanning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isScanning
}
