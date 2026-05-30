package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/university-bus-tracker/shared/types"
)

// RespondWithError sends a standardized error response
func RespondWithError(c *gin.Context, statusCode int, code, message string, details interface{}) {
	c.JSON(statusCode, types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:      code,
			Message:   message,
			Details:   details,
			Timestamp: time.Now(),
		},
	})
}

// RespondWithJSON sends a JSON response
func RespondWithJSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}
