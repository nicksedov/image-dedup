package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
)

// PrepareOcrImage opens an image, scales it by scaleFactor and rotates
// clockwise by the given angle (in degrees). Returns JPEG-encoded bytes.
func PrepareOcrImage(imagePath string, scaleFactor float64, angle float64) ([]byte, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Scale by scaleFactor
	if scaleFactor != 1.0 {
		newWidth := int(float64(img.Bounds().Dx()) * scaleFactor)
		newHeight := int(float64(img.Bounds().Dy()) * scaleFactor)
		if newWidth > 0 && newHeight > 0 {
			img = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
		}
	}

	// Rotate counter-clockwise by angle
	if angle != 0 {
		img = imaging.Rotate(img, angle, color.Black)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}
