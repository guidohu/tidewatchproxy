package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"tide_watch_proxy/pkg/store"
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
	redisClient        *redis.Client
	stormglassAPIKey   string
	bigDataCloudAPIKey string
	useCache           bool
	customLocations    map[string]string
	debug              bool
	ctx                context.Context
	locationStore      *store.LocationStore
	invalidKeysMutex   sync.RWMutex
	invalidKeys        map[string]time.Time
}

func NewHandler(redisClient *redis.Client, stormglassAPIKey string, bigDataCloudAPIKey string, useCache bool, customLocations map[string]string, debug bool, locationStore *store.LocationStore) *Handler {
	return &Handler{
		redisClient:        redisClient,
		stormglassAPIKey:   stormglassAPIKey,
		bigDataCloudAPIKey: bigDataCloudAPIKey,
		useCache:           useCache,
		customLocations:    customLocations,
		debug:              debug,
		ctx:                context.Background(),
		locationStore:      locationStore,
		invalidKeys:        make(map[string]time.Time),
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
	"windDirection":           true,
	"windSpeed":               true,
}
