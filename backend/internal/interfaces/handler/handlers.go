package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"image-toolkit/internal/application/imaging"
	"image-toolkit/internal/domain"
	"image-toolkit/internal/interfaces/dto"
	"image-toolkit/internal/interfaces/i18n"
	"image-toolkit/internal/interfaces/middleware"

	"github.com/gin-gonic/gin"
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

				thumb, err := imaging.GenerateThumbnail(filePath, s.thumbnailCache)
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

	thumbnail, err := imaging.GenerateThumbnail(path, s.thumbnailCache)
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
		Message:     string(i18n.MsgFolderAdded),
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

				thumb, err := imaging.GenerateThumbnail(filePath, s.thumbnailCache)
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

// --- App Settings Handlers ---

// handleGetSettings returns the current application settings
func (s *Server) handleGetSettings(c *gin.Context) {
	var settings domain.AppSettings
	if result := s.db.First(&settings, 1); result.Error != nil {
		c.JSON(http.StatusOK, dto.AppSettingsDTO{Theme: "light", Language: "en", TrashDir: ""})
		return
	}
	c.JSON(http.StatusOK, dto.AppSettingsDTO{
		Theme:    settings.Theme,
		Language: settings.Language,
		TrashDir: settings.TrashDir,
	})
}

// handleUpdateSettings updates the application settings
func (s *Server) handleUpdateSettings(c *gin.Context) {
	var req dto.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	validThemes := map[string]bool{"light": true, "dark": true}
	validLanguages := map[string]bool{"en": true, "ru": true}

	if req.Theme != "" && !validThemes[req.Theme] {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImageInvalidTheme))
		return
	}
	if req.Language != "" && !validLanguages[req.Language] {
		c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgImageInvalidLanguage))
		return
	}

	var settings domain.AppSettings
	if result := s.db.First(&settings, 1); result.Error != nil {
		settings = domain.AppSettings{ID: 1, Theme: "light", Language: "en"}
	}

	if req.Theme != "" {
		settings.Theme = req.Theme
	}
	if req.Language != "" {
		settings.Language = req.Language
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

	s.db.Save(&settings)

	c.JSON(http.StatusOK, dto.AppSettingsDTO{
		Theme:    settings.Theme,
		Language: settings.Language,
		TrashDir: settings.TrashDir,
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
		TrashDir: settings.TrashDir,
	})
}

// handleUpdateUserSettings updates the current user's settings
func (s *Server) handleUpdateUserSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, i18n.ErrorResponse(i18n.MsgAuthUnauthorized))
		return
	}

	var req dto.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, i18n.CreateValidationError(i18n.ValidationError))
		return
	}

	validThemes := map[string]bool{"light": true, "dark": true}
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

	s.db.Save(&settings)

	c.JSON(http.StatusOK, dto.UserSettingsDTO{
		Theme:    settings.Theme,
		Language: settings.Language,
		TrashDir: settings.TrashDir,
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
	query.Order("image_metadata.date_taken DESC").Offset(offset).Limit(pageSize).Find(&results)

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

			// Get distinct days that have images in this month
			var days []int
			s.db.Raw(`
				SELECT DISTINCT CAST(strftime('%d', date_taken) AS INTEGER) as day
				FROM image_metadata
				WHERE date_taken >= ? AND date_taken < ? AND date_taken IS NOT NULL
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
