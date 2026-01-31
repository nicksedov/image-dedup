package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
}

// DuplicateGroupView represents a duplicate group for template rendering
type DuplicateGroupView struct {
	Index     int
	Hash      string
	Size      int64
	SizeHuman string
	Files     []FileView
	Thumbnail string
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
	groups, err := findDuplicates(s.db)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to find duplicates: %v", err)
		return
	}

	var groupViews []DuplicateGroupView
	totalFiles := 0

	for i, g := range groups {
		var fileViews []FileView
		var thumbnail string

		for j, f := range g.Files {
			// Generate thumbnail from the first valid file
			if j == 0 {
				thumb, err := generateThumbnail(f.Path, s.thumbnailCache)
				if err == nil {
					thumbnail = thumb
				}
			}

			fileViews = append(fileViews, FileView{
				ID:       f.ID,
				Path:     f.Path,
				FileName: filepath.Base(f.Path),
				DirPath:  filepath.Dir(f.Path),
				ModTime:  f.ModTime.Format("2006-01-02 15:04:05"),
			})
			totalFiles++
		}

		groupViews = append(groupViews, DuplicateGroupView{
			Index:     i + 1,
			Hash:      g.Hash,
			Size:      g.Size,
			SizeHuman: formatSize(g.Size),
			Files:     fileViews,
			Thumbnail: thumbnail,
		})
	}

	data := TemplateData{
		Groups:       groupViews,
		TotalFiles:   totalFiles,
		TotalGroups:  len(groupViews),
		ScannedDirs:  s.scanDirs,
		LastScanTime: time.Now().Format("2006-01-02 15:04:05"),
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
	FilePaths []string `json:"filePaths"`
	OutputDir string   `json:"outputDir"`
	TrashDir  string   `json:"trashDir"`
}

// handleGenerateScript generates a bash script for moving files
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

	// Generate bash script
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n\n")
	sb.WriteString("# Image Dedup - File Removal Script\n")
	sb.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("# Files to move: %d\n\n", len(req.FilePaths)))

	// Create trash directory
	sb.WriteString("# Create trash directory\n")
	sb.WriteString(fmt.Sprintf("TRASH_DIR=\"%s\"\n", req.TrashDir))
	sb.WriteString("mkdir -p \"$TRASH_DIR\"\n\n")

	sb.WriteString("# Move files to trash\n")
	for _, path := range req.FilePaths {
		// Escape special characters in path
		escapedPath := strings.ReplaceAll(path, "\"", "\\\"")
		escapedPath = strings.ReplaceAll(escapedPath, "$", "\\$")

		// Generate unique destination name to avoid overwrites
		baseName := filepath.Base(path)
		sb.WriteString(fmt.Sprintf("mv \"%s\" \"$TRASH_DIR/%s\" 2>/dev/null && echo \"Moved: %s\" || echo \"Failed: %s\"\n",
			escapedPath, baseName, baseName, baseName))
	}

	sb.WriteString("\necho \"Done! Moved files are in: $TRASH_DIR\"\n")

	script := sb.String()

	// Save to file
	scriptPath := filepath.Join(req.OutputDir, "remove_duplicates.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save script: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Script generated successfully",
		"scriptPath": scriptPath,
		"fileCount":  len(req.FilePaths),
	})
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
