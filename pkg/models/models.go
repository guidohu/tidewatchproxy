package models

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
	Data []DenseTidePoint `json:"data"`
}

type DenseTidePoint struct {
	Timestamp int64   `json:"ts"`
	Height    float64 `json:"h"`
	Type      string  `json:"t,omitempty"`
}
