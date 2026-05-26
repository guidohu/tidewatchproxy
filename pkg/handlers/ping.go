package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HandlePing processes ping requests
// @Summary Ping proxy server
// @Description Record uuid and version of clients
// @Tags ping
// @Param uuid query string true "Client UUID"
// @Param version query string true "Client Version X.X.X"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /ping [get]
// @Router /ping [post]
func (h *Handler) HandlePing(c *gin.Context) {
	c.Set("backend", "Ping")

	uuid := c.Query("uuid")
	version := c.Query("version")

	if uuid == "" || version == "" {
		var body struct {
			UUID    string `json:"uuid"`
			Version string `json:"version"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			if uuid == "" {
				uuid = body.UUID
			}
			if version == "" {
				version = body.Version
			}
		}
	}

	uuid = strings.TrimSpace(uuid)
	version = strings.TrimSpace(version)

	if uuid == "" || version == "" {
		c.Set("error_type", "Missing Parameters")
		c.JSON(http.StatusBadRequest, gin.H{"error": "uuid and version are required"})
		return
	}

	if !isValidVersion(version) {
		c.Set("error_type", "Invalid Version Format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "version must be in X.X.X format"})
		return
	}

	h.locationStore.LogPing(uuid, version)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func isValidVersion(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}
