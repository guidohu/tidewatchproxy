package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"tide_watch_proxy/pkg/store"

	"github.com/gin-gonic/gin"
)

func setupTestStore(t *testing.T) (*store.LocationStore, string) {
	dbFile := "test_dashboard_metrics.db"
	s, err := store.NewLocationStore(dbFile)
	if err != nil {
		t.Fatalf("Failed to create location store: %v", err)
	}
	return s, dbFile
}

func teardownTestStore(s *store.LocationStore, dbFile string) {
	s.Close()
	os.Remove(dbFile)
	os.Remove(dbFile + "-shm")
	os.Remove(dbFile + "-wal")
}

func TestHandleLocationsAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, dbFile := setupTestStore(t)
	defer teardownTestStore(s, dbFile)

	// Give the store some direct db writes to bypass async flush issues for testing
	_, err := s.DB().Exec("INSERT INTO locations (lat, lng, count) VALUES (34.05, -118.25, 1)")
	if err != nil {
		t.Fatalf("Failed to seed db: %v", err)
	}
	_, err = s.DB().Exec("INSERT INTO requests (timestamp, backend, status_code, lat, lng, is_cache_hit) VALUES (datetime('now'), 'stormglass', 200, 34.05, -118.25, 0)")
	if err != nil {
		t.Fatalf("Failed to seed requests: %v", err)
	}

	h := NewDashboardHandler(s)
	r := gin.Default()
	r.GET("/dashboard/api/locations", h.HandleLocationsAPI)

	// Test case 1: Default days parameter (0)
	req, _ := http.NewRequest("GET", "/dashboard/api/locations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w.Code)
	}

	var locations []store.LocationStats
	if err := json.Unmarshal(w.Body.Bytes(), &locations); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(locations) == 0 {
		t.Errorf("Expected to find locations, found 0")
	}

	// Test case 2: Valid days parameter (1)
	req2, _ := http.NewRequest("GET", "/dashboard/api/locations?days=1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w2.Code)
	}

	var locations2 []store.LocationStats
	if err := json.Unmarshal(w2.Body.Bytes(), &locations2); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(locations2) == 0 {
		t.Errorf("Expected to find locations with days=1, found 0")
	}

    // Test case 3: Invalid days parameter (abc)
	req3, _ := http.NewRequest("GET", "/dashboard/api/locations?days=abc", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w3.Code)
	}
}

func TestHandleLocationsAPI_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, dbFile := setupTestStore(t)
	defer teardownTestStore(s, dbFile)

	// Close the DB immediately to trigger error
	s.DB().Close()

	h := NewDashboardHandler(s)
	r := gin.Default()
	r.GET("/dashboard/api/locations", h.HandleLocationsAPI)

	req, _ := http.NewRequest("GET", "/dashboard/api/locations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500 on db error, got %v", w.Code)
	}
}

func TestHandleStatsAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, dbFile := setupTestStore(t)
	defer teardownTestStore(s, dbFile)

	// Give the store some direct db writes to bypass async flush issues for testing
	_, err := s.DB().Exec("INSERT INTO requests (timestamp, backend, status_code, lat, lng, is_cache_hit) VALUES (datetime('now'), 'stormglass', 200, 34.05, -118.25, 0)")
	if err != nil {
		t.Fatalf("Failed to seed db: %v", err)
	}

	h := NewDashboardHandler(s)
	r := gin.Default()
	r.GET("/dashboard/api/stats", h.HandleStatsAPI)

	// Test case 1: Default days parameter
	req, _ := http.NewRequest("GET", "/dashboard/api/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w.Code)
	}

	var stats []store.BackendStats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(stats) == 0 {
		t.Errorf("Expected to find stats, found 0")
	}

	// Test case 2: Valid days parameter
	req2, _ := http.NewRequest("GET", "/dashboard/api/stats?days=1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %v", w2.Code)
	}

	var stats2 []store.BackendStats
	if err := json.Unmarshal(w2.Body.Bytes(), &stats2); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(stats2) == 0 {
		t.Errorf("Expected to find stats with days=1, found 0")
	}
}

func TestHandleStatsAPI_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, dbFile := setupTestStore(t)
	defer teardownTestStore(s, dbFile)

	// Close the DB immediately to trigger error
	s.DB().Close()

	h := NewDashboardHandler(s)
	r := gin.Default()
	r.GET("/dashboard/api/stats", h.HandleStatsAPI)

	req, _ := http.NewRequest("GET", "/dashboard/api/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500 on db error, got %v", w.Code)
	}
}
