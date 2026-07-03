package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"booking-service/config"
	"booking-service/db"
	"booking-service/model"
	"booking-service/redis"
	"booking-service/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type BookingHandler struct {
	svc *service.BookingService
	cfg *config.Config
}

func NewBookingHandler(svc *service.BookingService, cfg *config.Config) *BookingHandler {
	return &BookingHandler{svc: svc, cfg: cfg}
}

func (ctrl *BookingHandler) AuthMiddleware() gin.HandlerFunc {
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

		c.Set("userID", userID)
		c.Next()
	}
}

func (ctrl *BookingHandler) ListConcerts(c *gin.Context) {
	concerts, err := ctrl.svc.GetConcerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, concerts)
}

type TicketCategoryInput struct {
	Name  string  `json:"name" binding:"required"`
	Price float64 `json:"price" binding:"required"`
	Seats int     `json:"seats" binding:"required,min=1"`
}

type CreateConcertRequest struct {
	Title            string                `json:"title" binding:"required"`
	Artist           string                `json:"artist" binding:"required"`
	Description      string                `json:"description"`
	Location         string                `json:"location" binding:"required"`
	Date             string                `json:"date" binding:"required"`
	TicketCategories []TicketCategoryInput `json:"ticket_categories"`
}

func (ctrl *BookingHandler) CreateConcert(c *gin.Context) {
	var req CreateConcertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsedDate, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use RFC3339 (e.g. 2026-12-31T20:00:00Z)"})
		return
	}

	var categories []model.TicketCategory
	for _, cat := range req.TicketCategories {
		categories = append(categories, model.TicketCategory{
			Name:           cat.Name,
			Price:          cat.Price,
			TotalSeats:     cat.Seats,
			AvailableSeats: cat.Seats,
		})
	}

	concert, err := ctrl.svc.CreateConcert(req.Title, req.Artist, req.Description, req.Location, parsedDate, categories)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, concert)
}

type CreateBookingRequest struct {
	Items []service.OrderItemInput `json:"items" binding:"required,dive"`
}

func (ctrl *BookingHandler) CreateBooking(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	booking, err := ctrl.svc.CreateBooking(userID, req.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, booking)
}

func (ctrl *BookingHandler) ConfirmPayment(c *gin.Context) {
	bookingIDStr := c.Param("id")
	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	booking, err := ctrl.svc.ConfirmPayment(uint(bookingID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment confirmed, tickets are successfully issued.",
		"booking": booking,
	})
}

func (ctrl *BookingHandler) Health(c *gin.Context) {
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
		"service":   "booking-service",
		"timestamp": time.Now().Format(time.RFC3339),
		"database":  dbStatus,
		"redis":     redisStatus,
	})
}
