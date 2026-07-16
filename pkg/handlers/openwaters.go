package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"tide_watch_proxy/pkg/models"
	"tide_watch_proxy/pkg/util"
)

// @Summary Get Tide Extremes (OpenWaters)
// @Description Fetch tide extremes (high/low) from OpenWaters API
// @Tags OpenWaters
// @Produce json
// @Param latitude query string true "Latitude"
// @Param longitude query string true "Longitude"
// @Param start query string false "Start time (Unix timestamp)"
// @Param end query string false "End time (Unix timestamp)"
// @Param datum query string false "Datum (LAT, MSL, MLLW)"
// @Param units query string false "Units (default: meters)"
// @Success 200 {object} models.DenseTideData
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security AppIdAuth
// @Router /tides/extremes [get]
func (h *Handler) HandleOpenWatersExtremes(c *gin.Context) {
	c.Set("backend", "OpenWaters")
	latitude := c.Query("latitude")
	longitude := c.Query("longitude")
	start := c.Query("start")
	end := c.Query("end")
	datum := c.Query("datum")
	units := c.DefaultQuery("units", "meters")

	latVal, latErr := strconv.ParseFloat(latitude, 64)
	lngVal, lngErr := strconv.ParseFloat(longitude, 64)

	if latitude == "" || longitude == "" || latErr != nil || lngErr != nil {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude must be valid numbers"})
		return
	}

	if !util.IsValidLatitude(latVal) || !util.IsValidLongitude(lngVal) {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude must be between -90 and 90, longitude between -180 and 180"})
		return
	}

	if datum != "" && datum != "LAT" && datum != "MSL" && datum != "MLLW" {
		c.Set("error_type", "Invalid Datum")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid datum. Allowed values: LAT, MSL, MLLW"})
		return
	}

	cacheKey := fmt.Sprintf("ow_extremes:%s:%s:%s:%s:%s:%s",
		latitude, longitude, start, end, datum, units)

	if h.useCache {
		if val, err := h.redisClient.Get(h.ctx, cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			c.Set("is_cache_hit", true)
			c.Data(http.StatusOK, "application/json", []byte(val))
			return
		}
	}

	url := fmt.Sprintf("%s/extremes?latitude=%s&longitude=%s&units=%s",
		OpenWatersBaseURL, latitude, longitude, units)

	if start != "" {
		if s, err := strconv.ParseInt(start, 10, 64); err == nil {
			url += "&start=" + time.Unix(s, 0).Format(time.RFC3339)
		}
	}
	if end != "" {
		if e, err := strconv.ParseInt(end, 10, 64); err == nil {
			url += "&end=" + time.Unix(e, 0).Format(time.RFC3339)
		}
	}
	if datum != "" {
		url += "&datum=" + datum
	}

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		c.Set("error_type", "OpenWaters Connection Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from OpenWaters"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	util.LogStormglass(h.debug, "GET", url, body)

	if resp.StatusCode != http.StatusOK {
		c.Set("error_type", fmt.Sprintf("OpenWaters HTTP %d", resp.StatusCode))
		c.Set("upstream_response", string(body))
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Station  *models.StationInfo `json:"station"`
		Extremes []struct {
			Time  string  `json:"time"`
			Level float64 `json:"level"`
			High  bool    `json:"high"`
		} `json:"extremes"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.Set("error_type", "Parse Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OpenWaters response"})
		return
	}

	dense := models.DenseTideData{
		Data:    make([]models.DenseTidePoint, 0),
		Station: raw.Station,
	}
	for _, e := range raw.Extremes {
		t, _ := time.Parse(time.RFC3339, e.Time)
		tType := "low"
		if e.High {
			tType = "high"
		}
		dense.Data = append(dense.Data, models.DenseTidePoint{
			Timestamp: t.Unix(),
			Height:    models.TideHeight(e.Level),
			Type:      tType,
		})
	}

	jsonData, _ := json.Marshal(dense)
	if h.useCache {
		h.redisClient.Set(h.ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}

// @Summary Get Tide Timeline (OpenWaters)
// @Description Fetch tide timeline data from OpenWaters API
// @Tags OpenWaters
// @Produce json
// @Param latitude query string true "Latitude"
// @Param longitude query string true "Longitude"
// @Param start query string false "Start time (Unix timestamp)"
// @Param end query string false "End time (Unix timestamp)"
// @Param datum query string false "Datum (LAT, MSL, MLLW)"
// @Param units query string false "Units (default: meters)"
// @Success 200 {object} models.DenseTideData
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security AppIdAuth
// @Router /tides/timeline [get]
func (h *Handler) HandleOpenWatersTimeline(c *gin.Context) {
	c.Set("backend", "OpenWaters")
	latitude := c.Query("latitude")
	longitude := c.Query("longitude")
	start := c.Query("start")
	end := c.Query("end")
	datum := c.Query("datum")
	units := c.DefaultQuery("units", "meters")

	if latitude == "" || longitude == "" {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude are required"})
		return
	}

	if datum != "" && datum != "LAT" && datum != "MSL" && datum != "MLLW" {
		c.Set("error_type", "Invalid Datum")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid datum. Allowed values: LAT, MSL, MLLW"})
		return
	}

	cacheKey := fmt.Sprintf("ow_timeline:%s:%s:%s:%s:%s:%s",
		latitude, longitude, start, end, datum, units)

	if h.useCache {
		if val, err := h.redisClient.Get(h.ctx, cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			c.Set("is_cache_hit", true)
			c.Data(http.StatusOK, "application/json", []byte(val))
			return
		}
	}

	url := fmt.Sprintf("%s/timeline?latitude=%s&longitude=%s&units=%s",
		OpenWatersBaseURL, latitude, longitude, units)

	if start != "" {
		if s, err := strconv.ParseInt(start, 10, 64); err == nil {
			url += "&start=" + time.Unix(s, 0).Format(time.RFC3339)
		}
	}
	if end != "" {
		if e, err := strconv.ParseInt(end, 10, 64); err == nil {
			url += "&end=" + time.Unix(e, 0).Format(time.RFC3339)
		}
	}
	if datum != "" {
		url += "&datum=" + datum
	}

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		c.Set("error_type", "OpenWaters Connection Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from OpenWaters"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	util.LogStormglass(h.debug, "GET", url, body)

	if resp.StatusCode != http.StatusOK {
		c.Set("error_type", fmt.Sprintf("OpenWaters HTTP %d", resp.StatusCode))
		c.Set("upstream_response", string(body))
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Station  *models.StationInfo `json:"station"`
		Timeline []struct {
			Time  string  `json:"time"`
			Level float64 `json:"level"`
		} `json:"timeline"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.Set("error_type", "Parse Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OpenWaters response"})
		return
	}

	if raw.Station != nil {
		raw.Station.ID = ""
		raw.Station.Region = ""
		raw.Station.Type = ""
	}

	dense := models.DenseTideData{
		Data:    make([]models.DenseTidePoint, 0),
		Station: raw.Station,
	}
	var lastProcessedHour time.Time
	for _, e := range raw.Timeline {
		t, _ := time.Parse(time.RFC3339, e.Time)
		if t.Minute() == 0 && t.Second() == 0 {
			hour := t.Truncate(time.Hour)
			if !hour.Equal(lastProcessedHour) {
				dense.Data = append(dense.Data, models.DenseTidePoint{
					Timestamp: t.Unix(),
					Height:    models.TideHeight(e.Level),
				})
				lastProcessedHour = hour
			}
		}
	}

	jsonData, _ := json.Marshal(dense)
	if h.useCache {
		h.redisClient.Set(h.ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}
