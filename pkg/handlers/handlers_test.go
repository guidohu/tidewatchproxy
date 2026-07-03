package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"

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

func TestStormglassInvalidKeyCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original transport
	origTransport := http.DefaultClient.Transport
	defer func() {
		http.DefaultClient.Transport = origTransport
	}()

	var callCount int

	// Mock Stormglass returning invalid key error
	http.DefaultClient.Transport = &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"errors":{"key":"API key is invalid"}}`)),
				Header:     make(http.Header),
			}, nil
		},
	}

	// 1. Test invalid key caching using local memory cache (redisClient is nil)
	h := NewHandler(nil, "default-key", "", false, nil, false, nil)
	r := gin.Default()
	r.GET("/weather/point", func(c *gin.Context) {
		c.Set("api_key", "bad-key-1")
		h.HandleWeather(c)
	})

	// First request -> Should call upstream (mock transport)
	req, _ := http.NewRequest("GET", "/weather/point?lat=10&lng=20&params=swellHeight", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected first request status 403, got %v", w.Code)
	}
	if w.Body.String() != `{"errors":{"key":"API key is invalid"}}` {
		t.Errorf("Expected first response body to be direct upstream error, got `%s`", w.Body.String())
	}
	if callCount != 1 {
		t.Errorf("Expected upstream to be called exactly 1 time, called %d times", callCount)
	}

	// Second request with same key -> Should not call upstream (cached as invalid)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected second request status 403, got %v", w.Code)
	}
	if w.Body.String() != `{"errors":{"key":"API key is invalid (cached)"}}` {
		t.Errorf("Expected second response body to be cached error, got `%s`", w.Body.String())
	}
	if callCount != 1 {
		t.Errorf("Expected upstream to NOT be called a second time (cached), call count: %d", callCount)
	}

	// Request with a different key -> Should call upstream
	r2 := gin.Default()
	r2.GET("/weather/point", func(c *gin.Context) {
		c.Set("api_key", "bad-key-2")
		h.HandleWeather(c)
	})
	req2, _ := http.NewRequest("GET", "/weather/point?lat=10&lng=20&params=swellHeight", nil)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected request with different key status 403, got %v", w2.Code)
	}
	if callCount != 2 {
		t.Errorf("Expected upstream to be called for different key, call count: %d", callCount)
	}
}

func TestMarkAPIKeyInvalid_Memory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, "", "", false, nil, false, nil)

	// Test 1: Empty string should not be marked
	h.markAPIKeyInvalid("")
	if h.isAPIKeyInvalid("") {
		t.Errorf("Expected empty API key to not be marked invalid")
	}

	// Test 2: Valid string should be marked and identified
	validKey := "test-key-memory"
	h.markAPIKeyInvalid(validKey)
	if !h.isAPIKeyInvalid(validKey) {
		t.Errorf("Expected API key to be marked invalid")
	}

	// Test 3: Expired key should be cleaned up and return false
	h.invalidKeysMutex.Lock()
	h.invalidKeys[validKey] = time.Now().Add(-1 * time.Hour)
	h.invalidKeysMutex.Unlock()

	if h.isAPIKeyInvalid(validKey) {
		t.Errorf("Expected expired API key to not be marked invalid")
	}
}

func TestMarkAPIKeyInvalid_Redis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when starting miniredis", err)
	}
	defer mr.Close()

	// Initialize redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	h := NewHandler(redisClient, "", "", true, nil, false, nil)

	// Test 1: Empty string should not be marked
	h.markAPIKeyInvalid("")
	if h.isAPIKeyInvalid("") {
		t.Errorf("Expected empty API key to not be marked invalid")
	}

	// Test 2: Valid string should be marked and identified
	validKey := "test-key-redis"
	h.markAPIKeyInvalid(validKey)
	if !h.isAPIKeyInvalid(validKey) {
		t.Errorf("Expected API key to be marked invalid")
	}

	// Test 3: Unmarked key should not be identified as invalid
	unmarkedKey := "unmarked-key-redis"
	if h.isAPIKeyInvalid(unmarkedKey) {
		t.Errorf("Expected unmarked API key to not be marked invalid")
	}

	// Test 4: Expired key in Redis
	expiredKey := "expired-key-redis"
	h.markAPIKeyInvalid(expiredKey)
	mr.FastForward(24 * time.Hour) // Fast forward time to expire the key

	// Fast forward local memory cache manually
	h.invalidKeysMutex.Lock()
	h.invalidKeys[expiredKey] = time.Now().Add(-1 * time.Hour)
	h.invalidKeysMutex.Unlock()

	if h.isAPIKeyInvalid(expiredKey) {
		t.Errorf("Expected expired API key to not be marked invalid")
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		// Valid inputs
		{"Valid format 1.0.0", "1.0.0", true},
		{"Valid format 0.0.0", "0.0.0", true},
		{"Valid format with large numbers", "99.99.99", true},
		{"Valid format with multi-digit numbers", "10.200.3000", true},

		// Invalid inputs
		{"Empty string", "", false},
		{"Only two parts", "1.0", false},
		{"Four parts", "1.0.0.0", false},
		{"Contains letters", "1.a.0", false},
		{"Negative numbers", "-1.0.0", false},
		{"Contains spaces", " 1.0.0", false},
		{"Contains spaces middle", "1. 0.0", false},
		{"Contains symbols", "1.*.0", false},
		{"Double dots", "1..0", false},
		{"Missing first part", ".0.0", false},
		{"Missing last part", "1.0.", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidVersion(tc.version)
			if result != tc.expected {
				t.Errorf("isValidVersion(%q) = %v; expected %v", tc.version, result, tc.expected)
			}
		})
	}
}
