package db

import (
	"database/sql"
	"fmt"
)

type Sender struct {
	Name     string
	Address  string
	Count    int
	Subjects []string
}

func GetSenders(db *sql.DB, alias string, min int, unsubscribeOnly bool) ([]Sender, error) {
	unsubscribeFilter := ""
	if unsubscribeOnly {
		unsubscribeFilter = "AND (unsubscribe_email != '' OR unsubscribe_url != '')"
	}

	rows, err := db.Query(fmt.Sprintf(`
		SELECT sender_name, sender_address, COUNT(*) as count
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
		if err := rows.Scan(&s.Name, &s.Address, &s.Count); err != nil {
			return nil, err
		}
		senders = append(senders, s)
	}

	for i, s := range senders {
		subjects, err := getSenderSubjects(db, alias, s.Address)
		if err != nil {
			return nil, err
		}
		senders[i].Subjects = subjects
	}

	return senders, nil
}

func getSenderSubjects(db *sql.DB, alias, address string) ([]string, error) {
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
