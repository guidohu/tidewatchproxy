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
	// Inside the 30 day retention window, so the startup cleanup cannot race us.
	insert("-20 days", 503, "old_failure")

	// Last 24 hours: 5 minute buckets, the 40 day old failure is out of range.
	trend, err := s.GetFailureTrend(1)
	if err != nil {
		t.Fatalf("GetFailureTrend failed: %v", err)
	}

	totals := map[string]int{}
	pointBuckets := map[string]bool{}
	for _, p := range trend.Points {
		totals[p.Reason] += p.Count
		pointBuckets[p.Bucket] = true
	}

	if totals["upstream_timeout"] != 3 {
		t.Errorf("Expected 3 upstream_timeout failures, got %d", totals["upstream_timeout"])
	}
	if totals["rate_limited"] != 1 {
		t.Errorf("Expected 1 rate_limited failure, got %d", totals["rate_limited"])
	}
	if _, ok := totals["old_failure"]; ok {
		t.Errorf("Expected the 20 day old failure to be outside the 24h window")
	}
	if totals[""] != 0 {
		t.Errorf("Expected successful requests to be excluded, got %d", totals[""])
	}
	// The -10 minutes and -3 hours failures must land in separate buckets.
	if len(pointBuckets) != 2 {
		t.Errorf("Expected 2 distinct buckets with failures, got %d: %v", len(pointBuckets), pointBuckets)
	}

	// The timeline must cover the whole 24 hours in 5 minute steps, including
	// the buckets where nothing failed.
	if len(trend.Buckets) != 289 {
		t.Errorf("Expected 289 buckets covering 24h in 5 minute steps, got %d", len(trend.Buckets))
	}
	assertGapless(t, trend.Buckets, "2006-01-02 15:04", 5*time.Minute)
	assertCoversPoints(t, trend)

	// Every bucket must be usable as a chart slot, so the last one is the
	// current 5 minute window.
	if last := trend.Buckets[len(trend.Buckets)-1]; last != time.Now().UTC().Truncate(5*time.Minute).Format("2006-01-02 15:04") {
		t.Errorf("Expected the timeline to end at the current bucket, got %s", last)
	}

	// 7 days: hourly buckets.
	trend7, err := s.GetFailureTrend(7)
	if err != nil {
		t.Fatalf("GetFailureTrend(7) failed: %v", err)
	}
	if len(trend7.Buckets) != 169 {
		t.Errorf("Expected 169 hourly buckets over 7 days, got %d", len(trend7.Buckets))
	}
	assertGapless(t, trend7.Buckets, "2006-01-02 15:04", time.Hour)
	assertCoversPoints(t, trend7)

	// 30 days: daily buckets.
	trend30, err := s.GetFailureTrend(30)
	if err != nil {
		t.Fatalf("GetFailureTrend(30) failed: %v", err)
	}
	if len(trend30.Buckets) != 31 {
		t.Errorf("Expected 31 daily buckets over 30 days, got %d", len(trend30.Buckets))
	}
	assertGapless(t, trend30.Buckets, "2006-01-02", 24*time.Hour)
	assertCoversPoints(t, trend30)

	// All time: anchored at the oldest failure, so the 20 day old one is both
	// included and spanned by the timeline.
	trendAll, err := s.GetFailureTrend(0)
	if err != nil {
		t.Fatalf("GetFailureTrend(0) failed: %v", err)
	}
	var foundOld bool
	for _, p := range trendAll.Points {
		if p.Reason == "old_failure" {
			foundOld = true
		}
	}
	if !foundOld {
		t.Errorf("Expected old_failure in the all time trend")
	}
	if len(trendAll.Buckets) != 21 {
		t.Errorf("Expected 21 daily buckets from the 20 day old failure to today, got %d", len(trendAll.Buckets))
	}
	assertGapless(t, trendAll.Buckets, "2006-01-02", 24*time.Hour)
	assertCoversPoints(t, trendAll)
}

func TestLocationStore_GetFailureTrend_NoFailures(t *testing.T) {
	dbFile := "test_metrics_failure_trend_empty.db"
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + "-shm")
	defer os.Remove(dbFile + "-wal")

	s, err := NewLocationStore(dbFile)
	if err != nil {
		t.Fatalf("Failed to create location store: %v", err)
	}
	defer s.Close()

	// A quiet window still needs a full axis to draw, otherwise the chart
	// collapses to nothing instead of showing that there were no errors.
	trend, err := s.GetFailureTrend(1)
	if err != nil {
		t.Fatalf("GetFailureTrend failed: %v", err)
	}
	if len(trend.Points) != 0 {
		t.Errorf("Expected no failure points, got %d", len(trend.Points))
	}
	if len(trend.Buckets) != 289 {
		t.Errorf("Expected a full 24h timeline even without failures, got %d buckets", len(trend.Buckets))
	}

	// All time has nothing to anchor on, so it stays empty.
	trendAll, err := s.GetFailureTrend(0)
	if err != nil {
		t.Fatalf("GetFailureTrend(0) failed: %v", err)
	}
	if len(trendAll.Buckets) != 0 {
		t.Errorf("Expected an empty all time timeline without failures, got %d buckets", len(trendAll.Buckets))
	}
}

// assertGapless verifies the buckets are ordered and exactly one step apart.
func assertGapless(t *testing.T, buckets []string, layout string, step time.Duration) {
	t.Helper()
	for i := 1; i < len(buckets); i++ {
		prev, err := time.ParseInLocation(layout, buckets[i-1], time.UTC)
		if err != nil {
			t.Fatalf("Failed to parse bucket %q: %v", buckets[i-1], err)
		}
		cur, err := time.ParseInLocation(layout, buckets[i], time.UTC)
		if err != nil {
			t.Fatalf("Failed to parse bucket %q: %v", buckets[i], err)
		}
		if got := cur.Sub(prev); got != step {
			t.Fatalf("Expected %v between %s and %s, got %v", step, buckets[i-1], buckets[i], got)
		}
	}
}

// assertCoversPoints verifies no counted bucket falls outside the timeline.
func assertCoversPoints(t *testing.T, trend *FailureTrend) {
	t.Helper()
	known := map[string]bool{}
	for _, b := range trend.Buckets {
		known[b] = true
	}
	for _, p := range trend.Points {
		if !known[p.Bucket] {
			t.Errorf("Point bucket %s is missing from the timeline", p.Bucket)
		}
	}
}
