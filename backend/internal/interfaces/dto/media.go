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
type ScanResponse struct {
	Message string `json:"message"`
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
type AddFolderResponse struct {
	Message     string           `json:"message"`
	Folder      GalleryFolderDTO `json:"folder"`
	ScanStarted bool             `json:"scanStarted"`
}

// RemoveFolderResponse is the JSON response for DELETE /api/folders/:id
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
	Theme    string `json:"theme"`
	Language string `json:"language"`
	TrashDir string `json:"trashDir"`
}

// UpdateSettingsRequest is the JSON request for PUT /api/settings
type UpdateSettingsRequest struct {
	Theme    string  `json:"theme"`
	Language string  `json:"language"`
	TrashDir *string `json:"trashDir"`
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
