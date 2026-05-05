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

		// Skip tracking if the request failed or returned error
		if c.Writer.Status() >= 400 {
			return
		}

		// Try to extract latitude and longitude
		latStr := c.Query("lat")
		if latStr == "" {
			latStr = c.Query("latitude")
		}

		lngStr := c.Query("lng")
		if lngStr == "" {
			lngStr = c.Query("longitude")
		}

		if latStr != "" && lngStr != "" {
			lat := util.MustParseFloat(latStr)
			lng := util.MustParseFloat(lngStr)

			if util.IsValidCoordinate(lat) && util.IsValidCoordinate(lng) {
				// Run in a goroutine so it doesn't block the response
				go s.LogLocation(lat, lng)
			}
		}
	}
}
