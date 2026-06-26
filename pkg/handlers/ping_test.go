package handlers

import (
	"testing"
)

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
