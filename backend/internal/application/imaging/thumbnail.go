package imaging

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/deepteams/webp"
	"github.com/disintegration/imaging"
)

const (
	maxThumbnailSize = 320
)

// ThumbnailCache stores generated thumbnails in memory
type ThumbnailCache struct {
	cache map[string]string
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

// GenerateThumbnail creates a thumbnail for an image file
func GenerateThumbnail(imagePath string, cache *ThumbnailCache) (string, error) {
	if cached, ok := cache.Get(imagePath); ok {
		return cached, nil
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var newWidth, newHeight int
	if width >= height {
		newWidth = maxThumbnailSize
		newHeight = 0
	} else {
		newWidth = 0
		newHeight = maxThumbnailSize
	}

	thumbnail := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	var buf bytes.Buffer
	err = webp.Encode(&buf, thumbnail, &webp.Options{Quality: 80})
	if err != nil {
		// Fallback to JPEG if WebP encoding fails
		buf.Reset()
		if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
			return "", fmt.Errorf("failed to encode thumbnail: %w", err)
		}
		base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
		result := "data:image/jpeg;base64," + base64Str
		cache.Set(imagePath, result)
		return result, nil
	}

	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	result := "data:image/webp;base64," + base64Str

	cache.Set(imagePath, result)

	return result, nil
}

// GetImageMimeType returns the MIME type based on file extension
func GetImageMimeType(path string) string {
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
