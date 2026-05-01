package handler

import (
	"image-toolkit/internal/interfaces/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter sets up the Gin router with all API routes
func (s *Server) SetupRouter(authMiddleware *middleware.AuthMiddleware, csrfProtection *middleware.CSRFProtection, authHandlers *AuthHandlers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Security headers middleware
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	})

	// CORS middleware
	r.Use(middleware.SetupCORS(s.config))

	// CSRF protection
	r.Use(csrfProtection.Middleware())

	// Public auth routes
	api := r.Group("/api")
	{
		// Auth endpoints (public)
		auth := api.Group("/auth")
		{
			auth.GET("/status", authHandlers.handleAuthStatus)
			auth.POST("/login", authHandlers.handleLogin)
			auth.POST("/bootstrap/setup", authHandlers.handleBootstrapSetup)
		}

		// Protected routes (require auth)
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			protected.POST("/auth/logout", authHandlers.handleLogout)
			protected.GET("/auth/me", authHandlers.handleMe)
			protected.POST("/auth/change-password", authHandlers.handleChangePassword)
			protected.PATCH("/users/me", authHandlers.handleUpdateProfile)

			// Existing endpoints (now protected)
			protected.GET("/duplicates", s.handleGetDuplicates)
			protected.POST("/scan", s.handleScan)
			protected.POST("/fast-scan", s.handleFastScan)
			protected.GET("/status", s.handleGetStatus)
			protected.POST("/delete-files", s.handleDeleteFiles)
			protected.GET("/thumbnail", s.handleThumbnail)
			protected.GET("/folder-patterns", s.handleGetFolderPatterns)
			protected.POST("/batch-delete", s.handleBatchDelete)
			protected.GET("/folders", s.handleGetFolders)
			protected.POST("/folders", s.handleAddFolder)
			protected.DELETE("/folders/:id", s.handleRemoveFolder)
			protected.GET("/gallery", s.handleGetGalleryImages)
			protected.GET("/gallery/calendar", s.handleGetGalleryCalendar)
			protected.GET("/gallery/calendar/month", s.handleGetCalendarMonthInfo)
			protected.GET("/image", s.handleServeImage)
			protected.GET("/ocr-image", s.handleServeOcrImage)
			protected.GET("/settings", s.handleGetSettings)
			protected.PUT("/settings", s.handleUpdateSettings)
			protected.GET("/user-settings", s.handleGetUserSettings)
			protected.PUT("/user-settings", s.handleUpdateUserSettings)
			protected.GET("/trash-info", s.handleGetTrashInfo)
			protected.POST("/trash-clean", s.handleCleanTrash)
			protected.GET("/image-metadata", s.handleGetImageMetadata)
			protected.GET("/metadata-status", s.handleGetMetadataStatus)
			protected.GET("/ocr-status", s.handleGetOCRStatus)
			protected.POST("/ocr/classify", s.handleStartOcrClassification)
			protected.GET("/ocr/classify-status", s.handleGetOcrClassificationStatus)
			protected.GET("/ocr/documents", s.handleGetOcrDocuments)
			protected.GET("/ocr/data", s.handleGetOcrData)

			// LLM OCR endpoints
			protected.GET("/llm/settings", s.handleGetLlmSettings)
			protected.PUT("/llm/settings", s.handleUpdateLlmSettings)
			protected.POST("/llm/recognize", s.handleLlmRecognize)
			protected.GET("/llm/recognition", s.handleGetLlmRecognition)
			protected.GET("/llm/models", s.handleGetLlmModels)

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.GET("/users", authHandlers.handleListUsers)
				admin.POST("/users", authHandlers.handleCreateUser)
				admin.PATCH("/users/:id", authHandlers.handleUpdateUser)
				admin.DELETE("/users/:id", authHandlers.handleDeleteUser)
				admin.POST("/users/:id/reset-password", authHandlers.handleResetPassword)
				admin.GET("/audit", authHandlers.handleAuditLogs)
			}
		}
	}

	return r
}
