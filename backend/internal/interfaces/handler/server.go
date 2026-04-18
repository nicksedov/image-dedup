package handler

import (
	"fmt"
	"strings"

	"image-dedup/internal/application/imaging"
	"image-dedup/internal/infrastructure/config"
	"image-dedup/internal/interfaces/dto"

	"gorm.io/gorm"
)

// Server holds the application state
type Server struct {
	db              *gorm.DB
	thumbnailCache  *imaging.ThumbnailCache
	scanManager     *imaging.ScanManager
	metadataManager *imaging.MetadataManager
	config          *config.AppConfig
}

// NewServer creates a new server instance
func NewServer(db *gorm.DB, scanManager *imaging.ScanManager, metadataManager *imaging.MetadataManager, cfg *config.AppConfig) *Server {
	return &Server{
		db:              db,
		thumbnailCache:  imaging.NewThumbnailCache(),
		scanManager:     scanManager,
		metadataManager: metadataManager,
		config:          cfg,
	}
}

// formatSize formats file size in human readable format
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// pathsConflict checks if two normalized (forward-slash) paths are the same,
// or if one is a parent/child of the other.
// Returns a non-empty reason string if there is a conflict, empty string otherwise.
func pathsConflict(a, b string) string {
	// Normalize: trim trailing slashes, lowercase for case-insensitive FS
	na := strings.TrimRight(strings.ToLower(a), "/")
	nb := strings.TrimRight(strings.ToLower(b), "/")

	if na == nb {
		return "same"
	}
	if strings.HasPrefix(na, nb+"/") {
		return "child" // a is child of b
	}
	if strings.HasPrefix(nb, na+"/") {
		return "parent" // a is parent of b
	}
	return ""
}

// sortStrings sorts a slice of strings in place
func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// sortPatternsByCount sorts patterns by duplicate count descending
func sortPatternsByCount(patterns []dto.FolderPattern) {
	for i := 0; i < len(patterns)-1; i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[i].DuplicateCount < patterns[j].DuplicateCount {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
}

// createPatternID creates a unique ID from sorted folder paths
func createPatternID(folders []string) string {
	return strings.Join(folders, "|")
}
