package util

import (
	"fmt"
	"testing"
	"time"
)

func TestRound(t *testing.T) {
	tests := []struct {
		val       float64
		precision int
		want      float64
	}{
		{1.23456, 2, 1.23},
		{1.23556, 2, 1.24},
		{-1.23456, 2, -1.23},
		{10.5, 0, 11},
	}

	for _, tt := range tests {
		if got := Round(tt.val, tt.precision); got != tt.want {
			t.Errorf("Round(%v, %v) = %v, want %v", tt.val, tt.precision, got, tt.want)
		}
	}
}

func TestIsValidCoordinate(t *testing.T) {
	tests := []struct {
		val  float64
		want bool
	}{
		{0, true},
		{180, true},
		{-180, true},
		{181, false},
		{-181, false},
	}

	for _, tt := range tests {
		if got := IsValidCoordinate(tt.val); got != tt.want {
			t.Errorf("IsValidCoordinate(%v) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestParseAndClampTime(t *testing.T) {
	// Base time for stability
	now := time.Now().Truncate(time.Hour)
	startUnix := now.Unix()
	startStr := fmt.Sprintf("%d", startUnix)

	// Test 1: Simple start
	s, e := ParseAndClampTime(startStr, "")
	if s.Unix() != startUnix {
		t.Errorf("Expected start %v, got %v", startUnix, s.Unix())
	}
	expectedEnd := now.Add(24 * time.Hour)
	if e.Unix() != expectedEnd.Unix() {
		t.Errorf("Expected default end %v, got %v", expectedEnd.Unix(), e.Unix())
	}

	// Test 2: Clamp to 7 days
	endUnix := now.Add(10 * 24 * time.Hour).Unix()
	endStr := fmt.Sprintf("%d", endUnix)
	_, e = ParseAndClampTime(startStr, endStr)
	maxEnd := now.Add(7 * 24 * time.Hour)
	if e.Unix() != maxEnd.Unix() {
		t.Errorf("Expected clamped end %v, got %v", maxEnd.Unix(), e.Unix())
	}
}
