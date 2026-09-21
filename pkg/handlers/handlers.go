package handlers

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"tide_watch_proxy/pkg/store"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

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
	quotaMutex         sync.RWMutex
	quotaExceeded      map[string]quotaCacheEntry
}

// quotaCacheEntry holds a cached "quota exceeded" upstream response for an API key.
type quotaCacheEntry struct {
	status int
	body   []byte
	expiry time.Time
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
		quotaExceeded:      make(map[string]quotaCacheEntry),
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

// setTransportError records a failed upstream call: a coarse reason for the
// dashboard and the full error text for the error log, so a connection failure
// can be told apart from a DNS, TLS or timeout failure after the fact.
func setTransportError(c *gin.Context, backend string, err error) {
	c.Set("error_type", backend+" "+classifyTransportError(err))
	c.Set("upstream_response", redactSecrets(err.Error()))
}

// classifyTransportError turns a transport failure into a short, stable label
// for the dashboard's failure-reason charts.
func classifyTransportError(err error) string {
	switch {
	case isDNSError(err):
		return "DNS Error"
	case isTLSError(err):
		return "TLS Error"
	case isTimeout(err):
		return "Timeout"
	default:
		return "Connection Error"
	}
}

func isDNSError(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr)
}

func isTLSError(err error) bool {
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return true
	}
	var recordErr tls.RecordHeaderError
	return errors.As(err, &recordErr)
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// secretQueryParam matches credentials that upstream URLs carry in the query
// string, so they do not end up in the error log or on the dashboard.
var secretQueryParam = regexp.MustCompile(`(?i)([?&](?:key|apikey|api_key|token|password)=)[^&"\s]+`)

func redactSecrets(s string) string {
	return secretQueryParam.ReplaceAllString(s, "${1}REDACTED")
}
