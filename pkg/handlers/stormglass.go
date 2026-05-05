package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"tide_watch_proxy/pkg/models"
	"tide_watch_proxy/pkg/util"
)

// @Summary Get Weather Data
// @Description Fetch weather/swell/wind data from Stormglass API
// @Tags Stormglass
// @Produce json
// @Param lat query string true "Latitude"
// @Param lng query string true "Longitude"
// @Param params query string true "Comma-separated parameters"
// @Param start query string false "Start time (Unix timestamp)"
// @Param end query string false "End time (Unix timestamp)"
// @Param source query string false "Source (default: noaa)"
// @Success 200 {object} models.DenseWeatherData
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security AppIdAuth
// @Security ApiKeyAuth
// @Router /v2/weather/point [get]
func (h *Handler) HandleWeather(c *gin.Context) {
	c.Set("backend", "Stormglass")
	lat := c.Query("lat")
	lng := c.Query("lng")
	params := c.Query("params")
	start := c.Query("start")
	end := c.Query("end")

	latVal, latErr := strconv.ParseFloat(lat, 64)
	lngVal, lngErr := strconv.ParseFloat(lng, 64)

	if lat == "" || lng == "" || latErr != nil || lngErr != nil {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must be valid numbers"})
		return
	}

	if !util.IsValidCoordinate(latVal) || !util.IsValidCoordinate(lngVal) {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "coordinates must be between -180 and 180"})
		return
	}

	source := c.DefaultQuery("source", "noaa")
	if source != "noaa" {
		c.Set("error_type", "Unsupported Source")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only 'noaa' source is supported so far"})
		return
	}

	// Filter params
	requestedParams := strings.Split(params, ",")
	var filteredParams []string
	for _, p := range requestedParams {
		if allowedWeatherParams[strings.TrimSpace(p)] {
			filteredParams = append(filteredParams, strings.TrimSpace(p))
		}
	}

	if len(filteredParams) == 0 {
		c.Set("error_type", "No Valid Params")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid parameters requested"})
		return
	}
	params = strings.Join(filteredParams, ",")

	startTime, endTime := util.ParseAndClampTime(start, end)

	cacheKey := fmt.Sprintf("weather:%.2f:%.2f:%s:%s:%d:%d",
		latVal, lngVal, params, source, startTime.Unix(), endTime.Unix())

	if h.useCache {
		if val, err := h.redisClient.Get(h.ctx, cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", []byte(val))
			return
		}
	}

	// Fetch from Stormglass
	apiKey := c.GetString("api_key")
	url := fmt.Sprintf("%s/v2/weather/point?lat=%.4f&lng=%.4f&params=%s&source=%s&start=%d&end=%d",
		StormglassBaseURL, latVal, lngVal, params, source, startTime.Unix(), endTime.Unix())

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Set("error_type", "Stormglass Connection Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Stormglass"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	util.LogStormglass(h.debug, "GET", url, body)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			c.Set("error_type", "Stormglass Auth Error")
		} else if resp.StatusCode == http.StatusTooManyRequests {
			c.Set("error_type", "Stormglass Rate Limit")
		} else {
			c.Set("error_type", fmt.Sprintf("Stormglass HTTP %d", resp.StatusCode))
		}
		c.Set("upstream_response", string(body))
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Hours []struct {
			Time        string `json:"time"`
			SwellHeight struct {
				NOAA float64 `json:"noaa"`
			} `json:"swellHeight"`
			SwellDirection struct {
				NOAA float64 `json:"noaa"`
			} `json:"swellDirection"`
			SwellPeriod struct {
				NOAA float64 `json:"noaa"`
			} `json:"swellPeriod"`
			SecondarySwellHeight struct {
				NOAA float64 `json:"noaa"`
			} `json:"secondarySwellHeight"`
			SecondarySwellDirection struct {
				NOAA float64 `json:"noaa"`
			} `json:"secondarySwellDirection"`
			SecondarySwellPeriod struct {
				NOAA float64 `json:"noaa"`
			} `json:"secondarySwellPeriod"`
			WindDirection struct {
				NOAA float64 `json:"noaa"`
			} `json:"windDirection"`
			WindSpeed struct {
				NOAA float64 `json:"noaa"`
			} `json:"windSpeed"`
		} `json:"hours"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.Set("error_type", "Parse Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Stormglass response"})
		return
	}

	requested := make(map[string]bool)
	for _, p := range filteredParams {
		requested[p] = true
	}

	dense := models.DenseWeatherData{Data: make([]models.DenseWeatherPoint, 0)}
	for _, h_raw := range raw.Hours {
		t, _ := time.Parse(time.RFC3339, h_raw.Time)
		point := models.DenseWeatherPoint{Timestamp: t.Unix()}

		if requested["swellHeight"] {
			point.H1 = util.ToPtr(h_raw.SwellHeight.NOAA)
		}
		if requested["swellDirection"] {
			point.D1 = util.ToPtr(h_raw.SwellDirection.NOAA)
		}
		if requested["swellPeriod"] {
			point.P1 = util.ToPtr(h_raw.SwellPeriod.NOAA)
		}
		if requested["secondarySwellHeight"] {
			point.H2 = util.ToPtr(h_raw.SecondarySwellHeight.NOAA)
		}
		if requested["secondarySwellDirection"] {
			point.D2 = util.ToPtr(h_raw.SecondarySwellDirection.NOAA)
		}
		if requested["secondarySwellPeriod"] {
			point.P2 = util.ToPtr(h_raw.SecondarySwellPeriod.NOAA)
		}
		if requested["windDirection"] {
			point.WD = util.ToPtr(h_raw.WindDirection.NOAA)
		}
		if requested["windSpeed"] {
			point.WS = util.ToPtr(h_raw.WindSpeed.NOAA)
		}

		dense.Data = append(dense.Data, point)
	}

	jsonData, _ := json.Marshal(dense)
	if h.useCache {
		h.redisClient.Set(h.ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}

// @Summary Get Tide Extremes (Stormglass)
// @Description Fetch tide extremes from Stormglass API
// @Tags Stormglass
// @Produce json
// @Param lat query string true "Latitude"
// @Param lng query string true "Longitude"
// @Param start query string false "Start time (Unix timestamp)"
// @Param end query string false "End time (Unix timestamp)"
// @Param datum query string false "Datum"
// @Success 200 {object} models.DenseTideData
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security AppIdAuth
// @Security ApiKeyAuth
// @Router /v2/tide/extremes/point [get]
func (h *Handler) HandleTides(c *gin.Context) {
	c.Set("backend", "Stormglass")
	lat := c.Query("lat")
	lng := c.Query("lng")
	start := c.Query("start")
	end := c.Query("end")
	datum := c.Query("datum")

	latVal, latErr := strconv.ParseFloat(lat, 64)
	lngVal, lngErr := strconv.ParseFloat(lng, 64)

	if lat == "" || lng == "" || latErr != nil || lngErr != nil {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must be valid numbers"})
		return
	}

	if !util.IsValidCoordinate(latVal) || !util.IsValidCoordinate(lngVal) {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "coordinates must be between -180 and 180"})
		return
	}

	startTime, endTime := util.ParseAndClampTime(start, end)

	cacheKey := fmt.Sprintf("tides:%.2f:%.2f:%d:%d:%s",
		latVal, lngVal, startTime.Unix(), endTime.Unix(), datum)

	if h.useCache {
		if val, err := h.redisClient.Get(h.ctx, cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", []byte(val))
			return
		}
	}

	apiKey := c.GetString("api_key")
	url := fmt.Sprintf("%s/v2/tide/extremes/point?lat=%s&lng=%s&start=%d&end=%d",
		StormglassBaseURL, lat, lng, startTime.Unix(), endTime.Unix())
	if datum != "" {
		url += "&datum=" + datum
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Set("error_type", "Stormglass Connection Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Stormglass"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	util.LogStormglass(h.debug, "GET", url, body)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			c.Set("error_type", "Stormglass Auth Error")
		} else if resp.StatusCode == http.StatusTooManyRequests {
			c.Set("error_type", "Stormglass Rate Limit")
		} else {
			c.Set("error_type", fmt.Sprintf("Stormglass HTTP %d", resp.StatusCode))
		}
		c.Set("upstream_response", string(body))
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Data []struct {
			Height float64 `json:"height"`
			Time   string  `json:"time"`
			Type   string  `json:"type"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.Set("error_type", "Parse Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Stormglass response"})
		return
	}

	dense := models.DenseTideData{Data: make([]models.DenseTidePoint, 0)}
	for _, d := range raw.Data {
		t, _ := time.Parse(time.RFC3339, d.Time)
		dense.Data = append(dense.Data, models.DenseTidePoint{
			Timestamp: t.Unix(),
			Height:    d.Height,
			Type:      d.Type,
		})
	}

	jsonData, _ := json.Marshal(dense)
	if h.useCache {
		h.redisClient.Set(h.ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}

// @Summary Get Sea Level (Stormglass)
// @Description Fetch sea level timeline from Stormglass API
// @Tags Stormglass
// @Produce json
// @Param lat query string true "Latitude"
// @Param lng query string true "Longitude"
// @Param start query string false "Start time (Unix timestamp)"
// @Param end query string false "End time (Unix timestamp)"
// @Param datum query string false "Datum"
// @Success 200 {object} models.DenseTideData
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security AppIdAuth
// @Security ApiKeyAuth
// @Router /v2/tide/sea-level/point [get]
func (h *Handler) HandleSeaLevel(c *gin.Context) {
	c.Set("backend", "Stormglass")
	lat := c.Query("lat")
	lng := c.Query("lng")
	start := c.Query("start")
	end := c.Query("end")
	datum := c.Query("datum")

	latVal, latErr := strconv.ParseFloat(lat, 64)
	lngVal, lngErr := strconv.ParseFloat(lng, 64)

	if lat == "" || lng == "" || latErr != nil || lngErr != nil {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must be valid numbers"})
		return
	}

	if !util.IsValidCoordinate(latVal) || !util.IsValidCoordinate(lngVal) {
		c.Set("error_type", "Invalid Coordinates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "coordinates must be between -180 and 180"})
		return
	}

	startTime, endTime := util.ParseAndClampTime(start, end)

	cacheKey := fmt.Sprintf("sealevel:%.2f:%.2f:%d:%d:%s",
		latVal, lngVal, startTime.Unix(), endTime.Unix(), datum)

	if h.useCache {
		if val, err := h.redisClient.Get(h.ctx, cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", []byte(val))
			return
		}
	}

	apiKey := c.GetString("api_key")
	url := fmt.Sprintf("%s/v2/tide/sea-level/point?lat=%.4f&lng=%.4f&start=%d&end=%d",
		StormglassBaseURL, latVal, lngVal, startTime.Unix(), endTime.Unix())
	if datum != "" {
		url += "&datum=" + datum
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Set("error_type", "Stormglass Connection Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Stormglass"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	util.LogStormglass(h.debug, "GET", url, body)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			c.Set("error_type", "Stormglass Auth Error")
		} else if resp.StatusCode == http.StatusTooManyRequests {
			c.Set("error_type", "Stormglass Rate Limit")
		} else {
			c.Set("error_type", fmt.Sprintf("Stormglass HTTP %d", resp.StatusCode))
		}
		c.Set("upstream_response", string(body))
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Data []struct {
			Sg   float64 `json:"sg"`
			Time string  `json:"time"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.Set("error_type", "Parse Error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Stormglass response"})
		return
	}

	dense := models.DenseTideData{Data: make([]models.DenseTidePoint, 0)}
	for _, d := range raw.Data {
		t, _ := time.Parse(time.RFC3339, d.Time)
		dense.Data = append(dense.Data, models.DenseTidePoint{
			Timestamp: t.Unix(),
			Height:    d.Sg,
		})
	}

	jsonData, _ := json.Marshal(dense)
	if h.useCache {
		h.redisClient.Set(h.ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}
