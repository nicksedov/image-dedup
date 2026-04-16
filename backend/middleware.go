package main

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupCORS configures CORS middleware based on AppConfig.
// If CORSOrigins contains "*", all origins are allowed dynamically
// (compatible with AllowCredentials).
func SetupCORS(config *AppConfig) gin.HandlerFunc {
	allowAll := len(config.CORSOrigins) == 1 && config.CORSOrigins[0] == "*"

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if allowAll {
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return true
		}
	} else {
		corsConfig.AllowOrigins = config.CORSOrigins
	}

	return cors.New(corsConfig)
}
