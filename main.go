package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var allowedWeatherParams = map[string]bool{
	"swellHeight":             true,
	"swellPeriod":             true,
	"swellDirection":          true,
	"secondarySwellHeight":    true,
	"secondarySwellPeriod":    true,
	"secondarySwellDirection": true,
}

var (
	redisClient      *redis.Client
	stormglassAPIKey string
	ctx              = context.Background()
	customLocations  = make(map[string]string)
	locationCache    = make(map[string]LocationResponse)
	locationCacheMu  sync.RWMutex
	useCache         bool
	allowedAppIDs    []string
)

const (
	StormglassBaseURL   = "https://api.stormglass.io"
	BigDataCloudBaseURL = "https://api.bigdatacloud.net"
	RedisAddr           = "redis:6379"
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
}

type DenseTideData struct {
	Data []DenseTidePoint `json:"data"`
}

type DenseTidePoint struct {
	Timestamp int64   `json:"ts"`
	Height    float64 `json:"h"`
}

func main() {
	// Flags with environment variables as defaults
	var (
		apiKeyFlag          = flag.String("stormglass-api-key", os.Getenv("STORMGLASS_API_KEY"), "Stormglass API key")
		redisAddrFlag       = flag.String("redis-addr", getEnv("REDIS_ADDR", RedisAddr), "Redis address")
		portFlag            = flag.String("port", getEnv("PORT", "8080"), "Port to listen on")
		customLocationsFlag = flag.String("custom-locations", "custom_locations.csv", "Path to custom locations CSV file")
		allowedAppIDsFlag   = flag.String("allowed-app-ids", getEnv("ALLOWED_APP_IDS", ""), "Comma separated list of allowed App IDs or API keys")
	)
	flag.BoolVar(&useCache, "use-cache", true, "Enable Redis caching")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables (used as defaults for flags):\n")
		fmt.Fprintf(os.Stderr, "  STORMGLASS_API_KEY  Fallback Stormglass API key\n")
		fmt.Fprintf(os.Stderr, "  REDIS_ADDR          Redis address\n")
		fmt.Fprintf(os.Stderr, "  PORT                Port to listen on\n")
		fmt.Fprintf(os.Stderr, "  ALLOWED_APP_IDS     Comma separated list of allowed App IDs or API keys\n")
	}
	flag.Parse()

	// Assign flag values
	stormglassAPIKey = *apiKeyFlag
	redisAddr := *redisAddrFlag
	port := *portFlag
	if *allowedAppIDsFlag != "" {
		allowedAppIDs = strings.Split(*allowedAppIDsFlag, ",")
		for i := range allowedAppIDs {
			allowedAppIDs[i] = strings.TrimSpace(allowedAppIDs[i])
		}
	}

	loadCustomLocations(*customLocationsFlag)

	// Initialize Redis
	if useCache {
		redisClient = redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})
		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			log.Printf("Warning: Could not connect to Redis at %s: %v. Caching disabled.", redisAddr, err)
			useCache = false
		} else {
			log.Printf("Connected to Redis at %s", redisAddr)
		}
	}

	r := gin.Default()

	r.GET("/v2/weather/point", authMiddleware(), handleWeather)
	r.GET("/v2/tide/extremes/point", authMiddleware(), handleTides)
	r.GET("/v2/tide/sea-level/point", authMiddleware(), handleSeaLevel)
	r.GET("/data/reverse-geocode-client", authMiddleware(), handleReverseGeocode)

	r.Run(":" + port)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func loadCustomLocations(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("No custom locations file found at %s. Skipping.", path)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		log.Printf("Warning: Could not open custom locations file at %s: %v", path, err)
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error: Failed to read custom locations CSV at %s: %v", path, err)
		return
	}

	for _, record := range records {
		if len(record) < 3 {
			continue
		}
		lat, _ := strconv.ParseFloat(strings.TrimSpace(record[0]), 64)
		lon, _ := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		name := strings.TrimSpace(record[2])

		key := fmt.Sprintf("%.2f,%.2f", round(lat, 2), round(lon, 2))
		customLocations[key] = name
	}
	log.Printf("Successfully loaded %d custom locations from %s", len(customLocations), path)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(allowedAppIDs) > 0 {
			appID := c.Request.Header.Get("X-App-Id")
			if appID == "" {
				appID = c.Query("app_id")
			}

			allowed := false
			for _, id := range allowedAppIDs {
				if id == appID {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: invalid or missing App ID / API Key"})
				return
			}
		}

		key := c.Request.Header.Get("Authorization")
		if key == "" {
			key = c.Query("key")
		}

		if key == "" {
			key = stormglassAPIKey
		}

		if key == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "No API key provided"})
			return
		}

		c.Set("api_key", key)
		c.Next()
	}
}

func handleWeather(c *gin.Context) {
	lat := c.Query("lat")
	lng := c.Query("lng")
	params := c.Query("params")
	start := c.Query("start")
	end := c.Query("end")

	latVal, latErr := strconv.ParseFloat(lat, 64)
	lngVal, lngErr := strconv.ParseFloat(lng, 64)

	if lat == "" || lng == "" || latErr != nil || lngErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must be valid numbers"})
		return
	}

	if !isValidCoordinate(latVal) || !isValidCoordinate(lngVal) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coordinates must be between -180 and 180"})
		return
	}

	source := c.DefaultQuery("source", "noaa")
	if source != "noaa" {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid parameters requested"})
		return
	}
	params = strings.Join(filteredParams, ",")

	startTime, endTime := parseAndClampTime(start, end)

	cacheKey := fmt.Sprintf("weather:%.2f:%.2f:%s:%s:%d:%d",
		latVal, lngVal, params, source, startTime.Unix(), endTime.Unix())

	if useCache {
		if val, err := redisClient.Get(ctx, cacheKey).Result(); err == nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Stormglass"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
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
		} `json:"hours"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Stormglass response"})
		return
	}

	requested := make(map[string]bool)
	for _, p := range filteredParams {
		requested[p] = true
	}

	dense := DenseWeatherData{Data: make([]DenseWeatherPoint, 0)}
	for _, h := range raw.Hours {
		t, _ := time.Parse(time.RFC3339, h.Time)
		point := DenseWeatherPoint{Timestamp: t.Unix()}

		if requested["swellHeight"] {
			point.H1 = toPtr(h.SwellHeight.NOAA)
		}
		if requested["swellDirection"] {
			point.D1 = toPtr(h.SwellDirection.NOAA)
		}
		if requested["swellPeriod"] {
			point.P1 = toPtr(h.SwellPeriod.NOAA)
		}
		if requested["secondarySwellHeight"] {
			point.H2 = toPtr(h.SecondarySwellHeight.NOAA)
		}
		if requested["secondarySwellDirection"] {
			point.D2 = toPtr(h.SecondarySwellDirection.NOAA)
		}
		if requested["secondarySwellPeriod"] {
			point.P2 = toPtr(h.SecondarySwellPeriod.NOAA)
		}

		dense.Data = append(dense.Data, point)
	}

	jsonData, _ := json.Marshal(dense)
	if useCache {
		redisClient.Set(ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}

func handleTides(c *gin.Context) {
	lat := c.Query("lat")
	lng := c.Query("lng")
	start := c.Query("start")
	end := c.Query("end")

	latVal, latErr := strconv.ParseFloat(lat, 64)
	lngVal, lngErr := strconv.ParseFloat(lng, 64)

	if lat == "" || lng == "" || latErr != nil || lngErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must be valid numbers"})
		return
	}

	if !isValidCoordinate(latVal) || !isValidCoordinate(lngVal) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coordinates must be between -180 and 180"})
		return
	}

	startTime, endTime := parseAndClampTime(start, end)

	cacheKey := fmt.Sprintf("tides:%.2f:%.2f:%d:%d",
		latVal, lngVal, startTime.Unix(), endTime.Unix())

	if useCache {
		if val, err := redisClient.Get(ctx, cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", []byte(val))
			return
		}
	}

	apiKey := c.GetString("api_key")
	url := fmt.Sprintf("%s/v2/tide/extremes/point?lat=%s&lng=%s&start=%d&end=%d",
		StormglassBaseURL, lat, lng, startTime.Unix(), endTime.Unix())

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Stormglass"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Data []struct {
			Height float64 `json:"height"`
			Time   string  `json:"time"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Stormglass response"})
		return
	}

	dense := DenseTideData{Data: make([]DenseTidePoint, 0)}
	for _, d := range raw.Data {
		t, _ := time.Parse(time.RFC3339, d.Time)
		dense.Data = append(dense.Data, DenseTidePoint{
			Timestamp: t.Unix(),
			Height:    d.Height,
		})
	}

	jsonData, _ := json.Marshal(dense)
	if useCache {
		redisClient.Set(ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}

func handleSeaLevel(c *gin.Context) {
	lat := c.Query("lat")
	lng := c.Query("lng")
	start := c.Query("start")
	end := c.Query("end")
	datum := c.Query("datum")

	latVal, latErr := strconv.ParseFloat(lat, 64)
	lngVal, lngErr := strconv.ParseFloat(lng, 64)

	if lat == "" || lng == "" || latErr != nil || lngErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must be valid numbers"})
		return
	}

	if !isValidCoordinate(latVal) || !isValidCoordinate(lngVal) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coordinates must be between -180 and 180"})
		return
	}

	startTime, endTime := parseAndClampTime(start, end)

	cacheKey := fmt.Sprintf("sealevel:%.2f:%.2f:%d:%d:%s",
		latVal, lngVal, startTime.Unix(), endTime.Unix(), datum)

	if useCache {
		if val, err := redisClient.Get(ctx, cacheKey).Result(); err == nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Stormglass"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	var raw struct {
		Data []struct {
			Sg   float64 `json:"sg"`
			Time string  `json:"time"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Stormglass response"})
		return
	}

	dense := DenseTideData{Data: make([]DenseTidePoint, 0)}
	for _, d := range raw.Data {
		t, _ := time.Parse(time.RFC3339, d.Time)
		dense.Data = append(dense.Data, DenseTidePoint{
			Timestamp: t.Unix(),
			Height:    d.Sg,
		})
	}

	jsonData, _ := json.Marshal(dense)
	if useCache {
		redisClient.Set(ctx, cacheKey, jsonData, time.Hour)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, dense)
}

func handleReverseGeocode(c *gin.Context) {
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")

	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude are required"})
		return
	}

	lat := mustParseFloat(latStr)
	lng := mustParseFloat(lngStr)
	key := fmt.Sprintf("%.2f,%.2f", round(lat, 2), round(lng, 2))

	// Check CSV first - this always works, regardless of useCache flag
	if name, ok := customLocations[key]; ok {
		log.Printf("CSV Match: Found custom location '%s' for key %s", name, key)
		c.JSON(http.StatusOK, LocationResponse{Locality: name})
		return
	}

	// Check memory cache if enabled
	if useCache {
		locationCacheMu.RLock()
		if cached, ok := locationCache[key]; ok {
			locationCacheMu.RUnlock()
			c.JSON(http.StatusOK, cached)
			return
		}
		locationCacheMu.RUnlock()
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

	filtered := LocationResponse{
		Locality:    raw.Locality,
		City:        raw.City,
		CountryName: raw.CountryName,
		CountryCode: raw.CountryCode,
	}

	// Cache result if enabled
	if useCache {
		locationCacheMu.Lock()
		locationCache[key] = filtered
		locationCacheMu.Unlock()
	}

	c.JSON(http.StatusOK, filtered)
}

func parseAndClampTime(start, end string) (time.Time, time.Time) {
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

func mustParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func round(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func isValidCoordinate(val float64) bool {
	return val >= -180 && val <= 180
}

func toPtr(f float64) *float64 {
	return &f
}
