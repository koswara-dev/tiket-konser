package middleware

import (
	"booking-service/helper"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			helper.WriteErrorResponse(c, http.StatusForbidden, "Forbidden: Role not found")
			c.Abort()
			return
		}

		role := roleVal.(string)
		isAllowed := false
		for _, r := range allowedRoles {
			if r == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			helper.WriteErrorResponse(c, http.StatusForbidden, "Forbidden: You do not have permission to access this resource")
			c.Abort()
			return
		}

		c.Next()
	}
}
