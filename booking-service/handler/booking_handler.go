package handler

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"booking-service/config"
	"booking-service/db"
	"booking-service/dto"
	"booking-service/helper"
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

		c.Set("userID", userID)
		c.Set("role", roleVal)
		c.Next()
	}
}

// ListConcerts godoc
// @Summary List concerts
// @Description Get a paginated list of concerts with optional case-insensitive search by title or artist
// @Tags Concerts
// @Produce json
// @Param search query string false "Search query (title or artist)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Limit per page (default: 10)"
// @Success 200 {object} dto.WebResponseWithPaging[[]dto.ConcertResponse]
// @Failure 500 {object} dto.WebResponse[any]
// @Router /concerts [get]
func (ctrl *BookingHandler) ListConcerts(c *gin.Context) {
	search := c.Query("search")

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	concerts, totalRows, err := ctrl.svc.GetConcerts(search, page, limit)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	var concertResps []dto.ConcertResponse
	for _, concert := range concerts {
		concertResps = append(concertResps, mapConcertResponse(concert))
	}

	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))
	pagingInfo := dto.PagingResponse{
		Page:       page,
		Limit:      limit,
		TotalRows:  totalRows,
		TotalPages: totalPages,
	}

	helper.WritePagingResponse(c, http.StatusOK, "Concerts retrieved successfully", concertResps, pagingInfo)
}

// CreateConcert godoc
// @Summary Create a concert
// @Description Add a new concert with ticket categories. Restricted to admin and staff.
// @Tags Concerts
// @Accept json
// @Produce json
// @Param request body dto.CreateConcertRequest true "Create concert request"
// @Security BearerAuth
// @Success 201 {object} dto.WebResponse[dto.ConcertResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 403 {object} dto.WebResponse[any]
// @Failure 500 {object} dto.WebResponse[any]
// @Router /concerts [post]
func (ctrl *BookingHandler) CreateConcert(c *gin.Context) {
	var req dto.CreateConcertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.WriteValidationErrorResponse(c, err)
		return
	}

	parsedDate, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, "invalid date format, use RFC3339 (e.g. 2026-12-31T20:00:00Z)")
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
		helper.WriteErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.WriteSuccessResponse(c, http.StatusCreated, "Concert created successfully", mapConcertResponse(*concert))
}

// CreateBooking godoc
// @Summary Create a booking
// @Description Book tickets for category classes. Requires authentication.
// @Tags Bookings
// @Accept json
// @Produce json
// @Param request body dto.CreateBookingRequest true "Create booking request"
// @Security BearerAuth
// @Success 201 {object} dto.WebResponse[dto.BookingResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 401 {object} dto.WebResponse[any]
// @Router /bookings [post]
func (ctrl *BookingHandler) CreateBooking(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		helper.WriteErrorResponse(c, http.StatusUnauthorized, "User unauthorized")
		return
	}
	userID := userIDVal.(uint)

	var req dto.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.WriteValidationErrorResponse(c, err)
		return
	}

	var serviceItems []service.OrderItemInput
	for _, item := range req.Items {
		serviceItems = append(serviceItems, service.OrderItemInput{
			TicketCategoryID: item.TicketCategoryID,
			Quantity:         item.Quantity,
		})
	}

	booking, err := ctrl.svc.CreateBooking(userID, serviceItems)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.WriteSuccessResponse(c, http.StatusCreated, "Booking created successfully", mapBookingResponse(*booking))
}

// ConfirmPayment godoc
// @Summary Confirm payment
// @Description Mark booking status as confirmed. Restricted to admin and staff.
// @Tags Bookings
// @Produce json
// @Param id path int true "Booking ID"
// @Security BearerAuth
// @Success 200 {object} dto.WebResponse[dto.BookingResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 403 {object} dto.WebResponse[any]
// @Router /bookings/{id}/pay [post]
func (ctrl *BookingHandler) ConfirmPayment(c *gin.Context) {
	bookingIDStr := c.Param("id")
	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, "invalid booking id")
		return
	}

	booking, err := ctrl.svc.ConfirmPayment(uint(bookingID))
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Payment confirmed, tickets are successfully issued.", mapBookingResponse(*booking))
}

// Health godoc
// @Summary Health check
// @Description Get service, database, and redis connection status
// @Tags Health
// @Produce json
// @Success 200 {object} dto.WebResponse[any]
// @Router /health [get]
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

	healthData := gin.H{
		"status":    "UP",
		"service":   "booking-service",
		"timestamp": time.Now().Format(time.RFC3339),
		"database":  dbStatus,
		"redis":     redisStatus,
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Health status", healthData)
}

func mapConcertResponse(concert model.Concert) dto.ConcertResponse {
	var cats []dto.TicketCategoryResponse
	for _, cat := range concert.TicketCategories {
		cats = append(cats, dto.TicketCategoryResponse{
			ID:             cat.ID,
			Name:           cat.Name,
			Price:          cat.Price,
			TotalSeats:     cat.TotalSeats,
			AvailableSeats: cat.AvailableSeats,
		})
	}

	return dto.ConcertResponse{
		ID:               concert.ID,
		Title:            concert.Title,
		Artist:           concert.Artist,
		Description:      concert.Description,
		Date:             concert.Date.Format(time.RFC3339),
		Location:         concert.Location,
		TicketCategories: cats,
	}
}

func mapBookingResponse(booking model.Booking) dto.BookingResponse {
	var items []dto.BookingItemResponse
	for _, item := range booking.BookingItems {
		var catResp dto.TicketCategoryResponse
		if item.TicketCategory.ID != 0 {
			catResp = dto.TicketCategoryResponse{
				ID:             item.TicketCategory.ID,
				Name:           item.TicketCategory.Name,
				Price:          item.TicketCategory.Price,
				TotalSeats:     item.TicketCategory.TotalSeats,
				AvailableSeats: item.TicketCategory.AvailableSeats,
			}
		}

		items = append(items, dto.BookingItemResponse{
			ID:               item.ID,
			TicketCategoryID: item.TicketCategoryID,
			TicketCategory:   catResp,
			Quantity:         item.Quantity,
			SubTotal:         item.SubTotal,
		})
	}

	return dto.BookingResponse{
		ID:           booking.ID,
		UserID:       booking.UserID,
		BookingDate:  booking.BookingDate.Format(time.RFC3339),
		TotalAmount:  booking.TotalAmount,
		Status:       booking.Status,
		BookingItems: items,
	}
}

// GetBooking godoc
// @Summary Get booking details
// @Description Retrieve booking order items and total pricing. Standard user can only fetch their own bookings. Admin/staff can fetch any.
// @Tags Bookings
// @Produce json
// @Param id path int true "Booking ID"
// @Security BearerAuth
// @Success 200 {object} dto.WebResponse[dto.BookingResponse]
// @Failure 400 {object} dto.WebResponse[any]
// @Failure 403 {object} dto.WebResponse[any]
// @Failure 404 {object} dto.WebResponse[any]
// @Router /bookings/{id} [get]
func (ctrl *BookingHandler) GetBooking(c *gin.Context) {
	bookingIDStr := c.Param("id")
	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusBadRequest, "invalid booking id")
		return
	}

	userIDVal, _ := c.Get("userID")
	roleVal, _ := c.Get("role")
	authenticatedUserID := userIDVal.(uint)
	role := roleVal.(string)

	booking, err := ctrl.svc.GetBookingByID(uint(bookingID))
	if err != nil {
		helper.WriteErrorResponse(c, http.StatusNotFound, "Booking not found")
		return
	}

	// IDOR Protection: a standard user can only retrieve their own booking
	// Bypassed if request has admin or staff roles
	if role != "admin" && role != "staff" && booking.UserID != authenticatedUserID {
		helper.WriteErrorResponse(c, http.StatusForbidden, "Forbidden: You are not authorized to view this booking")
		return
	}

	helper.WriteSuccessResponse(c, http.StatusOK, "Booking retrieved successfully", mapBookingResponse(*booking))
}
