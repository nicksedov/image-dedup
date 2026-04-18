package imaging

import (
	"fmt"
	"log"
	"sync"
	"time"

	"image-dedup/internal/domain"
	"image-dedup/internal/infrastructure/geocoder"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MetadataStatusResponse is the JSON response for GET /api/metadata-status
type MetadataStatusResponse struct {
	Processing     bool   `json:"processing"`
	Progress       string `json:"progress"`
	FilesProcessed int    `json:"filesProcessed"`
}

// MetadataManager manages background EXIF metadata extraction
type MetadataManager struct {
	mu             sync.RWMutex
	isProcessing   bool
	progress       string
	filesProcessed int
	db             *gorm.DB
	geocoder       *geocoder.Geocoder
	workers        int
	ticker         *time.Ticker
	stopChan       chan struct{}
}

// NewMetadataManager creates a new MetadataManager and starts the periodic extraction loop.
func NewMetadataManager(db *gorm.DB, geo *geocoder.Geocoder, workers int, intervalMinutes int) *MetadataManager {
	mm := &MetadataManager{
		db:       db,
		geocoder: geo,
		workers:  workers,
		stopChan: make(chan struct{}),
	}

	if intervalMinutes > 0 {
		mm.ticker = time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		go func() {
			for {
				select {
				case <-mm.ticker.C:
					if err := mm.StartExtraction(); err != nil {
						// Already processing, skip this tick
					}
				case <-mm.stopChan:
					return
				}
			}
		}()
	}

	return mm
}

// Stop shuts down the periodic extraction loop.
func (mm *MetadataManager) Stop() {
	if mm.ticker != nil {
		mm.ticker.Stop()
	}
	close(mm.stopChan)
}

// StartExtraction launches an asynchronous metadata extraction pass.
func (mm *MetadataManager) StartExtraction() error {
	mm.mu.Lock()
	if mm.isProcessing {
		mm.mu.Unlock()
		return fmt.Errorf("metadata extraction already in progress")
	}
	mm.isProcessing = true
	mm.progress = "Starting metadata extraction..."
	mm.filesProcessed = 0
	mm.mu.Unlock()

	go func() {
		mm.processUnextracted()

		mm.mu.Lock()
		mm.isProcessing = false
		mm.progress = "Metadata extraction complete"
		mm.mu.Unlock()
	}()

	return nil
}

// processUnextracted finds images without metadata and extracts it.
func (mm *MetadataManager) processUnextracted() {
	// Find images that have no metadata row or have stale metadata
	var images []domain.ImageFile
	mm.db.Raw(`
		SELECT image_files.* FROM image_files
		LEFT JOIN image_metadata ON image_metadata.image_file_id = image_files.id
		WHERE image_metadata.id IS NULL
		   OR image_metadata.updated_at < image_files.updated_at
		ORDER BY image_files.id
	`).Scan(&images)

	total := len(images)
	if total == 0 {
		mm.mu.Lock()
		mm.progress = "No images need metadata extraction"
		mm.mu.Unlock()
		return
	}

	mm.mu.Lock()
	mm.progress = fmt.Sprintf("Extracting metadata: 0/%d", total)
	mm.mu.Unlock()

	log.Printf("Metadata extraction: %d images to process", total)

	type metadataResult struct {
		imageFileID uint
		metadata    *domain.ImageMetadata
	}

	jobs := make(chan domain.ImageFile, total)
	results := make(chan metadataResult, total)

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < mm.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for img := range jobs {
				meta, err := extractMetadata(img.Path)
				if err != nil {
					log.Printf("Metadata extraction failed for %s: %v", img.Path, err)
					continue
				}
				meta.ImageFileID = img.ID

				// Reverse geocode if GPS data is present
				if meta.GPSLatitude != nil && meta.GPSLongitude != nil {
					country, city := mm.geocoder.ReverseGeocode(*meta.GPSLatitude, *meta.GPSLongitude)
					meta.GeoCountry = country
					meta.GeoCity = city
				}

				results <- metadataResult{imageFileID: img.ID, metadata: meta}
			}
		}()
	}

	// Send jobs
	go func() {
		for _, img := range images {
			jobs <- img
		}
		close(jobs)
	}()

	// Collect results in background
	go func() {
		wg.Wait()
		close(results)
	}()

	// Batch upsert results
	batch := make([]*domain.ImageMetadata, 0, 50)
	count := 0
	for r := range results {
		batch = append(batch, r.metadata)
		count++

		mm.mu.Lock()
		mm.filesProcessed = count
		mm.progress = fmt.Sprintf("Extracting metadata: %d/%d", count, total)
		mm.mu.Unlock()

		if len(batch) >= 50 {
			mm.upsertBatch(batch)
			batch = batch[:0]
		}
	}

	// Flush remaining
	if len(batch) > 0 {
		mm.upsertBatch(batch)
	}

	log.Printf("Metadata extraction complete: %d images processed", count)
}

// upsertBatch inserts or updates a batch of metadata records.
func (mm *MetadataManager) upsertBatch(batch []*domain.ImageMetadata) {
	for _, meta := range batch {
		mm.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "image_file_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"width", "height", "camera_model", "lens_model", "iso",
				"aperture", "shutter_speed", "focal_length", "date_taken",
				"orientation", "color_space", "software",
				"gps_latitude", "gps_longitude", "geo_country", "geo_city",
				"updated_at",
			}),
		}).Create(meta)
	}
}

// GetStatus returns the current metadata extraction status.
func (mm *MetadataManager) GetStatus() MetadataStatusResponse {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return MetadataStatusResponse{
		Processing:     mm.isProcessing,
		Progress:       mm.progress,
		FilesProcessed: mm.filesProcessed,
	}
}

// IsProcessing returns whether metadata extraction is currently running.
func (mm *MetadataManager) IsProcessing() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.isProcessing
}
