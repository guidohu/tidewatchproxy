package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AppIDMiddleware(allowedAppIDs []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(allowedAppIDs) > 0 {
			appID := c.Request.Header.Get("X-App-Id")
			if appID == "" {
				appID = c.Query("app_id")
			}

			allowed := false
			for _, id := range allowedAppIDs {
				if id == appID {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: invalid or missing App ID"})
				return
			}
		}
		c.Next()
	}
}

func AuthMiddleware(stormglassAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Request.Header.Get("Authorization")
		if key == "" {
			key = c.Query("key")
		}

		if key == "" {
			key = stormglassAPIKey
		}

		if key == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "No API key provided"})
			return
		}

		c.Set("api_key", key)
		c.Next()
	}
}
