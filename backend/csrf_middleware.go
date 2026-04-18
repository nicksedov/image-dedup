package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// CSRFTokenHeader is the header name for CSRF token
	CSRFTokenHeader = "X-CSRF-Token"

	// CSRFTokenCookie is the cookie name for CSRF token
	CSRFTokenCookie = "csrf_token"
)

// CSRFProtection implements CSRF protection for cookie-based auth
type CSRFProtection struct {
	// Paths that don't require CSRF validation (safe methods + login)
	skipPaths []string
}

// NewCSRFProtection creates a new CSRF protection middleware
func NewCSRFProtection() *CSRFProtection {
	return &CSRFProtection{
		skipPaths: []string{
			"/api/auth/login",
			"/api/auth/status",
			"/api/auth/me",
			"/api/auth/logout",
			"/api/auth/bootstrap/setup",
			"/api/auth/change-password",
			"/api/users/me",
			"/api/admin/users",
			"/api/admin/users/:id",
			"/api/settings",
			"/api/batch-delete",
		},
	}
}

// ShouldSkipCSRF checks if a path should skip CSRF validation
func (p *CSRFProtection) ShouldSkipCSRF(path string) bool {
	for _, skipPath := range p.skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// Middleware returns the CSRF protection gin middleware
func (p *CSRFProtection) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only validate state-changing methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// Generate CSRF token if not present
			p.ensureCSRFToken(c)
			c.Next()
			return
		}

		// Skip CSRF for login endpoint
		if p.ShouldSkipCSRF(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Validate CSRF token
		tokenFromHeader := c.GetHeader(CSRFTokenHeader)
		tokenFromCookie, err := c.Cookie(CSRFTokenCookie)

		if err != nil || tokenFromHeader == "" || tokenFromHeader != tokenFromCookie {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token validation failed"})
			c.Abort()
			return
		}

		// Also validate Origin/Referer headers
		origin := c.GetHeader("Origin")
		if origin != "" {
			referer := c.GetHeader("Referer")
			if referer != "" && !strings.HasPrefix(referer, origin) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Origin validation failed"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ensureCSRFToken ensures a CSRF token cookie exists
func (p *CSRFProtection) ensureCSRFToken(c *gin.Context) {
	_, err := c.Cookie(CSRFTokenCookie)
	if err != nil {
		// Generate new CSRF token
		token, err := generateCSRFToken()
		if err != nil {
			return
		}

		c.SetCookie(CSRFTokenCookie, token, 3600, "/", "", false, true)
	}
}

// generateCSRFToken generates a random CSRF token
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// SetCSRFToken sets a new CSRF token in the response
func SetCSRFToken(c *gin.Context) (string, error) {
	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}

	c.SetCookie(CSRFTokenCookie, token, 3600, "/", "", false, true)
	return token, nil
}
