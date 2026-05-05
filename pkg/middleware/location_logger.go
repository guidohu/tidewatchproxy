package middleware

import (
	"bytes"
	"io"
	"log"

	"github.com/gin-gonic/gin"
	"tide_watch_proxy/pkg/store"
	"tide_watch_proxy/pkg/util"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func LocationLogger(s *store.LocationStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

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
			// Try to get error type and upstream response from context if set by handler
			if e, exists := c.Get("error_type"); exists {
				errorType = e.(string)
			} else {
				errorType = "Unknown Error"
			}

			upstreamResp := ""
			if ur, exists := c.Get("upstream_response"); exists {
				upstreamResp = ur.(string)
			}

			// Detailed error logging
			errLog := store.ErrorLog{
				Method:           c.Request.Method,
				Path:             c.Request.URL.Path,
				Query:            c.Request.URL.RawQuery,
				StatusCode:       status,
				RequestBody:  string(reqBody),
				ResponseBody: blw.body.String(),
				UpstreamResponse: upstreamResp,
				Backend:      backendStr,
				ErrorType:    errorType,
			}

			// Log to database
			go s.LogError(errLog)

			// Log to command line
			log.Printf("[ERROR] %s %s | Status: %d | Backend: %s | Error: %s", 
				errLog.Method, errLog.Path, errLog.StatusCode, errLog.Backend, errLog.ErrorType)
			if errLog.RequestBody != "" {
				log.Printf("[ERROR] Request Body: %s", errLog.RequestBody)
			}
			if errLog.UpstreamResponse != "" {
				log.Printf("[ERROR] Upstream Response: %s", errLog.UpstreamResponse)
			}
			log.Printf("[ERROR] Response Body sent to client: %s", errLog.ResponseBody)
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
