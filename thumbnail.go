package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
)

const (
	maxThumbnailSize = 128
)

// ThumbnailCache stores generated thumbnails in memory
type ThumbnailCache struct {
	cache map[string]string // path -> base64 encoded thumbnail
	mu    sync.RWMutex
}

// NewThumbnailCache creates a new thumbnail cache
func NewThumbnailCache() *ThumbnailCache {
	return &ThumbnailCache{
		cache: make(map[string]string),
	}
}

// Get returns a cached thumbnail if available
func (tc *ThumbnailCache) Get(path string) (string, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	thumb, ok := tc.cache[path]
	return thumb, ok
}

// Set stores a thumbnail in the cache
func (tc *ThumbnailCache) Set(path, thumbnail string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[path] = thumbnail
}

// generateThumbnail creates a thumbnail for an image file
// Returns base64-encoded JPEG data
func generateThumbnail(imagePath string, cache *ThumbnailCache) (string, error) {
	// Check cache first
	if cached, ok := cache.Get(imagePath); ok {
		return cached, nil
	}

	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions (max 128px on longest side)
	var newWidth, newHeight int
	if width >= height {
		newWidth = maxThumbnailSize
		newHeight = 0 // Preserve aspect ratio
	} else {
		newWidth = 0 // Preserve aspect ratio
		newHeight = maxThumbnailSize
	}

	// Resize the image using Lanczos filter for high quality
	thumbnail := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	// Encode to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 80}); err != nil {
		return "", fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	// Convert to base64
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	result := "data:image/jpeg;base64," + base64Str

	// Store in cache
	cache.Set(imagePath, result)

	return result, nil
}

// getImageMimeType returns the MIME type based on file extension
func getImageMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	case ".tiff", ".tif":
		return "image/tiff"
	default:
		return "image/jpeg"
	}
}

// init registers additional image formats
func init() {
	// Register additional image formats
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
}
