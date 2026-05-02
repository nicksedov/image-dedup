package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Status represents the OCR service status
type Status struct {
	HealthStatus HealthStatus `json:"healthStatus"`
	Error        string       `json:"error,omitempty"`
	LastCheck    time.Time    `json:"lastCheck"`
}

// HealthStatus represents the health status of OCR service
type HealthStatus string

const (
	HealthStatusUnknown   HealthStatus = "unknown"
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ClassifyParams holds query parameters for the classify endpoint
type ClassifyParams struct {
	ConfidenceThreshold float32
	Level               string
	MinTokenCount       int
	Lang                string
}

// DefaultClassifyParams returns default parameters matching openapi.yaml spec
func DefaultClassifyParams() *ClassifyParams {
	return &ClassifyParams{
		ConfidenceThreshold: 0.65,
		Level:               "RIL_TEXTLINE",
		MinTokenCount:       40,
		Lang:                "eng+rus",
	}
}

// ClassifyResponse matches the OCR classifier API response
type ClassifyResponse struct {
	MeanConfidence     float32       `json:"mean_confidence"`
	WeightedConfidence float32       `json:"weighted_confidence"`
	TokenCount         int           `json:"token_count"`
	Boxes              []BoundingBox `json:"boxes"`
	Angle              int           `json:"angle"`
	BoundingBoxWidth   int           `json:"bounding_box_width"`
	BoundingBoxHeight  int           `json:"bounding_box_height"`
	ScaleFactor        float32       `json:"scale_factor"`
	IsTextDocument     bool          `json:"is_text_document"`
}

// BoundingBox represents a detected text region
type BoundingBox struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Word       string  `json:"word"`
	Confidence float32 `json:"confidence"`
}

// Client is an interface for OCR classifier service
type Client interface {
	// CheckHealth checks if OCR service is available
	CheckHealth(ctx context.Context) (HealthStatus, error)
	// GetStatus returns the current OCR status
	GetStatus() Status
	// StartHealthCheck starts the periodic health check in background
	StartHealthCheck(intervalSeconds int)
	// StopHealthCheck stops the periodic health check
	StopHealthCheck()
	// Classify sends an image to the OCR classifier and returns results
	Classify(ctx context.Context, image io.Reader, contentType string, params *ClassifyParams) (*ClassifyResponse, error)
}

type clientImpl struct {
	baseURL    string
	httpClient *http.Client
	status     Status
	stopCheck  chan struct{}
	isRunning  bool
}

// NewClient creates a new OCR client
func NewClient(host string, port string) Client {
	baseURL := fmt.Sprintf("http://%s:%s/ocr-classifier/api", host, port)
	return &clientImpl{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		status: Status{
			HealthStatus: HealthStatusUnknown,
			LastCheck:    time.Now(),
		},
		stopCheck: make(chan struct{}),
	}
}

// CheckHealth checks if OCR service is available
func (c *clientImpl) CheckHealth(ctx context.Context) (HealthStatus, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return HealthStatusUnhealthy, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("failed to parse response: %w", err)
	}

	if result["status"] == "ok" {
		return HealthStatusHealthy, nil
	}

	return HealthStatusUnhealthy, fmt.Errorf("health check returned non-OK status")
}

// GetStatus returns the current OCR status
func (c *clientImpl) GetStatus() Status {
	c.status.LastCheck = time.Now()
	return c.status
}

// StartHealthCheck starts the periodic health check in background
func (c *clientImpl) StartHealthCheck(intervalSeconds int) {
	if c.isRunning {
		return
	}

	c.isRunning = true
	go c.healthCheckLoop(intervalSeconds)
}

// StopHealthCheck stops the periodic health check
func (c *clientImpl) StopHealthCheck() {
	if !c.isRunning {
		return
	}

	close(c.stopCheck)
	c.isRunning = false
}

func (c *clientImpl) healthCheckLoop(intervalSeconds int) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCheck:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			status, err := c.CheckHealth(ctx)
			cancel()

			c.status.LastCheck = time.Now()
			if err != nil {
				c.status.HealthStatus = HealthStatusUnhealthy
				c.status.Error = err.Error()
			} else {
				c.status.HealthStatus = status
				c.status.Error = ""
			}
		}
	}
}

// Classify sends an image to the OCR classifier and returns results
func (c *clientImpl) Classify(ctx context.Context, image io.Reader, contentType string, params *ClassifyParams) (*ClassifyResponse, error) {
	// Build URL with query parameters
	queryParams := url.Values{}
	if params != nil {
		queryParams.Set("confidence_threshold", fmt.Sprintf("%.2f", params.ConfidenceThreshold))
		if params.Level != "" {
			queryParams.Set("level", params.Level)
		}
		if params.MinTokenCount > 0 {
			queryParams.Set("min_token_count", fmt.Sprintf("%d", params.MinTokenCount))
		}
		if params.Lang != "" {
			queryParams.Set("lang", params.Lang)
		}
	} else {
		// Use defaults
		defaults := DefaultClassifyParams()
		queryParams.Set("confidence_threshold", fmt.Sprintf("%.2f", defaults.ConfidenceThreshold))
		queryParams.Set("level", defaults.Level)
		queryParams.Set("min_token_count", fmt.Sprintf("%d", defaults.MinTokenCount))
		queryParams.Set("lang", defaults.Lang)
	}

	classifyURL := fmt.Sprintf("%s/v1/classify?%s", c.baseURL, queryParams.Encode())

	// Read image data into buffer
	imgData, err := io.ReadAll(image)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Create POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, classifyURL, bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type
	if contentType == "" {
		contentType = "image/jpeg"
	}
	req.Header.Set("Content-Type", contentType)

	// Use longer timeout for OCR processing (90s) - OCR can be slow for large images
	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OCR API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result ClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse OCR response: %w", err)
	}

	return &result, nil
}
