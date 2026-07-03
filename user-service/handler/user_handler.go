package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"user-service/config"
	"user-service/db"
	"user-service/redis"
	"user-service/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer <token>"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been blacklisted or invalidated"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("token", tokenString)
		c.Set("exp", exp)
		c.Next()
	}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (ctrl *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.svc.Register(req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email for the OTP code.",
		"user":    user,
	})
}

type VerifyRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

func (ctrl *UserHandler) VerifyOTP(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.svc.VerifyOTP(req.Email, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account successfully verified and activated."})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (ctrl *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := ctrl.svc.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}

func (ctrl *UserHandler) Logout(c *gin.Context) {
	tokenString, _ := c.Get("token")
	expVal, _ := c.Get("exp")

	tokenStr := tokenString.(string)
	exp := expVal.(int64)

	err := ctrl.svc.Logout(tokenStr, exp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out from current session."})
}

func (ctrl *UserHandler) ForceLogout(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	err := ctrl.svc.ForceLogout(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out from all devices."})
}

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

	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"service":   "user-service",
		"timestamp": time.Now().Format(time.RFC3339),
		"database":  dbStatus,
		"redis":     redisStatus,
	})
}
