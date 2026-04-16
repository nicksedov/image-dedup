package main

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// BootstrapService handles the initial system setup logic
type BootstrapService struct {
	db                *gorm.DB
	bootstrapLogin    string
	bootstrapPassword string
}

// NewBootstrapService creates a new bootstrap service
func NewBootstrapService(db *gorm.DB, bootstrapLogin, bootstrapPassword string) *BootstrapService {
	return &BootstrapService{
		db:                db,
		bootstrapLogin:    bootstrapLogin,
		bootstrapPassword: bootstrapPassword,
	}
}

// IsBootstrapMode returns true if no users exist in the database
func (s *BootstrapService) IsBootstrapMode() (bool, error) {
	var count int64
	if err := s.db.Model(&User{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// ValidateBootstrapCredentials checks if the provided credentials match the bootstrap admin
func (s *BootstrapService) ValidateBootstrapCredentials(login, password string) bool {
	return login == s.bootstrapLogin && password == s.bootstrapPassword
}

// CreateBootstrapAdmin creates the first admin user in the database and completes bootstrap
func (s *BootstrapService) CreateBootstrapAdmin(newPassword, displayName string) (*User, error) {
	// Double-check we're still in bootstrap mode
	isBootstrap, err := s.IsBootstrapMode()
	if err != nil {
		return nil, fmt.Errorf("failed to check bootstrap mode: %w", err)
	}
	if !isBootstrap {
		return nil, fmt.Errorf("bootstrap mode is already disabled")
	}

	// Hash the new password
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create the admin user
	user := User{
		Login:              s.bootstrapLogin,
		DisplayName:        displayName,
		Role:               RoleAdmin,
		PasswordHash:       passwordHash,
		IsActive:           true,
		MustChangePassword: false,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	// Set last login time
	now := time.Now()
	s.db.Model(&user).Update("last_login_at", now)

	return &user, nil
}

// GetBootstrapLogin returns the bootstrap admin login (for display purposes)
func (s *BootstrapService) GetBootstrapLogin() string {
	return s.bootstrapLogin
}
