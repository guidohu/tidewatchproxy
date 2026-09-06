package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"tide_watch_proxy/pkg/util"

	_ "modernc.org/sqlite"
)

type LocationStore struct {
	db           *sql.DB
	requestChan  chan requestEntry
	locationChan chan locationEntry
	errorChan    chan ErrorLog
	pingChan     chan pingEntry
	stopChan     chan struct{}

	// Prepared statements
	stmtRequest     *sql.Stmt
	stmtLocation    *sql.Stmt
	stmtError       *sql.Stmt
	stmtPingRequest *sql.Stmt
	stmtUserVersion *sql.Stmt
}

type pingEntry struct {
	UUID    string
	Version string
}

type requestEntry struct {
	backend    string
	statusCode int
	errorType  string
	lat        float64
	lng        float64
	isCacheHit bool
}

type locationEntry struct {
	lat float64
	lng float64
}

type LocationStats struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Count int     `json:"count"`
}

type BackendStats struct {
	Backend      string `json:"backend"`
	Success      int    `json:"success"`
	Failed       int    `json:"failed"`
	CacheSuccess int    `json:"cache_success"`
	CacheFailed  int    `json:"cache_failed"`
	Locations    int    `json:"locations"`
}

type FailureReason struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type UsageStats struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

func NewLocationStore(dbPath string) (*LocationStore, error) {
	// Add busy timeout to handle concurrent writes
	dsn := fmt.Sprintf("%s?_busy_timeout=5000", dbPath)
	log.Printf("Opening SQLite database at %s...", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// For SQLite in WAL mode, we can allow multiple concurrent readers.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Explicitly enable WAL mode and other performance settings
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	_, err = db.Exec("PRAGMA synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}
	_, _ = db.Exec("PRAGMA cache_size=-64000") // 64MB cache
	_, _ = db.Exec("PRAGMA temp_store=MEMORY")
	_, _ = db.Exec("PRAGMA mmap_size=268435456") // 256MB mmap

	log.Printf("Connected to database (WAL mode enabled). Initializing tables...")

	// Create tables if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS locations (
			lat REAL,
			lng REAL,
			count INTEGER,
			PRIMARY KEY (lat, lng)
		);
		CREATE TABLE IF NOT EXISTS requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			backend TEXT,
			status_code INTEGER,
			error_type TEXT,
			lat REAL,
			lng REAL,
			is_cache_hit INTEGER DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS error_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			method TEXT,
			path TEXT,
			query TEXT,
			status_code INTEGER,
			request_body TEXT,
			response_body TEXT,
			upstream_response TEXT,
			backend TEXT,
			error_type TEXT
		);
		CREATE TABLE IF NOT EXISTS user_versions (
			uuid TEXT PRIMARY KEY,
			version TEXT,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS ping_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			uuid TEXT,
			version TEXT
		);

		-- Performance Indexes
		CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
		CREATE INDEX IF NOT EXISTS idx_requests_backend ON requests(backend);
		CREATE INDEX IF NOT EXISTS idx_requests_lat_lng ON requests(lat, lng);
		CREATE INDEX IF NOT EXISTS idx_error_logs_timestamp ON error_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_ping_requests_timestamp ON ping_requests(timestamp);
		CREATE INDEX IF NOT EXISTS idx_user_versions_last_seen ON user_versions(last_seen);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create tables and indexes: %w", err)
	}
	log.Printf("Tables initialized. Running migrations...")

	// Migrations: Add columns if they don't exist
	_, _ = db.Exec("ALTER TABLE requests ADD COLUMN timestamp DATETIME DEFAULT CURRENT_TIMESTAMP")
	_, _ = db.Exec("ALTER TABLE requests ADD COLUMN lat REAL")
	_, _ = db.Exec("ALTER TABLE requests ADD COLUMN lng REAL")
	_, _ = db.Exec("ALTER TABLE requests ADD COLUMN is_cache_hit INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE error_logs ADD COLUMN upstream_response TEXT")

	// Ensure existing rows have a timestamp if it was NULL
	_, _ = db.Exec("UPDATE requests SET timestamp = CURRENT_TIMESTAMP WHERE timestamp IS NULL")

	log.Printf("Database initialization complete.")

	// Prepare statements
	stmtRequest, err := db.Prepare(`
		INSERT INTO requests (backend, status_code, error_type, lat, lng, is_cache_hit)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request statement: %w", err)
	}

	stmtLocation, err := db.Prepare(`
		INSERT INTO locations (lat, lng, count)
		VALUES (?, ?, 1)
		ON CONFLICT(lat, lng) DO UPDATE SET count = count + 1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare location statement: %w", err)
	}

	stmtError, err := db.Prepare(`
		INSERT INTO error_logs (method, path, query, status_code, request_body, response_body, upstream_response, backend, error_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare error statement: %w", err)
	}

	stmtPingRequest, err := db.Prepare(`
		INSERT INTO ping_requests (uuid, version)
		VALUES (?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare ping request statement: %w", err)
	}

	stmtUserVersion, err := db.Prepare(`
		INSERT INTO user_versions (uuid, version, last_seen)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(uuid) DO UPDATE SET version = ?, last_seen = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare user version statement: %w", err)
	}

	store := &LocationStore{
		db:              db,
		requestChan:     make(chan requestEntry, 1000),
		locationChan:    make(chan locationEntry, 1000),
		errorChan:       make(chan ErrorLog, 100),
		pingChan:        make(chan pingEntry, 1000),
		stopChan:        make(chan struct{}),
		stmtRequest:     stmtRequest,
		stmtLocation:    stmtLocation,
		stmtError:       stmtError,
		stmtPingRequest: stmtPingRequest,
		stmtUserVersion: stmtUserVersion,
	}

	// Start background worker
	go store.runWorker()

	// Start background cleanup
	go store.startCleanupTask()

	return store, nil
}

func (s *LocationStore) DB() *sql.DB {
	return s.db
}

func (s *LocationStore) Close() error {
	close(s.stopChan)
	s.stmtRequest.Close()
	s.stmtLocation.Close()
	s.stmtError.Close()
	s.stmtPingRequest.Close()
	s.stmtUserVersion.Close()
	return s.db.Close()
}

func (s *LocationStore) runWorker() {
	var (
		requests  []requestEntry
		locations []locationEntry
		errors    []ErrorLog
		pings     []pingEntry
	)

	flush := func() {
		if len(requests) == 0 && len(locations) == 0 && len(errors) == 0 && len(pings) == 0 {
			return
		}

		start := time.Now()
		tx, err := s.db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			return
		}

		if len(requests) > 0 {
			txStmt := tx.Stmt(s.stmtRequest)
			for _, req := range requests {
				var latVal, lngVal interface{}
				if req.lat != 0 || req.lng != 0 {
					latVal = util.Round(req.lat, 2)
					lngVal = util.Round(req.lng, 2)
				}
				txStmt.Exec(req.backend, req.statusCode, req.errorType, latVal, lngVal, req.isCacheHit)
			}
		}

		if len(locations) > 0 {
			txStmt := tx.Stmt(s.stmtLocation)
			for _, loc := range locations {
				qLat := util.Round(loc.lat, 2)
				qLng := util.Round(loc.lng, 2)
				txStmt.Exec(qLat, qLng)
			}
		}

		if len(errors) > 0 {
			txStmt := tx.Stmt(s.stmtError)
			for _, entry := range errors {
				txStmt.Exec(
					entry.Method, entry.Path, entry.Query, entry.StatusCode,
					entry.RequestBody, entry.ResponseBody, entry.UpstreamResponse,
					entry.Backend, entry.ErrorType,
				)
			}
		}

		if len(pings) > 0 {
			txPingReq := tx.Stmt(s.stmtPingRequest)
			txUserVer := tx.Stmt(s.stmtUserVersion)
			for _, p := range pings {
				txPingReq.Exec(p.UUID, p.Version)
				txUserVer.Exec(p.UUID, p.Version, p.Version)
			}
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
		}

		if len(requests)+len(locations)+len(errors)+len(pings) > 10 {
			log.Printf("Flushed %d items to DB in %v", len(requests)+len(locations)+len(errors)+len(pings), time.Since(start))
		}

		requests = nil
		locations = nil
		errors = nil
		pings = nil
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case req := <-s.requestChan:
			requests = append(requests, req)
			if len(requests) >= 200 {
				flush()
			}
		case loc := <-s.locationChan:
			locations = append(locations, loc)
			if len(locations) >= 200 {
				flush()
			}
		case err := <-s.errorChan:
			errors = append(errors, err)
			if len(errors) >= 50 {
				flush()
			}
		case p := <-s.pingChan:
			pings = append(pings, p)
			if len(pings) >= 200 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.stopChan:
			flush()
			return
		}
	}
}

func (s *LocationStore) startCleanupTask() {
	// Run cleanup immediately on start
	s.CleanupOldLogs()

	// Then every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	for range ticker.C {
		s.CleanupOldLogs()
	}
}

func (s *LocationStore) CleanupOldLogs() {
	// Clean up requests and error logs older than 30 days
	_, err := s.db.Exec("DELETE FROM requests WHERE timestamp < datetime('now', '-30 days')")
	if err != nil {
		log.Printf("Error cleaning up requests: %v", err)
	}
	_, err = s.db.Exec("DELETE FROM error_logs WHERE timestamp < datetime('now', '-30 days')")
	if err != nil {
		log.Printf("Error cleaning up error logs: %v", err)
	}
	_, err = s.db.Exec("DELETE FROM ping_requests WHERE timestamp < datetime('now', '-30 days')")
	if err != nil {
		log.Printf("Error cleaning up ping requests: %v", err)
	}
}

func (s *LocationStore) LogLocation(lat, lng float64) {
	select {
	case s.locationChan <- locationEntry{lat: lat, lng: lng}:
	default:
		log.Printf("Warning: locationChan full, dropping location log")
	}
}

func (s *LocationStore) GetAllLocations(days int) ([]LocationStats, error) {
	query := `SELECT lat, lng, count FROM locations`
	var args []interface{}

	if days > 0 {
		query = `
			SELECT lat, lng, COUNT(*) as count 
			FROM requests 
			WHERE timestamp >= datetime('now', ?) AND lat IS NOT NULL AND lng IS NOT NULL
			GROUP BY lat, lng`
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []LocationStats
	for rows.Next() {
		var st LocationStats
		if err := rows.Scan(&st.Lat, &st.Lng, &st.Count); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, nil
}

func (s *LocationStore) LogRequest(backend string, statusCode int, errorType string, lat, lng float64, isCacheHit bool) {
	select {
	case s.requestChan <- requestEntry{
		backend:    backend,
		statusCode: statusCode,
		errorType:  errorType,
		lat:        lat,
		lng:        lng,
		isCacheHit: isCacheHit,
	}:
	default:
		log.Printf("Warning: requestChan full, dropping request log")
	}
}

type ErrorLog struct {
	ID               int    `json:"id"`
	Timestamp        string `json:"timestamp"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	Query            string `json:"query"`
	StatusCode       int    `json:"status_code"`
	RequestBody      string `json:"request_body"`
	ResponseBody     string `json:"response_body"`
	UpstreamResponse string `json:"upstream_response"`
	Backend          string `json:"backend"`
	ErrorType        string `json:"error_type"`
}

func (s *LocationStore) LogError(entry ErrorLog) {
	select {
	case s.errorChan <- entry:
	default:
		log.Printf("Warning: errorChan full, dropping error log")
	}
}

func (s *LocationStore) GetErrorLogs(days int) ([]ErrorLog, error) {
	query := `SELECT id, timestamp, method, path, query, status_code, request_body, response_body, upstream_response, backend, error_type FROM error_logs`
	var args []interface{}

	if days > 0 {
		query += " WHERE timestamp >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	query += " ORDER BY timestamp DESC LIMIT 50"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ErrorLog
	for rows.Next() {
		var l ErrorLog
		err := rows.Scan(&l.ID, &l.Timestamp, &l.Method, &l.Path, &l.Query, &l.StatusCode, &l.RequestBody, &l.ResponseBody, &l.UpstreamResponse, &l.Backend, &l.ErrorType)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (s *LocationStore) GetBackendStats(days int) ([]BackendStats, error) {
	query := `
		SELECT 
			backend, 
			SUM(CASE WHEN status_code < 400 THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as failed,
			SUM(CASE WHEN status_code < 400 AND is_cache_hit = 1 THEN 1 ELSE 0 END) as cache_success,
			SUM(CASE WHEN status_code >= 400 AND is_cache_hit = 1 THEN 1 ELSE 0 END) as cache_failed,
			COUNT(DISTINCT lat || ',' || lng) as locations
		FROM requests`
	var args []interface{}

	if days > 0 {
		query += " WHERE timestamp >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	query += " GROUP BY backend"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []BackendStats
	for rows.Next() {
		var st BackendStats
		if err := rows.Scan(&st.Backend, &st.Success, &st.Failed, &st.CacheSuccess, &st.CacheFailed, &st.Locations); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, nil
}

func (s *LocationStore) GetFailureReasons(days int) ([]FailureReason, error) {
	query := `
		SELECT error_type, COUNT(*) as count
		FROM requests
		WHERE status_code >= 400 AND error_type IS NOT NULL AND error_type != ''`
	var args []interface{}

	if days > 0 {
		query += " AND timestamp >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	query += `
		GROUP BY error_type
		ORDER BY count DESC
		LIMIT 10`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reasons []FailureReason
	for rows.Next() {
		var r FailureReason
		if err := rows.Scan(&r.Reason, &r.Count); err != nil {
			return nil, err
		}
		reasons = append(reasons, r)
	}
	return reasons, nil
}

type FailureTrendStats struct {
	Bucket string `json:"bucket"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// GetFailureTrend returns failure counts per error type over time, bucketed the
// same way as GetUsageStats. Only the most frequent error types are reported
// individually; the remainder is aggregated into an "Other" series so the chart
// legend stays readable.
func (s *LocationStore) GetFailureTrend(days int) ([]FailureTrendStats, error) {
	var bucketExpr, timeFilter string
	var args []interface{}

	if days <= 1 && days != 0 {
		// 24 hours, 5 minute buckets
		bucketExpr = "strftime('%Y-%m-%d %H:%M', datetime((strftime('%s', timestamp) / 300) * 300, 'unixepoch'))"
		timeFilter = " AND timestamp >= datetime('now', '-24 hours')"
	} else if days <= 7 && days != 0 {
		// 7 days, 1 hour buckets
		bucketExpr = "strftime('%Y-%m-%d %H:00', timestamp)"
		timeFilter = " AND timestamp >= datetime('now', '-7 days')"
	} else {
		// 30 days or All Time, 1 day buckets
		bucketExpr = "strftime('%Y-%m-%d', timestamp)"
		if days > 0 {
			timeFilter = " AND timestamp >= datetime('now', ?)"
			args = append(args, fmt.Sprintf("-%d days", days))
		}
	}

	query := `
		WITH failures AS (
			SELECT ` + bucketExpr + ` as bucket, error_type
			FROM requests
			WHERE status_code >= 400 AND error_type IS NOT NULL AND error_type != ''` + timeFilter + `
		),
		top_reasons AS (
			SELECT error_type
			FROM failures
			GROUP BY error_type
			ORDER BY COUNT(*) DESC
			LIMIT 10
		)
		SELECT
			bucket,
			CASE WHEN error_type IN (SELECT error_type FROM top_reasons) THEN error_type ELSE 'Other' END as reason,
			COUNT(*) as count
		FROM failures
		GROUP BY bucket, reason
		ORDER BY bucket`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []FailureTrendStats
	for rows.Next() {
		var st FailureTrendStats
		if err := rows.Scan(&st.Bucket, &st.Reason, &st.Count); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, nil
}

func (s *LocationStore) GetUsageStats(days int) ([]UsageStats, error) {
	var query string
	var args []interface{}

	if days <= 1 && days != 0 {
		// 24 hours, 5 minute buckets
		query = `
			SELECT 
				strftime('%Y-%m-%d %H:%M', datetime((strftime('%s', timestamp) / 300) * 300, 'unixepoch')) as bucket,
				COUNT(*) as count
			FROM requests
			WHERE timestamp >= datetime('now', '-24 hours')
			GROUP BY bucket
			ORDER BY bucket`
	} else if days <= 7 && days != 0 {
		// 7 days, 1 hour buckets
		query = `
			SELECT 
				strftime('%Y-%m-%d %H:00', timestamp) as bucket,
				COUNT(*) as count
			FROM requests
			WHERE timestamp >= datetime('now', '-7 days')
			GROUP BY bucket
			ORDER BY bucket`
	} else {
		// 30 days or All Time, 1 day buckets
		query = `
			SELECT 
				strftime('%Y-%m-%d', timestamp) as bucket,
				COUNT(*) as count
			FROM requests`
		if days > 0 {
			query += " WHERE timestamp >= datetime('now', '-" + fmt.Sprintf("%d", days) + " days')"
		}
		query += " GROUP BY bucket ORDER BY bucket"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []UsageStats
	for rows.Next() {
		var st UsageStats
		if err := rows.Scan(&st.Bucket, &st.Count); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, nil
}

func (s *LocationStore) LogPing(uuid, version string) {
	select {
	case s.pingChan <- pingEntry{UUID: uuid, Version: version}:
	default:
		log.Printf("Warning: pingChan full, dropping ping log")
	}
}

type VersionCount struct {
	Version string `json:"version"`
	Count   int    `json:"count"`
}

type UsersPerVersion struct {
	Last24h []VersionCount `json:"last_24h"`
	Last7d  []VersionCount `json:"last_7d"`
	Last30d []VersionCount `json:"last_30d"`
}

func (s *LocationStore) GetUsersPerVersion(days int) ([]VersionCount, error) {
	query := `
		SELECT version, COUNT(*) as count 
		FROM user_versions`
	var args []interface{}

	if days > 0 {
		query += " WHERE last_seen >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	query += `
		GROUP BY version
		ORDER BY count DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []VersionCount
	for rows.Next() {
		var vc VersionCount
		if err := rows.Scan(&vc.Version, &vc.Count); err != nil {
			return nil, err
		}
		counts = append(counts, vc)
	}
	return counts, nil
}

type PingUsageStats struct {
	Bucket  string `json:"bucket"`
	Version string `json:"version"`
	Count   int    `json:"count"`
}

func (s *LocationStore) GetPingUsageStats(days int) ([]PingUsageStats, error) {
	var query string
	var args []interface{}

	if days <= 1 && days != 0 {
		// 24 hours, 5 minute buckets
		query = `
			SELECT 
				strftime('%Y-%m-%d %H:%M', datetime((strftime('%s', timestamp) / 300) * 300, 'unixepoch')) as bucket,
				version,
				COUNT(*) as count
			FROM ping_requests
			WHERE timestamp >= datetime('now', '-24 hours')
			GROUP BY bucket, version
			ORDER BY bucket`
	} else if days <= 7 && days != 0 {
		// 7 days, 1 hour buckets
		query = `
			SELECT 
				strftime('%Y-%m-%d %H:00', timestamp) as bucket,
				version,
				COUNT(*) as count
			FROM ping_requests
			WHERE timestamp >= datetime('now', '-7 days')
			GROUP BY bucket, version
			ORDER BY bucket`
	} else {
		// 30 days or All Time, 1 day buckets
		query = `
			SELECT 
				strftime('%Y-%m-%d', timestamp) as bucket,
				version,
				COUNT(*) as count
			FROM ping_requests`
		if days > 0 {
			query += " WHERE timestamp >= datetime('now', '-" + fmt.Sprintf("%d", days) + " days')"
		}
		query += " GROUP BY bucket, version ORDER BY bucket"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PingUsageStats
	for rows.Next() {
		var st PingUsageStats
		if err := rows.Scan(&st.Bucket, &st.Version, &st.Count); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, nil
}
