package imaging

import (
	"fmt"
	"log"
	"math"

	"image-toolkit/internal/domain"

	"gorm.io/gorm"
)

// OcrMigrationManager handles migration of existing OCR classifications
// to add bounding_box_width and bounding_box_height fields
type OcrMigrationManager struct {
	db *gorm.DB
}

// NewOcrMigrationManager creates a new migration manager
func NewOcrMigrationManager(db *gorm.DB) *OcrMigrationManager {
	return &OcrMigrationManager{db: db}
}

// RunMigration calculates and updates bounding_box_width and bounding_box_height
// for existing OCR classifications based on original image dimensions, scale_factor, and angle
func (om *OcrMigrationManager) RunMigration() error {
	// Find all classifications that need migration
	// (where bounding_box_width or bounding_box_height is 0 or null)
	var classifications []domain.OcrClassification
	if err := om.db.Where("bounding_box_width = 0 OR bounding_box_width IS NULL").Find(&classifications).Error; err != nil {
		return fmt.Errorf("failed to query classifications for migration: %w", err)
	}

	if len(classifications) == 0 {
		log.Println("OCR migration: no classifications need migration")
		return nil
	}

	log.Printf("OCR migration: found %d classifications to update", len(classifications))

	// Process each classification
	for i, classification := range classifications {
		// Get original image dimensions from metadata
		var metadata domain.ImageMetadata
		if err := om.db.Where("image_file_id = ?", classification.ImageFileID).First(&metadata).Error; err != nil {
			log.Printf("OCR migration: failed to load metadata for image %d: %v", classification.ImageFileID, err)
			continue
		}

		// Calculate bounding box dimensions based on angle and scale factor
		originalWidth := float32(metadata.Width)
		originalHeight := float32(metadata.Height)
		scaleFactor := classification.ScaleFactor

		// Calculate rotated dimensions
		var rotatedWidth, rotatedHeight float32

		switch classification.Angle {
		case 0, 180:
			rotatedWidth = originalWidth
			rotatedHeight = originalHeight
		case 90, 270:
			rotatedWidth = originalHeight
			rotatedHeight = originalWidth
		default:
			// For arbitrary angles, use affine transformation formulas
			// After rotating a rectangle by angle θ, the bounding box dimensions are:
			// newWidth = |originalWidth * cos(θ)| + |originalHeight * sin(θ)|
			// newHeight = |originalWidth * sin(θ)| + |originalHeight * cos(θ)|
			angleRad := float32(classification.Angle) * float32(math.Pi) / 180.0
			cosAngle := float32(math.Cos(float64(angleRad)))
			sinAngle := float32(math.Sin(float64(angleRad)))

			rotatedWidth = float32(math.Abs(float64(originalWidth*cosAngle))) + float32(math.Abs(float64(originalHeight*sinAngle)))
			rotatedHeight = float32(math.Abs(float64(originalWidth*sinAngle))) + float32(math.Abs(float64(originalHeight*cosAngle)))
		}

		// Apply scale factor and convert to int
		boundingBoxWidth := int(rotatedWidth * scaleFactor)
		boundingBoxHeight := int(rotatedHeight * scaleFactor)

		// Update the classification
		if err := om.db.Model(&classification).Updates(map[string]interface{}{
			"bounding_box_width":  boundingBoxWidth,
			"bounding_box_height": boundingBoxHeight,
		}).Error; err != nil {
			log.Printf("OCR migration: failed to update classification %d: %v", classification.ID, err)
			continue
		}

		// Log progress every 100 records
		if (i+1)%100 == 0 {
			log.Printf("OCR migration: processed %d/%d classifications", i+1, len(classifications))
		}
	}

	log.Printf("OCR migration: completed updating %d classifications", len(classifications))
	return nil
}

// GetMigrationStatus returns migration statistics
func (om *OcrMigrationManager) GetMigrationStatus() (total int, migrated int, err error) {
	var total64, migrated64 int64
	// Count total classifications
	if err := om.db.Model(&domain.OcrClassification{}).Count(&total64).Error; err != nil {
		return 0, 0, err
	}

	// Count classifications that have been migrated (non-zero bounding_box_width)
	if err := om.db.Model(&domain.OcrClassification{}).
		Where("bounding_box_width > 0").
		Count(&migrated64).Error; err != nil {
		return 0, 0, err
	}

	return int(total64), int(migrated64), nil
}

// EnsureMigrationNeeded checks if migration is needed for any classifications
func (om *OcrMigrationManager) EnsureMigrationNeeded() (bool, error) {
	var count int64
	if err := om.db.Model(&domain.OcrClassification{}).
		Where("bounding_box_width = 0 OR bounding_box_width IS NULL").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
