package models

import (
	"encoding/json"
	"testing"
)

func TestTideHeight_MarshalJSON(t *testing.T) {
	tests := []struct {
		input    TideHeight
		expected string
	}{
		{input: TideHeight(1.234), expected: `1.23`},
		{input: TideHeight(1.236), expected: `1.24`},
		{input: TideHeight(1.2), expected: `1.20`},
		{input: TideHeight(0.0), expected: `0.00`},
		{input: TideHeight(-0.456), expected: `-0.46`},
	}

	for _, tt := range tests {
		data, err := json.Marshal(tt.input)
		if err != nil {
			t.Errorf("json.Marshal failed for %v: %v", tt.input, err)
			continue
		}
		if string(data) != tt.expected {
			t.Errorf("For %v, expected JSON %q, got %q", tt.input, tt.expected, string(data))
		}
	}
}

func TestTideHeight_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected TideHeight
	}{
		{input: `1.23`, expected: TideHeight(1.23)},
		{input: `1.236`, expected: TideHeight(1.236)},
		{input: `0`, expected: TideHeight(0.0)},
		{input: `-0.46`, expected: TideHeight(-0.46)},
	}

	for _, tt := range tests {
		var output TideHeight
		err := json.Unmarshal([]byte(tt.input), &output)
		if err != nil {
			t.Errorf("json.Unmarshal failed for %q: %v", tt.input, err)
			continue
		}
		if output != tt.expected {
			t.Errorf("For %q, expected TideHeight %v, got %v", tt.input, tt.expected, output)
		}
	}
}
