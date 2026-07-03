package middleware

import (
	"log"
	"net/http"
	"user-service/helper"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERED] %v", err)
				helper.WriteErrorResponse(c, http.StatusInternalServerError, "Internal Server Error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
