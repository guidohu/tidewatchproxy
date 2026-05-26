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
	users, err := s.GetUsersPerVersion()
	if err != nil {
		t.Fatalf("GetUsersPerVersion failed: %v", err)
	}

	// We expect 1 user on 1.0.1 (user-2) and 1 user on 1.1.0 (user-1, upgraded)
	// Let's count them
	var count101, count110 int
	for _, vc := range users.Last24h {
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
