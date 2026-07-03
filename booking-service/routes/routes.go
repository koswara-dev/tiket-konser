package routes

import (
	"booking-service/handler"
	"booking-service/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(bookingHandler *handler.BookingHandler) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.ErrorHandler())

	r.GET("/health", bookingHandler.Health)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/concerts", bookingHandler.ListConcerts)
		v1.POST("/concerts", bookingHandler.AuthMiddleware(), middleware.RequireRoles("admin", "staff"), bookingHandler.CreateConcert)

		secured := v1.Group("/")
		secured.Use(bookingHandler.AuthMiddleware())
		{
			secured.POST("/bookings", bookingHandler.CreateBooking)
			secured.GET("/bookings/:id", bookingHandler.GetBooking)
			secured.POST("/bookings/:id/pay", middleware.RequireRoles("admin", "staff"), bookingHandler.ConfirmPayment)
		}
	}

	return r
}
