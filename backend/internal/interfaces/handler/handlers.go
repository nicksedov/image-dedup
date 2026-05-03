package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"image-toolkit/internal/application/imaging"
	"image-toolkit/internal/application/thumbnail"
	"image-toolkit/internal/domain"
	"image-toolkit/internal/infrastructure/llm"
	"image-toolkit/internal/interfaces/dto"
	"image-toolkit/internal/interfaces/i18n"
	"image-toolkit/internal/interfaces/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// handleGetDuplicates returns paginated duplicate groups as JSON
func (s *Server) handleGetDuplicates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

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

	offset := (page - 1) * pageSize
	groups, totalGroups, totalFiles, err := imaging.FindDuplicatesPaginated(s.db, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgScanDuplicateFailed))
		return
	}

	totalPages := (totalGroups + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Prepare group DTOs with parallel thumbnail generation
	groupDTOs := make([]dto.DuplicateGroupDTO, len(groups))
	pageFiles := 0

	for _, g := range groups {
		pageFiles += len(g.Files)
	}

	const maxWorkers = 16
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i, g := range groups {
		fileDTOs := make([]dto.FileDTO, len(g.Files))
		for j, f := range g.Files {
			fileDTOs[j] = dto.FileDTO{
				ID:       f.ID,
				Path:     f.Path,
				FileName: filepath.Base(f.Path),
				DirPath:  filepath.Dir(f.Path),
				ModTime:  f.ModTime.Format("2006-01-02 15:04:05"),
			}
		}

		groupDTOs[i] = dto.DuplicateGroupDTO{
			Index:     offset + i + 1,
			Hash:      g.Hash,
			Size:      g.Size,
			SizeHuman: formatSize(g.Size),
			Files:     fileDTOs,
		}

		if len(g.Files) > 0 {
			wg.Add(1)
			go func(idx int, filePath string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				var thumb string
				var err error

				// Use thumbnail service if available
				if s.thumbnailService != nil {
					thumb, err = s.thumbnailService.GetOrGenerate(filePath)
				} else {
					thumb, err = imaging.GenerateThumbnail(filePath, s.thumbnailCache)
				}

				if err == nil {
					groupDTOs[idx].Thumbnail = thumb
				}
			}(i, g.Files[0].Path)
		}
	}

	wg.Wait()

	// Get scanned dirs from gallery folders
	var galleryFolders []domain.GalleryFolder
	s.db.Order("created_at").Find(&galleryFolders)
	scannedDirs := make([]string, len(galleryFolders))
	for i, f := range galleryFolders {
		scannedDirs[i] = f.Path
	}

	response := dto.DuplicatesResponse{
		Groups:      groupDTOs,
		TotalFiles:  totalFiles,
		PageFiles:   pageFiles,
		TotalGroups: totalGroups,
		ScannedDirs: scannedDirs,
		CurrentPage: page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasPrevPage: page > 1,
		HasNextPage: page < totalPages,
		PageSizes:   validPageSizes,
	}

	c.JSON(http.StatusOK, response)
}

// handleScan triggers an async scan of directories
func (s *Server) handleScan(c *gin.Context) {
	if err := s.scanManager.StartScan(); err != nil {
		c.JSON(http.StatusConflict, i18n.ErrorResponse(i18n.MsgScanFailed))
		return
	}
	c.JSON(http.StatusAccepted, dto.ScanResponse{Message: string(i18n.MsgScanStarted)})
}

// handleFastScan triggers an async fast scan of directories
// Fast scan only computes hash when file record doesn't exist or size differs
func (s *Server) handleFastScan(c *gin.Context) {
	result := s.scanManager.FastScanGallery()
	c.JSON(http.StatusOK, dto.FastScanResponse{
		Message:   string(i18n.MsgScanStarted),
		Unchanged: result.Unchanged,
		Modified:  result.Modified,
		Created:   result.Created,
		Deleted:   result.Deleted,
		Total:     result.TotalChecked,
	})
}

// handleGetStatus returns the current scan status
func (s *Server) handleGetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.scanManager.GetStatus())
}

// handleThumbnail serves a thumbnail for a specific file
func (s *Server) handleThumbnail(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImagePathRequired))
		return
	}

	var thumbnail string
	var err error

	// Use thumbnail service if available
	if s.thumbnailService != nil {
		thumbnail, err = s.thumbnailService.GetOrGenerate(path)
	} else {
		thumbnail, err = imaging.GenerateThumbnail(path, s.thumbnailCache)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgImageThumbnailFailed))
		return
	}

	c.JSON(http.StatusOK, dto.ThumbnailResponse{Thumbnail: thumbnail})
}

// handleDeleteFiles deletes selected files directly (moves to trash)
func (s *Server) handleDeleteFiles(c *gin.Context) {
	var req dto.DeleteFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	if len(req.FilePaths) == 0 {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgScanNoFilesSelected))
		return
	}

	var successCount, failedCount int
	var failedFiles []string

	if req.TrashDir != "" {
		if err := os.MkdirAll(req.TrashDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgScanTrashDirFailed))
			return
		}

		for _, filePath := range req.FilePaths {
			baseName := filepath.Base(filePath)
			destPath := filepath.Join(req.TrashDir, baseName)

			if _, err := os.Stat(destPath); err == nil {
				ext := filepath.Ext(baseName)
				nameWithoutExt := strings.TrimSuffix(baseName, ext)
				destPath = filepath.Join(req.TrashDir, nameWithoutExt+"_"+time.Now().Format("20060102_150405")+ext)
			}

			if err := os.Rename(filePath, destPath); err != nil {
				failedCount++
				failedFiles = append(failedFiles, baseName+": "+err.Error())
				continue
			}

			s.db.Where("path = ?", filepath.ToSlash(filePath)).Delete(&domain.ImageFile{})
			successCount++
		}
	} else {
		for _, filePath := range req.FilePaths {
			baseName := filepath.Base(filePath)

			if err := os.Remove(filePath); err != nil {
				failedCount++
				failedFiles = append(failedFiles, baseName+": "+err.Error())
				continue
			}

			s.db.Where("path = ?", filepath.ToSlash(filePath)).Delete(&domain.ImageFile{})
			successCount++
		}
	}

	c.JSON(http.StatusOK, dto.DeleteFilesResponse{
		Success:     successCount,
		Failed:      failedCount,
		FailedFiles: failedFiles,
	})
}

// handleGetFolderPatterns returns all unique folder patterns from duplicates
func (s *Server) handleGetFolderPatterns(c *gin.Context) {
	groups, _, _, err := imaging.FindDuplicatesPaginated(s.db, 0, 100000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgScanDuplicateFailed))
		return
	}

	patternMap := make(map[string]*dto.FolderPattern)

	for _, group := range groups {
		folderSet := make(map[string]bool)
		for _, file := range group.Files {
			dir := filepath.Dir(file.Path)
			folderSet[dir] = true
		}

		folders := make([]string, 0, len(folderSet))
		for folder := range folderSet {
			folders = append(folders, folder)
		}

		sortStrings(folders)

		patternID := createPatternID(folders)

		if existing, ok := patternMap[patternID]; ok {
			existing.DuplicateCount++
			existing.TotalFiles += len(group.Files)
		} else {
			patternMap[patternID] = &dto.FolderPattern{
				ID:             patternID,
				Folders:        folders,
				DuplicateCount: 1,
				TotalFiles:     len(group.Files),
			}
		}
	}

	patterns := make([]dto.FolderPattern, 0, len(patternMap))
	for _, p := range patternMap {
		patterns = append(patterns, *p)
	}

	sortPatternsByCount(patterns)

	c.JSON(http.StatusOK, dto.FolderPatternsResponse{Patterns: patterns})
}

// handleBatchDelete applies batch deletion rules to all matching duplicates
func (s *Server) handleBatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	if len(req.Rules) == 0 {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	ruleMap := make(map[string]string)
	for _, rule := range req.Rules {
		ruleMap[rule.PatternID] = rule.KeepFolder
	}

	groups, _, _, err := imaging.FindDuplicatesPaginated(s.db, 0, 100000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgScanDuplicateFailed))
		return
	}

	var successCount, failedCount int
	var failedFiles []string

	if req.TrashDir != "" {
		if err := os.MkdirAll(req.TrashDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgScanTrashDirFailed))
			return
		}
	}

	for _, group := range groups {
		folderSet := make(map[string]bool)
		for _, file := range group.Files {
			dir := filepath.Dir(file.Path)
			folderSet[dir] = true
		}

		folders := make([]string, 0, len(folderSet))
		for folder := range folderSet {
			folders = append(folders, folder)
		}
		sortStrings(folders)

		patternID := createPatternID(folders)

		keepFolder, hasRule := ruleMap[patternID]
		if !hasRule {
			continue
		}

		for _, file := range group.Files {
			fileDir := filepath.Dir(file.Path)
			if fileDir == keepFolder {
				continue
			}

			if req.TrashDir != "" {
				baseName := filepath.Base(file.Path)
				destPath := filepath.Join(req.TrashDir, baseName)

				if _, err := os.Stat(destPath); err == nil {
					ext := filepath.Ext(baseName)
					nameWithoutExt := strings.TrimSuffix(baseName, ext)
					destPath = filepath.Join(req.TrashDir, nameWithoutExt+"_"+time.Now().Format("20060102_150405_000")+ext)
				}

				if err := os.Rename(file.Path, destPath); err != nil {
					failedCount++
					failedFiles = append(failedFiles, filepath.Base(file.Path)+": "+err.Error())
					continue
				}
			} else {
				if err := os.Remove(file.Path); err != nil {
					failedCount++
					failedFiles = append(failedFiles, filepath.Base(file.Path)+": "+err.Error())
					continue
				}
			}

			s.db.Where("path = ?", filepath.ToSlash(file.Path)).Delete(&domain.ImageFile{})
			successCount++
		}
	}

	c.JSON(http.StatusOK, dto.BatchDeleteResponse{
		Success:     successCount,
		Failed:      failedCount,
		FailedFiles: failedFiles,
	})
}

// --- Gallery Folder Handlers ---

// handleGetFolders returns all gallery folders
func (s *Server) handleGetFolders(c *gin.Context) {
	var folders []domain.GalleryFolder
	s.db.Order("created_at").Find(&folders)

	folderDTOs := make([]dto.GalleryFolderDTO, len(folders))
	for i, f := range folders {
		var count int64
		prefix := f.Path + "/"
		s.db.Model(&domain.ImageFile{}).Where("path LIKE ?", prefix+"%").Count(&count)

		folderDTOs[i] = dto.GalleryFolderDTO{
			ID:        f.ID,
			Path:      f.Path,
			FileCount: int(count),
			CreatedAt: f.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, dto.GalleryFoldersResponse{
		Folders:      folderDTOs,
		TotalFolders: len(folderDTOs),
	})
}

// handleAddFolder adds a new gallery folder and triggers a scan
func (s *Server) handleAddFolder(c *gin.Context) {
	var req dto.AddFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgFolderPathRequired))
		return
	}

	// Validate directory exists
	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgFolderInvalidPath))
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgFolderCannotAccessPath))
		return
	}
	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgFolderNotDirectory))
		return
	}

	normalizedPath := filepath.ToSlash(absPath)

	// Check conflict with trash directory
	var settings domain.AppSettings
	if result := s.db.First(&settings, 1); result.Error == nil && settings.TrashDir != "" {
		if reason := pathsConflict(normalizedPath, settings.TrashDir); reason != "" {
			c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgFolderConflictTrash))
			return
		}
	}

	folder := domain.GalleryFolder{Path: normalizedPath}
	if result := s.db.Create(&folder); result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate") || strings.Contains(result.Error.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, i18n.ErrorResponse(i18n.MsgFolderAlreadyInGallery))
			return
		}
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgFolderAddFailed))
		return
	}

	// Trigger background scan for this folder
	scanStarted := true
	if err := s.scanManager.ScanSingleDir(normalizedPath); err != nil {
		scanStarted = false
	}

	c.JSON(http.StatusOK, dto.AddFolderResponse{
		Message: string(i18n.MsgFolderAdded),
		Folder: dto.GalleryFolderDTO{
			ID:        folder.ID,
			Path:      folder.Path,
			FileCount: 0,
			CreatedAt: folder.CreatedAt.Format("2006-01-02 15:04:05"),
		},
		ScanStarted: scanStarted,
	})
}

// handleRemoveFolder removes a gallery folder and its files from the database
func (s *Server) handleRemoveFolder(c *gin.Context) {
	id := c.Param("id")

	var folder domain.GalleryFolder
	if result := s.db.First(&folder, id); result.Error != nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgFolderNotFound))
		return
	}

	// Delete all image files under this folder
	prefix := folder.Path + "/"
	result := s.db.Where("path LIKE ?", prefix+"%").Delete(&domain.ImageFile{})
	filesRemoved := int(result.RowsAffected)

	// Delete the folder record
	s.db.Delete(&folder)

	c.JSON(http.StatusOK, dto.RemoveFolderResponse{
		Message:      string(i18n.MsgFolderRemoved),
		FilesRemoved: filesRemoved,
	})
}

// handleGetGalleryImages returns paginated gallery images
func (s *Server) handleGetGalleryImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	view := c.DefaultQuery("view", "list")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var totalImages int64
	s.db.Model(&domain.ImageFile{}).Count(&totalImages)

	totalPages := (int(totalImages) + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * pageSize

	var files []domain.ImageFile
	s.db.Order("path").Offset(offset).Limit(pageSize).Find(&files)

	imageDTOs := make([]dto.GalleryImageDTO, len(files))
	for i, f := range files {
		imageDTOs[i] = dto.GalleryImageDTO{
			ID:        f.ID,
			Path:      f.Path,
			FileName:  filepath.Base(f.Path),
			DirPath:   filepath.Dir(f.Path),
			Size:      f.Size,
			SizeHuman: formatSize(f.Size),
			ModTime:   f.ModTime.Format("2006-01-02 15:04:05"),
		}
	}

	// Generate thumbnails in parallel if thumbnail or folders view
	if (view == "thumbnails" || view == "folders") && len(files) > 0 {
		const maxWorkers = 16
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, maxWorkers)

		for i, f := range files {
			wg.Add(1)
			go func(idx int, filePath string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				var thumb string
				var err error

				// Use thumbnail service if available
				if s.thumbnailService != nil {
					thumb, err = s.thumbnailService.GetOrGenerate(filePath)
				} else {
					thumb, err = imaging.GenerateThumbnail(filePath, s.thumbnailCache)
				}

				if err == nil {
					imageDTOs[idx].Thumbnail = thumb
				}
			}(i, f.Path)
		}
		wg.Wait()
	}

	c.JSON(http.StatusOK, dto.GalleryImagesResponse{
		Images:      imageDTOs,
		TotalImages: int(totalImages),
		CurrentPage: page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNextPage: page < totalPages,
	})
}

// handleServeImage serves a full-size image file
func (s *Server) handleServeImage(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImagePathRequired))
		return
	}

	// Security: verify the path is within a gallery folder
	var folders []domain.GalleryFolder
	s.db.Find(&folders)

	allowed := false
	for _, f := range folders {
		if strings.HasPrefix(path, f.Path+"/") || strings.HasPrefix(path, f.Path+"\\") {
			allowed = true
			break
		}
	}
	if !allowed {
		c.JSON(http.StatusForbidden, i18n.ErrorResponse(i18n.MsgImageAccessDenied))
		return
	}

	// Convert slash path to OS path for file serving
	osPath := filepath.FromSlash(path)

	if _, err := os.Stat(osPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgImageNotFound))
		return
	}

	c.File(osPath)
}

// handleServeOcrImage serves an image scaled and rotated for OCR overlay display
func (s *Server) handleServeOcrImage(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImagePathRequired))
		return
	}

	angleStr := c.DefaultQuery("angle", "0")
	scaleFactorStr := c.DefaultQuery("scaleFactor", "1")

	angle, err := strconv.ParseFloat(angleStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImagePathRequired))
		return
	}

	scaleFactor, err := strconv.ParseFloat(scaleFactorStr, 64)
	if err != nil || scaleFactor <= 0 {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImagePathRequired))
		return
	}

	// Security: verify the path is within a gallery folder
	var folders []domain.GalleryFolder
	s.db.Find(&folders)

	allowed := false
	for _, f := range folders {
		if strings.HasPrefix(path, f.Path+"/") || strings.HasPrefix(path, f.Path+"\\") {
			allowed = true
			break
		}
	}
	if !allowed {
		c.JSON(http.StatusForbidden, i18n.ErrorResponse(i18n.MsgImageAccessDenied))
		return
	}

	osPath := filepath.FromSlash(path)

	data, err := imaging.PrepareOcrImage(osPath, scaleFactor, angle)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgImageNotFound))
		} else {
			c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgImageThumbnailFailed))
		}
		return
	}

	c.Data(http.StatusOK, "image/webp", data)
}

// --- App Settings Handlers ---

// handleGetSettings returns the current application settings
func (s *Server) handleGetSettings(c *gin.Context) {
	var settings domain.AppSettings
	if result := s.db.First(&settings, 1); result.Error != nil {
		c.JSON(http.StatusOK, dto.AppSettingsDTO{TrashDir: ""})
		return
	}
	c.JSON(http.StatusOK, dto.AppSettingsDTO{
		TrashDir:           settings.TrashDir,
		ThumbnailCachePath: settings.ThumbnailCachePath,
		ThumbnailCacheSize: settings.ThumbnailCacheSize,
	})
}

// handleUpdateSettings updates the application settings
func (s *Server) handleUpdateSettings(c *gin.Context) {
	var req dto.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	var settings domain.AppSettings
	if result := s.db.First(&settings, 1); result.Error != nil {
		settings = domain.AppSettings{ID: 1}
	}

	if req.TrashDir != nil {
		newTrashDir := strings.TrimSpace(*req.TrashDir)
		if newTrashDir != "" {
			// Normalize the trash dir path
			absTrash, err := filepath.Abs(newTrashDir)
			if err != nil {
				c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImageInvalidTrashPath))
				return
			}
			normalizedTrash := filepath.ToSlash(absTrash)

			// Check conflict with all gallery folders
			var galleryFolders []domain.GalleryFolder
			s.db.Find(&galleryFolders)
			for _, gf := range galleryFolders {
				if reason := pathsConflict(normalizedTrash, gf.Path); reason != "" {
					c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImageTrashConflict))
					return
				}
			}
			settings.TrashDir = normalizedTrash
		} else {
			settings.TrashDir = ""
		}
	}
	if req.ThumbnailCachePath != nil {
		newCachePath := strings.TrimSpace(*req.ThumbnailCachePath)
		if newCachePath != "" {
			// Normalize the cache path
			absCache, err := filepath.Abs(newCachePath)
			if err != nil {
				c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImageInvalidTrashPath))
				return
			}
			normalizedCache := filepath.ToSlash(absCache)
			settings.ThumbnailCachePath = normalizedCache

			// Update thumbnail service if available
			if s.thumbnailService != nil {
				log.Printf("Updating thumbnail cache path from %s to %s", s.thumbnailService.Stats().CacheDir, normalizedCache)
				if err := s.thumbnailService.UpdateCachePath(normalizedCache); err != nil {
					log.Printf("Failed to update thumbnail cache path: %v", err)
				} else {
					log.Printf("Thumbnail cache path updated successfully. New stats: %+v", s.thumbnailService.Stats())
				}
			} else {
				log.Printf("Thumbnail service is nil, cannot update cache path")
			}
		} else {
			settings.ThumbnailCachePath = ""
		}
	}

	s.db.Save(&settings)

	c.JSON(http.StatusOK, dto.AppSettingsDTO{
		TrashDir:           settings.TrashDir,
		ThumbnailCachePath: settings.ThumbnailCachePath,
		ThumbnailCacheSize: settings.ThumbnailCacheSize,
	})
}

// --- User Settings Handlers ---

// handleGetUserSettings returns the current user's settings
func (s *Server) handleGetUserSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, i18n.ErrorResponse(i18n.MsgAuthUnauthorized))
		return
	}

	var settings domain.UserSettings
	if result := s.db.FirstOrCreate(&settings, domain.UserSettings{UserID: user.ID}); result.Error != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgAuthInternalError))
		return
	}

	c.JSON(http.StatusOK, dto.UserSettingsDTO{
		Theme:    settings.Theme,
		Language: settings.Language,
	})
}

// handleUpdateUserSettings updates the current user's settings
func (s *Server) handleUpdateUserSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, i18n.ErrorResponse(i18n.MsgAuthUnauthorized))
		return
	}

	var req dto.UpdateUserSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	validThemes := map[string]bool{
		"light-purple":  true,
		"dark-purple":   true,
		"light-green":   true,
		"dark-green":    true,
		"light-blue":    true,
		"dark-blue":     true,
		"light-orange":  true,
		"dark-orange":   true,
		"dark-contrast": true,
	}
	validLanguages := map[string]bool{"en": true, "ru": true}

	if req.Theme != "" && !validThemes[req.Theme] {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImageInvalidTheme))
		return
	}
	if req.Language != "" && !validLanguages[req.Language] {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImageInvalidLanguage))
		return
	}

	var settings domain.UserSettings
	result := s.db.FirstOrCreate(&settings, domain.UserSettings{UserID: user.ID})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgAuthInternalError))
		return
	}

	if req.Theme != "" {
		settings.Theme = req.Theme
	}
	if req.Language != "" {
		settings.Language = req.Language
	}

	s.db.Save(&settings)

	c.JSON(http.StatusOK, dto.UserSettingsDTO{
		Theme:    settings.Theme,
		Language: settings.Language,
	})
}

// handleGetTrashInfo returns information about files in the trash directory
func (s *Server) handleGetTrashInfo(c *gin.Context) {
	var settings domain.AppSettings
	if result := s.db.First(&settings, 1); result.Error != nil || settings.TrashDir == "" {
		c.JSON(http.StatusOK, dto.TrashInfoResponse{FileCount: 0, TotalSize: 0, TotalSizeHuman: "0 B"})
		return
	}

	info, err := os.Stat(settings.TrashDir)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusOK, dto.TrashInfoResponse{FileCount: 0, TotalSize: 0, TotalSizeHuman: "0 B"})
		return
	}

	entries, err := os.ReadDir(settings.TrashDir)
	if err != nil {
		c.JSON(http.StatusOK, dto.TrashInfoResponse{FileCount: 0, TotalSize: 0, TotalSizeHuman: "0 B"})
		return
	}

	var fileCount int
	var totalSize int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileCount++
		if fi, err := entry.Info(); err == nil {
			totalSize += fi.Size()
		}
	}

	c.JSON(http.StatusOK, dto.TrashInfoResponse{
		FileCount:      fileCount,
		TotalSize:      totalSize,
		TotalSizeHuman: formatSize(totalSize),
	})
}

// handleCleanTrash removes all files from the trash directory
func (s *Server) handleCleanTrash(c *gin.Context) {
	var settings domain.AppSettings
	if result := s.db.First(&settings, 1); result.Error != nil || settings.TrashDir == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgTrashNotConfigured))
		return
	}

	info, err := os.Stat(settings.TrashDir)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgTrashNotExists))
		return
	}

	entries, err := os.ReadDir(settings.TrashDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgTrashReadFailed))
		return
	}

	var deleted, failed int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(settings.TrashDir, entry.Name())
		if err := os.Remove(filePath); err != nil {
			failed++
		} else {
			deleted++
		}
	}

	c.JSON(http.StatusOK, dto.CleanTrashResponse{
		Deleted: deleted,
		Failed:  failed,
	})
}

// handleGetImageMetadata returns EXIF metadata for a single image
func (s *Server) handleGetImageMetadata(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path required"})
		return
	}

	// Find the image file in DB
	var imageFile domain.ImageFile
	if result := s.db.Where("path = ?", path).First(&imageFile); result.Error != nil {
		c.JSON(http.StatusOK, dto.ImageMetadataResponse{Found: false})
		return
	}

	// Find metadata for this image
	var meta domain.ImageMetadata
	if result := s.db.Where("image_file_id = ?", imageFile.ID).First(&meta); result.Error != nil {
		c.JSON(http.StatusOK, dto.ImageMetadataResponse{Found: false})
		return
	}

	// Build the DTO
	metaDTO := &dto.ImageMetadataDTO{
		Width:        meta.Width,
		Height:       meta.Height,
		Dimensions:   fmt.Sprintf("%d \u00d7 %d", meta.Width, meta.Height),
		CameraModel:  meta.CameraModel,
		LensModel:    meta.LensModel,
		ISO:          meta.ISO,
		Aperture:     meta.Aperture,
		ShutterSpeed: meta.ShutterSpeed,
		FocalLength:  meta.FocalLength,
		Orientation:  meta.Orientation,
		ColorSpace:   meta.ColorSpace,
		Software:     meta.Software,
		GPSLatitude:  meta.GPSLatitude,
		GPSLongitude: meta.GPSLongitude,
		GeoCountry:   meta.GeoCountry,
		GeoCity:      meta.GeoCity,
		HasGPS:       meta.GPSLatitude != nil && meta.GPSLongitude != nil,
		HasExif:      imaging.HasExifData(&meta),
	}

	if meta.DateTaken != nil {
		metaDTO.DateTaken = meta.DateTaken.Format("2006-01-02 15:04:05")
	}

	c.JSON(http.StatusOK, dto.ImageMetadataResponse{Found: true, Metadata: metaDTO})
}

// handleGetMetadataStatus returns the current metadata extraction status
func (s *Server) handleGetMetadataStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.metadataManager.GetStatus())
}

// handleGetGalleryCalendar returns paginated gallery images grouped by date taken
func (s *Server) handleGetGalleryCalendar(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	startDate := c.Query("startDate") // "YYYY-MM-DD" or empty
	endDate := c.Query("endDate")     // "YYYY-MM-DD" or empty
	monthYear := c.Query("monthYear") // "YYYY-MM" for calendar widget

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	// Query: join image_files with image_metadata where date_taken is not null
	// Order by date_taken DESC (newest first)
	type imageWithDate struct {
		domain.ImageFile
		DateTaken time.Time
	}

	query := s.db.Table("image_files").
		Select("image_files.*, image_metadata.date_taken").
		Joins("INNER JOIN image_metadata ON image_metadata.image_file_id = image_files.id").
		Where("image_metadata.date_taken IS NOT NULL")

	// Apply date range filter
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("image_metadata.date_taken >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// End of the end date
			endOfDay := t.Add(24*time.Hour - time.Second)
			query = query.Where("image_metadata.date_taken <= ?", endOfDay)
		}
	}

	// Count total
	var totalImages int64
	query.Count(&totalImages)

	// Paginate
	offset := (page - 1) * pageSize
	var results []imageWithDate
	query.Order("image_metadata.date_taken ASC").Offset(offset).Limit(pageSize).Find(&results)

	// Group by date
	type dateGroup struct {
		date   time.Time
		images []domain.ImageFile
	}
	groupsMap := make(map[string]*dateGroup)
	var dateOrder []string

	for _, r := range results {
		dateStr := r.DateTaken.Format("2006-01-02")
		if _, ok := groupsMap[dateStr]; !ok {
			groupsMap[dateStr] = &dateGroup{date: r.DateTaken}
			dateOrder = append(dateOrder, dateStr)
		}
		groupsMap[dateStr].images = append(groupsMap[dateStr].images, r.ImageFile)
	}

	// Build response groups
	groupDTOs := make([]dto.CalendarDateGroup, 0, len(dateOrder))
	for _, dateStr := range dateOrder {
		g := groupsMap[dateStr]
		imageDTOs := make([]dto.GalleryImageDTO, len(g.images))
		for i, f := range g.images {
			imageDTOs[i] = dto.GalleryImageDTO{
				ID:        f.ID,
				Path:      f.Path,
				FileName:  filepath.Base(f.Path),
				DirPath:   filepath.Dir(f.Path),
				Size:      f.Size,
				SizeHuman: formatSize(f.Size),
				ModTime:   f.ModTime.Format("2006-01-02 15:04:05"),
			}
		}

		// Generate thumbnails in parallel
		if len(g.images) > 0 {
			const maxWorkers = 16
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, maxWorkers)

			for i, f := range g.images {
				wg.Add(1)
				go func(idx int, filePath string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					thumb, err := imaging.GenerateThumbnail(filePath, s.thumbnailCache)
					if err == nil {
						imageDTOs[idx].Thumbnail = thumb
					}
				}(i, f.Path)
			}
			wg.Wait()
		}

		// Human-readable label
		label := g.date.Format("Monday, January 2, 2006")

		groupDTOs = append(groupDTOs, dto.CalendarDateGroup{
			Date:       dateStr,
			Label:      label,
			ImageCount: len(g.images),
			Images:     imageDTOs,
		})
	}

	// Get date range
	var dateRange dto.CalendarDateRange
	var minDate, maxDate *time.Time
	s.db.Raw("SELECT MIN(date_taken), MAX(date_taken) FROM image_metadata WHERE date_taken IS NOT NULL").Row().Scan(&minDate, &maxDate)
	if minDate != nil {
		dateRange.MinDate = minDate.Format("2006-01-02")
	}
	if maxDate != nil {
		dateRange.MaxDate = maxDate.Format("2006-01-02")
	}
	dateRange.TotalWithDate = int(totalImages)

	// Get month info for calendar widget
	var months []dto.CalendarMonthInfo
	if monthYear != "" {
		if t, err := time.Parse("2006-01", monthYear); err == nil {
			year := t.Year()
			month := int(t.Month())
			nextMonth := t.AddDate(0, 1, 0)

			// Get distinct days that have images in this month (PostgreSQL)
			var days []int
			s.db.Raw(`
				SELECT DISTINCT CAST(EXTRACT(DAY FROM date_taken) AS INTEGER) as day
				FROM image_metadata
				WHERE date_taken >= $1 AND date_taken < $2 AND date_taken IS NOT NULL
				ORDER BY day
			`, t, nextMonth).Pluck("day", &days)

			months = append(months, dto.CalendarMonthInfo{
				Year:  year,
				Month: month,
				Days:  days,
			})
		}
	}

	totalPages := (int(totalImages) + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, dto.GalleryCalendarResponse{
		Groups:      groupDTOs,
		TotalImages: int(totalImages),
		TotalGroups: len(groupDTOs),
		HasMore:     page < totalPages,
		DateRange:   dateRange,
		Months:      months,
	})
}

// handleGetCalendarMonthInfo returns days with image counts for a specific month (lightweight, no thumbnails)
func (s *Server) handleGetCalendarMonthInfo(c *gin.Context) {
	monthYear := c.Query("monthYear") // "YYYY-MM"
	if monthYear == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "monthYear parameter is required"})
		return
	}

	t, err := time.Parse("2006-01", monthYear)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monthYear format. Use YYYY-MM"})
		return
	}

	year := t.Year()
	month := int(t.Month())
	nextMonth := t.AddDate(0, 1, 0)

	// Get day-level counts: how many images per day in this month (PostgreSQL)
	type dayCount struct {
		Day   int `json:"day"`
		Count int `json:"count"`
	}

	var dayCounts []dayCount
	s.db.Raw(`
		SELECT 
			CAST(EXTRACT(DAY FROM date_taken) AS INTEGER) as day,
			COUNT(*) as count
		FROM image_metadata
		WHERE date_taken >= $1 AND date_taken < $2 AND date_taken IS NOT NULL
		GROUP BY EXTRACT(DAY FROM date_taken)
		ORDER BY day
	`, t, nextMonth).Scan(&dayCounts)

	// Build days array (only days that have images)
	days := make([]int, 0, len(dayCounts))
	for _, dc := range dayCounts {
		days = append(days, dc.Day)
	}

	// Get total images in this month
	var totalInMonth int
	s.db.Raw(`
		SELECT COUNT(*) FROM image_metadata
		WHERE date_taken >= $1 AND date_taken < $2 AND date_taken IS NOT NULL
	`, t, nextMonth).Scan(&totalInMonth)

	c.JSON(http.StatusOK, gin.H{
		"year":      year,
		"month":     month,
		"days":      days,
		"dayCounts": dayCounts,
		"total":     totalInMonth,
	})
}

// handleGetOCRStatus returns the current OCR classifier status
func (s *Server) handleGetOCRStatus(c *gin.Context) {
	if s.ocrClient == nil || !s.config.OCREnabled {
		c.JSON(http.StatusOK, dto.OCRStatusResponse{
			Status: dto.OCRStatus{
				Enabled: false,
				Health:  "disabled",
			},
		})
		return
	}

	status := s.ocrClient.GetStatus()
	c.JSON(http.StatusOK, dto.OCRStatusResponse{
		Status: dto.OCRStatus{
			Enabled:    true,
			Health:     string(status.HealthStatus),
			LastCheck:  status.LastCheck.Format("2006-01-02 15:04:05"),
			Error:      status.Error,
			ServiceURL: fmt.Sprintf("http://%s:%s", s.config.OCRHost, s.config.OCRPort),
		},
	})
}

// handleStartOcrClassification starts the OCR classification process
func (s *Server) handleStartOcrClassification(c *gin.Context) {
	if s.ocrManager == nil {
		c.JSON(http.StatusServiceUnavailable, i18n.ErrorResponse(i18n.MsgOcrFailed))
		return
	}

	// Default to full scan (non-incremental)
	incremental := false
	if err := s.ocrManager.StartClassification(incremental); err != nil {
		c.JSON(http.StatusConflict, i18n.ErrorResponse(i18n.MsgOcrAlreadyRunning))
		return
	}

	c.JSON(http.StatusAccepted, dto.ScanResponse{
		Message: string(i18n.MsgOcrStarted),
	})
}

// handleStartOcrClassificationIncremental starts OCR classification for new/changed files only
func (s *Server) handleStartOcrClassificationIncremental(c *gin.Context) {
	if s.ocrManager == nil {
		c.JSON(http.StatusServiceUnavailable, i18n.ErrorResponse(i18n.MsgOcrFailed))
		return
	}

	if err := s.ocrManager.StartClassification(true); err != nil {
		c.JSON(http.StatusConflict, i18n.ErrorResponse(i18n.MsgOcrAlreadyRunning))
		return
	}

	c.JSON(http.StatusAccepted, dto.ScanResponse{
		Message: string(i18n.MsgOcrStarted),
	})
}

// handleStopOcrClassification requests a graceful stop of OCR classification
func (s *Server) handleStopOcrClassification(c *gin.Context) {
	if s.ocrManager == nil {
		c.JSON(http.StatusServiceUnavailable, i18n.ErrorResponse(i18n.MsgOcrFailed))
		return
	}

	if !s.ocrManager.IsProcessing() {
		c.JSON(http.StatusConflict, i18n.ErrorResponse(i18n.MsgOcrNotRunning))
		return
	}

	s.ocrManager.StopClassification()

	c.JSON(http.StatusOK, dto.ScanResponse{
		Message: "OCR classification stopping",
	})
}

// handleGetOcrClassificationStatus returns the OCR classification progress
func (s *Server) handleGetOcrClassificationStatus(c *gin.Context) {
	if s.ocrManager == nil {
		c.JSON(http.StatusOK, dto.OcrClassificationStatusResponse{
			Processing: false,
			Progress:   "OCR classification disabled",
		})
		return
	}

	status := s.ocrManager.GetStatus()
	c.JSON(http.StatusOK, status)
}

// handleGetOcrDocuments returns paginated list of images classified as text documents
func (s *Server) handleGetOcrDocuments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

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

	// Query documents classified as text documents
	offset := (page - 1) * pageSize

	var total int64
	s.db.Table("ocr_classifications").
		Joins("JOIN image_files ON image_files.id = ocr_classifications.image_file_id").
		Where("ocr_classifications.is_text_document = true").
		Count(&total)

	var results []struct {
		ID                 uint
		ImageFileID        uint
		Path               string
		Size               int64
		Hash               string
		ModTime            time.Time
		MeanConfidence     float32
		WeightedConfidence float32
		TokenCount         int
		Angle              int
		ScaleFactor        float32
	}

	if err := s.db.Table("ocr_classifications").
		Select("image_files.id, image_files.path, image_files.size, image_files.hash, image_files.mod_time, ocr_classifications.image_file_id, ocr_classifications.mean_confidence, ocr_classifications.weighted_confidence, ocr_classifications.token_count, ocr_classifications.angle, ocr_classifications.scale_factor").
		Joins("JOIN image_files ON image_files.id = ocr_classifications.image_file_id").
		Where("ocr_classifications.is_text_document = true").
		Order("image_files.id").
		Offset(offset).
		Limit(pageSize).
		Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgScanFailed))
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Build DTOs with thumbnails
	docs := make([]dto.OcrDocumentDTO, len(results))
	for i, r := range results {
		docs[i] = dto.OcrDocumentDTO{
			ID:                 r.ID,
			ImageFileID:        r.ImageFileID,
			Path:               r.Path,
			FileName:           filepath.Base(r.Path),
			DirPath:            filepath.Dir(r.Path),
			Size:               r.Size,
			SizeHuman:          formatSize(r.Size),
			ModTime:            r.ModTime.Format("2006-01-02 15:04:05"),
			MeanConfidence:     r.MeanConfidence,
			WeightedConfidence: r.WeightedConfidence,
			TokenCount:         r.TokenCount,
			Angle:              r.Angle,
			ScaleFactor:        r.ScaleFactor,
		}
	}

	// Generate thumbnails in parallel
	const maxWorkers = 16
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i, doc := range docs {
		if doc.Path == "" {
			continue
		}
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var thumb string
			var err error

			// Use thumbnail service if available
			if s.thumbnailService != nil {
				thumb, err = s.thumbnailService.GetOrGenerate(path)
			} else {
				thumb, err = imaging.GenerateThumbnail(path, s.thumbnailCache)
			}

			if err == nil {
				docs[idx].Thumbnail = thumb
			}
		}(i, doc.Path)
	}
	wg.Wait()

	c.JSON(http.StatusOK, dto.OcrDocumentsResponse{
		Documents:   docs,
		Total:       int(total),
		CurrentPage: page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNextPage: page < totalPages,
	})
}

// handleGetOcrData returns OCR classification data and bounding boxes for a specific image
func (s *Server) handleGetOcrData(c *gin.Context) {
	imagePath := c.Query("path")
	if imagePath == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgOcrImagePathRequired))
		return
	}

	// Find classification
	var classification domain.OcrClassification
	if err := s.db.Table("ocr_classifications").
		Joins("JOIN image_files ON image_files.id = ocr_classifications.image_file_id").
		Where("image_files.path = ?", imagePath).
		First(&classification).Error; err != nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgOcrDataNotFound))
		return
	}

	// Find bounding boxes
	var boxes []domain.OcrBoundingBox
	s.db.Where("classification_id = ?", classification.ID).Find(&boxes)

	// Convert to DTOs
	boxDTOs := make([]dto.BoundingBoxDTO, len(boxes))
	for i, b := range boxes {
		boxDTOs[i] = dto.BoundingBoxDTO{
			X:          b.X,
			Y:          b.Y,
			Width:      b.Width,
			Height:     b.Height,
			Word:       b.Word,
			Confidence: b.Confidence,
		}
	}

	c.JSON(http.StatusOK, dto.OcrDataResponse{
		ImagePath:         imagePath,
		Angle:             classification.Angle,
		ScaleFactor:       classification.ScaleFactor,
		BoundingBoxWidth:  classification.BoundingBoxWidth,
		BoundingBoxHeight: classification.BoundingBoxHeight,
		Boxes:             boxDTOs,
	})
}

// handleGetLlmSettings returns LLM settings
func (s *Server) handleGetLlmSettings(c *gin.Context) {
	var settings domain.LlmSettings
	if err := s.db.First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, dto.LlmSettingsDTO{
				Provider: "ollama",
				ApiUrl:   "http://localhost:11434",
				Model:    "minicpm-v",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgLlmOcrSettingsNotFound))
		return
	}

	// Don't expose API key in full, mask it
	apiKey := settings.ApiKey
	if len(apiKey) > 8 {
		apiKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
	}

	c.JSON(http.StatusOK, dto.LlmSettingsDTO{
		ID:       settings.ID,
		Provider: settings.Provider,
		ApiUrl:   settings.ApiUrl,
		ApiKey:   apiKey,
		Model:    settings.Model,
		Enabled:  settings.Enabled,
	})
}

// handleUpdateLlmSettings updates LLM settings
func (s *Server) handleUpdateLlmSettings(c *gin.Context) {
	var req dto.UpdateLlmSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.ValidationError))
		return
	}

	var settings domain.LlmSettings
	err := s.db.First(&settings).Error

	if err == gorm.ErrRecordNotFound {
		// Create new settings
		settings = domain.LlmSettings{
			Provider: req.Provider,
			ApiUrl:   req.ApiUrl,
			ApiKey:   req.ApiKey,
			Model:    req.Model,
			Enabled:  req.Enabled,
		}
		if err := s.db.Create(&settings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgLlmOcrSettingsSaveFailed))
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgLlmOcrSettingsSaveFailed))
		return
	} else {
		// Update existing settings
		s.db.Model(&settings).Updates(map[string]interface{}{
			"provider": req.Provider,
			"api_url":  req.ApiUrl,
			"api_key":  req.ApiKey,
			"model":    req.Model,
			"enabled":  req.Enabled,
		})
	}

	c.JSON(http.StatusOK, map[string]string{"message": string(i18n.MsgLlmOcrSettingsSaved)})
}

// handleLlmRecognize starts LLM-based OCR recognition asynchronously
func (s *Server) handleLlmRecognize(c *gin.Context) {
	var req dto.LlmOcrRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.ValidationError))
		return
	}

	// Get LLM settings
	var settings domain.LlmSettings
	if err := s.db.First(&settings).Error; err != nil || !settings.Enabled {
		c.JSON(http.StatusServiceUnavailable, i18n.ErrorResponse(i18n.MsgLlmOcrNotEnabled))
		return
	}

	// Get image file ID from path
	var imageFile domain.ImageFile
	if err := s.db.Where("path = ?", req.ImagePath).First(&imageFile).Error; err != nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgOcrDataNotFound))
		return
	}

	// Check if already recognized (return cached result unless force re-recognition)
	if !req.Force && s.llmOcrService != nil {
		existing, _ := s.llmOcrService.GetRecognition(imageFile.ID)
		if existing != nil && existing.Success {
			c.JSON(http.StatusOK, dto.LlmRecognizeStatusResponse{
				Status:           "completed",
				MarkdownContent:  existing.MarkdownContent,
				Language:         existing.Language,
				Provider:         existing.Provider,
				Model:            existing.Model,
				ProcessingTimeMs: existing.ProcessingTimeMs,
			})
			return
		}
	}

	// Create LLM client
	llmClient, err := llm.NewClient(settings.Provider, settings.ApiUrl, settings.ApiKey, settings.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgLlmOcrRecognitionFailed))
		return
	}

	// Start async recognition
	if s.llmOcrService == nil {
		c.JSON(http.StatusServiceUnavailable, i18n.ErrorResponse(i18n.MsgLlmOcrRecognitionFailed))
		return
	}

	started := s.llmOcrService.StartRecognizeAsync(imageFile.ID, llmClient, settings)
	if started {
		c.JSON(http.StatusAccepted, dto.LlmRecognizeStatusResponse{
			Status: "processing",
		})
	} else {
		// Already processing this image
		c.JSON(http.StatusAccepted, dto.LlmRecognizeStatusResponse{
			Status: "processing",
		})
	}
}

// handleLlmRecognizeStatus returns the status of an async LLM recognition task
func (s *Server) handleLlmRecognizeStatus(c *gin.Context) {
	imagePath := c.Query("path")
	if imagePath == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgOcrImagePathRequired))
		return
	}

	// Get image file ID
	var imageFile domain.ImageFile
	if err := s.db.Where("path = ?", imagePath).First(&imageFile).Error; err != nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgOcrDataNotFound))
		return
	}

	if s.llmOcrService == nil {
		c.JSON(http.StatusOK, dto.LlmRecognizeStatusResponse{Status: "not_found"})
		return
	}

	taskStatus := s.llmOcrService.GetRecognizeStatus(imageFile.ID)
	if taskStatus == nil {
		// No active task — check if there's a result in DB
		existing, _ := s.llmOcrService.GetRecognition(imageFile.ID)
		if existing != nil && existing.Success {
			c.JSON(http.StatusOK, dto.LlmRecognizeStatusResponse{
				Status:           "completed",
				MarkdownContent:  existing.MarkdownContent,
				Language:         existing.Language,
				Provider:         existing.Provider,
				Model:            existing.Model,
				ProcessingTimeMs: existing.ProcessingTimeMs,
			})
			return
		}
		c.JSON(http.StatusOK, dto.LlmRecognizeStatusResponse{Status: "not_found"})
		return
	}

	switch taskStatus.Status {
	case "processing":
		c.JSON(http.StatusOK, dto.LlmRecognizeStatusResponse{Status: "processing"})
	case "completed":
		resp := dto.LlmRecognizeStatusResponse{
			Status: "completed",
		}
		if taskStatus.Result != nil {
			resp.MarkdownContent = taskStatus.Result.MarkdownContent
			resp.Language = taskStatus.Result.Language
			resp.Provider = taskStatus.Result.Provider
			resp.Model = taskStatus.Result.Model
			resp.ProcessingTimeMs = taskStatus.Result.ProcessingTimeMs
		}
		c.JSON(http.StatusOK, resp)
	case "failed":
		c.JSON(http.StatusOK, dto.LlmRecognizeStatusResponse{
			Status: "failed",
			Error:  taskStatus.Error,
		})
	default:
		c.JSON(http.StatusOK, dto.LlmRecognizeStatusResponse{Status: "not_found"})
	}
}

// handleGetLlmRecognition retrieves LLM OCR recognition for an image
func (s *Server) handleGetLlmRecognition(c *gin.Context) {
	imagePath := c.Query("path")
	if imagePath == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgOcrImagePathRequired))
		return
	}

	// Get image file ID
	var imageFile domain.ImageFile
	if err := s.db.Where("path = ?", imagePath).First(&imageFile).Error; err != nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgOcrDataNotFound))
		return
	}

	// Get recognition
	if s.llmOcrService == nil {
		c.JSON(http.StatusOK, dto.LlmOcrDataResponse{Found: false})
		return
	}

	recognition, err := s.llmOcrService.GetRecognition(imageFile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgLlmOcrRecognitionFailed))
		return
	}

	if recognition == nil {
		c.JSON(http.StatusOK, dto.LlmOcrDataResponse{Found: false})
		return
	}

	c.JSON(http.StatusOK, dto.LlmOcrDataResponse{
		Found:            true,
		MarkdownContent:  recognition.MarkdownContent,
		Language:         recognition.Language,
		Provider:         recognition.Provider,
		Model:            recognition.Model,
		ProcessingTimeMs: recognition.ProcessingTimeMs,
		Success:          recognition.Success,
		Error:            recognition.Error,
		CreatedAt:        recognition.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// handleGetLlmModels returns a list of available LLM models from the configured server
func (s *Server) handleGetLlmModels(c *gin.Context) {
	// Get LLM settings
	var settings domain.LlmSettings
	if err := s.db.First(&settings).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.LlmModelsResponse{
			Success:  false,
			Error:    "LLM settings not configured",
			Provider: "",
		})
		return
	}

	if !settings.Enabled {
		c.JSON(http.StatusServiceUnavailable, dto.LlmModelsResponse{
			Success:  false,
			Error:    "LLM recognition is not enabled",
			Provider: settings.Provider,
		})
		return
	}

	// Create LLM client
	llmClient, err := llm.NewClient(settings.Provider, settings.ApiUrl, settings.ApiKey, settings.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.LlmModelsResponse{
			Success:  false,
			Error:    err.Error(),
			Provider: settings.Provider,
		})
		return
	}

	// Fetch models
	models, err := llmClient.ListModels()
	if err != nil {
		c.JSON(http.StatusOK, dto.LlmModelsResponse{
			Success:  false,
			Error:    err.Error(),
			Provider: settings.Provider,
		})
		return
	}

	// Convert to DTO
	modelDTOs := make([]dto.LlmModelDTO, len(models))
	for i, m := range models {
		modelDTOs[i] = dto.LlmModelDTO{
			ID:   m.ID,
			Name: m.Name,
			Size: m.Size,
		}
	}

	c.JSON(http.StatusOK, dto.LlmModelsResponse{
		Success:  true,
		Models:   modelDTOs,
		Provider: settings.Provider,
	})
}

// handleThumbnailCacheStats возвращает статистику кэша миниатюр
func (s *Server) handleThumbnailCacheStats(c *gin.Context) {
	if s.thumbnailService == nil {
		log.Printf("Thumbnail stats requested: service is nil")
		c.JSON(http.StatusOK, thumbnail.ThumbnailStats{})
		return
	}

	stats := s.thumbnailService.Stats()
	log.Printf("Thumbnail stats: %+v", stats)
	c.JSON(http.StatusOK, stats)
}

// handleThumbnailCacheInvalidate удаляет миниатюру из кэша
func (s *Server) handleThumbnailCacheInvalidate(c *gin.Context) {
	var req dto.InvalidateThumbnailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	if req.FilePath == "" {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImagePathRequired))
		return
	}

	if s.thumbnailService == nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgScanDuplicateFailed))
		return
	}

	if err := s.thumbnailService.Invalidate(req.FilePath); err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgImageThumbnailFailed))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "thumbnail invalidated"})
}

// handleThumbnailCacheInvalidateAll удаляет все миниатюры из кэша
func (s *Server) handleThumbnailCacheInvalidateAll(c *gin.Context) {
	if s.thumbnailService == nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgScanDuplicateFailed))
		return
	}

	if err := s.thumbnailService.InvalidateAll(); err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgImageThumbnailFailed))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all thumbnails invalidated"})
}

// handleThumbnailCacheWarmup предварительно генерирует миниатюры для файлов
func (s *Server) handleThumbnailCacheWarmup(c *gin.Context) {
	var req dto.WarmupThumbnailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	if len(req.FilePaths) == 0 {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgScanNoFilesSelected))
		return
	}

	if s.thumbnailService == nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgScanDuplicateFailed))
		return
	}

	if err := s.thumbnailService.Warmup(req.FilePaths); err != nil {
		c.JSON(http.StatusInternalServerError, i18n.ErrorResponse(i18n.MsgImageThumbnailFailed))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "thumbnails warmed up"})
}

// handleThumbnailCacheEnable включает кэш миниатюр
func (s *Server) handleThumbnailCacheEnable(c *gin.Context) {
	if s.thumbnailService == nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgThumbnailCacheNotAvailable))
		return
	}

	s.thumbnailService.Enable()
	c.JSON(http.StatusOK, gin.H{"message": "thumbnail cache enabled"})
}

// handleThumbnailCacheDisable выключает кэш миниатюр
func (s *Server) handleThumbnailCacheDisable(c *gin.Context) {
	if s.thumbnailService == nil {
		c.JSON(http.StatusNotFound, i18n.ErrorResponse(i18n.MsgThumbnailCacheNotAvailable))
		return
	}

	s.thumbnailService.Disable()
	c.JSON(http.StatusOK, gin.H{"message": "thumbnail cache disabled"})
}
