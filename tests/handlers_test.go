package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"tide_watch_proxy/pkg/handlers"
	"tide_watch_proxy/pkg/models"
)

func TestHandleOpenWatersExtremes_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewHandler(nil, "", false, nil, false)

	r := gin.Default()
	r.GET("/tides/extremes", h.HandleOpenWatersExtremes)

	// Test 1: Missing params
	req, _ := http.NewRequest("GET", "/tides/extremes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %v", w.Code)
	}

	// Test 2: Invalid datum
	req, _ = http.NewRequest("GET", "/tides/extremes?latitude=0&longitude=0&datum=INVALID", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid datum, got %v", w.Code)
	}
}

// More tests could be added here by mocking the http.DefaultClient
// or using a library like 'gherkin' or 'sqlmock' / 'redismock'.
