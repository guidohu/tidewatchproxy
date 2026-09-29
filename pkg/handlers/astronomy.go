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

// HandleAstronomy calculates sunrise, sunset and the current moon phase for a
// location and date, computed locally rather than fetched from an upstream
// provider.
//
// @Summary Get Astronomy Data
// @Description Calculate sunrise, sunset and moon phase for a location and date. Computed locally, no upstream API call.
// @Tags Astronomy
// @Produce json
// @Param lat query string true "Latitude"
// @Param lng query string true "Longitude"
// @Param date query string false "Date (Unix timestamp, default: now)"
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

	sunTimes := suncalc.GetTimes(date, latVal, lngVal)
	illumination := suncalc.GetMoonIllumination(date)

	c.JSON(http.StatusOK, models.AstronomyResponse{
		Sunrise:       sunTimes[suncalc.Sunrise].Value.Unix(),
		Sunset:        sunTimes[suncalc.Sunset].Value.Unix(),
		MoonPhase:     util.Round(illumination.Phase, 4),
		MoonPhaseName: moonPhaseName(illumination.Phase),
	})
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
