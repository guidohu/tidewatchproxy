package models

import (
	"fmt"
	"strconv"
)

type LocationResponse struct {
	Locality    string `json:"locality,omitempty"`
	City        string `json:"city,omitempty"`
	CountryName string `json:"countryName,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

type DenseWeatherData struct {
	Data []DenseWeatherPoint `json:"data"`
}

type DenseWeatherPoint struct {
	Timestamp int64    `json:"ts"`
	H1        *float64 `json:"h1,omitempty"`
	D1        *float64 `json:"d1,omitempty"`
	P1        *float64 `json:"p1,omitempty"`
	H2        *float64 `json:"h2,omitempty"`
	D2        *float64 `json:"d2,omitempty"`
	P2        *float64 `json:"p2,omitempty"`
	WD        *float64 `json:"wd,omitempty"`
	WS        *float64 `json:"ws,omitempty"`
}

type DenseTideData struct {
	Data    []DenseTidePoint `json:"data"`
	Station *StationInfo     `json:"station,omitempty"`
}

type DenseTidePoint struct {
	Timestamp int64      `json:"ts"`
	Height    TideHeight `json:"h"`
	Type      string     `json:"t,omitempty"`
}

// TideHeight represents a float64 value that marshals to JSON with exactly 2 decimal places.
type TideHeight float64

// MarshalJSON formats TideHeight with exactly 2 decimal places.
func (h TideHeight) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.2f", h)), nil
}

// UnmarshalJSON parses a JSON float into TideHeight.
func (h *TideHeight) UnmarshalJSON(data []byte) error {
	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	*h = TideHeight(val)
	return nil
}

// AstronomyResponse holds a daily sunrise/sunset/moon-phase forecast for a
// given location, computed locally rather than fetched from an upstream
// provider.
type AstronomyResponse struct {
	Data []AstronomyDay `json:"data"`
}

// AstronomyDay holds the sunrise/sunset times and moon phase for a single
// calendar day (UTC).
type AstronomyDay struct {
	Date          int64   `json:"ts"`
	Sunrise       int64   `json:"sr"`
	Sunset        int64   `json:"ss"`
	MoonPhase     float64 `json:"mp"`
	MoonPhaseName string  `json:"mpn"`
}

type StationInfo struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Region  string `json:"region,omitempty"`
	Country string `json:"country,omitempty"`
	Type    string `json:"type,omitempty"`
}
