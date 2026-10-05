package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sixdouglas/suncalc"

	"tide_watch_proxy/pkg/models"
	"tide_watch_proxy/pkg/util"
)

// astronomyForecastDays is the number of daily entries returned by
// HandleAstronomy.
const astronomyForecastDays = 7

// HandleAstronomy calculates sunrise, sunset and moon phase for a location,
// at daily resolution for the next astronomyForecastDays days starting from
// the given date. Computed locally rather than fetched from an upstream
// provider.
//
// @Summary Get Astronomy Forecast
// @Description Calculate sunrise, sunset and moon phase for a location, one entry per day for the next 7 days. Computed locally, no upstream API call.
// @Tags Astronomy
// @Produce json
// @Param lat query string true "Latitude"
// @Param lng query string true "Longitude"
// @Param date query string false "Start date (Unix timestamp, default: today)"
// @Success 200 {object} models.AstronomyResponse
// @Failure 400 {object} map[string]string "Bad Request"
// @Security AppIdAuth
// @Router /astronomy [get]
func (h *Handler) HandleAstronomy(c *gin.Context) {
	c.Set("backend", "Astronomy")
	lat := c.Query("lat")
	lng := c.Query("lng")

	latVal, latErr := strconv.ParseFloat(lat, 64)
	lngVal, lngErr := strconv.ParseFloat(lng, 64)

	if lat == "" || lng == "" || latErr != nil || lngErr != nil {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must be valid numbers"})
		return
	}

	if !util.IsValidLatitude(latVal) || !util.IsValidLongitude(lngVal) {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude must be between -90 and 90, longitude between -180 and 180"})
		return
	}

	date := time.Now()
	if dateStr := c.Query("date"); dateStr != "" {
		ts, err := strconv.ParseInt(dateStr, 10, 64)
		if err != nil {
			c.Set("error_type", "Invalid Date")
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be a valid Unix timestamp"})
			return
		}
		date = time.Unix(ts, 0)
	}

	start := startOfUTCDay(date)
	days := make([]models.AstronomyDay, 0, astronomyForecastDays)
	for i := 0; i < astronomyForecastDays; i++ {
		day := start.AddDate(0, 0, i)
		sunTimes := suncalc.GetTimes(day, latVal, lngVal)
		illumination := suncalc.GetMoonIllumination(day)

		days = append(days, models.AstronomyDay{
			Date:          day.Unix(),
			Sunrise:       sunTimes[suncalc.Sunrise].Value.Unix(),
			Sunset:        sunTimes[suncalc.Sunset].Value.Unix(),
			MoonPhase:     util.Round(illumination.Phase, 4),
			MoonPhaseName: moonPhaseName(illumination.Phase),
		})
	}

	c.JSON(http.StatusOK, models.AstronomyResponse{Data: days})
}

// startOfUTCDay truncates t to midnight UTC of its calendar day.
func startOfUTCDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// moonPhaseName maps a suncalc moon phase value (0-1, where 0 and 1 are new
// moon and 0.5 is full moon) to one of the eight named lunar phases.
func moonPhaseName(phase float64) string {
	switch {
	case phase < 0.03 || phase >= 0.97:
		return "new_moon"
	case phase < 0.22:
		return "waxing_crescent"
	case phase < 0.28:
		return "first_quarter"
	case phase < 0.47:
		return "waxing_gibbous"
	case phase < 0.53:
		return "full_moon"
	case phase < 0.72:
		return "waning_gibbous"
	case phase < 0.78:
		return "last_quarter"
	default:
		return "waning_crescent"
	}
}
