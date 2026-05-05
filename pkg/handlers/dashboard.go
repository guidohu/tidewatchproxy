package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tide_watch_proxy/pkg/store"
)

type DashboardHandler struct {
	store *store.LocationStore
}

func NewDashboardHandler(s *store.LocationStore) *DashboardHandler {
	return &DashboardHandler{store: s}
}

func (h *DashboardHandler) HandleLocationsAPI(c *gin.Context) {
	locations, err := h.store.GetAllLocations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch locations"})
		return
	}
	c.JSON(http.StatusOK, locations)
}

func (h *DashboardHandler) HandleDashboard(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Requested GPS Coordinates</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" crossorigin=""/>
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js" crossorigin=""></script>
    <style>
        body { padding: 0; margin: 0; }
        html, body, #map { height: 100%; width: 100vw; }
    </style>
</head>
<body>
    <div id="map"></div>
    <script>
        var map = L.map('map').setView([0, 0], 2);

        L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
            maxZoom: 19,
            attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        }).addTo(map);

        fetch('/api/locations')
            .then(response => response.json())
            .then(data => {
                if (!data) return;
                data.forEach(loc => {
                    var radius = Math.max(10, Math.min(100, loc.count * 2));
                    L.circleMarker([loc.lat, loc.lng], {
                        color: 'red',
                        fillColor: '#f03',
                        fillOpacity: 0.5,
                        radius: radius
                    }).addTo(map).bindPopup("Requests: " + loc.count);
                });
            });
    </script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
