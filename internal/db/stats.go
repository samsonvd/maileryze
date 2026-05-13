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
		var oldest, newest, lastFetch sql.NullTime
		if err := rows.Scan(&alias, &provider, &count, &oldest, &newest, &lastFetch); err != nil {
			return nil, err
		}
		s := AliasStats{Provider: provider, Count: count}
		if oldest.Valid {
			s.OldestEmail = oldest.Time
		}
		if newest.Valid {
			s.NewestEmail = newest.Time
		}
		if lastFetch.Valid {
			s.LastFetchedAt = lastFetch.Time
		}
		stats.Aliases[alias] = s
	}

	return stats, nil
}
