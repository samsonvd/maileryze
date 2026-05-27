package db

import (
	"database/sql"
	"fmt"
)

type EmailMatch struct {
	SenderName    string
	SenderAddress string
	Subject       string
}

func FindEmails(db *sql.DB, alias, query string, unsubscribeOnly bool) ([]EmailMatch, error) {
	unsubscribeFilter := ""
	if unsubscribeOnly {
		unsubscribeFilter = "AND (unsubscribe_email != '' OR unsubscribe_url != '')"
	}

	pattern := "%" + query + "%"
	rows, err := db.Query(fmt.Sprintf(`
		SELECT sender_name, sender_address, subject
		FROM emails
		WHERE alias = ?
		  AND (sender_name LIKE ? OR sender_address LIKE ? OR subject LIKE ?)
		  %s
		ORDER BY sender_address, received_at DESC`, unsubscribeFilter),
		alias, pattern, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("querying emails: %w", err)
	}
	defer rows.Close()

	var matches []EmailMatch
	for rows.Next() {
		var m EmailMatch
		if err := rows.Scan(&m.SenderName, &m.SenderAddress, &m.Subject); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}
