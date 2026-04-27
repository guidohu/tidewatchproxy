package handlers

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"
	"tide_watch_proxy/pkg/models"
)

type Config struct {
	RedisClient      *redis.Client
	StormglassAPIKey string
	UseCache         bool
	CustomLocations  map[string]string
	Debug            bool
}

// Handler holds all dependencies for API request processing
type Handler struct {
	redisClient      *redis.Client
	stormglassAPIKey string
	useCache         bool
	customLocations  map[string]string
	locationCache    map[string]models.LocationResponse
	locationCacheMu  sync.RWMutex
	debug            bool
	ctx              context.Context
}

func NewHandler(redisClient *redis.Client, stormglassAPIKey string, useCache bool, customLocations map[string]string, debug bool) *Handler {
	return &Handler{
		redisClient:      redisClient,
		stormglassAPIKey: stormglassAPIKey,
		useCache:         useCache,
		customLocations:  customLocations,
		locationCache:    make(map[string]models.LocationResponse),
		debug:            debug,
		ctx:              context.Background(),
	}
}

const (
	StormglassBaseURL   = "https://api.stormglass.io"
	BigDataCloudBaseURL = "https://api.bigdatacloud.net"
	OpenWatersBaseURL   = "https://api.openwaters.io/tides"
)

var allowedWeatherParams = map[string]bool{
	"swellHeight":             true,
	"swellPeriod":             true,
	"swellDirection":          true,
	"secondarySwellHeight":    true,
	"secondarySwellPeriod":    true,
	"secondarySwellDirection": true,
}
