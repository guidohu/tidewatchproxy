package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tide_watch_proxy/pkg/models"
	"tide_watch_proxy/pkg/util"

	"github.com/gin-gonic/gin"
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
// @Router /data/reverse-geocode [get]
func (h *Handler) HandleReverseGeocode(c *gin.Context) {
	c.Set("backend", "BigDataCloud")
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")

	latVal, latErr := strconv.ParseFloat(latStr, 64)
	lngVal, lngErr := strconv.ParseFloat(lngStr, 64)

	if latStr == "" || lngStr == "" || latErr != nil || lngErr != nil {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude must be valid numbers"})
		return
	}

	if !util.IsValidLatitude(latVal) || !util.IsValidLongitude(lngVal) {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude must be between -90 and 90, longitude between -180 and 180"})
		return
	}

	lat := latVal
	lng := lngVal
	key := fmt.Sprintf("%.2f,%.2f", util.Round(lat, 2), util.Round(lng, 2))

	cacheKey := "geocode:" + key

	// Check CSV first - this always works, regardless of useCache flag
	if name, ok := h.customLocations[key]; ok {
		log.Printf("CSV Match: Found custom location '%s' for key %s", name, key)
		c.Set("is_cache_hit", true)
		resp := models.LocationResponse{Locality: name}
		// Write through to Redis so custom locations show up in the dashboard
		// cache table; the entry is never read for serving since the CSV is
		// checked first.
		if h.useCache {
			jsonData, _ := json.Marshal(resp)
			h.redisClient.Set(h.ctx, cacheKey, jsonData, 30*24*time.Hour)
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	// Check Redis cache if enabled
	if h.useCache {
		if val, err := h.redisClient.Get(h.ctx, cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			var cached models.LocationResponse
			if err := json.Unmarshal([]byte(val), &cached); err == nil {
				c.Set("is_cache_hit", true)
				c.JSON(http.StatusOK, cached)
				return
			}
		}
	}

	// Fetch from BigDataCloud
	url := fmt.Sprintf("%s/data/reverse-geocode?latitude=%s&longitude=%s&localityLanguage=en&key=%s",
		BigDataCloudBaseURL, latStr, lngStr, h.bigDataCloudAPIKey)

	resp, err := httpClient.Get(url)
	if err != nil {
		c.Set("error_type", "BigDataCloud Connection Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from BigDataCloud"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.Set("error_type", fmt.Sprintf("BigDataCloud HTTP %d", resp.StatusCode))
		c.Set("upstream_response", string(body))
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Locality    string `json:"locality"`
		City        string `json:"city"`
		CountryName string `json:"countryName"`
		CountryCode string `json:"countryCode"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.Set("error_type", "Parse Error")
		c.Set("upstream_response", string(body))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse BigDataCloud response"})
		return
	}

	filtered := models.LocationResponse{
		Locality:    raw.Locality,
		City:        raw.City,
		CountryName: raw.CountryName,
		CountryCode: raw.CountryCode,
	}

	// Cache result if enabled (30 days)
	if h.useCache {
		jsonData, _ := json.Marshal(filtered)
		h.redisClient.Set(h.ctx, cacheKey, jsonData, 30*24*time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, filtered)
}

// GeocodeCacheEntry represents a cached reverse-geocoding result for the dashboard.
type GeocodeCacheEntry struct {
	Locality    string  `json:"locality"`
	City        string  `json:"city"`
	CountryName string  `json:"countryName"`
	CountryCode string  `json:"countryCode"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

// HandleGeocodeCacheAPI returns all reverse-geocoding entries currently in the Redis cache.
func (h *Handler) HandleGeocodeCacheAPI(c *gin.Context) {
	entries := []GeocodeCacheEntry{}
	if !h.useCache {
		c.JSON(http.StatusOK, entries)
		return
	}

	var cursor uint64
	for {
		keys, next, err := h.redisClient.Scan(h.ctx, cursor, "geocode:*", 100).Result()
		if err != nil {
			log.Printf("Error scanning geocode cache: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan geocode cache"})
			return
		}

		if len(keys) > 0 {
			vals, err := h.redisClient.MGet(h.ctx, keys...).Result()
			if err != nil {
				log.Printf("Error fetching geocode cache values: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch geocode cache values"})
				return
			}
			for i, val := range vals {
				str, ok := val.(string)
				if !ok {
					continue
				}
				var loc models.LocationResponse
				if err := json.Unmarshal([]byte(str), &loc); err != nil {
					continue
				}
				coords := strings.Split(strings.TrimPrefix(keys[i], "geocode:"), ",")
				if len(coords) != 2 {
					continue
				}
				lat, latErr := strconv.ParseFloat(coords[0], 64)
				lng, lngErr := strconv.ParseFloat(coords[1], 64)
				if latErr != nil || lngErr != nil {
					continue
				}
				entries = append(entries, GeocodeCacheEntry{
					Locality:    loc.Locality,
					City:        loc.City,
					CountryName: loc.CountryName,
					CountryCode: loc.CountryCode,
					Lat:         lat,
					Lng:         lng,
				})
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}

	c.JSON(http.StatusOK, entries)
}
