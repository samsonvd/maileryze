package db

import (
	"database/sql"
	"fmt"
	"time"
)

type AliasStats struct {
	Provider      string
	Count         int
	OldestEmail   time.Time
	NewestEmail   time.Time
	LastFetchedAt time.Time
}

type Stats struct {
	Aliases map[string]AliasStats
}

func GetStats(db *sql.DB) (*Stats, error) {
	stats := &Stats{Aliases: make(map[string]AliasStats)}

	rows, err := db.Query(`
		SELECT alias, provider, COUNT(*), MIN(received_at), MAX(received_at), MAX(fetched_at)
		FROM emails
		GROUP BY alias
		ORDER BY alias`)
	if err != nil {
		return nil, fmt.Errorf("querying stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var alias, provider string
		var count int
		var oldest, newest, lastFetch sql.NullString
		if err := rows.Scan(&alias, &provider, &count, &oldest, &newest, &lastFetch); err != nil {
			return nil, err
		}
		s := AliasStats{Provider: provider, Count: count}
		if oldest.Valid {
			s.OldestEmail, _ = parseTime(oldest.String)
		}
		if newest.Valid {
			s.NewestEmail, _ = parseTime(newest.String)
		}
		if lastFetch.Valid {
			s.LastFetchedAt, _ = parseTime(lastFetch.String)
		}
		stats.Aliases[alias] = s
	}

	return stats, nil
}

var timeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
}

func parseTime(s string) (time.Time, error) {
	for _, f := range timeFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %q", s)
}
