package auth

import (
	"crypto/sha256"
	"fmt"
	"time"

	"image-dedup/internal/domain"

	"gorm.io/gorm"
)

// SessionConfig holds session lifetime configuration
type SessionConfig struct {
	IdleTimeout     time.Duration // Session expires after this period of inactivity
	AbsoluteTimeout time.Duration // Session expires after this time regardless of activity
	CookieMaxAge    int           // Max-Age attribute for persistent cookie (seconds)
	TokenLength     int           // Length of the random token in bytes
}

// DefaultSessionConfig returns the default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		IdleTimeout:     30 * 24 * time.Hour, // 30 days
		AbsoluteTimeout: 90 * 24 * time.Hour, // 90 days
		CookieMaxAge:    30 * 24 * 60 * 60,   // 30 days in seconds
		TokenLength:     64,                  // 64 bytes = 512 bits
	}
}

// SessionRepository handles database operations for sessions
type SessionRepository struct {
	db     *gorm.DB
	config *SessionConfig
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *gorm.DB, config *SessionConfig) *SessionRepository {
	return &SessionRepository{
		db:     db,
		config: config,
	}
}

// CreateSession creates a new session for a user and returns the session token
func (r *SessionRepository) CreateSession(userID uint, ipAddress, userAgent string) (string, error) {
	token, err := GenerateSecureToken(r.config.TokenLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	// Hash the token for storage
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(token)))

	now := time.Now()
	session := domain.Session{
		UserID:       userID,
		SessionToken: tokenHash,
		CreatedAt:    now,
		LastSeenAt:   now,
		ExpiresAt:    now.Add(r.config.AbsoluteTimeout),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := r.db.Create(&session).Error; err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// GetSession retrieves a session by token and validates it
func (r *SessionRepository) GetSession(token string) (*domain.Session, error) {
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(token)))

	var session domain.Session
	if err := r.db.Where("session_token = ? AND revoked_at IS NULL", tokenHash).First(&session).Error; err != nil {
		return nil, err
	}

	// Check if session has expired
	now := time.Now()
	if now.After(session.ExpiresAt) {
		// Mark as revoked
		r.db.Model(&session).Update("revoked_at", now)
		return nil, gorm.ErrRecordNotFound
	}

	// Check idle timeout
	if now.Sub(session.LastSeenAt) > r.config.IdleTimeout {
		r.db.Model(&session).Update("revoked_at", now)
		return nil, gorm.ErrRecordNotFound
	}

	return &session, nil
}

// UpdateLastSeen updates the last_seen_at timestamp for a session
func (r *SessionRepository) UpdateLastSeen(token string) error {
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(token)))
	return r.db.Model(&domain.Session{}).
		Where("session_token = ? AND revoked_at IS NULL", tokenHash).
		Update("last_seen_at", time.Now()).Error
}

// RevokeSession marks a session as revoked
func (r *SessionRepository) RevokeSession(token string) error {
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(token)))
	now := time.Now()
	return r.db.Model(&domain.Session{}).
		Where("session_token = ?", tokenHash).
		Updates(map[string]interface{}{
			"revoked_at": now,
		}).Error
}

// RevokeAllUserSessions revokes all sessions for a user
func (r *SessionRepository) RevokeAllUserSessions(userID uint) error {
	now := time.Now()
	return r.db.Model(&domain.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// CleanupExpiredSessions removes expired and revoked sessions from the database
func (r *SessionRepository) CleanupExpiredSessions() error {
	now := time.Now()
	return r.db.Where("expires_at < ? OR revoked_at IS NOT NULL", now).Delete(&domain.Session{}).Error
}

// GetSessionConfig returns the session configuration
func (r *SessionRepository) GetSessionConfig() *SessionConfig {
	return r.config
}
