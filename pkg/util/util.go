package util

import (
	"encoding/json"
	"log"
	"math"
	"strconv"
	"time"
)

func GetEnv(key, fallback string) string {
	// Note: We might want to pass this in from main, but for now we'll keep it here
	// and maybe move it to main later. Actually, helpers shouldn't depend on os env if possible.
	return fallback
}

func Round(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func MustParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func IsValidCoordinate(val float64) bool {
	return val >= -180 && val <= 180
}

func ToPtr(f float64) *float64 {
	return &f
}

func ParseAndClampTime(start, end string) (time.Time, time.Time) {
	var startTime, endTime time.Time

	if start != "" {
		if s, err := strconv.ParseInt(start, 10, 64); err == nil {
			startTime = time.Unix(s, 0)
		} else {
			startTime, _ = time.Parse(time.RFC3339, start)
		}
	} else {
		startTime = time.Now()
	}

	if end != "" {
		if e, err := strconv.ParseInt(end, 10, 64); err == nil {
			endTime = time.Unix(e, 0)
		} else {
			endTime, _ = time.Parse(time.RFC3339, end)
		}
	} else {
		endTime = startTime.Add(24 * time.Hour)
	}

	// Rounding
	startTime = startTime.Truncate(time.Hour)
	endTime = endTime.Add(time.Hour - 1).Truncate(time.Hour)

	// Clamping to 7 days
	maxEnd := startTime.Add(7 * 24 * time.Hour)
	if endTime.After(maxEnd) {
		endTime = maxEnd
	}

	return startTime, endTime
}

func LogStormglass(debug bool, method, url string, rawResponse []byte) {
	if !debug {
		return
	}

	var prettyBody string
	var obj interface{}
	if err := json.Unmarshal(rawResponse, &obj); err == nil {
		if pretty, err := json.MarshalIndent(obj, "", "  "); err == nil {
			prettyBody = string(pretty)
		} else {
			prettyBody = string(rawResponse)
		}
	} else {
		prettyBody = string(rawResponse)
	}

	log.Printf("DEBUG Stormglass Request: %s %s", method, url)
	log.Printf("DEBUG Stormglass Response:\n%s", prettyBody)
}
