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

func NewLocationStore(dbPath string) (*LocationStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS locations (
			lat REAL,
			lng REAL,
			count INTEGER,
			PRIMARY KEY (lat, lng)
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

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

func (s *LocationStore) GetAllLocations() ([]LocationStats, error) {
	rows, err := s.db.Query(`SELECT lat, lng, count FROM locations`)
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
