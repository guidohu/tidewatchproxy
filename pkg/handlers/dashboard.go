package handlers

import (
	_ "embed"
	"log"
	"net/http"
	"strconv"

	"tide_watch_proxy/pkg/store"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	store *store.LocationStore
}

func NewDashboardHandler(s *store.LocationStore) *DashboardHandler {
	return &DashboardHandler{store: s}
}

func (h *DashboardHandler) HandleLocationsAPI(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	locations, err := h.store.GetAllLocations(days)
	if err != nil {
		log.Printf("Error fetching locations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch locations"})
		return
	}
	log.Printf("Fetched %d locations for last %d days", len(locations), days)
	c.JSON(http.StatusOK, locations)
}

func (h *DashboardHandler) HandleStatsAPI(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	stats, err := h.store.GetBackendStats(days)
	if err != nil {
		log.Printf("Error fetching stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}
	log.Printf("Fetched stats for %d backends for last %d days", len(stats), days)
	c.JSON(http.StatusOK, stats)
}

func (h *DashboardHandler) HandleFailureReasonsAPI(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	reasons, err := h.store.GetFailureReasons(days)
	if err != nil {
		log.Printf("Error fetching failure reasons: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch failure reasons"})
		return
	}
	log.Printf("Fetched %d failure reasons for last %d days", len(reasons), days)
	c.JSON(http.StatusOK, reasons)
}

func (h *DashboardHandler) HandleErrorLogsAPI(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	logs, err := h.store.GetErrorLogs(days)
	if err != nil {
		log.Printf("Error fetching error logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch error logs"})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *DashboardHandler) HandleUsageAPI(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	stats, err := h.store.GetUsageStats(days)
	if err != nil {
		log.Printf("Error fetching usage stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch usage stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *DashboardHandler) HandleUsersPerVersionAPI(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	stats, err := h.store.GetUsersPerVersion(days)
	if err != nil {
		log.Printf("Error fetching users per version stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users per version stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *DashboardHandler) HandlePingUsageAPI(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	stats, err := h.store.GetPingUsageStats(days)
	if err != nil {
		log.Printf("Error fetching ping usage stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ping usage stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

//go:embed dashboard.html
var dashboardHTML []byte

func (h *DashboardHandler) HandleDashboard(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", dashboardHTML)
}
