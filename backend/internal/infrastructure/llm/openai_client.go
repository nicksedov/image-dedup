package llm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAIClient implements Client for OpenAI-compatible API
type OpenAIClient struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(baseURL, apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Timeout: 5 * time.Minute,
	}
}

// openAIRequest represents OpenAI chat completion request
type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string          `json:"role"`
	Content []openAIContent `json:"content"`
}

type openAIContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

// openAIResponse represents OpenAI chat completion response
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Recognize performs OCR using OpenAI-compatible API
func (c *OpenAIClient) Recognize(imagePath string, systemPrompt string) (string, error) {
	// Read image file
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	// Determine image format
	mediaType := "image/jpeg"
	ext := ""
	if len(imagePath) > 4 {
		ext = imagePath[len(imagePath)-4:]
	}
	switch ext {
	case ".png":
		mediaType = "image/png"
	case ".gif":
		mediaType = "image/gif"
	case ".web":
		mediaType = "image/webp"
	case ".tiff":
		mediaType = "image/tiff"
	}

	// Encode image to base64 data URL
	base64Img := base64.StdEncoding.EncodeToString(imgData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, base64Img)

	// Prepare request
	req := openAIRequest{
		Model: c.Model,
		Messages: []openAIMessage{
			{
				Role: "system",
				Content: []openAIContent{
					{
						Type: "text",
						Text: systemPrompt,
					},
				},
			},
			{
				Role: "user",
				Content: []openAIContent{
					{
						Type: "text",
						Text: "Perform OCR on this image and return markdown content.",
					},
					{
						Type: "image_url",
						ImageURL: &openAIImageURL{
							URL: dataURL,
						},
					},
				},
			},
		},
		MaxTokens: 4000,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	client := &http.Client{Timeout: c.Timeout}
	httpReq, err := http.NewRequest("POST", c.BaseURL+"/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// openAIModelsResponse represents OpenAI models list response
type openAIModelsResponse struct {
	Data []openAIModel `json:"data"`
}

type openAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
}

// ListModels returns a list of available models from OpenAI-compatible server
func (c *OpenAIClient) ListModels() ([]ModelInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	httpReq, err := http.NewRequest("GET", c.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var modelsResp openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]ModelInfo, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		models[i] = ModelInfo{
			ID:   m.ID,
			Name: m.ID,
		}
	}

	return models, nil
}
