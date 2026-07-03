package routes

import (
	"booking-service/handler"
	"booking-service/middleware"
	_ "booking-service/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter(bookingHandler *handler.BookingHandler) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.ErrorHandler())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
