package middleware

import (
	"github.com/gin-gonic/gin"
	"tide_watch_proxy/pkg/store"
	"tide_watch_proxy/pkg/util"
)

func LocationLogger(s *store.LocationStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Proceed with the request
		c.Next()

		status := c.Writer.Status()
		backend, _ := c.Get("backend")
		backendStr, ok := backend.(string)
		if !ok {
			backendStr = "Unknown"
		}

		errorType := ""
		if status >= 400 {
			// Try to get error type from context if set by handler
			if e, exists := c.Get("error_type"); exists {
				errorType = e.(string)
			} else {
				errorType = "Unknown Error"
			}
		}

		// Try to extract latitude and longitude for the map
		latStr := c.Query("lat")
		if latStr == "" {
			latStr = c.Query("latitude")
		}

		lngStr := c.Query("lng")
		if lngStr == "" {
			lngStr = c.Query("longitude")
		}

		var lat, lng float64
		if latStr != "" && lngStr != "" {
			lat = util.MustParseFloat(latStr)
			lng = util.MustParseFloat(lngStr)
		}

		// Log request metrics (including coordinates if available)
		go s.LogRequest(backendStr, status, errorType, lat, lng)

		// Update aggregated location count only for successful requests
		if status < 400 && lat != 0 && lng != 0 {
			if util.IsValidCoordinate(lat) && util.IsValidCoordinate(lng) {
				go s.LogLocation(lat, lng)
			}
		}
	}
}
