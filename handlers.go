package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/charmap"
	"gorm.io/gorm"
)

// Server holds the application state
type Server struct {
	db             *gorm.DB
	thumbnailCache *ThumbnailCache
	scanDirs       []string
}

// NewServer creates a new server instance
func NewServer(db *gorm.DB, scanDirs []string) *Server {
	return &Server{
		db:             db,
		thumbnailCache: NewThumbnailCache(),
		scanDirs:       scanDirs,
	}
}

// TemplateData represents data passed to the HTML template
type TemplateData struct {
	Groups       []DuplicateGroupView
	TotalFiles   int
	TotalGroups  int
	ScannedDirs  []string
	LastScanTime string
	// Pagination
	CurrentPage  int
	PageSize     int
	TotalPages   int
	HasPrevPage  bool
	HasNextPage  bool
	PrevPage     int
	NextPage     int
	PageSizes    []int
}

// DuplicateGroupView represents a duplicate group for template rendering
type DuplicateGroupView struct {
	Index     int
	Hash      string
	Size      int64
	SizeHuman string
	Files     []FileView
	Thumbnail template.URL
}

// FileView represents a file for template rendering
type FileView struct {
	ID       uint
	Path     string
	FileName string
	DirPath  string
	ModTime  string
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

// handleIndex renders the main page
func (s *Server) handleIndex(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	// Validate page size
	validPageSizes := []int{50, 100, 250, 500}
	isValidPageSize := false
	for _, ps := range validPageSizes {
		if pageSize == ps {
			isValidPageSize = true
			break
		}
	}
	if !isValidPageSize {
		pageSize = 50
	}

	if page < 1 {
		page = 1
	}

	// Fetch only the groups needed for this page
	offset := (page - 1) * pageSize
	groups, totalGroups, err := findDuplicatesPaginated(s.db, offset, pageSize)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to find duplicates: %v", err)
		return
	}

	// Calculate pagination
	totalPages := (totalGroups + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Prepare group views with parallel thumbnail generation
	groupViews := make([]DuplicateGroupView, len(groups))
	totalFiles := 0

	// Count total files first (fast operation)
	for _, g := range groups {
		totalFiles += len(g.Files)
	}

	// Generate thumbnails in parallel (up to 16 workers)
	const maxWorkers = 16
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i, g := range groups {
		// Prepare file views (fast, no I/O)
		fileViews := make([]FileView, len(g.Files))
		for j, f := range g.Files {
			fileViews[j] = FileView{
				ID:       f.ID,
				Path:     f.Path,
				FileName: filepath.Base(f.Path),
				DirPath:  filepath.Dir(f.Path),
				ModTime:  f.ModTime.Format("2006-01-02 15:04:05"),
			}
		}

		groupViews[i] = DuplicateGroupView{
			Index:     offset + i + 1,
			Hash:      g.Hash,
			Size:      g.Size,
			SizeHuman: formatSize(g.Size),
			Files:     fileViews,
		}

		// Generate thumbnail in parallel
		if len(g.Files) > 0 {
			wg.Add(1)
			go func(idx int, filePath string) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				thumb, err := generateThumbnail(filePath, s.thumbnailCache)
				if err == nil {
					groupViews[idx].Thumbnail = template.URL(thumb)
				}
			}(i, g.Files[0].Path)
		}
	}

	wg.Wait()

	data := TemplateData{
		Groups:       groupViews,
		TotalFiles:   totalFiles,
		TotalGroups:  totalGroups,
		ScannedDirs:  s.scanDirs,
		LastScanTime: time.Now().Format("2006-01-02 15:04:05"),
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		HasPrevPage:  page > 1,
		HasNextPage:  page < totalPages,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		PageSizes:    validPageSizes,
	}

	c.HTML(http.StatusOK, "index.html", data)
}

// handleScan triggers a new scan of directories
func (s *Server) handleScan(c *gin.Context) {
	progressChan := make(chan string, 100)

	go func() {
		// First cleanup missing files
		cleanupMissingFiles(s.db, progressChan)

		// Then scan all directories
		for _, dir := range s.scanDirs {
			scanDirectory(s.db, dir, progressChan)
		}
		close(progressChan)
	}()

	// Drain the channel (we could implement SSE for real-time progress)
	for range progressChan {
	}

	c.Redirect(http.StatusSeeOther, "/")
}

// GenerateScriptRequest represents the request for script generation
type GenerateScriptRequest struct {
	FilePaths  []string `json:"filePaths"`
	OutputDir  string   `json:"outputDir"`
	TrashDir   string   `json:"trashDir"`
	ScriptType string   `json:"scriptType"` // "bash" or "windows"
}

// handleGenerateScript generates a script for moving files
func (s *Server) handleGenerateScript(c *gin.Context) {
	var req GenerateScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.FilePaths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files selected"})
		return
	}

	if req.OutputDir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Output directory not specified"})
		return
	}

	if req.TrashDir == "" {
		req.TrashDir = filepath.Join(req.OutputDir, "trash")
	}

	if req.ScriptType == "" {
		req.ScriptType = "bash"
	}

	var script string
	var scriptPath string
	var scriptBytes []byte

	if req.ScriptType == "windows" {
		// Convert paths to Windows format (backslashes)
		windowsPaths := make([]string, len(req.FilePaths))
		for i, p := range req.FilePaths {
			windowsPaths[i] = strings.ReplaceAll(p, "/", "\\")
		}
		windowsTrashDir := strings.ReplaceAll(req.TrashDir, "/", "\\")

		script = generateWindowsScript(windowsPaths, windowsTrashDir)
		scriptPath = filepath.Join(req.OutputDir, "remove_duplicates.ps1")

		// Encode script in Windows-1251
		encoder := charmap.Windows1251.NewEncoder()
		encoded, err := encoder.String(script)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to encode script: %v", err)})
			return
		}
		scriptBytes = []byte(encoded)
	} else {
		script = generateBashScript(req.FilePaths, req.TrashDir)
		scriptPath = filepath.Join(req.OutputDir, "remove_duplicates.sh")
		scriptBytes = []byte(script)
	}

	// Save to file
	if err := os.WriteFile(scriptPath, scriptBytes, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save script: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Script generated successfully",
		"scriptPath": scriptPath,
		"fileCount":  len(req.FilePaths),
	})
}

// generateBashScript creates a bash script for Unix/Linux/macOS
func generateBashScript(filePaths []string, trashDir string) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n\n")
	sb.WriteString("# Image Dedup - File Removal Script\n")
	sb.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("# Files to move: %d\n\n", len(filePaths)))

	// Create trash directory
	sb.WriteString("# Create trash directory\n")
	sb.WriteString(fmt.Sprintf("TRASH_DIR=\"%s\"\n", trashDir))
	sb.WriteString("mkdir -p \"$TRASH_DIR\"\n\n")

	sb.WriteString("# Move files to trash\n")
	for _, path := range filePaths {
		// Escape special characters in path
		escapedPath := strings.ReplaceAll(path, "\"", "\\\"")
		escapedPath = strings.ReplaceAll(escapedPath, "$", "\\$")

		// Generate unique destination name to avoid overwrites
		baseName := filepath.Base(path)
		sb.WriteString(fmt.Sprintf("mv \"%s\" \"$TRASH_DIR/%s\" 2>/dev/null && echo \"Moved: %s\" || echo \"Failed: %s\"\n",
			escapedPath, baseName, baseName, baseName))
	}

	sb.WriteString("\necho \"Done! Moved files are in: $TRASH_DIR\"\n")
	return sb.String()
}

// generateWindowsScript creates a PowerShell script for Windows
func generateWindowsScript(filePaths []string, trashDir string) string {
	var sb strings.Builder
	sb.WriteString("# Image Dedup - File Removal Script (PowerShell)\n")
	sb.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("# Files to move: %d\n\n", len(filePaths)))

	// Create trash directory
	sb.WriteString("# Create trash directory\n")
	// Escape backslashes for PowerShell string
	escapedTrashDir := strings.ReplaceAll(trashDir, "'", "''")
	sb.WriteString(fmt.Sprintf("$TrashDir = '%s'\n", escapedTrashDir))
	sb.WriteString("if (-not (Test-Path -Path $TrashDir)) {\n")
	sb.WriteString("    New-Item -ItemType Directory -Path $TrashDir -Force | Out-Null\n")
	sb.WriteString("}\n\n")

	sb.WriteString("# Move files to trash\n")
	for _, path := range filePaths {
		// Escape single quotes for PowerShell
		escapedPath := strings.ReplaceAll(path, "'", "''")
		baseName := filepath.Base(path)
		escapedBaseName := strings.ReplaceAll(baseName, "'", "''")

		sb.WriteString(fmt.Sprintf("try {\n"))
		sb.WriteString(fmt.Sprintf("    Move-Item -Path '%s' -Destination (Join-Path $TrashDir '%s') -Force\n", escapedPath, escapedBaseName))
		sb.WriteString(fmt.Sprintf("    Write-Host \"Moved: %s\" -ForegroundColor Green\n", baseName))
		sb.WriteString(fmt.Sprintf("} catch {\n"))
		sb.WriteString(fmt.Sprintf("    Write-Host \"Failed: %s - $_\" -ForegroundColor Red\n", baseName))
		sb.WriteString(fmt.Sprintf("}\n\n"))
	}

	sb.WriteString("Write-Host \"\"\n")
	sb.WriteString("Write-Host \"Done! Moved files are in: $TrashDir\" -ForegroundColor Cyan\n")
	sb.WriteString("Write-Host \"Press any key to exit...\"\n")
	sb.WriteString("$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')\n")
	return sb.String()
}

// handleThumbnail serves a thumbnail for a specific file
func (s *Server) handleThumbnail(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.String(http.StatusBadRequest, "Path required")
		return
	}

	thumbnail, err := generateThumbnail(path, s.thumbnailCache)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate thumbnail: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"thumbnail": thumbnail})
}

// SetupRouter sets up the Gin router with all routes
func (s *Server) SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Load HTML templates
	r.SetHTMLTemplate(template.Must(template.ParseFiles("templates/index.html")))

	// Routes
	r.GET("/", s.handleIndex)
	r.POST("/scan", s.handleScan)
	r.POST("/generate-script", s.handleGenerateScript)
	r.GET("/thumbnail", s.handleThumbnail)

	return r
}
