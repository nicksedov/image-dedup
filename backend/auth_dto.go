package main

// --- Auth API DTOs ---

// LoginRequest represents the login request body
type LoginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthStatusResponse represents the auth status response
type AuthStatusResponse struct {
	IsAuthenticated bool   `json:"isAuthenticated"`
	IsBootstrapMode bool   `json:"isBootstrapMode"`
	CSRFToken       string `json:"csrfToken,omitempty"`
}

// UserDTO represents user data in API responses (excludes sensitive fields)
type UserDTO struct {
	ID                 uint     `json:"id"`
	Login              string   `json:"login"`
	DisplayName        string   `json:"displayName"`
	Role               UserRole `json:"role"`
	IsActive           bool     `json:"isActive"`
	MustChangePassword bool     `json:"mustChangePassword"`
	CreatedAt          string   `json:"createdAt"`
	LastLoginAt        *string  `json:"lastLoginAt"`
}

// ToUserDTO converts a User to UserDTO
func ToUserDTO(u *User) UserDTO {
	dto := UserDTO{
		ID:                 u.ID,
		Login:              u.Login,
		DisplayName:        u.DisplayName,
		Role:               u.Role,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
		CreatedAt:          u.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if u.LastLoginAt != nil {
		s := u.LastLoginAt.Format("2006-01-02 15:04:05")
		dto.LastLoginAt = &s
	}
	return dto
}

// ChangePasswordRequest represents the change password request
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// BootstrapSetupRequest represents the bootstrap admin setup request
type BootstrapSetupRequest struct {
	NewPassword string `json:"newPassword" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
}

// --- User Management API DTOs ---

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Login       string   `json:"login" binding:"required"`
	DisplayName string   `json:"displayName" binding:"required"`
	Role        UserRole `json:"role" binding:"required"`
	Password    string   `json:"password" binding:"required"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	DisplayName *string   `json:"displayName"`
	Role        *UserRole `json:"role"`
	IsActive    *bool     `json:"isActive"`
}

// UpdateProfileRequest represents the request to update own profile
type UpdateProfileRequest struct {
	DisplayName string `json:"displayName" binding:"required"`
}

// UsersListResponse represents the response for listing users
type UsersListResponse struct {
	Users []UserDTO `json:"users"`
	Total int       `json:"total"`
}

// ResetPasswordRequest represents the admin password reset request
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" binding:"required"`
}

// AuditLogDTO represents an audit log entry in API responses
type AuditLogDTO struct {
	ID          uint   `json:"id"`
	ActorUserID *uint  `json:"actorUserId"`
	Action      string `json:"action"`
	TargetType  string `json:"targetType"`
	TargetID    *uint  `json:"targetId"`
	Meta        string `json:"meta"`
	CreatedAt   string `json:"createdAt"`
}

// ToAuditLogDTO converts an AuditLog to DTO
func ToAuditLogDTO(log *AuditLog) AuditLogDTO {
	return AuditLogDTO{
		ID:          log.ID,
		ActorUserID: log.ActorUserID,
		Action:      string(log.Action),
		TargetType:  log.TargetType,
		TargetID:    log.TargetID,
		Meta:        log.Meta,
		CreatedAt:   log.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// AuditLogsResponse represents the response for audit logs
type AuditLogsResponse struct {
	Logs  []AuditLogDTO `json:"logs"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
}
