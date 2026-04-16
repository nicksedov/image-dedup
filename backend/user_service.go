package main

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UserService handles user management operations
type UserService struct {
	db          *gorm.DB
	sessionRepo *SessionRepository
}

// NewUserService creates a new user service
func NewUserService(db *gorm.DB, sessionRepo *SessionRepository) *UserService {
	return &UserService{
		db:          db,
		sessionRepo: sessionRepo,
	}
}

// CreateUserInput contains the data needed to create a user
type CreateUserInput struct {
	Login       string   `json:"login" binding:"required"`
	DisplayName string   `json:"displayName" binding:"required"`
	Role        UserRole `json:"role" binding:"required"`
	Password    string   `json:"password" binding:"required"`
}

// CreateUser creates a new user (admin action)
func (s *UserService) CreateUser(adminID uint, input *CreateUserInput) (*User, error) {
	// Verify admin exists
	var admin User
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return nil, err
	}
	if admin.Role != RoleAdmin {
		return nil, ErrForbidden
	}

	// Validate role
	if input.Role != RoleAdmin && input.Role != RoleUser {
		return nil, errors.New("invalid role")
	}

	// Validate password length (min 8, max 128)
	if len(input.Password) < 8 || len(input.Password) > 128 {
		return nil, errors.New("password must be between 8 and 128 characters")
	}

	// Check if login already exists
	var existing User
	if err := s.db.Where("login = ?", input.Login).First(&existing).Error; err == nil {
		return nil, errors.New("user with this login already exists")
	}

	// Hash password
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := User{
		Login:              input.Login,
		DisplayName:        input.DisplayName,
		Role:               input.Role,
		PasswordHash:       passwordHash,
		IsActive:           true,
		MustChangePassword: true,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id uint) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers returns all users
func (s *UserService) ListUsers() ([]User, error) {
	var users []User
	if err := s.db.Order("created_at desc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserInput contains updatable user fields
type UpdateUserInput struct {
	DisplayName *string   `json:"displayName"`
	Role        *UserRole `json:"role"`
	IsActive    *bool     `json:"isActive"`
}

// UpdateUser updates a user (admin action)
func (s *UserService) UpdateUser(adminID, userID uint, input *UpdateUserInput) (*User, error) {
	// Verify admin exists
	var admin User
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return nil, err
	}
	if admin.Role != RoleAdmin {
		return nil, ErrForbidden
	}

	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	// Prevent modifying the last admin
	if input.Role != nil && *input.Role != RoleAdmin && user.Role == RoleAdmin {
		adminCount, err := s.countAdmins()
		if err != nil {
			return nil, err
		}
		if adminCount <= 1 {
			return nil, errors.New("cannot demote the last admin")
		}
	}

	// Prevent deactivating the last admin
	if input.IsActive != nil && !*input.IsActive && user.Role == RoleAdmin {
		adminCount, err := s.countAdmins()
		if err != nil {
			return nil, err
		}
		if adminCount <= 1 {
			return nil, errors.New("cannot deactivate the last admin")
		}
	}

	updates := make(map[string]interface{})
	if input.DisplayName != nil {
		updates["display_name"] = *input.DisplayName
	}
	if input.Role != nil {
		updates["role"] = *input.Role
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// If deactivated, revoke all sessions
	if input.IsActive != nil && !*input.IsActive {
		s.sessionRepo.RevokeAllUserSessions(user.ID)
	}

	return &user, nil
}

// DeleteUser deletes a user (admin action)
func (s *UserService) DeleteUser(adminID, userID uint) error {
	// Verify admin exists
	var admin User
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return err
	}
	if admin.Role != RoleAdmin {
		return ErrForbidden
	}

	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	// Prevent deleting the last admin
	if user.Role == RoleAdmin {
		adminCount, err := s.countAdmins()
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return errors.New("cannot delete the last admin")
		}
	}

	// Revoke all sessions
	s.sessionRepo.RevokeAllUserSessions(user.ID)

	// Delete user
	return s.db.Delete(&user).Error
}

// UpdateProfile updates the current user's own profile
func (s *UserService) UpdateProfile(userID uint, displayName string) (*User, error) {
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&user).Update("display_name", displayName).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// countAdmins returns the number of active admin users
func (s *UserService) countAdmins() (int64, error) {
	var count int64
	err := s.db.Model(&User{}).Where("role = ? AND is_active = ?", RoleAdmin, true).Count(&count).Error
	return count, err
}

// CreateAuditLog creates an audit log entry
func CreateAuditLog(db *gorm.DB, actorUserID *uint, action AuditAction, targetType string, targetID *uint, meta string) error {
	log := AuditLog{
		ActorUserID: actorUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Meta:        meta,
		CreatedAt:   time.Now(),
	}
	return db.Create(&log).Error
}

// ListAuditLogs returns audit logs with pagination
func ListAuditLogs(db *gorm.DB, page, pageSize int) ([]AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	var total int64
	if err := db.Model(&AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []AuditLog
	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
