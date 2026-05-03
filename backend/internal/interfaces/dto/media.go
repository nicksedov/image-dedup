package dto

// --- Duplicates API ---

// DuplicatesResponse is the JSON response for GET /api/duplicates
type DuplicatesResponse struct {
	Groups      []DuplicateGroupDTO `json:"groups"`
	TotalFiles  int                 `json:"totalFiles"`
	PageFiles   int                 `json:"pageFiles"`
	TotalGroups int                 `json:"totalGroups"`
	ScannedDirs []string            `json:"scannedDirs"`
	CurrentPage int                 `json:"currentPage"`
	PageSize    int                 `json:"pageSize"`
	TotalPages  int                 `json:"totalPages"`
	HasPrevPage bool                `json:"hasPrevPage"`
	HasNextPage bool                `json:"hasNextPage"`
	PageSizes   []int               `json:"pageSizes"`
}

// DuplicateGroupDTO represents a duplicate group in JSON responses
type DuplicateGroupDTO struct {
	Index     int       `json:"index"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	SizeHuman string    `json:"sizeHuman"`
	Files     []FileDTO `json:"files"`
	Thumbnail string    `json:"thumbnail"`
}

// FileDTO represents a file in JSON responses
type FileDTO struct {
	ID       uint   `json:"id"`
	Path     string `json:"path"`
	FileName string `json:"fileName"`
	DirPath  string `json:"dirPath"`
	ModTime  string `json:"modTime"`
}

// --- Scan API ---

// ScanResponse is the JSON response for POST /api/scan
// Message is a i18n key string (e.g., "scan.started")
type ScanResponse struct {
	Message string `json:"message"`
}

// FastScanResponse is the JSON response for POST /api/fast-scan
type FastScanResponse struct {
	Message   string `json:"message"`
	Unchanged int    `json:"unchanged"` // Files that exist and haven't changed
	Modified  int    `json:"modified"`  // Files that were modified (size changed)
	Created   int    `json:"created"`   // New files added
	Deleted   int    `json:"deleted"`   // Records removed from DB (files no longer exist)
	Total     int    `json:"total"`     // Total checked (modified + created)
}

// --- Delete Files API ---

// DeleteFilesRequest represents the request for direct file deletion
type DeleteFilesRequest struct {
	FilePaths []string `json:"filePaths"`
	TrashDir  string   `json:"trashDir"`
}

// DeleteFilesResponse represents the response from file deletion
type DeleteFilesResponse struct {
	Success     int      `json:"success"`
	Failed      int      `json:"failed"`
	FailedFiles []string `json:"failedFiles,omitempty"`
}

// --- Folder Patterns API ---

// FolderPattern represents a unique combination of folders containing duplicates
type FolderPattern struct {
	ID             string   `json:"id"`
	Folders        []string `json:"folders"`
	DuplicateCount int      `json:"duplicateCount"`
	TotalFiles     int      `json:"totalFiles"`
}

// FolderPatternsResponse represents the response for folder patterns
type FolderPatternsResponse struct {
	Patterns []FolderPattern `json:"patterns"`
}

// --- Batch Delete API ---

// BatchDeleteRequest represents a request for batch deletion
type BatchDeleteRequest struct {
	Rules    []BatchDeleteRule `json:"rules"`
	TrashDir string            `json:"trashDir"`
}

// BatchDeleteRule specifies which folder to keep for a pattern
type BatchDeleteRule struct {
	PatternID  string `json:"patternId"`
	KeepFolder string `json:"keepFolder"`
}

// BatchDeleteResponse represents the response from batch deletion
type BatchDeleteResponse struct {
	Success     int      `json:"success"`
	Failed      int      `json:"failed"`
	FailedFiles []string `json:"failedFiles,omitempty"`
}

// --- Thumbnail API ---

// ThumbnailResponse is the JSON response for GET /api/thumbnail
type ThumbnailResponse struct {
	Thumbnail string `json:"thumbnail"`
}

// ThumbnailCacheStatsResponse статистика кэша миниатюр
type ThumbnailCacheStatsResponse struct {
	TotalSize    int64  `json:"totalSize"`
	TotalFiles   int    `json:"totalFiles"`
	CacheDir     string `json:"cacheDir"`
	Enabled      bool   `json:"enabled"`
	Initialized  bool   `json:"initialized"`
}

// InvalidateThumbnailRequest запрос на удаление миниатюры
type InvalidateThumbnailRequest struct {
	FilePath string `json:"filePath" binding:"required"`
}

// WarmupThumbnailsRequest запрос на предварительную генерацию миниатюр
type WarmupThumbnailsRequest struct {
	FilePaths []string `json:"filePaths" binding:"required"`
}

// ThumbnailCacheStatusResponse статус кэша миниатюр
type ThumbnailCacheStatusResponse struct {
	Enabled     bool   `json:"enabled"`
	CacheDir    string `json:"cacheDir"`
	FilesCount  int    `json:"filesCount"`
	TotalSize   int64  `json:"totalSize"`
	TotalSizeHuman string `json:"totalSizeHuman"`
}

// --- Gallery Folders API ---

// GalleryFolderDTO represents a gallery folder in JSON responses
type GalleryFolderDTO struct {
	ID        uint   `json:"id"`
	Path      string `json:"path"`
	FileCount int    `json:"fileCount"`
	CreatedAt string `json:"createdAt"`
}

// GalleryFoldersResponse is the JSON response for GET /api/folders
type GalleryFoldersResponse struct {
	Folders      []GalleryFolderDTO `json:"folders"`
	TotalFolders int                `json:"totalFolders"`
}

// AddFolderRequest represents the request for adding a gallery folder
type AddFolderRequest struct {
	Path string `json:"path" binding:"required"`
}

// AddFolderResponse is the JSON response for POST /api/folders
// Message is a i18n key string (e.g., "folder.added")
type AddFolderResponse struct {
	Message     string           `json:"message"`
	Folder      GalleryFolderDTO `json:"folder"`
	ScanStarted bool             `json:"scanStarted"`
}

// RemoveFolderResponse is the JSON response for DELETE /api/folders/:id
// Message is a i18n key string (e.g., "folder.removed")
type RemoveFolderResponse struct {
	Message      string `json:"message"`
	FilesRemoved int    `json:"filesRemoved"`
}

// --- Gallery Images API ---

// GalleryImageDTO represents an image in the gallery browser
type GalleryImageDTO struct {
	ID        uint   `json:"id"`
	Path      string `json:"path"`
	FileName  string `json:"fileName"`
	DirPath   string `json:"dirPath"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"sizeHuman"`
	ModTime   string `json:"modTime"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// GalleryImagesResponse is the JSON response for GET /api/gallery
type GalleryImagesResponse struct {
	Images      []GalleryImageDTO `json:"images"`
	TotalImages int               `json:"totalImages"`
	CurrentPage int               `json:"currentPage"`
	PageSize    int               `json:"pageSize"`
	TotalPages  int               `json:"totalPages"`
	HasNextPage bool              `json:"hasNextPage"`
}

// --- App Settings API ---

// AppSettingsDTO is the JSON response for GET /api/settings
type AppSettingsDTO struct {
	Theme              string `json:"theme"`
	Language           string `json:"language"`
	TrashDir           string `json:"trashDir"`
	ThumbnailCachePath string `json:"thumbnailCachePath,omitempty"`
	ThumbnailCacheSize int    `json:"thumbnailCacheSize,omitempty"`
}

// UserSettingsDTO is the JSON response for user settings
type UserSettingsDTO struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
	TrashDir string `json:"trashDir"`
}

// UpdateSettingsRequest is the JSON request for PUT /api/settings
type UpdateSettingsRequest struct {
	Theme              string  `json:"theme"`
	Language           string  `json:"language"`
	TrashDir           *string `json:"trashDir"`
	ThumbnailCachePath *string `json:"thumbnailCachePath,omitempty"`
}

// --- Trash API ---

// TrashInfoResponse is the JSON response for GET /api/trash-info
type TrashInfoResponse struct {
	FileCount      int    `json:"fileCount"`
	TotalSize      int64  `json:"totalSize"`
	TotalSizeHuman string `json:"totalSizeHuman"`
}

// CleanTrashResponse is the JSON response for POST /api/trash-clean
type CleanTrashResponse struct {
	Deleted int `json:"deleted"`
	Failed  int `json:"failed"`
}

// --- Image Metadata API ---

// ImageMetadataDTO represents image EXIF metadata and geolocation in JSON responses
type ImageMetadataDTO struct {
	Width        int      `json:"width"`
	Height       int      `json:"height"`
	Dimensions   string   `json:"dimensions"`
	CameraModel  string   `json:"cameraModel"`
	LensModel    string   `json:"lensModel"`
	ISO          int      `json:"iso"`
	Aperture     string   `json:"aperture"`
	ShutterSpeed string   `json:"shutterSpeed"`
	FocalLength  string   `json:"focalLength"`
	DateTaken    string   `json:"dateTaken"`
	Orientation  int      `json:"orientation"`
	ColorSpace   string   `json:"colorSpace"`
	Software     string   `json:"software"`
	GPSLatitude  *float64 `json:"gpsLatitude"`
	GPSLongitude *float64 `json:"gpsLongitude"`
	GeoCountry   string   `json:"geoCountry"`
	GeoCity      string   `json:"geoCity"`
	HasGPS       bool     `json:"hasGps"`
	HasExif      bool     `json:"hasExif"`
}

// ImageMetadataResponse is the JSON response for GET /api/image-metadata
type ImageMetadataResponse struct {
	Found    bool              `json:"found"`
	Metadata *ImageMetadataDTO `json:"metadata,omitempty"`
}

// --- Gallery Calendar API ---

// CalendarDateGroup represents a group of images for a single date
type CalendarDateGroup struct {
	Date       string            `json:"date"`  // "YYYY-MM-DD"
	Label      string            `json:"label"` // Human-readable label
	ImageCount int               `json:"imageCount"`
	Images     []GalleryImageDTO `json:"images"`
}

// CalendarDateRange represents the min/max date range for all images with EXIF dates
type CalendarDateRange struct {
	MinDate       string `json:"minDate"` // "YYYY-MM-DD" or empty
	MaxDate       string `json:"maxDate"` // "YYYY-MM-DD" or empty
	TotalWithDate int    `json:"totalWithDate"`
}

// CalendarMonthInfo represents which days in a month have images
type CalendarMonthInfo struct {
	Year  int   `json:"year"`
	Month int   `json:"month"` // 1-12
	Days  []int `json:"days"`  // Days that have images (1-31)
}

// GalleryCalendarResponse is the JSON response for GET /api/gallery/calendar
type GalleryCalendarResponse struct {
	Groups      []CalendarDateGroup `json:"groups"`
	TotalImages int                 `json:"totalImages"`
	TotalGroups int                 `json:"totalGroups"`
	HasMore     bool                `json:"hasMore"`
	DateRange   CalendarDateRange   `json:"dateRange"`
	// Month info for the calendar widget (current page's months)
	Months []CalendarMonthInfo `json:"months"`
}

// --- OCR Status API ---

// OCRStatus represents the current status of OCR classifier service
type OCRStatus struct {
	Enabled    bool   `json:"enabled"`
	Health     string `json:"health"`
	LastCheck  string `json:"lastCheck,omitempty"`
	Error      string `json:"error,omitempty"`
	ServiceURL string `json:"serviceUrl,omitempty"`
}

// OCRStatusResponse is the JSON response for GET /api/ocr/status
type OCRStatusResponse struct {
	Status OCRStatus `json:"status"`
}

// --- OCR Classification API ---

// OcrClassificationStatusResponse for GET /api/ocr/classify-status
type OcrClassificationStatusResponse struct {
	Processing     bool   `json:"processing"`
	Progress       string `json:"progress"`
	FilesProcessed int    `json:"filesProcessed"`
	TotalFiles     int    `json:"totalFiles"`
}

// OcrDocumentDTO represents an image classified as a text document
type OcrDocumentDTO struct {
	ID                 uint    `json:"id"`
	ImageFileID        uint    `json:"imageFileId"`
	Path               string  `json:"path"`
	FileName           string  `json:"fileName"`
	DirPath            string  `json:"dirPath"`
	Size               int64   `json:"size"`
	SizeHuman          string  `json:"sizeHuman"`
	ModTime            string  `json:"modTime"`
	Thumbnail          string  `json:"thumbnail,omitempty"`
	ThumbnailCachePath string  `json:"thumbnailCachePath,omitempty"`
	MeanConfidence     float32 `json:"meanConfidence"`
	WeightedConfidence float32 `json:"weightedConfidence"`
	TokenCount         int     `json:"tokenCount"`
	Angle              int     `json:"angle"`
	ScaleFactor        float32 `json:"scaleFactor"`
}

// OcrDocumentsResponse for GET /api/ocr/documents
type OcrDocumentsResponse struct {
	Documents   []OcrDocumentDTO `json:"documents"`
	Total       int              `json:"total"`
	CurrentPage int              `json:"currentPage"`
	PageSize    int              `json:"pageSize"`
	TotalPages  int              `json:"totalPages"`
	HasNextPage bool             `json:"hasNextPage"`
}

// BoundingBoxDTO for OCR bounding box data
type BoundingBoxDTO struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Word       string  `json:"word"`
	Confidence float32 `json:"confidence"`
}

// OcrDataResponse for GET /api/ocr/data
type OcrDataResponse struct {
	ImagePath   string           `json:"imagePath"`
	Angle       int              `json:"angle"`
	ScaleFactor float32          `json:"scaleFactor"`
	Boxes       []BoundingBoxDTO `json:"boxes"`
}

// --- LLM Settings API ---

// LlmSettingsDTO for LLM settings responses
type LlmSettingsDTO struct {
	ID       uint   `json:"id"`
	Provider string `json:"provider"`
	ApiUrl   string `json:"apiUrl"`
	ApiKey   string `json:"apiKey"`
	Model    string `json:"model"`
	Enabled  bool   `json:"enabled"`
}

// UpdateLlmSettingsRequest for PUT /api/llm/settings
type UpdateLlmSettingsRequest struct {
	Provider string `json:"provider"`
	ApiUrl   string `json:"apiUrl"`
	ApiKey   string `json:"apiKey"`
	Model    string `json:"model"`
	Enabled  bool   `json:"enabled"`
}

// LlmModelDTO represents an available LLM model
type LlmModelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
}

// LlmModelsResponse for GET /api/llm/models
type LlmModelsResponse struct {
	Success  bool          `json:"success"`
	Models   []LlmModelDTO `json:"models"`
	Error    string        `json:"error,omitempty"`
	Provider string        `json:"provider"`
}

// --- LLM OCR API ---

// LlmOcrRequest for POST /api/llm/recognize
type LlmOcrRequest struct {
	ImagePath string `json:"imagePath" binding:"required"`
	Force     bool   `json:"force"`
}

// LlmOcrResponse for POST /api/llm/recognize
type LlmOcrResponse struct {
	Success          bool   `json:"success"`
	MarkdownContent  string `json:"markdownContent,omitempty"`
	Language         string `json:"language"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	ProcessingTimeMs int    `json:"processingTimeMs"`
	Error            string `json:"error,omitempty"`
}

// LlmOcrDataResponse for GET /api/llm/recognition
type LlmOcrDataResponse struct {
	Found            bool   `json:"found"`
	MarkdownContent  string `json:"markdownContent,omitempty"`
	Language         string `json:"language,omitempty"`
	Provider         string `json:"provider,omitempty"`
	Model            string `json:"model,omitempty"`
	ProcessingTimeMs int    `json:"processingTimeMs,omitempty"`
	Success          bool   `json:"success,omitempty"`
	Error            string `json:"error,omitempty"`
	CreatedAt        string `json:"createdAt,omitempty"`
}

// LlmRecognizeStatusResponse for GET /api/llm/recognize-status
type LlmRecognizeStatusResponse struct {
	Status           string `json:"status"` // "processing", "completed", "failed", "not_found"
	MarkdownContent  string `json:"markdownContent,omitempty"`
	Language         string `json:"language,omitempty"`
	Provider         string `json:"provider,omitempty"`
	Model            string `json:"model,omitempty"`
	ProcessingTimeMs int    `json:"processingTimeMs,omitempty"`
	Error            string `json:"error,omitempty"`
}
