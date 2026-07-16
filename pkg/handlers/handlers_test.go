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
	origTransport := httpClient.Transport
	defer func() {
		httpClient.Transport = origTransport
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

	httpClient.Transport = &mockRoundTripper{
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
	origTransport := httpClient.Transport
	defer func() {
		httpClient.Transport = origTransport
	}()

	var callCount int

	// Mock Stormglass returning invalid key error
	httpClient.Transport = &mockRoundTripper{
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

func TestStormglassQuotaExceededCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original transport
	origTransport := httpClient.Transport
	defer func() {
		httpClient.Transport = origTransport
	}()

	const quotaBody = `{"errors":{"key":"API quota exceeded"},"meta":{"dailyQuota":10,"requestCount":41}}`

	var callCount int

	// Mock Stormglass returning a quota-exceeded error
	httpClient.Transport = &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(quotaBody)),
				Header:     make(http.Header),
			}, nil
		},
	}

	// Use local memory cache (redisClient is nil)
	h := NewHandler(nil, "default-key", "", false, nil, false, nil)
	r := gin.Default()
	r.GET("/weather/point", func(c *gin.Context) {
		c.Set("api_key", "quota-key-1")
		h.HandleWeather(c)
	})

	// First request -> Should call upstream and return the quota error verbatim
	req, _ := http.NewRequest("GET", "/weather/point?lat=10&lng=20&params=swellHeight", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected first request status 429, got %v", w.Code)
	}
	if w.Body.String() != quotaBody {
		t.Errorf("Expected first response body to be direct upstream error, got `%s`", w.Body.String())
	}
	if callCount != 1 {
		t.Errorf("Expected upstream to be called exactly 1 time, called %d times", callCount)
	}

	// Second request with same key -> Should not call upstream (cached quota)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected second request status 429, got %v", w.Code)
	}
	if w.Body.String() != quotaBody {
		t.Errorf("Expected second response body to be cached quota error, got `%s`", w.Body.String())
	}
	if w.Header().Get("X-Cache") != "HIT" {
		t.Errorf("Expected cached quota response to set X-Cache: HIT, got `%s`", w.Header().Get("X-Cache"))
	}
	if callCount != 1 {
		t.Errorf("Expected upstream to NOT be called a second time (cached), call count: %d", callCount)
	}

	// Request with a different key -> Should call upstream
	r2 := gin.Default()
	r2.GET("/weather/point", func(c *gin.Context) {
		c.Set("api_key", "quota-key-2")
		h.HandleWeather(c)
	})
	req2, _ := http.NewRequest("GET", "/weather/point?lat=10&lng=20&params=swellHeight", nil)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)

	if callCount != 2 {
		t.Errorf("Expected upstream to be called for different key, call count: %d", callCount)
	}
}

func TestOpenWatersCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when starting miniredis", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// Save original transport
	origTransport := httpClient.Transport
	defer func() {
		httpClient.Transport = origTransport
	}()

	extremesJSON := `{
		"station": {"name": "Key West", "country": "USA"},
		"extremes": [{"time": "2026-06-02T10:00:00Z", "level": 0.45, "high": true}]
	}`

	var callCount int
	httpClient.Transport = &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(extremesJSON)),
				Header:     make(http.Header),
			}, nil
		},
	}

	h := NewHandler(redisClient, "", "", true, nil, false, nil)
	r := gin.Default()
	r.GET("/tides/extremes", h.HandleOpenWatersExtremes)

	req, _ := http.NewRequest("GET", "/tides/extremes?latitude=24.55&longitude=-81.8&datum=MLLW", nil)

	// First request -> cache MISS, calls upstream
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected first request status 200, got %v", w.Code)
	}
	if w.Header().Get("X-Cache") != "MISS" {
		t.Errorf("Expected first request X-Cache: MISS, got `%s`", w.Header().Get("X-Cache"))
	}
	if callCount != 1 {
		t.Errorf("Expected upstream to be called exactly 1 time, called %d times", callCount)
	}
	firstBody := w.Body.String()

	// Second identical request -> cache HIT, no upstream call, identical body
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected second request status 200, got %v", w.Code)
	}
	if w.Header().Get("X-Cache") != "HIT" {
		t.Errorf("Expected second request X-Cache: HIT, got `%s`", w.Header().Get("X-Cache"))
	}
	if callCount != 1 {
		t.Errorf("Expected upstream to NOT be called a second time (cached), call count: %d", callCount)
	}
	if w.Body.String() != firstBody {
		t.Errorf("Expected cached body to match first response.\nfirst: %s\ncached: %s", firstBody, w.Body.String())
	}

	// Request with different coordinates -> cache MISS, calls upstream again
	req2, _ := http.NewRequest("GET", "/tides/extremes?latitude=10&longitude=20&datum=MLLW", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if callCount != 2 {
		t.Errorf("Expected upstream to be called for different coordinates, call count: %d", callCount)
	}
}

func TestHandleGeocodeCacheAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when starting miniredis", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	h := NewHandler(redisClient, "", "", true, nil, false, nil)

	r := gin.Default()
	r.GET("/dashboard/api/geocode-cache", h.HandleGeocodeCacheAPI)

	// Test 1: Empty cache returns empty list
	req, _ := http.NewRequest("GET", "/dashboard/api/geocode-cache", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %v", w.Code)
	}
	var entries []GeocodeCacheEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}

	// Test 2: Cached entries are returned with parsed coordinates
	mr.Set("geocode:47.37,8.54", `{"locality":"Zurich District","city":"Zurich","countryName":"Switzerland","countryCode":"CH"}`)
	mr.Set("geocode:-33.86,151.21", `{"locality":"Sydney Harbour","city":"Sydney","countryName":"Australia","countryCode":"AU"}`)
	// Non-geocode keys must be ignored
	mr.Set("invalid_api_key:some-key", "1")

	req, _ = http.NewRequest("GET", "/dashboard/api/geocode-cache", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %v", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	found := false
	for _, e := range entries {
		if e.City == "Zurich" {
			found = true
			if e.CountryName != "Switzerland" {
				t.Errorf("Expected country name 'Switzerland', got '%s'", e.CountryName)
			}
			if e.CountryCode != "CH" {
				t.Errorf("Expected country code 'CH', got '%s'", e.CountryCode)
			}
			if e.Locality != "Zurich District" {
				t.Errorf("Expected locality 'Zurich District', got '%s'", e.Locality)
			}
			if e.Lat != 47.37 || e.Lng != 8.54 {
				t.Errorf("Expected coordinates 47.37, 8.54, got %v, %v", e.Lat, e.Lng)
			}
		}
	}
	if !found {
		t.Errorf("Expected to find Zurich entry in response")
	}

	// Test 3: Caching disabled returns empty list
	hNoCache := NewHandler(nil, "", "", false, nil, false, nil)
	rNoCache := gin.Default()
	rNoCache.GET("/dashboard/api/geocode-cache", hNoCache.HandleGeocodeCacheAPI)

	req, _ = http.NewRequest("GET", "/dashboard/api/geocode-cache", nil)
	w = httptest.NewRecorder()
	rNoCache.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with caching disabled, got %v", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with caching disabled, got %d", len(entries))
	}
}
