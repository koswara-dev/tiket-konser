package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"user-service/config"
	"user-service/db"
	"user-service/dto"
	"user-service/helper"
	"user-service/redis"
	"user-service/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	svc *service.UserService
	cfg *config.Config
}

func NewUserHandler(svc *service.UserService, cfg *config.Config) *UserHandler {
	return &UserHandler{svc: svc, cfg: cfg}
}

func (ctrl *UserHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			helper.WriteErrorResponse(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			helper.WriteErrorResponse(c, http.StatusUnauthorized, "Authorization header format must be Bearer <token>")
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(ctrl.cfg.JwtSecret), nil
		})

		if err != nil || !token.Valid {
			helper.WriteErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			helper.WriteErrorResponse(c, http.StatusUnauthorized, "Invalid token claims")
			c.Abort()
			return
		}

		subVal := claims["sub"]
		var userID uint
		switch val := subVal.(type) {
		case float64:
			userID = uint(val)
		case string:
			parsed, _ := strconv.ParseUint(val, 10, 32)
			userID = uint(parsed)
		}

		roleVal, _ := claims["role"].(string)

		var iat int64
		if val, exists := claims["iat"]; exists {
			if floatVal, ok := val.(float64); ok {
				iat = int64(floatVal)
			}
		}

		var exp int64
		if val, exists := claims["exp"]; exists {
			if floatVal, ok := val.(float64); ok {
				exp = int64(floatVal)
			}
		}

		valid, err := ctrl.svc.IsTokenValid(userID, tokenString, iat)
		if err != nil || !valid {
			helper.WriteErrorResponse(c, http.StatusUnauthorized, "Token has been blacklisted or invalidated")
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("role", roleVal)
		c.Set("token", tokenString)
		c.Set("exp", exp)
		c.Next()
	}
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with name, email, password, and optional role
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register request"
// @Success 201 {object} dto.WebResponse[dto.UserResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 500 {object} dto.WebResponse[any]
// @Router /auth/register [post]
func (ctrl *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.WriteValidationErrorResponse(c, err)
		return
	}

	user, err := ctrl.svc.Register(req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	userResp := dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	helper.WriteSuccessResponse(c, http.StatusCreated, "Registration successful. Please check your email for the OTP code.", userResp)
}

// VerifyOTP godoc
// @Summary Verify user registration OTP
// @Description Verify the OTP code sent to user email to activate the account
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.VerifyOTPRequest true "Verify OTP request"
// @Success 200 {object} dto.WebResponse[any]
// @Failure 400 {object} dto.WebResponse[any]
// @Router /auth/verify-otp [post]
func (ctrl *UserHandler) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.WriteValidationErrorResponse(c, err)
		return
	}

	err := ctrl.svc.VerifyOTP(req.Email, req.Code)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Account successfully verified and activated.", nil)
}

// Login godoc
// @Summary User login
// @Description Login with email and password to receive a JWT token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login request"
// @Success 200 {object} dto.WebResponse[dto.LoginResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 401 {object} dto.WebResponse[any]
// @Router /auth/login [post]
func (ctrl *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.WriteValidationErrorResponse(c, err)
		return
	}

	token, user, err := ctrl.svc.Login(req.Email, req.Password)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	loginResp := dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Login successful", loginResp)
}

// Logout godoc
// @Summary User logout
// @Description Invalidate the current session token (blacklists it in Redis)
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.WebResponse[any]
// @Failure 401 {object} dto.WebResponse[any]
// @Failure 500 {object} dto.WebResponse[any]
// @Router /auth/logout [post]
func (ctrl *UserHandler) Logout(c *gin.Context) {
	tokenString, _ := c.Get("token")
	expVal, _ := c.Get("exp")

	tokenStr := tokenString.(string)
	exp := expVal.(int64)

	err := ctrl.svc.Logout(tokenStr, exp)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Successfully logged out from current session.", nil)
}

// ForceLogout godoc
// @Summary Force logout from all devices
// @Description Invalidate all active tokens for the user
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.WebResponse[any]
// @Failure 401 {object} dto.WebResponse[any]
// @Failure 500 {object} dto.WebResponse[any]
// @Router /auth/force-logout [post]
func (ctrl *UserHandler) ForceLogout(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		helper.WriteErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	userID := userIDVal.(uint)

	err := ctrl.svc.ForceLogout(userID)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Successfully logged out from all devices.", nil)
}

// GetProfile godoc
// @Summary Get user profile
// @Description Fetch user profile details. A standard user can only retrieve their own ID. Admin/staff can retrieve any.
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Security BearerAuth
// @Success 200 {object} dto.WebResponse[dto.UserResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 403 {object} dto.WebResponse[any]
// @Failure 404 {object} dto.WebResponse[any]
// @Router /users/{id} [get]
func (ctrl *UserHandler) GetProfile(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	userIDVal, _ := c.Get("userID")
	roleVal, _ := c.Get("role")
	authenticatedUserID := userIDVal.(uint)
	role := roleVal.(string)

	// IDOR Protection: Standard user can only query their own profile ID
	// Bypassed if request has admin or staff roles
	if role != "admin" && role != "staff" && authenticatedUserID != uint(targetID) {
		helper.WriteErrorResponse(c, http.StatusForbidden, "Forbidden: You are not authorized to view this profile")
		return
	}

	user, err := ctrl.svc.GetUserByID(uint(targetID))
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	userResp := dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "User profile retrieved successfully", userResp)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update profile details (name, password). Standard user can only update their own. Admin/staff can update any.
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body dto.UpdateProfileRequest true "Update profile request"
// @Security BearerAuth
// @Success 200 {object} dto.WebResponse[dto.UserResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 403 {object} dto.WebResponse[any]
// @Failure 404 {object} dto.WebResponse[any]
// @Failure 500 {object} dto.WebResponse[any]
// @Router /users/{id} [put]
func (ctrl *UserHandler) UpdateProfile(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	userIDVal, _ := c.Get("userID")
	roleVal, _ := c.Get("role")
	authenticatedUserID := userIDVal.(uint)
	role := roleVal.(string)

	// IDOR Protection: Standard user can only update their own profile ID
	// Bypassed if request has admin or staff roles
	if role != "admin" && role != "staff" && authenticatedUserID != uint(targetID) {
		helper.WriteErrorResponse(c, http.StatusForbidden, "Forbidden: You are not authorized to update this profile")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.WriteValidationErrorResponse(c, err)
		return
	}

	user, err := ctrl.svc.GetUserByID(uint(targetID))
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}
		user.Password = string(hashedPassword)
	}

	err = ctrl.svc.UpdateUser(user)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	userResp := dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Profile updated successfully", userResp)
}

// ListUsers godoc
// @Summary List all users
// @Description Retrieve a list of all registered users. Restricted to admin and staff.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.WebResponse[[]dto.UserResponse]
// @Failure 403 {object} dto.WebResponse[any]
// @Failure 500 {object} dto.WebResponse[any]
// @Router /users [get]
func (ctrl *UserHandler) ListUsers(c *gin.Context) {
	users, err := ctrl.svc.ListUsers()
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	var userResps []dto.UserResponse
	for _, user := range users {
		userResps = append(userResps, dto.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		})
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Users retrieved successfully", userResps)
}

// Health godoc
// @Summary Health check
// @Description Get service, database, and redis connection status
// @Tags Health
// @Produce json
// @Success 200 {object} dto.WebResponse[any]
// @Router /health [get]
func (ctrl *UserHandler) Health(c *gin.Context) {
	dbStatus := "connected"
	if db.DB == nil {
		dbStatus = "disconnected"
	} else {
		sqlDB, err := db.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
		}
	}

	redisStatus := "connected"
	if redis.Client == nil {
		redisStatus = "disconnected"
	} else {
		_, err := redis.Client.Ping(redis.Ctx).Result()
		if err != nil {
			redisStatus = "disconnected"
		}
	}

	healthData := gin.H{
		"status":    "UP",
		"service":   "user-service",
		"timestamp": time.Now().Format(time.RFC3339),
		"database":  dbStatus,
		"redis":     redisStatus,
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Health status", healthData)
}
