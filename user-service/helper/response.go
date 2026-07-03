package helper

import (
	"errors"
	"net/http"

	"user-service/dto"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func WriteSuccessResponse(c *gin.Context, code int, message string, data any) {
	c.JSON(code, dto.WebResponse[any]{
		Code:    code,
		Status:  http.StatusText(code),
		Message: message,
		Data:    data,
	})
}

func WriteErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, dto.WebResponse[any]{
		Code:    code,
		Status:  http.StatusText(code),
		Message: message,
	})
}

func WriteValidationErrorResponse(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	var errs []dto.ValidationError
	if errors.As(err, &ve) {
		for _, fe := range ve {
			errs = append(errs, dto.ValidationError{
				Field: fe.Field(),
				Error: fe.Tag(),
			})
		}
	} else {
		// Fallback for general binding errors
		errs = append(errs, dto.ValidationError{
			Field: "body",
			Error: err.Error(),
		})
	}

	c.JSON(http.StatusBadRequest, dto.WebResponse[any]{
		Code:    http.StatusBadRequest,
		Status:  http.StatusText(http.StatusBadRequest),
		Message: "Validation failed",
		Errors:  errs,
	})
}
