package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleOpenWatersExtremes_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", false, nil, false)

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

func TestHandleOpenWatersTimeline_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", false, nil, false)
	r := gin.Default()
	r.GET("/tides/timeline", h.HandleOpenWatersTimeline)

	// Test 1: Missing params
	req, _ := http.NewRequest("GET", "/tides/timeline", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %v", w.Code)
	}

	// Test 2: Invalid datum
	req, _ = http.NewRequest("GET", "/tides/timeline?latitude=0&longitude=0&datum=INVALID", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid datum, got %v", w.Code)
	}
}

func TestHandleWeather_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", false, nil, false)
	r := gin.Default()
	r.GET("/weather/point", h.HandleWeather)

	// Test 1: Missing params
	req, _ := http.NewRequest("GET", "/weather/point", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %v", w.Code)
	}

	// Test 2: Invalid coordinates
	req, _ = http.NewRequest("GET", "/weather/point?lat=200&lng=0", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid coordinates, got %v", w.Code)
	}

	// Test 3: No valid weather params
	req, _ = http.NewRequest("GET", "/weather/point?lat=0&lng=0&params=invalid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for no valid weather params, got %v", w.Code)
	}
}

func TestHandleTides_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", false, nil, false)
	r := gin.Default()
	r.GET("/tides/stormglass", h.HandleTides)

	// Test 1: Missing params
	req, _ := http.NewRequest("GET", "/tides/stormglass", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %v", w.Code)
	}
}

func TestHandleSeaLevel_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", false, nil, false)
	r := gin.Default()
	r.GET("/sealevel", h.HandleSeaLevel)

	// Test 1: Missing params
	req, _ := http.NewRequest("GET", "/sealevel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %v", w.Code)
	}
}

func TestHandleReverseGeocode_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", false, nil, false)
	r := gin.Default()
	r.GET("/geocoding", h.HandleReverseGeocode)

	// Test 1: Missing params
	req, _ := http.NewRequest("GET", "/geocoding", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %v", w.Code)
	}
}

// More tests could be added here by mocking the http.DefaultClient
// or using a library like 'gherkin' or 'sqlmock' / 'redismock'.
