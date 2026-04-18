package imaging

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	_ "github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"image-dedup/internal/domain"
)

// extractMetadata reads EXIF metadata and image dimensions from a file.
// It always attempts to get dimensions; EXIF fields are best-effort.
func extractMetadata(filePath string) (*domain.ImageMetadata, error) {
	meta := &domain.ImageMetadata{}

	// Get image dimensions (works for all supported formats)
	if w, h, err := getImageDimensions(filePath); err == nil {
		meta.Width = w
		meta.Height = h
	}

	// Attempt EXIF extraction (only JPEG and TIFF have EXIF)
	extractExifFields(filePath, meta)

	return meta, nil
}

// getImageDimensions reads only the image header to get width and height.
func getImageDimensions(filePath string) (int, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// extractExifFields attempts to read all EXIF fields from the file.
// Each field is extracted independently; failures are silently ignored.
func extractExifFields(filePath string, meta *domain.ImageMetadata) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return
	}

	// Camera model
	if tag, err := x.Get(exif.Model); err == nil {
		meta.CameraModel = tagString(tag)
	}

	// Lens model
	if tag, err := x.Get(exif.LensModel); err == nil {
		meta.LensModel = tagString(tag)
	}

	// ISO
	if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if v, err := tag.Int(0); err == nil {
			meta.ISO = v
		}
	}

	// Aperture (FNumber)
	if tag, err := x.Get(exif.FNumber); err == nil {
		if num, denom, err := tag.Rat2(0); err == nil && denom != 0 {
			val := float64(num) / float64(denom)
			meta.Aperture = fmt.Sprintf("f/%.1f", val)
		}
	}

	// Shutter speed (ExposureTime)
	if tag, err := x.Get(exif.ExposureTime); err == nil {
		if num, denom, err := tag.Rat2(0); err == nil && denom != 0 {
			meta.ShutterSpeed = formatExposureTime(num, denom)
		}
	}

	// Focal length
	if tag, err := x.Get(exif.FocalLength); err == nil {
		if num, denom, err := tag.Rat2(0); err == nil && denom != 0 {
			val := float64(num) / float64(denom)
			if val == math.Trunc(val) {
				meta.FocalLength = fmt.Sprintf("%.0fmm", val)
			} else {
				meta.FocalLength = fmt.Sprintf("%.1fmm", val)
			}
		}
	}

	// Date taken
	if dt, err := x.DateTime(); err == nil {
		t := dt.UTC()
		meta.DateTaken = &t
	}

	// Orientation
	if tag, err := x.Get(exif.Orientation); err == nil {
		if v, err := tag.Int(0); err == nil {
			meta.Orientation = v
		}
	}

	// Color space
	if tag, err := x.Get(exif.ColorSpace); err == nil {
		if v, err := tag.Int(0); err == nil {
			meta.ColorSpace = colorSpaceName(v)
		}
	}

	// Software
	if tag, err := x.Get(exif.Software); err == nil {
		meta.Software = tagString(tag)
	}

	// GPS coordinates
	if lat, lng, err := x.LatLong(); err == nil {
		meta.GPSLatitude = &lat
		meta.GPSLongitude = &lng
	}
}

// tagString extracts a clean string from an EXIF tag.
func tagString(tag *tiff.Tag) string {
	if tag.Format() == tiff.StringVal {
		s, err := tag.StringVal()
		if err == nil {
			return s
		}
	}
	return tag.String()
}

// formatExposureTime formats a rational exposure time value.
func formatExposureTime(num, denom int64) string {
	if num == 0 {
		return "0s"
	}
	if num >= denom {
		val := float64(num) / float64(denom)
		if val == math.Trunc(val) {
			return fmt.Sprintf("%.0fs", val)
		}
		return fmt.Sprintf("%.1fs", val)
	}
	// Show as fraction, e.g. "1/250s"
	simplified := denom / num
	return fmt.Sprintf("1/%ds", simplified)
}

// colorSpaceName maps EXIF ColorSpace integer to a human-readable name.
func colorSpaceName(v int) string {
	switch v {
	case 1:
		return "sRGB"
	case 65535:
		return "Uncalibrated"
	default:
		return fmt.Sprintf("Unknown (%d)", v)
	}
}

// HasExifData returns true if any meaningful EXIF field is populated.
func HasExifData(meta *domain.ImageMetadata) bool {
	return meta.CameraModel != "" || meta.LensModel != "" || meta.ISO != 0 ||
		meta.Aperture != "" || meta.ShutterSpeed != "" || meta.FocalLength != "" ||
		meta.DateTaken != nil || meta.Software != "" || meta.GPSLatitude != nil
}
