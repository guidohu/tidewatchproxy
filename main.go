package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var (
	redisClient *redis.Client
	useCache    bool
	proxyAPIKey string
	ctx         = context.Background()
)

const (
	SurflineBaseURL = "https://services.surfline.com"
	RedisAddr       = "redis:6379"
)

// Data models for filtered response
type Spot struct {
	ID   string  `json:"_id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type MapviewResponse struct {
	Data struct {
		Spots []Spot `json:"spots"`
	} `json:"data"`
}

type TidePoint struct {
	Timestamp int64   `json:"timestamp"`
	Type      string  `json:"type"`
	Height    float64 `json:"height"`
}

type TideForecastResponse struct {
	Associated struct {
		Units struct {
			TideHeight string `json:"tideHeight"`
		} `json:"units"`
	} `json:"associated"`
	Data struct {
		Tides []TidePoint `json:"tides"`
	} `json:"data"`
}

type Swell struct {
	Height    float64 `json:"height"`
	Period    float64 `json:"period"`
	Direction float64 `json:"direction"`
	Impact    float64 `json:"impact"`
}

type WavePoint struct {
	Timestamp int64   `json:"timestamp"`
	Swells    []Swell `json:"swells"`
}

type WaveForecastResponse struct {
	Associated struct {
		Units struct {
			SwellHeight string `json:"swellHeight"`
		} `json:"units"`
		Location struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"location"`
	} `json:"associated"`
	Data struct {
		Wave []WavePoint `json:"wave"`
	} `json:"data"`
}

type rateLimiter struct {
	tokens int
	last   time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*rateLimiter)
)

func rateLimitMiddleware() gin.HandlerFunc {
	// Cleanup stale IP entries
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, limiter := range clients {
				if time.Since(limiter.last) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.Request.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = c.ClientIP()
		} else {
			parts := strings.Split(ip, ",")
			ip = strings.TrimSpace(parts[0])
		}

		mu.Lock()
		now := time.Now()
		limiter, exists := clients[ip]
		if !exists {
			limiter = &rateLimiter{tokens: 10, last: now}
			clients[ip] = limiter
		} else {
			// Refill tokens: 10 per minute = 1 token per 6 seconds
			elapsed := now.Sub(limiter.last)
			tokensToAdd := int(elapsed.Seconds() / 6.0)
			if tokensToAdd > 0 {
				limiter.tokens += tokensToAdd
				if limiter.tokens > 10 {
					limiter.tokens = 10
				}
				limiter.last = limiter.last.Add(time.Duration(tokensToAdd*6) * time.Second)
			}
		}

		allowed := false
		if limiter.tokens > 0 {
			limiter.tokens--
			allowed = true
		}
		mu.Unlock()

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded. Maximum 10 requests per minute per IP."})
			return
		}
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Request.Header.Get("x-api-key")
		if key == "" {
			authHeader := c.Request.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if proxyAPIKey == "" {
			c.Next()
			return
		}

		if key != proxyAPIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid API Key"})
			return
		}

		c.Next()
	}
}

func main() {
	flag.BoolVar(&useCache, "use-cache", true, "Enable Redis caching")
	flag.StringVar(&proxyAPIKey, "api-key", os.Getenv("API_KEY"), "API Key for client authentication")
	flag.Parse()

	if proxyAPIKey == "" {
		log.Println("WARNING: Proxy API key is not set. API key authentication is disabled and all requests will be allowed.")
	}

	// Initialize Redis
	if useCache {
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = RedisAddr
		}
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
	
	// Apply middlewares
	r.Use(rateLimitMiddleware())
	r.Use(authMiddleware())

	r.GET("/kbyg/mapview", handleMapview)
	r.GET("/kbyg/spots/forecasts/tides", handleTides)
	r.GET("/kbyg/spots/forecasts/wave", handleWave)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func handleMapview(c *gin.Context) {
	cacheKey := fmt.Sprintf("mapview:%s", c.Request.URL.RawQuery)
	ttl := 24 * time.Hour

	proxyRequest(c, "/kbyg/mapview", cacheKey, ttl, func(data []byte) (interface{}, error) {
		var raw struct {
			Data struct {
				Spots []struct {
					ID   string  `json:"_id"`
					Name string  `json:"name"`
					Lat  float64 `json:"lat"`
					Lon  float64 `json:"lon"`
					// Ignore other fields
				} `json:"spots"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}

		var filtered MapviewResponse
		for _, s := range raw.Data.Spots {
			filtered.Data.Spots = append(filtered.Data.Spots, Spot{
				ID:   s.ID,
				Name: s.Name,
				Lat:  s.Lat,
				Lon:  s.Lon,
			})
		}
		return filtered, nil
	})
}

func handleTides(c *gin.Context) {
	cacheKey := fmt.Sprintf("tides:%s", c.Request.URL.RawQuery)
	ttl := 24 * time.Hour

	proxyRequest(c, "/kbyg/spots/forecasts/tides", cacheKey, ttl, func(data []byte) (interface{}, error) {
		var raw struct {
			Associated struct {
				Units struct {
					TideHeight string `json:"tideHeight"`
				} `json:"units"`
			} `json:"associated"`
			Data struct {
				Tides []TidePoint `json:"tides"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}

		var filtered TideForecastResponse
		filtered.Associated.Units.TideHeight = raw.Associated.Units.TideHeight
		filtered.Data.Tides = raw.Data.Tides
		return filtered, nil
	})
}

func handleWave(c *gin.Context) {
	cacheKey := fmt.Sprintf("wave:%s", c.Request.URL.RawQuery)
	ttl := 1 * time.Hour

	proxyRequest(c, "/kbyg/spots/forecasts/wave", cacheKey, ttl, func(data []byte) (interface{}, error) {
		var raw struct {
			Associated struct {
				Units struct {
					SwellHeight string `json:"swellHeight"`
				} `json:"units"`
				Location struct {
					Lat float64 `json:"lat"`
					Lon float64 `json:"lon"`
				} `json:"location"`
			} `json:"associated"`
			Data struct {
				Wave []struct {
					Timestamp int64 `json:"timestamp"`
					Swells    []struct {
						Height    float64 `json:"height"`
						Period    float64 `json:"period"`
						Direction float64 `json:"direction"`
						Impact    float64 `json:"impact"`
					} `json:"swells"`
				} `json:"wave"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}

		var filtered WaveForecastResponse
		filtered.Associated.Units.SwellHeight = raw.Associated.Units.SwellHeight
		filtered.Associated.Location.Lat = raw.Associated.Location.Lat
		filtered.Associated.Location.Lon = raw.Associated.Location.Lon

		for _, w := range raw.Data.Wave {
			point := WavePoint{Timestamp: w.Timestamp}

			// Sort swells by impact descending
			sort.Slice(w.Swells, func(i, j int) bool {
				return w.Swells[i].Impact > w.Swells[j].Impact
			})

			for i, s := range w.Swells {
				if i >= 3 {
					break // Only top 3 swells needed
				}
				point.Swells = append(point.Swells, Swell{
					Height:    s.Height,
					Period:    s.Period,
					Direction: s.Direction,
					Impact:    s.Impact,
				})
			}
			filtered.Data.Wave = append(filtered.Data.Wave, point)
		}
		return filtered, nil
	})
}

func proxyRequest(c *gin.Context, path, cacheKey string, ttl time.Duration, filter func([]byte) (interface{}, error)) {
	if useCache {
		val, err := redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", []byte(val))
			return
		}
	}

	// Fetch from Surfline
	resp, err := http.Get(SurflineBaseURL + path + "?" + c.Request.URL.RawQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch from Surfline"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}

	filteredData, err := filter(body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to filter data"})
		return
	}

	jsonBytes, err := json.Marshal(filteredData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal filtered data"})
		return
	}

	if useCache {
		err := redisClient.Set(ctx, cacheKey, jsonBytes, ttl).Err()
		if err != nil {
			log.Printf("Warning: Could not set cache for key %s: %v", cacheKey, err)
		}
	}

	c.Header("X-Cache", "MISS")
	c.Data(http.StatusOK, "application/json", jsonBytes)
}
