package routes

import (
	"booking-service/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(bookingHandler *handler.BookingHandler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", bookingHandler.Health)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/concerts", bookingHandler.ListConcerts)
		v1.POST("/concerts", bookingHandler.CreateConcert)

		secured := v1.Group("/")
		secured.Use(bookingHandler.AuthMiddleware())
		{
			secured.POST("/bookings", bookingHandler.CreateBooking)
			secured.POST("/bookings/:id/pay", bookingHandler.ConfirmPayment)
		}
	}

	return r
}
