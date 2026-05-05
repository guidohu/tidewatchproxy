package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "tide_watch_proxy/docs" // Import generated docs
	"tide_watch_proxy/pkg/handlers"
	"tide_watch_proxy/pkg/middleware"
	"tide_watch_proxy/pkg/store"
	"tide_watch_proxy/pkg/util"
)

// @title Tide Watch Proxy API
// @version 1.0
// @description Proxy server for the Tide Watch Garmin application, providing weather, tides, and geocoding. If server runs with API key restriction, clients need to provide an API key.
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey AppIdAuth
// @in header
// @name X-App-Id
// @description Allowed App ID for accessing the API.

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Stormglass API Key for weather and some tide endpoints.

var (
	redisClient      *redis.Client
	stormglassAPIKey string
	ctx              = context.Background()
	customLocations  = make(map[string]string)
	useCache         bool
	debug            bool
	allowedAppIDs    []string
)

func main() {
	// Flags with environment variables as defaults
	var (
		apiKeyFlag          = flag.String("stormglass-api-key", os.Getenv("STORMGLASS_API_KEY"), "Stormglass API key")
		redisAddrFlag       = flag.String("redis-addr", getEnv("REDIS_ADDR", "redis:6379"), "Redis address")
		portFlag            = flag.String("port", getEnv("PORT", "8080"), "Port to listen on")
		customLocationsFlag = flag.String("custom-locations", "custom_locations.csv", "Path to custom locations CSV file")
		allowedAppIDsFlag   = flag.String("allowed-app-ids", getEnv("ALLOWED_APP_IDS", ""), "Comma separated list of allowed App IDs or API keys")
		dbPathFlag          = flag.String("db-path", getEnv("DB_PATH", "metrics.db"), "Path to SQLite database for statistics")
	)
	flag.BoolVar(&useCache, "use-cache", true, "Enable Redis caching")
	flag.BoolVar(&debug, "debug", getEnv("DEBUG", "false") == "true", "Print raw Stormglass API responses to console")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables (used as defaults for flags):\n")
		fmt.Fprintf(os.Stderr, "  STORMGLASS_API_KEY  Fallback Stormglass API key\n")
		fmt.Fprintf(os.Stderr, "  REDIS_ADDR          Redis address\n")
		fmt.Fprintf(os.Stderr, "  PORT                Port to listen on\n")
		fmt.Fprintf(os.Stderr, "  ALLOWED_APP_IDS     Comma separated list of allowed App IDs or API keys\n")
		fmt.Fprintf(os.Stderr, "  DB_PATH             Path to SQLite database for statistics\n")
		fmt.Fprintf(os.Stderr, "  DEBUG               Set to 'true' to enable debug logging\n")
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

	log.Printf("Starting Tide Watch Proxy...")
	log.Printf("Port: %s", port)
	log.Printf("Redis Address: %s", redisAddr)
	log.Printf("Stormglass API Key: %s", stormglassAPIKey)
	log.Printf("Allowed App IDs: %v", allowedAppIDs)
	log.Printf("Custom Locations File: %s", *customLocationsFlag)
	log.Printf("SQLite DB Path: %s", *dbPathFlag)
	log.Printf("Debug Mode: %v", debug)

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

	// Initialize SQLite Location Store
	locationStore, err := store.NewLocationStore(*dbPathFlag)
	if err != nil {
		log.Fatalf("Failed to initialize location store: %v", err)
	}

	// Initialize Handler
	h := handlers.NewHandler(redisClient, stormglassAPIKey, useCache, customLocations, debug)
	dashboardHandler := handlers.NewDashboardHandler(locationStore)

	r := gin.Default()

	// Location logging middleware
	loggerMiddleware := middleware.LocationLogger(locationStore)

	r.GET("/v2/weather/point", middleware.AppIDMiddleware(allowedAppIDs), middleware.AuthMiddleware(stormglassAPIKey), loggerMiddleware, h.HandleWeather)
	r.GET("/v2/tide/extremes/point", middleware.AppIDMiddleware(allowedAppIDs), middleware.AuthMiddleware(stormglassAPIKey), loggerMiddleware, h.HandleTides)
	r.GET("/v2/tide/sea-level/point", middleware.AppIDMiddleware(allowedAppIDs), middleware.AuthMiddleware(stormglassAPIKey), loggerMiddleware, h.HandleSeaLevel)
	r.GET("/tides/extremes", middleware.AppIDMiddleware(allowedAppIDs), loggerMiddleware, h.HandleOpenWatersExtremes)
	r.GET("/tides/timeline", middleware.AppIDMiddleware(allowedAppIDs), loggerMiddleware, h.HandleOpenWatersTimeline)
	r.GET("/data/reverse-geocode-client", middleware.AppIDMiddleware(allowedAppIDs), loggerMiddleware, h.HandleReverseGeocode)

	// Dashboard routes
	r.GET("/dashboard", dashboardHandler.HandleDashboard)
	r.GET("/api/locations", dashboardHandler.HandleLocationsAPI)

	// Swagger documentation route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

		key := fmt.Sprintf("%.2f,%.2f", util.Round(lat, 2), util.Round(lon, 2))
		customLocations[key] = name
	}
	log.Printf("Successfully loaded %d custom locations from %s", len(customLocations), path)
}
