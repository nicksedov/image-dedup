package domain

import (
	"time"
)

// UserRole represents the role of a user
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

// User represents a user account in the system
type User struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Login              string     `gorm:"uniqueIndex;size:255;not null" json:"login"`
	DisplayName        string     `gorm:"size:255;not null" json:"displayName"`
	Role               UserRole   `gorm:"size:50;not null;default:user" json:"role"`
	PasswordHash       string     `gorm:"not null" json:"-"`
	IsActive           bool       `gorm:"default:true" json:"isActive"`
	MustChangePassword bool       `gorm:"default:false" json:"mustChangePassword"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	LastLoginAt        *time.Time `json:"lastLoginAt"`
}

// UserSettings represents user-specific application settings
type UserSettings struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"userId"`
	Theme     string    `gorm:"default:light;not null" json:"theme"`
	Language  string    `gorm:"default:en;not null" json:"language"`
	TrashDir  string    `gorm:"default:''" json:"trashDir"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Session represents an active user session
type Session struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"userId"`
	SessionToken string     `gorm:"uniqueIndex;size:255;not null" json:"-"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastSeenAt   time.Time  `json:"lastSeenAt"`
	ExpiresAt    time.Time  `gorm:"index" json:"expiresAt"`
	IPAddress    string     `gorm:"size:45" json:"-"`
	UserAgent    string     `gorm:"size:500" json:"-"`
	RevokedAt    *time.Time `json:"-"`
}

// AuditAction represents the type of audit action
type AuditAction string

const (
	ActionLogin             AuditAction = "login"
	ActionLogout            AuditAction = "logout"
	ActionLoginFailed       AuditAction = "login_failed"
	ActionCreateUser        AuditAction = "create_user"
	ActionUpdateUser        AuditAction = "update_user"
	ActionDeleteUser        AuditAction = "delete_user"
	ActionResetPassword     AuditAction = "reset_password"
	ActionChangePassword    AuditAction = "change_password"
	ActionDeactivateUser    AuditAction = "deactivate_user"
	ActionActivateUser      AuditAction = "activate_user"
	ActionBootstrapComplete AuditAction = "bootstrap_complete"
)

// AuditLog records security and administrative events
type AuditLog struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	ActorUserID *uint       `gorm:"index" json:"actorUserId"`
	Action      AuditAction `gorm:"size:50;not null" json:"action"`
	TargetType  string      `gorm:"size:100" json:"targetType"`
	TargetID    *uint       `json:"targetId"`
	Meta        string      `gorm:"type:jsonb" json:"meta"`
	CreatedAt   time.Time   `json:"createdAt"`
}

// AuthError represents authentication error types
type AuthError string

func (e AuthError) Error() string {
	return string(e)
}

const (
	ErrInvalidCredentials AuthError = "Неверный логин или пароль"
	ErrUserDeactivated    AuthError = "Неверный логин или пароль"
	ErrRateLimited        AuthError = "Слишком много попыток входа. Попробуйте позже"
	ErrForbidden          AuthError = "Недостаточно прав"
)
