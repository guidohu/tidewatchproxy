package handlers

import (
	"log"
	"net/http"
	"strconv"

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

func (h *DashboardHandler) HandleDashboard(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Tide Watch Proxy Dashboard</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" crossorigin=""/>
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js" crossorigin=""></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: #f5f7fa;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }
        .card {
            background: white;
            padding: 20px;
            border-radius: 12px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.05);
            margin-bottom: 20px;
        }
        #map { height: 500px; border-radius: 12px; width: 100%; }
        .bottom-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        h1 { margin: 0; font-size: 1.5rem; color: #1e293b; }
        h2 { margin-top: 0; color: #64748b; font-size: 1rem; text-transform: uppercase; letter-spacing: 0.05em; }
        .chart-container { position: relative; height: 250px; }
        
        .stats-widget {
            display: flex;
            align-items: center;
            justify-content: space-around;
            height: 100%;
            min-height: 200px;
        }
        .stat-item {
            display: flex;
            flex-direction: column;
            align-items: center;
        }
        .stat-value {
            font-size: 3rem;
            font-weight: 800;
            margin-bottom: 5px;
        }
        .stat-label {
            font-size: 0.8rem;
            color: #64748b;
            text-transform: uppercase;
            font-weight: 600;
            letter-spacing: 0.05em;
        }
        .success { color: #10b981; }
        .failed { color: #ef4444; }
        .total { color: #1e293b; }

        .error-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
            font-size: 0.85rem;
        }
        .error-table th, .error-table td {
            text-align: left;
            padding: 12px;
            border-bottom: 1px solid #e2e8f0;
        }
        .error-table th {
            background-color: #f8fafc;
            color: #64748b;
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.75rem;
        }
        .error-table tr:hover { background-color: #f1f5f9; }
        .status-badge {
            padding: 4px 8px;
            border-radius: 4px;
            font-weight: 600;
            background-color: #fee2e2;
            color: #ef4444;
        }
        .code-block {
            max-width: 300px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            font-family: monospace;
            color: #475569;
            background: #f8fafc;
            padding: 2px 4px;
            border-radius: 4px;
            cursor: help;
        }

        select {
            padding: 8px 12px;
            border-radius: 6px;
            border: 1px solid #cbd5e1;
            background-color: white;
            font-size: 0.9rem;
            color: #334155;
            cursor: pointer;
            box-shadow: 0 1px 2px rgba(0,0,0,0.05);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Tide Watch Proxy Dashboard</h1>
            <select id="timeframe" onchange="updateDashboard()">
                <option value="0">All Time</option>
                <option value="1">Last 24 Hours</option>
                <option value="7">Last 7 Days</option>
                <option value="30">Last 30 Days</option>
            </select>
        </div>

        <div class="card">
            <h2>Request Map</h2>
            <div id="map"></div>
        </div>

        <div class="bottom-grid">
            <div class="card">
                <h2>Top Failure Reasons</h2>
                <div class="chart-container">
                    <canvas id="failureChart"></canvas>
                </div>
            </div>
            <div class="card">
                <h2>API Performance</h2>
                <div class="stats-widget">
                    <div class="stat-item">
                        <div class="stat-value success" id="successValue">0</div>
                        <div class="stat-label">Success</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-value failed" id="failedValue">0</div>
                        <div class="stat-label">Failures</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-value total" id="totalValue">0</div>
                        <div class="stat-label">Total</div>
                    </div>
                </div>
            </div>
        </div>

        <div class="card">
            <h2>Recent Error Logs</h2>
            <table class="error-table">
                <thead>
                    <tr>
                        <th>Time</th>
                        <th>Backend</th>
                        <th>Path</th>
                        <th>Status</th>
                        <th>Error Type</th>
                        <th>Request</th>
                        <th>Upstream Resp</th>
                        <th>Response sent</th>
                    </tr>
                </thead>
                <tbody id="errorLogBody">
                    <tr><td colspan="8" style="text-align:center">No error logs available</td></tr>
                </tbody>
            </table>
        </div>
    </div>

    <script>
        let failureChart;
        let markers = [];

        // Map Initialization
        var map = L.map('map').setView([20, 0], 2);
        L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
            maxZoom: 19,
            attribution: '&copy; OpenStreetMap'
        }).addTo(map);

        function updateDashboard() {
            const days = document.getElementById('timeframe').value;
            
            // Clear markers
            markers.forEach(m => map.removeLayer(m));
            markers = [];

            // Fetch Locations
            fetch('/dashboard/api/locations?days=' + days)
                .then(res => res.json())
                .then(data => {
                    if (!data) return;
                    data.forEach(loc => {
                        var radius = Math.max(5, Math.min(20, loc.count));
                        var m = L.circleMarker([loc.lat, loc.lng], {
                            color: '#3b82f6',
                            fillColor: '#3b82f6',
                            fillOpacity: 0.5,
                            radius: radius
                        }).addTo(map).bindPopup("<b>Lat:</b> " + loc.lat + "<br><b>Lng:</b> " + loc.lng + "<br><b>Requests:</b> " + loc.count);
                        markers.push(m);
                    });
                });

            // Fetch Backend Stats
            fetch('/dashboard/api/stats?days=' + days)
                .then(res => res.json())
                .then(data => {
                    if (!data) data = [];
                    const success = data.reduce((a, b) => a + b.success, 0);
                    const failed = data.reduce((a, b) => a + b.failed, 0);
                    const total = success + failed;

                    document.getElementById('totalValue').innerText = total.toLocaleString();
                    document.getElementById('successValue').innerText = success.toLocaleString();
                    document.getElementById('failedValue').innerText = failed.toLocaleString();
                });

            // Fetch Failure Reasons
            fetch('/dashboard/api/reasons?days=' + days)
                .then(res => res.json())
                .then(data => {
                    if (!data) data = [];
                    if (failureChart) failureChart.destroy();
                    failureChart = new Chart(document.getElementById('failureChart'), {
                        type: 'doughnut',
                        data: {
                            labels: data.map(d => d.reason),
                            datasets: [{
                                data: data.map(d => d.count),
                                backgroundColor: ['#ef4444', '#f59e0b', '#3b82f6', '#8b5cf6', '#ec4899', '#6b7280']
                            }]
                        },
                        options: {
                            responsive: true,
                            maintainAspectRatio: false,
                            plugins: { 
                                legend: { 
                                    position: 'right',
                                    labels: { boxWidth: 12, font: { size: 11 } }
                                } 
                            }
                        }
                    });
                });

            // Fetch Error Logs
            fetch('/dashboard/api/errors?days=' + days)
                .then(res => res.json())
                .then(data => {
                    const body = document.getElementById('errorLogBody');
                    if (!data || data.length === 0) {
                        body.innerHTML = '<tr><td colspan="8" style="text-align:center">No error logs available</td></tr>';
                        return;
                    }
                    body.innerHTML = data.map(log => {
                        return '<tr>' +
                            '<td>' + new Date(log.timestamp).toLocaleString() + '</td>' +
                            '<td>' + log.backend + '</td>' +
                            '<td><code>' + log.path + '</code></td>' +
                            '<td><span class="status-badge">' + log.status_code + '</span></td>' +
                            '<td>' + log.error_type + '</td>' +
                            '<td><div class="code-block" title="' + (log.request_body || '').replace(/"/g, '&quot;') + '">' + (log.request_body || 'N/A') + '</div></td>' +
                            '<td><div class="code-block" title="' + (log.upstream_response || '').replace(/"/g, '&quot;') + '">' + (log.upstream_response || 'N/A') + '</div></td>' +
                            '<td><div class="code-block" title="' + (log.response_body || '').replace(/"/g, '&quot;') + '">' + log.response_body + '</div></td>' +
                        '</tr>';
                    }).join('');
                });
        }

        // Initial load
        updateDashboard();
    </script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
