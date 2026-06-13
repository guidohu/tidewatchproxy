package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleOpenWatersExtremes_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", "", false, nil, false, nil)

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
	h := NewHandler(nil, "", "", false, nil, false, nil)
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
	h := NewHandler(nil, "", "", false, nil, false, nil)
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
	h := NewHandler(nil, "", "", false, nil, false, nil)
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
	h := NewHandler(nil, "", "", false, nil, false, nil)
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
	h := NewHandler(nil, "", "", false, nil, false, nil)
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

func TestHandlePing_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", "", false, nil, false, nil)
	r := gin.Default()
	r.GET("/ping", h.HandlePing)

	// Test 1: Missing params
	req, _ := http.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %v", w.Code)
	}

	// Test 2: Invalid version format
	req, _ = http.NewRequest("GET", "/ping?uuid=test-uuid&version=1.0", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid version format, got %v", w.Code)
	}

	// Test 3: Invalid version letters
	req, _ = http.NewRequest("GET", "/ping?uuid=test-uuid&version=1.a.0", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid version letters, got %v", w.Code)
	}

	// Test 4: Valid ping
	// Note: We pass a mock or nil locationStore which works fine for validation checks but would panic on write.
	// Since HandlePing will call h.locationStore.LogPing, we'll verify it returns 400 for errors beforehand.
}

type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestHandleOpenWaters_Parsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", "", false, nil, false, nil)
	r := gin.Default()
	r.GET("/tides/timeline", h.HandleOpenWatersTimeline)
	r.GET("/tides/extremes", h.HandleOpenWatersExtremes)

	// Save original transport
	origTransport := http.DefaultClient.Transport
	defer func() {
		http.DefaultClient.Transport = origTransport
	}()

	timelineJSON := `{
		"datum": "MLLW",
		"units": "meters",
		"station": {
			"id": "8722588",
			"name": "Key West",
			"region": "Florida",
			"country": "USA",
			"continent": "North America",
			"timezone": "America/New_York",
			"type": "reference",
			"latitude": 24.55,
			"longitude": -81.8
		},
		"distance": 120.5,
		"timeline": [
			{
				"time": "2026-06-02T10:00:00Z",
				"level": 0.45
			}
		]
	}`

	extremesJSON := `{
		"datum": "MLLW",
		"units": "meters",
		"station": {
			"id": "8722588",
			"name": "Key West",
			"region": "Florida",
			"country": "USA",
			"continent": "North America",
			"timezone": "America/New_York",
			"type": "reference",
			"latitude": 24.55,
			"longitude": -81.8
		},
		"distance": 120.5,
		"extremes": [
			{
				"time": "2026-06-02T10:00:00Z",
				"level": 0.45,
				"high": true
			}
		]
	}`

	http.DefaultClient.Transport = &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			var respStr string
			if strings.Contains(req.URL.Path, "/timeline") {
				respStr = timelineJSON
			} else if strings.Contains(req.URL.Path, "/extremes") {
				respStr = extremesJSON
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(respStr)),
				Header:     make(http.Header),
			}, nil
		},
	}

	// 1. Test timeline parsing
	req, _ := http.NewRequest("GET", "/tides/timeline?latitude=24.55&longitude=-81.8&datum=MLLW", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w.Code)
	}

	var timelineResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &timelineResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	station, ok := timelineResp["station"].(map[string]interface{})
	if !ok {
		t.Fatalf("Station field missing or invalid in timeline response")
	}

	if station["name"] != "Key West" || station["country"] != "USA" {
		t.Errorf("Unexpected station fields: %v", station)
	}

	if _, exists := station["id"]; exists {
		t.Errorf("id field should have been filtered out")
	}
	if _, exists := station["region"]; exists {
		t.Errorf("region field should have been filtered out")
	}
	if _, exists := station["type"]; exists {
		t.Errorf("type field should have been filtered out")
	}

	// Ensure filtered out fields are NOT present
	if _, exists := station["latitude"]; exists {
		t.Errorf("Latitude field should have been filtered out")
	}
	if _, exists := station["longitude"]; exists {
		t.Errorf("Longitude field should have been filtered out")
	}
	if _, exists := station["timezone"]; exists {
		t.Errorf("Timezone field should have been filtered out")
	}

	// 2. Test extremes parsing
	req, _ = http.NewRequest("GET", "/tides/extremes?latitude=24.55&longitude=-81.8&datum=MLLW", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w.Code)
	}

	var extremesResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &extremesResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	station, ok = extremesResp["station"].(map[string]interface{})
	if !ok {
		t.Fatalf("Station field missing or invalid in extremes response")
	}

	if station["id"] != "8722588" || station["name"] != "Key West" || station["region"] != "Florida" || station["country"] != "USA" || station["type"] != "reference" {
		t.Errorf("Unexpected station fields: %v", station)
	}
}
