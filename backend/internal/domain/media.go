package domain

import (
	"path/filepath"
	"strings"
	"time"
)

// ImageFile represents an image file in the database
type ImageFile struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Path      string    `gorm:"uniqueIndex;not null" json:"path"`
	Size      int64     `gorm:"not null;index:idx_size_hash" json:"size"`
	Hash      string    `gorm:"not null;index:idx_size_hash" json:"hash"`
	ModTime   time.Time `gorm:"not null" json:"modTime"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DuplicateGroup represents a group of duplicate images
type DuplicateGroup struct {
	Hash  string
	Size  int64
	Files []ImageFile
}

// SupportedExtensions contains all supported image file extensions
var SupportedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
	".webp": true,
}

// IsImageFile checks if a file is a supported image based on extension
func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return SupportedExtensions[ext]
}

// ImageMetadata stores extracted EXIF metadata and geolocation for an image
type ImageMetadata struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ImageFileID  uint       `gorm:"uniqueIndex;not null" json:"imageFileId"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	CameraModel  string     `json:"cameraModel"`
	LensModel    string     `json:"lensModel"`
	ISO          int        `json:"iso"`
	Aperture     string     `json:"aperture"`
	ShutterSpeed string     `json:"shutterSpeed"`
	FocalLength  string     `json:"focalLength"`
	DateTaken    *time.Time `json:"dateTaken"`
	Orientation  int        `json:"orientation"`
	ColorSpace   string     `json:"colorSpace"`
	Software     string     `json:"software"`
	GPSLatitude  *float64   `json:"gpsLatitude"`
	GPSLongitude *float64   `json:"gpsLongitude"`
	GeoCountry   string     `json:"geoCountry"`
	GeoCity      string     `json:"geoCity"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// GalleryFolder represents a configured gallery folder in the database
type GalleryFolder struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Path      string    `gorm:"uniqueIndex;not null" json:"path"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AppSettings stores global application settings (singleton, ID=1)
type AppSettings struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Theme              string    `gorm:"default:light-purple;not null" json:"theme"`
	Language           string    `gorm:"default:en;not null" json:"language"`
	TrashDir           string    `gorm:"default:''" json:"trashDir"`
	ThumbnailCachePath string    `gorm:"default:''" json:"thumbnailCachePath"`
	ThumbnailCacheSize int       `gorm:"default:0" json:"thumbnailCacheSize"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// OcrClassification stores OCR classification results for an image
type OcrClassification struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	ImageFileID        uint      `gorm:"uniqueIndex;not null" json:"imageFileId"`
	IsTextDocument     bool      `gorm:"not null;default:false;index:idx_is_text_doc" json:"isTextDocument"`
	MeanConfidence     float32   `json:"meanConfidence"`
	WeightedConfidence float32   `json:"weightedConfidence"`
	TokenCount         int       `json:"tokenCount"`
	Angle              int       `json:"angle"`
	ScaleFactor        float32   `json:"scaleFactor"`
	BoundingBoxWidth   int       `json:"boundingBoxWidth"`
	BoundingBoxHeight  int       `json:"boundingBoxHeight"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// OcrBoundingBox stores bounding box data for OCR text regions
type OcrBoundingBox struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	ClassificationID uint    `gorm:"index;not null" json:"classificationId"`
	X                int     `json:"x"`
	Y                int     `json:"y"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	Word             string  `json:"word"`
	Confidence       float32 `json:"confidence"`
}

// LlmSettings stores VL LLM connection settings (singleton, ID=1)
type LlmSettings struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Provider  string    `gorm:"default:ollama;not null" json:"provider"` // "ollama" or "openai"
	ApiUrl    string    `gorm:"not null" json:"apiUrl"`
	ApiKey    string    `gorm:"default:''" json:"apiKey"` // Empty for Ollama
	Model     string    `gorm:"not null" json:"model"`
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// OcrLlmRecognition stores VL LLM OCR recognition results
type OcrLlmRecognition struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	ImageFileID         uint      `gorm:"uniqueIndex;not null" json:"imageFileId"`
	OcrClassificationID uint      `gorm:"index" json:"ocrClassificationId"`
	Language            string    `gorm:"not null" json:"language"` // "en", "ru", etc.
	MarkdownContent     string    `gorm:"type:text;not null" json:"markdownContent"`
	Provider            string    `json:"provider"`         // Which provider was used
	Model               string    `json:"model"`            // Which model was used
	ProcessingTimeMs    int       `json:"processingTimeMs"` // Processing time in milliseconds
	Error               string    `json:"error"`            // Error message if failed
	Success             bool      `gorm:"default:false" json:"success"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}
