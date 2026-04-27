package imaging

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"image-toolkit/internal/domain"
	"image-toolkit/internal/infrastructure/ocr"

	"gorm.io/gorm"
)

// OcrManager handles background OCR classification of images
type OcrManager struct {
	mu             sync.RWMutex
	isProcessing   bool
	progress       string
	filesProcessed int
	totalFiles     int
	db             *gorm.DB
	ocrClient      ocr.Client
	workers        int
}

// OcrStatusResponse represents the OCR processing status
type OcrStatusResponse struct {
	Processing     bool   `json:"processing"`
	Progress       string `json:"progress"`
	FilesProcessed int    `json:"filesProcessed"`
	TotalFiles     int    `json:"totalFiles"`
}

// NewOcrManager creates a new OCR manager
func NewOcrManager(db *gorm.DB, ocrClient ocr.Client, workers int) *OcrManager {
	return &OcrManager{
		db:        db,
		ocrClient: ocrClient,
		workers:   workers,
	}
}

// StartClassification starts the OCR classification process in background
func (om *OcrManager) StartClassification() error {
	om.mu.Lock()
	if om.isProcessing {
		om.mu.Unlock()
		return fmt.Errorf("OCR classification already in progress")
	}
	om.isProcessing = true
	om.progress = "Starting OCR classification..."
	om.filesProcessed = 0
	om.totalFiles = 0
	om.mu.Unlock()

	go om.processUnclassified()

	return nil
}

// GetStatus returns the current OCR processing status
func (om *OcrManager) GetStatus() OcrStatusResponse {
	om.mu.RLock()
	defer om.mu.RUnlock()

	return OcrStatusResponse{
		Processing:     om.isProcessing,
		Progress:       om.progress,
		FilesProcessed: om.filesProcessed,
		TotalFiles:     om.totalFiles,
	}
}

// IsProcessing returns true if OCR classification is currently running
func (om *OcrManager) IsProcessing() bool {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.isProcessing
}

// processUnclassified finds images without OCR classification and processes them
func (om *OcrManager) processUnclassified() {
	defer func() {
		om.mu.Lock()
		om.isProcessing = false
		om.progress = "OCR classification complete"
		om.mu.Unlock()
	}()

	// Find images without classification or with stale classification
	var images []domain.ImageFile
	query := om.db.Table("image_files").
		Select("image_files.*").
		Joins("LEFT JOIN ocr_classifications ON ocr_classifications.image_file_id = image_files.id").
		Where("ocr_classifications.id IS NULL OR ocr_classifications.updated_at < image_files.updated_at").
		Order("image_files.id")

	if err := query.Find(&images).Error; err != nil {
		log.Printf("OCR: failed to query unclassified images: %v", err)
		return
	}

	om.mu.Lock()
	om.totalFiles = len(images)
	om.mu.Unlock()

	if len(images) == 0 {
		om.mu.Lock()
		om.progress = "No unclassified images found"
		om.isProcessing = false
		om.mu.Unlock()
		return
	}

	om.mu.Lock()
	om.progress = fmt.Sprintf("Found %d images to classify", len(images))
	om.mu.Unlock()

	// Process images using worker pool
	type ocrResult struct {
		image          domain.ImageFile
		classification *domain.OcrClassification
		boxes          []domain.OcrBoundingBox
		err            error
	}

	jobs := make(chan domain.ImageFile, len(images))
	results := make(chan ocrResult, len(images))

	var wg sync.WaitGroup
	for w := 0; w < om.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for img := range jobs {
				result := ocrResult{image: img}

				// Open image file
				file, err := os.Open(img.Path)
				if err != nil {
					result.err = fmt.Errorf("failed to open file: %w", err)
					results <- result
					continue
				}

				// Determine content type based on extension
				contentType := "image/jpeg"
				if len(img.Path) > 4 {
					ext := img.Path[len(img.Path)-4:]
					if ext == ".png" {
						contentType = "image/png"
					}
				}

				// Call OCR API
				ctx := context.Background()
				ocrResp, err := om.ocrClient.Classify(ctx, file, contentType, ocr.DefaultClassifyParams())
				file.Close()

				if err != nil {
					result.err = fmt.Errorf("OCR classification failed: %w", err)
					results <- result
					continue
				}

				// Create classification record
				classification := &domain.OcrClassification{
					ImageFileID:        img.ID,
					IsTextDocument:     ocrResp.IsTextDocument,
					MeanConfidence:     ocrResp.MeanConfidence,
					WeightedConfidence: ocrResp.WeightedConfidence,
					TokenCount:         ocrResp.TokenCount,
					Angle:              ocrResp.Angle,
					ScaleFactor:        ocrResp.ScaleFactor,
				}

				// Create bounding box records only for text documents
				var boxes []domain.OcrBoundingBox
				if ocrResp.IsTextDocument && len(ocrResp.Boxes) > 0 {
					// We need to save the classification first to get its ID
					// So we'll handle boxes in a separate step
					for _, box := range ocrResp.Boxes {
						boxes = append(boxes, domain.OcrBoundingBox{
							X:          box.X,
							Y:          box.Y,
							Width:      box.Width,
							Height:     box.Height,
							Word:       box.Word,
							Confidence: box.Confidence,
						})
					}
				}

				result.classification = classification
				result.boxes = boxes
				results <- result
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

	// Close results channel when all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results with batch DB writes
	const batchSize = 50
	var toCreate []domain.OcrClassification
	var boxesToCreate []domain.OcrBoundingBox
	count := 0

	for result := range results {
		count++

		if result.err != nil {
			log.Printf("OCR: error processing %s: %v", result.image.Path, result.err)
			om.mu.Lock()
			om.filesProcessed = count
			om.progress = fmt.Sprintf("Error on %s: %v", result.image.Path, result.err)
			om.mu.Unlock()
			continue
		}

		if result.classification != nil {
			toCreate = append(toCreate, *result.classification)
		}

		// Store boxes with reference to the classification (will be set after insert)
		if len(result.boxes) > 0 && result.classification != nil {
			// We'll link boxes after classification is inserted
			for i := range result.boxes {
				boxesToCreate = append(boxesToCreate, result.boxes[i])
			}
		}

		om.mu.Lock()
		om.filesProcessed = count
		om.progress = fmt.Sprintf("Classifying: %d/%d", count, om.totalFiles)
		om.mu.Unlock()

		// Batch write classifications
		if len(toCreate) >= batchSize {
			om.saveClassificationBatch(&toCreate, &boxesToCreate)
		}
	}

	// Flush remaining
	if len(toCreate) > 0 || len(boxesToCreate) > 0 {
		om.saveClassificationBatch(&toCreate, &boxesToCreate)
	}

	om.mu.Lock()
	om.progress = fmt.Sprintf("OCR classification complete: %d/%d images processed", count, om.totalFiles)
	om.mu.Unlock()
}

// saveClassificationBatch saves a batch of classifications and their bounding boxes
func (om *OcrManager) saveClassificationBatch(classifications *[]domain.OcrClassification, boxes *[]domain.OcrBoundingBox) {
	if len(*classifications) == 0 {
		*classifications = (*classifications)[:0]
		*boxes = (*boxes)[:0]
		return
	}

	// Save classifications and their bounding boxes sequentially
	// Each classification must be saved first to get its ID for bounding boxes
	boxIdx := 0
	for i := range *classifications {
		if err := om.db.Create(&(*classifications)[i]).Error; err != nil {
			log.Printf("OCR: failed to save classification for image %d: %v", (*classifications)[i].ImageFileID, err)
			continue
		}

		// Save bounding boxes for this classification
		classificationID := (*classifications)[i].ID
		for boxIdx < len(*boxes) {
			// Link box to the current classification
			(*boxes)[boxIdx].ClassificationID = classificationID
			if err := om.db.Create(&(*boxes)[boxIdx]).Error; err != nil {
				log.Printf("OCR: failed to save bounding box for classification %d: %v", classificationID, err)
			}
			boxIdx++
		}
	}

	*classifications = (*classifications)[:0]
	*boxes = (*boxes)[:0]
}
