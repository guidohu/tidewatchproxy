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

func (h *Handler) HandleOpenWatersExtremes(c *gin.Context) {
	latitude := c.Query("latitude")
	longitude := c.Query("longitude")
	start := c.Query("start")
	end := c.Query("end")
	datum := c.Query("datum")
	units := c.DefaultQuery("units", "meters")

	if latitude == "" || longitude == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude are required"})
		return
	}

	if datum != "" && datum != "LAT" && datum != "MSL" && datum != "MLLW" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid datum. Allowed values: LAT, MSL, MLLW"})
		return
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from OpenWaters"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	util.LogStormglass(h.debug, "GET", url, body)

	if resp.StatusCode != http.StatusOK {
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Extremes []struct {
			Time  string  `json:"time"`
			Level float64 `json:"level"`
			High  bool    `json:"high"`
		} `json:"extremes"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OpenWaters response"})
		return
	}

	dense := models.DenseTideData{Data: make([]models.DenseTidePoint, 0)}
	for _, e := range raw.Extremes {
		t, _ := time.Parse(time.RFC3339, e.Time)
		tType := "low"
		if e.High {
			tType = "high"
		}
		dense.Data = append(dense.Data, models.DenseTidePoint{
			Timestamp: t.Unix(),
			Height:    e.Level,
			Type:      tType,
		})
	}

	c.JSON(http.StatusOK, dense)
}

func (h *Handler) HandleOpenWatersTimeline(c *gin.Context) {
	latitude := c.Query("latitude")
	longitude := c.Query("longitude")
	start := c.Query("start")
	end := c.Query("end")
	datum := c.Query("datum")
	units := c.DefaultQuery("units", "meters")

	if latitude == "" || longitude == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude are required"})
		return
	}

	if datum != "" && datum != "LAT" && datum != "MSL" && datum != "MLLW" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid datum. Allowed values: LAT, MSL, MLLW"})
		return
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from OpenWaters"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	util.LogStormglass(h.debug, "GET", url, body)

	if resp.StatusCode != http.StatusOK {
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Timeline []struct {
			Time  string  `json:"time"`
			Level float64 `json:"level"`
		} `json:"timeline"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OpenWaters response"})
		return
	}

	dense := models.DenseTideData{Data: make([]models.DenseTidePoint, 0)}
	var lastProcessedHour time.Time
	for _, e := range raw.Timeline {
		t, _ := time.Parse(time.RFC3339, e.Time)
		if t.Minute() == 0 && t.Second() == 0 {
			hour := t.Truncate(time.Hour)
			if !hour.Equal(lastProcessedHour) {
				dense.Data = append(dense.Data, models.DenseTidePoint{
					Timestamp: t.Unix(),
					Height:    e.Level,
				})
				lastProcessedHour = hour
			}
		}
	}

	c.JSON(http.StatusOK, dense)
}
