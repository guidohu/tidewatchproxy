package store

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
	"tide_watch_proxy/pkg/util"
)

type LocationStore struct {
	db *sql.DB
}

type LocationStats struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Count int     `json:"count"`
}

type BackendStats struct {
	Backend string `json:"backend"`
	Success int    `json:"success"`
	Failed  int    `json:"failed"`
}

type FailureReason struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

func NewLocationStore(dbPath string) (*LocationStore, error) {
	// Add busy timeout to handle concurrent writes
	dsn := fmt.Sprintf("%s?_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// For SQLite, a single connection is often better to avoid "database is locked" errors,
	// especially when multiple goroutines are performing writes.
	db.SetMaxOpenConns(1)

	// Enable WAL mode for better concurrency
	_, _ = db.Exec("PRAGMA journal_mode=WAL")

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
			lng REAL
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Migrations: Add columns if they don't exist (ignore errors if they already do)
	_, _ = db.Exec("ALTER TABLE requests ADD COLUMN timestamp DATETIME DEFAULT CURRENT_TIMESTAMP")
	_, _ = db.Exec("ALTER TABLE requests ADD COLUMN lat REAL")
	_, _ = db.Exec("ALTER TABLE requests ADD COLUMN lng REAL")

	// Ensure existing rows have a timestamp if it was NULL
	_, _ = db.Exec("UPDATE requests SET timestamp = CURRENT_TIMESTAMP WHERE timestamp IS NULL")

	return &LocationStore{db: db}, nil
}

func (s *LocationStore) LogLocation(lat, lng float64) {
	// Quantize to ~1.1km resolution (2 decimal places)
	qLat := util.Round(lat, 2)
	qLng := util.Round(lng, 2)

	// Upsert
	_, err := s.db.Exec(`
		INSERT INTO locations (lat, lng, count)
		VALUES (?, ?, 1)
		ON CONFLICT(lat, lng) DO UPDATE SET count = count + 1
	`, qLat, qLng)

	if err != nil {
		log.Printf("Error logging location: %v", err)
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

func (s *LocationStore) LogRequest(backend string, statusCode int, errorType string, lat, lng float64) {
	var latVal, lngVal interface{}
	if lat != 0 || lng != 0 {
		latVal = util.Round(lat, 2)
		lngVal = util.Round(lng, 2)
	}

	_, err := s.db.Exec(`
		INSERT INTO requests (backend, status_code, error_type, lat, lng)
		VALUES (?, ?, ?, ?, ?)
	`, backend, statusCode, errorType, latVal, lngVal)
	if err != nil {
		log.Printf("Error logging request: %v", err)
	}
}

func (s *LocationStore) GetBackendStats(days int) ([]BackendStats, error) {
	query := `
		SELECT 
			backend, 
			SUM(CASE WHEN status_code < 400 THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as failed
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
		if err := rows.Scan(&st.Backend, &st.Success, &st.Failed); err != nil {
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
