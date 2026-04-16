package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthHandlers contains all authentication-related handlers
type AuthHandlers struct {
	authService *AuthService
	bootstrap   *BootstrapService
	userService *UserService
	sessionRepo *SessionRepository
	db          *gorm.DB
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers(authService *AuthService, bootstrap *BootstrapService, userService *UserService, sessionRepo *SessionRepository, db *gorm.DB) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
		bootstrap:   bootstrap,
		userService: userService,
		sessionRepo: sessionRepo,
		db:          db,
	}
}

// handleAuthStatus returns the current authentication status
func (h *AuthHandlers) handleAuthStatus(c *gin.Context) {
	isBootstrap, err := h.bootstrap.IsBootstrapMode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Try to get user from session
	user := GetCurrentUser(c)
	if user != nil {
		c.JSON(http.StatusOK, gin.H{
			"isAuthenticated": true,
			"isBootstrapMode": false,
			"user":            ToUserDTO(user),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"isAuthenticated": false,
		"isBootstrapMode": isBootstrap,
	})
}

// handleLogin authenticates a user and creates a session
func (h *AuthHandlers) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный логин или пароль"})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	result, err := h.authService.Login(req.Login, req.Password, ipAddress, userAgent)
	if err != nil {
		if err == ErrRateLimited {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}

	if result.IsBootstrap {
		// Bootstrap login - return bootstrap session info
		config := h.sessionRepo.GetSessionConfig()
		c.SetCookie(SessionCookieName, "bootstrap", config.CookieMaxAge, "/", "", true, true)
		c.JSON(http.StatusOK, gin.H{
			"isBootstrap": true,
			"message":     "Bootstrap mode - please complete initial setup",
		})
		return
	}

	// Set session cookie
	config := h.sessionRepo.GetSessionConfig()
	c.SetCookie(
		SessionCookieName,
		result.Token,
		config.CookieMaxAge,
		"/",
		"",
		true, // secure - requires HTTPS (set false for dev, true in prod)
		true, // httpOnly - not accessible via JS
	)

	// Create audit log
	CreateAuditLog(h.db, &result.User.ID, ActionLogin, "user", &result.User.ID, fmt.Sprintf(`{"ip": "%s"}`, ipAddress))

	c.JSON(http.StatusOK, gin.H{
		"user": ToUserDTO(result.User),
	})
}

// handleLogout revokes the current session
func (h *AuthHandlers) handleLogout(c *gin.Context) {
	user := GetCurrentUser(c)
	if user != nil {
		token, _ := c.Cookie(SessionCookieName)
		h.authService.Logout(token)
		CreateAuditLog(h.db, &user.ID, ActionLogout, "user", &user.ID, "")
	}

	// Clear cookie
	c.SetCookie(SessionCookieName, "", -1, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"message": "Выход выполнен"})
}

// handleMe returns the current user's profile
func (h *AuthHandlers) handleMe(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": ToUserDTO(user),
	})
}

// handleChangePassword changes the current user's password
func (h *AuthHandlers) handleChangePassword(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	// Validate new password length
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать от 8 до 128 символов"})
		return
	}

	if err := h.authService.ChangePassword(user.ID, req.OldPassword, req.NewPassword); err != nil {
		if err == ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный текущий пароль"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось изменить пароль"})
		return
	}

	CreateAuditLog(h.db, &user.ID, ActionChangePassword, "user", &user.ID, "")

	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменен"})
}

// handleBootstrapSetup completes the bootstrap initialization
func (h *AuthHandlers) handleBootstrapSetup(c *gin.Context) {
	var req BootstrapSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	// Validate password
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать от 8 до 128 символов"})
		return
	}

	// Create admin user
	user, err := h.bootstrap.CreateBootstrapAdmin(req.NewPassword, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Revoke bootstrap cookie and create real session
	token, err := h.sessionRepo.CreateSession(user.ID, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать сессию"})
		return
	}

	// Set proper session cookie
	config := h.sessionRepo.GetSessionConfig()
	c.SetCookie(
		SessionCookieName,
		token,
		config.CookieMaxAge,
		"/",
		"",
		true,
		true,
	)

	// Audit log
	CreateAuditLog(h.db, &user.ID, ActionBootstrapComplete, "system", nil, `{"admin_login": "`+user.Login+`"}`)

	c.JSON(http.StatusOK, gin.H{
		"user":    ToUserDTO(user),
		"message": "Первичная настройка завершена",
	})
}

// --- Admin Handlers ---

// handleListUsers returns all users (admin only)
func (h *AuthHandlers) handleListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить список пользователей"})
		return
	}

	userDTOs := make([]UserDTO, len(users))
	for i, u := range users {
		userDTOs[i] = ToUserDTO(&u)
	}

	c.JSON(http.StatusOK, gin.H{
		"users": userDTOs,
		"total": len(userDTOs),
	})
}

// handleCreateUser creates a new user (admin only)
func (h *AuthHandlers) handleCreateUser(c *gin.Context) {
	admin := GetCurrentUser(c)

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	// Validate password length
	if len(req.Password) < 8 || len(req.Password) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать от 8 до 128 символов"})
		return
	}

	// Validate role
	if req.Role != RoleAdmin && req.Role != RoleUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверная роль"})
		return
	}

	input := &CreateUserInput{
		Login:       req.Login,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		Password:    req.Password,
	}

	user, err := h.userService.CreateUser(admin.ID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	CreateAuditLog(h.db, &admin.ID, ActionCreateUser, "user", &user.ID, fmt.Sprintf(`{"login": "%s", "role": "%s"}`, user.Login, user.Role))

	c.JSON(http.StatusCreated, gin.H{
		"user":    ToUserDTO(user),
		"message": "Пользователь создан",
	})
}

// handleUpdateUser updates a user (admin only)
func (h *AuthHandlers) handleUpdateUser(c *gin.Context) {
	admin := GetCurrentUser(c)

	id := c.Param("id")
	var userID uint
	if _, err := fmt.Sscanf(id, "%d", &userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	input := &UpdateUserInput{
		DisplayName: req.DisplayName,
		Role:        req.Role,
		IsActive:    req.IsActive,
	}

	user, err := h.userService.UpdateUser(admin.ID, userID, input)
	if err != nil {
		if strings.Contains(err.Error(), "last admin") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить пользователя"})
		return
	}

	// Audit
	action := ActionUpdateUser
	if input.IsActive != nil && !*input.IsActive {
		action = ActionDeactivateUser
	} else if input.IsActive != nil && *input.IsActive {
		action = ActionActivateUser
	}
	CreateAuditLog(h.db, &admin.ID, action, "user", &user.ID, "")

	c.JSON(http.StatusOK, gin.H{
		"user":    ToUserDTO(user),
		"message": "Пользователь обновлен",
	})
}

// handleDeleteUser deletes a user (admin only)
func (h *AuthHandlers) handleDeleteUser(c *gin.Context) {
	admin := GetCurrentUser(c)

	id := c.Param("id")
	var userID uint
	if _, err := fmt.Sscanf(id, "%d", &userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	if err := h.userService.DeleteUser(admin.ID, userID); err != nil {
		if strings.Contains(err.Error(), "last admin") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось удалить пользователя"})
		return
	}

	CreateAuditLog(h.db, &admin.ID, ActionDeleteUser, "user", &userID, "")

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь удален"})
}

// handleResetPassword resets a user's password (admin only)
func (h *AuthHandlers) handleResetPassword(c *gin.Context) {
	admin := GetCurrentUser(c)

	id := c.Param("id")
	var userID uint
	if _, err := fmt.Sscanf(id, "%d", &userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	if len(req.NewPassword) < 8 || len(req.NewPassword) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пароль должен содержать от 8 до 128 символов"})
		return
	}

	if err := h.authService.AdminResetPassword(admin.ID, userID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сбросить пароль"})
		return
	}

	CreateAuditLog(h.db, &admin.ID, ActionResetPassword, "user", &userID, "")

	c.JSON(http.StatusOK, gin.H{"message": "Пароль сброшен"})
}

// handleUpdateProfile updates the current user's profile
func (h *AuthHandlers) handleUpdateProfile(c *gin.Context) {
	user := GetCurrentUser(c)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	updatedUser, err := h.userService.UpdateProfile(user.ID, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить профиль"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": ToUserDTO(updatedUser),
	})
}

// handleAuditLogs returns audit logs (admin only)
func (h *AuthHandlers) handleAuditLogs(c *gin.Context) {
	page := 1
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}

	logs, total, err := ListAuditLogs(h.db, page, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить журнал"})
		return
	}

	dtoLogs := make([]AuditLogDTO, len(logs))
	for i, log := range logs {
		dtoLogs[i] = ToAuditLogDTO(&log)
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  dtoLogs,
		"total": total,
		"page":  page,
	})
}

// handleGetCSRFToken returns a new CSRF token
func (h *AuthHandlers) handleGetCSRFToken(c *gin.Context) {
	token, err := SetCSRFToken(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать CSRF токен"})
		return
	}

	// Encode as JSON for the client to read
	type CSRFResponse struct {
		Token string `json:"token"`
	}

	// Use json.Marshal to ensure proper encoding
	resp := CSRFResponse{Token: token}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, resp)
}
