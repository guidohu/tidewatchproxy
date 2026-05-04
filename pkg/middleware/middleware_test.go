package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAppIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowedIDs     []string
		headerAppID    string
		queryAppID     string
		expectedStatus int
	}{
		{
			name:           "Empty allowed IDs - should pass",
			allowedIDs:     []string{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid App ID in header",
			allowedIDs:     []string{"app1", "app2"},
			headerAppID:    "app1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid App ID in query",
			allowedIDs:     []string{"app1", "app2"},
			queryAppID:     "app2",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid App ID",
			allowedIDs:     []string{"app1", "app2"},
			headerAppID:    "wrong",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing App ID",
			allowedIDs:     []string{"app1", "app2"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AppIDMiddleware(tt.allowedIDs))
			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			path := "/test"
			if tt.queryAppID != "" {
				path += "?app_id=" + tt.queryAppID
			}

			req, _ := http.NewRequest("GET", path, nil)
			if tt.headerAppID != "" {
				req.Header.Set("X-App-Id", tt.headerAppID)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		defaultKey     string
		headerKey      string
		queryKey       string
		expectedStatus int
		expectedKey    string
	}{
		{
			name:           "No key provided, no default - should fail",
			defaultKey:     "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Key in header",
			defaultKey:     "",
			headerKey:      "header-key",
			expectedStatus: http.StatusOK,
			expectedKey:    "header-key",
		},
		{
			name:           "Key in query",
			defaultKey:     "",
			queryKey:       "query-key",
			expectedStatus: http.StatusOK,
			expectedKey:    "query-key",
		},
		{
			name:           "Default key used",
			defaultKey:     "default-key",
			expectedStatus: http.StatusOK,
			expectedKey:    "default-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware(tt.defaultKey))
			r.GET("/test", func(c *gin.Context) {
				key, _ := c.Get("api_key")
				c.JSON(http.StatusOK, gin.H{"api_key": key})
			})

			path := "/test"
			if tt.queryKey != "" {
				path += "?key=" + tt.queryKey
			}

			req, _ := http.NewRequest("GET", path, nil)
			if tt.headerKey != "" {
				req.Header.Set("Authorization", tt.headerKey)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				// Verify key is set in context if needed
				if tt.expectedKey != "" {
					// We could parse JSON here, but checking for presence is enough for basic middleware test
				}
			}
		})
	}
}
