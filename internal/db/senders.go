package db

import (
	"database/sql"
	"fmt"
)

type Sender struct {
	Name             string
	Address          string
	Count            int
	Subjects         []string
	UnsubscribeURL   string
	UnsubscribeEmail string
}

// GetSenders returns senders for an alias sorted by email count descending.
// When unsubscribeOnly is true only senders with a List-Unsubscribe header are returned.
// Subjects are loaded per sender via a follow-up query.
func GetSenders(db *sql.DB, alias string, min int, unsubscribeOnly bool) ([]Sender, error) {
	unsubscribeFilter := ""
	if unsubscribeOnly {
		unsubscribeFilter = "AND (unsubscribe_email != '' OR unsubscribe_url != '')"
	}

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			sender_name,
			sender_address,
			COUNT(*) as count,
			COALESCE(MAX(CASE WHEN unsubscribe_url   != '' THEN unsubscribe_url   END), '') as unsub_url,
			COALESCE(MAX(CASE WHEN unsubscribe_email != '' THEN unsubscribe_email END), '') as unsub_email
		FROM emails
		WHERE alias = ? %s
		GROUP BY sender_address
		HAVING count >= ?
		ORDER BY count DESC`, unsubscribeFilter),
		alias, min)
	if err != nil {
		return nil, fmt.Errorf("querying senders: %w", err)
	}
	defer rows.Close()

	var senders []Sender
	for rows.Next() {
		var s Sender
		if err := rows.Scan(&s.Name, &s.Address, &s.Count, &s.UnsubscribeURL, &s.UnsubscribeEmail); err != nil {
			return nil, err
		}
		senders = append(senders, s)
	}

	for i, s := range senders {
		subjects, err := GetSenderSubjects(db, alias, s.Address)
		if err != nil {
			return nil, err
		}
		senders[i].Subjects = subjects
	}

	return senders, nil
}

// GetSendersBasic returns senders without loading subjects — faster for the TUI
// which loads subjects lazily in the detail view.
func GetSendersBasic(db *sql.DB, alias string) ([]Sender, error) {
	rows, err := db.Query(`
		SELECT
			sender_name,
			sender_address,
			COUNT(*) as count,
			COALESCE(MAX(CASE WHEN unsubscribe_url   != '' THEN unsubscribe_url   END), '') as unsub_url,
			COALESCE(MAX(CASE WHEN unsubscribe_email != '' THEN unsubscribe_email END), '') as unsub_email
		FROM emails
		WHERE alias = ?
		GROUP BY sender_address
		ORDER BY count DESC`,
		alias)
	if err != nil {
		return nil, fmt.Errorf("querying senders: %w", err)
	}
	defer rows.Close()

	var senders []Sender
	for rows.Next() {
		var s Sender
		if err := rows.Scan(&s.Name, &s.Address, &s.Count, &s.UnsubscribeURL, &s.UnsubscribeEmail); err != nil {
			return nil, err
		}
		senders = append(senders, s)
	}
	return senders, nil
}

// GetSenderSubjects returns the distinct subject lines for a sender.
func GetSenderSubjects(db *sql.DB, alias, address string) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT subject
		FROM emails
		WHERE alias = ? AND sender_address = ?
		ORDER BY subject`,
		alias, address)
	if err != nil {
		return nil, fmt.Errorf("querying subjects for %s: %w", address, err)
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, err
		}
		subjects = append(subjects, subject)
	}
	return subjects, nil
}
