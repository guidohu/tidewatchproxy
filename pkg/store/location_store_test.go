package store

import (
	"os"
	"testing"
	"time"
)

func TestLocationStore_PingLoggingAndQuery(t *testing.T) {
	dbFile := "test_metrics_ping.db"
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + "-shm")
	defer os.Remove(dbFile + "-wal")

	s, err := NewLocationStore(dbFile)
	if err != nil {
		t.Fatalf("Failed to create location store: %v", err)
	}
	defer s.Close()

	// Log some pings
	s.LogPing("user-1", "1.0.0")
	s.LogPing("user-2", "1.0.1")
	s.LogPing("user-1", "1.1.0") // User-1 upgraded to 1.1.0

	// Since LogPing writes asynchronously to channel, we flush manually or wait a bit
	// Let's sleep a moment to let the background worker flush
	time.Sleep(6 * time.Second)

	// Verify user versions table
	users, err := s.GetUsersPerVersion(1)
	if err != nil {
		t.Fatalf("GetUsersPerVersion failed: %v", err)
	}

	// We expect 1 user on 1.0.1 (user-2) and 1 user on 1.1.0 (user-1, upgraded)
	// Let's count them
	var count101, count110 int
	for _, vc := range users {
		if vc.Version == "1.0.1" {
			count101 = vc.Count
		} else if vc.Version == "1.1.0" {
			count110 = vc.Count
		}
	}

	if count101 != 1 {
		t.Errorf("Expected 1 user on version 1.0.1, got %d", count101)
	}
	if count110 != 1 {
		t.Errorf("Expected 1 user on version 1.1.0, got %d", count110)
	}

	// Verify ping usage trend over time
	stats, err := s.GetPingUsageStats(1) // 1 day timeframe
	if err != nil {
		t.Fatalf("GetPingUsageStats failed: %v", err)
	}

	if len(stats) == 0 {
		t.Fatalf("Expected some ping usage stats, got 0")
	}
}

func TestLocationStore_GetFailureTrend(t *testing.T) {
	dbFile := "test_metrics_failure_trend.db"
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + "-shm")
	defer os.Remove(dbFile + "-wal")

	s, err := NewLocationStore(dbFile)
	if err != nil {
		t.Fatalf("Failed to create location store: %v", err)
	}
	defer s.Close()

	// Insert directly so we control the timestamps.
	insert := func(offset string, statusCode int, errorType string) {
		_, err := s.DB().Exec(
			"INSERT INTO requests (timestamp, backend, status_code, error_type) VALUES (datetime('now', ?), 'test', ?, ?)",
			offset, statusCode, errorType,
		)
		if err != nil {
			t.Fatalf("Failed to insert request: %v", err)
		}
	}

	insert("-10 minutes", 500, "upstream_timeout")
	insert("-10 minutes", 500, "upstream_timeout")
	insert("-10 minutes", 429, "rate_limited")
	insert("-3 hours", 500, "upstream_timeout")
	insert("-10 minutes", 200, "") // success, must be excluded
	insert("-40 days", 503, "ancient_failure")

	// Last 24 hours: 5 minute buckets, the 40 day old failure is out of range.
	stats, err := s.GetFailureTrend(1)
	if err != nil {
		t.Fatalf("GetFailureTrend failed: %v", err)
	}

	totals := map[string]int{}
	buckets := map[string]bool{}
	for _, st := range stats {
		totals[st.Reason] += st.Count
		buckets[st.Bucket] = true
	}

	if totals["upstream_timeout"] != 3 {
		t.Errorf("Expected 3 upstream_timeout failures, got %d", totals["upstream_timeout"])
	}
	if totals["rate_limited"] != 1 {
		t.Errorf("Expected 1 rate_limited failure, got %d", totals["rate_limited"])
	}
	if _, ok := totals["ancient_failure"]; ok {
		t.Errorf("Expected the 40 day old failure to be outside the 24h window")
	}
	if totals[""] != 0 {
		t.Errorf("Expected successful requests to be excluded, got %d", totals[""])
	}
	// The -10 minutes and -3 hours failures must land in separate buckets.
	if len(buckets) != 2 {
		t.Errorf("Expected 2 distinct time buckets, got %d: %v", len(buckets), buckets)
	}

	// All Time: the 40 day old failure is included.
	allStats, err := s.GetFailureTrend(0)
	if err != nil {
		t.Fatalf("GetFailureTrend(0) failed: %v", err)
	}
	var foundAncient bool
	for _, st := range allStats {
		if st.Reason == "ancient_failure" {
			foundAncient = true
		}
	}
	if !foundAncient {
		t.Errorf("Expected ancient_failure in the all time trend")
	}

	// The remaining timeframes must at least run and stay within their window.
	for _, days := range []int{7, 30} {
		if _, err := s.GetFailureTrend(days); err != nil {
			t.Fatalf("GetFailureTrend(%d) failed: %v", days, err)
		}
	}
}
