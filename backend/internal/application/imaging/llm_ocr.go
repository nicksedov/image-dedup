package imaging

import (
	"fmt"
	"strings"
	"time"

	"image-toolkit/internal/domain"
	"image-toolkit/internal/infrastructure/llm"

	"gorm.io/gorm"
)

// LlmOcrService handles VL LLM-based OCR recognition
type LlmOcrService struct {
	db *gorm.DB
}

// NewLlmOcrService creates a new LLM OCR service
func NewLlmOcrService(db *gorm.DB) *LlmOcrService {
	return &LlmOcrService{db: db}
}

// RecognizeResult holds the result of LLM recognition
type RecognizeResult struct {
	Success          bool
	MarkdownContent  string
	Language         string
	Provider         string
	Model            string
	ProcessingTimeMs int
	Error            string
}

// RecognizeWithLlm performs OCR using VL LLM
func (s *LlmOcrService) RecognizeWithLlm(imageFileID uint, client llm.Client, settings domain.LlmSettings) (*RecognizeResult, error) {
	// Step 1: Get OCR classification to detect language
	var classification domain.OcrClassification
	if err := s.db.Where("image_file_id = ?", imageFileID).First(&classification).Error; err != nil {
		return nil, fmt.Errorf("failed to get OCR classification: %w", err)
	}

	// Step 2: Detect language from classification data
	language := s.detectLanguage(classification)

	// Step 3: Build system prompt
	systemPrompt := buildOcrSystemPrompt(language)

	// Step 4: Get image path
	var imageFile domain.ImageFile
	if err := s.db.First(&imageFile, imageFileID).Error; err != nil {
		return nil, fmt.Errorf("failed to get image file: %w", err)
	}

	// Step 5: Call LLM
	startTime := time.Now()
	markdownContent, err := client.Recognize(imageFile.Path, systemPrompt)
	processingTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		// Save failed result using UPSERT
		recognition := domain.OcrLlmRecognition{
			ImageFileID:         imageFileID,
			OcrClassificationID: classification.ID,
			Language:            language,
			MarkdownContent:     "",
			Provider:            settings.Provider,
			Model:               settings.Model,
			ProcessingTimeMs:    processingTime,
			Error:               err.Error(),
			Success:             false,
		}
		s.db.
			Where("image_file_id = ?", imageFileID).
			Assign(recognition).
			FirstOrCreate(&recognition)
		return nil, err
	}

	// Step 6: Save successful result using UPSERT
	recognition := domain.OcrLlmRecognition{
		ImageFileID:         imageFileID,
		OcrClassificationID: classification.ID,
		Language:            language,
		MarkdownContent:     markdownContent,
		Provider:            settings.Provider,
		Model:               settings.Model,
		ProcessingTimeMs:    processingTime,
		Error:               "",
		Success:             true,
	}
	if err := s.db.
		Where("image_file_id = ?", imageFileID).
		Assign(recognition).
		FirstOrCreate(&recognition).Error; err != nil {
		return nil, fmt.Errorf("failed to save recognition result: %w", err)
	}

	return &RecognizeResult{
		Success:          true,
		MarkdownContent:  markdownContent,
		Language:         language,
		Provider:         settings.Provider,
		Model:            settings.Model,
		ProcessingTimeMs: processingTime,
		Error:            "",
	}, nil
}

// GetRecognition retrieves LLM OCR recognition for an image
func (s *LlmOcrService) GetRecognition(imageFileID uint) (*domain.OcrLlmRecognition, error) {
	var recognition domain.OcrLlmRecognition
	if err := s.db.Where("image_file_id = ?", imageFileID).Order("id DESC").First(&recognition).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &recognition, nil
}

// detectLanguage determines document language from OCR classification data
func (s *LlmOcrService) detectLanguage(classification domain.OcrClassification) string {
	// Get bounding boxes to analyze words for language detection
	var boxes []domain.OcrBoundingBox
	s.db.Where("classification_id = ?", classification.ID).Find(&boxes)

	if len(boxes) == 0 {
		return "en" // Default to English
	}

	// Simple language detection based on character sets
	russianCount := 0
	englishCount := 0

	for _, box := range boxes {
		word := strings.ToLower(box.Word)
		for _, ch := range word {
			// Cyrillic range
			if ch >= 0x0400 && ch <= 0x04FF {
				russianCount++
			}
			// Latin range
			if ch >= 0x0061 && ch <= 0x007A {
				englishCount++
			}
		}
	}

	if russianCount > englishCount {
		return "ru"
	}
	return "en"
}

// buildOcrSystemPrompt creates the system prompt for VL LLM
func buildOcrSystemPrompt(language string) string {
	langName := "English"
	if language == "ru" {
		langName = "Russian"
	}

	return fmt.Sprintf(`You are an OCR model. Convert the document in %s language to markdown format, preserving formatting for headings, paragraphs, tables, and formulas.

Rules:
1. Output ONLY the document content - NO comments, explanations, or meta-text
2. Preserve the original structure: headings, lists, tables, formulas
3. Use proper markdown syntax (# for H1, ## for H2, etc.)
4. For tables, use markdown table format (| column | column |)
5. For formulas, use LaTeX format ($formula$ for inline, $$formula$$ for block)
6. Maintain the original language - do NOT translate
7. If text is unclear, mark it as [unclear]
8. Do NOT add "Here is the document" or "This document contains" - just output the content

Return only the markdown content, nothing else.`, langName)
}
