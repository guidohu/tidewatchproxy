package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"tide_watch_proxy/pkg/models"
	"tide_watch_proxy/pkg/util"
)

// @Summary Reverse Geocode
// @Description Get location name from latitude and longitude using BigDataCloud
// @Tags Geocoding
// @Produce json
// @Param latitude query string true "Latitude"
// @Param longitude query string true "Longitude"
// @Success 200 {object} models.LocationResponse
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security AppIdAuth
// @Router /data/reverse-geocode-client [get]
func (h *Handler) HandleReverseGeocode(c *gin.Context) {
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")

	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude are required"})
		return
	}

	lat := util.MustParseFloat(latStr)
	lng := util.MustParseFloat(lngStr)
	key := fmt.Sprintf("%.2f,%.2f", util.Round(lat, 2), util.Round(lng, 2))

	// Check CSV first - this always works, regardless of useCache flag
	if name, ok := h.customLocations[key]; ok {
		log.Printf("CSV Match: Found custom location '%s' for key %s", name, key)
		c.JSON(http.StatusOK, models.LocationResponse{Locality: name})
		return
	}

	// Check memory cache if enabled
	if h.useCache {
		h.locationCacheMu.RLock()
		if cached, ok := h.locationCache[key]; ok {
			h.locationCacheMu.RUnlock()
			c.JSON(http.StatusOK, cached)
			return
		}
		h.locationCacheMu.RUnlock()
	}

	// Fetch from BigDataCloud
	url := fmt.Sprintf("%s/data/reverse-geocode-client?latitude=%s&longitude=%s&localityLanguage=en",
		BigDataCloudBaseURL, latStr, lngStr)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from BigDataCloud"})
		return
	}
	defer resp.Body.Close()

	var raw struct {
		Locality    string `json:"locality"`
		City        string `json:"city"`
		CountryName string `json:"countryName"`
		CountryCode string `json:"countryCode"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse BigDataCloud response"})
		return
	}

	filtered := models.LocationResponse{
		Locality:    raw.Locality,
		City:        raw.City,
		CountryName: raw.CountryName,
		CountryCode: raw.CountryCode,
	}

	// Cache result if enabled
	if h.useCache {
		h.locationCacheMu.Lock()
		h.locationCache[key] = filtered
		h.locationCacheMu.Unlock()
	}

	c.JSON(http.StatusOK, filtered)
}
