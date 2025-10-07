package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth creates an API key authentication middleware
func APIKeyAuth(validAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Bearer token format
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token == validAPIKey {
					c.Next()
					return
				}
			}
		}

		// Check x-goog-api-key header (for Gemini compatibility)
		if apiKey := c.GetHeader("x-goog-api-key"); apiKey == validAPIKey {
			c.Next()
			return
		}

		// Check query parameter
		if apiKey := c.Query("key"); apiKey == validAPIKey {
			c.Next()
			return
		}

		// Unauthorized
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or missing API key",
		})
		c.Abort()
	}
}
