package routes

import (
	"user-service/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(userHandler *handler.UserHandler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", userHandler.Health)

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/verify-otp", userHandler.VerifyOTP)
			auth.POST("/login", userHandler.Login)
		}

		securedAuth := v1.Group("/auth")
		securedAuth.Use(userHandler.AuthMiddleware())
		{
			securedAuth.POST("/logout", userHandler.Logout)
			securedAuth.POST("/force-logout", userHandler.ForceLogout)
		}
	}

	return r
}
