package routes

import (
	"user-service/handler"
	"user-service/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(userHandler *handler.UserHandler) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.ErrorHandler())

	r.GET("/health", userHandler.Health)

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/verify-otp", userHandler.VerifyOTP)
			auth.POST("/login", userHandler.Login)
		}

		secured := v1.Group("/")
		secured.Use(userHandler.AuthMiddleware())
		{
			secured.POST("/auth/logout", userHandler.Logout)
			secured.POST("/auth/force-logout", userHandler.ForceLogout)
			secured.GET("/users/:id", userHandler.GetProfile)
			secured.PUT("/users/:id", userHandler.UpdateProfile)
			secured.GET("/users", middleware.RequireRoles("admin", "staff"), userHandler.ListUsers)
		}
	}

	return r
}
